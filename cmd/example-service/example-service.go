package main

import (
	"context"
	"expvar"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rumsrami/example-service/cmd/example-service/internal/handlers"
	"github.com/rumsrami/example-service/internal/db"
	"github.com/rumsrami/example-service/internal/platform/broker"
	"github.com/rumsrami/example-service/internal/platform/web"
)

const (
	// environment variables
	appEnv       = "APP_ENV"
	tableName    = "TABLE_NAME"
	slackWebHook = "SLACK_WEBHOOK"
	serviceName  = "CHAT"
	portENV      = "PORT"
	// environment variables values
	productionEnvironment = "production"
	stagingEnvironment    = "staging"
	localEnvironment      = "local"
	expvarAppBuildKey     = "build"
	// initialization errors
	errGeneratingConfigUsage   = "generating config usage"
	errGeneratingConfigVersion = "generating config version"
	errParsingConfiguration    = "parsing config"
	errGeneratingConfig        = "generating config for output"
	errDatabaseConnection      = "Error connecting to db"
	errInternalServer          = "internal server error"
	errShutdown                = "integrity issue caused shutdown"
	errGracefulShutdown        = "could not stop server gracefully"
	errNatsServer              = "nats server error"
	errNatsBroker              = "nats broker error"
	errAWSSession              = "aws session error"
	errDynamoDb                = "aws dynamodb unknown error"
	errGoProcesses             = "error running go process"
)

func start() error {
	// =========================================================================
	// Checks if the env variable "APP_ENV" is set
	// in production and staging this is obtained from app.yml & app_st.yml
	// this is set to either staging or production to control logging format
	// For local development it is set to "local" where logging is only to Std out
	if env := os.Getenv(appEnv); env != "" {
		environment = env
	}

	// =========================================================================
	// StdOut logging
	noColor := true

	if environment == localEnvironment {
		noColor = false
	}

	textLogger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: noColor})
	stOutLogger := textLogger.With().Str("service", serviceName).Str("build", build).Logger()

	stOutLogger.Info().Msgf(fmt.Sprintf("main : Current Environment : %s", environment))

	// =========================================================================
	// Configuration
	var cfg struct {
		conf.Version
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:9000"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
		}
		ZAuth struct {
			// used with the authentication middleware
			// to verify the jwt token
			Authority string `conf:"default:https://localhost.auth0.com/"`
			Audience  string `conf:"default:http://localhost:9000"`
		}
	}
	cfg.Version.SVN = build
	cfg.Version.Desc = "copyright information here"

	if err := conf.Parse(os.Args[1:], serviceName, &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage(serviceName, &cfg)
			if err != nil {
				return errors.Wrap(err, errGeneratingConfigUsage)
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString(serviceName, &cfg)
			if err != nil {
				return errors.Wrap(err, errGeneratingConfigVersion)
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, errParsingConfiguration)
	}

	// Update the port if running in Heroku
	if port := os.Getenv(portENV); port != "" {
		cfg.Web.APIHost = "0.0.0.0:" + port
	}

	// =========================================================================
	// App Starting

	// Print the build version for our logs. Also expose it under /debug/vars.
	expvar.NewString(expvarAppBuildKey).Set(build)
	stOutLogger.Info().Msgf(fmt.Sprintf("main : Initializing : Application version %q", build))

	defer func() {
		stOutLogger.Info().Msgf("main : Completed")
	}()

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, errGeneratingConfig)
	}
	stOutLogger.Info().Msgf(fmt.Sprintf("main : Config :\n%v\n", out))

	// =========================================================================
	// Start a run group and add services one by one
	var g run.Group

	// =========================================================================
	// =========================================================================
	// Start the HTTP Server
	// =========================================================================
	// =========================================================================

	stOutLogger.Info().Msgf("main : Initializing : HTTP Server support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// create a new mux
	app := web.NewApp(shutdown)

	// create a new http server with the mux as handler
	httpServer := http.Server{
		Handler: app,
	}

	// channel to listen for server errors to trigger shutdown
	serverErrors := make(chan error, 1)

	{
		ln, _ := net.Listen("tcp", cfg.Web.APIHost)
		g.Add(func() error {
			return httpServer.Serve(ln)
		}, func(error) {
			shutdown <- os.Interrupt
		})
	}

	stOutLogger.Info().Msgf(fmt.Sprintf("main : Started : HTTP Server listening on %s", cfg.Web.APIHost))

	// =========================================================================
	// =========================================================================
	// Start Debug Service
	// =========================================================================
	// =========================================================================
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.

	if environment == localEnvironment {
		stOutLogger.Info().Msgf("main : Initializing : Debugging support")

		{
			ln, _ := net.Listen("tcp", cfg.Web.DebugHost)
			g.Add(func() error {
				return http.Serve(ln, http.DefaultServeMux)
			}, func(error) {
				_ = ln.Close()
			})
		}

		stOutLogger.Info().Msgf(fmt.Sprintf("main : Started : Debuging Listening %s", cfg.Web.DebugHost))
	}

	errGroup := make(chan error)

	go func() {
		errGroup <- g.Run()
	}()

	// =========================================================================
	// Start AWS Session
	// -> Commented out as it needs AWS credentials in Environmen

	// sess, err := session.NewSession()
	// if err != nil {
	// 	return errors.Wrap(err, errAWSSession)
	// }

	//=========================================================================
	// Start Nats Client

	// create the message broker running on nats
	stOutLogger.Info().Msgf("main : Initializing : NATS Client support")

	natsClient, err := broker.NewClient(natsServiceName)
	if err != nil {
		return errors.Wrap(err, errNatsBroker)
	}

	stOutLogger.Info().Msgf("main : Started : NATS Client support")

	// =========================================================================
	// Start Database

	stOutLogger.Info().Msgf("main : Initializing : Database support")

	// In-memory database
	database := db.NewDatabase()
	{
		g.Add(func() error {
			return database.Run()
		}, func(error) {
			database.Stop()
		})
	}

	/*
		 ---> Dynamo db start commented out as it needs credentials

		if tableNameFromEnv := os.Getenv(tableName); tableNameFromEnv != "" {
			dbTableName = tableNameFromEnv
		}

		// Create aws services here
		dyn := dynamodb.New(sess)

		input := &dynamodb.DescribeTableInput{
			TableName: aws.String(dbTableName),
		}

		_, err = dyn.DescribeTable(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case dynamodb.ErrCodeResourceNotFoundException:
					return errors.Wrap(aerr, dynamodb.ErrCodeResourceNotFoundException)
				case dynamodb.ErrCodeInternalServerError:
					return errors.Wrap(aerr, dynamodb.ErrCodeInternalServerError)
				default:
					return errors.Wrap(aerr, errDynamoDb)
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				stOutLogger.Error().Msgf("main : Dynamodb table error", err.Error())
			}
		}


	*/

	stOutLogger.Info().Msgf("main : Started : Database support")

	// =========================================================================
	// Start Routing Service

	stOutLogger.Info().Msgf("main : Initializing : Routing support")

	handlers.Mount(build, database, cfg.ZAuth.Authority, cfg.ZAuth.Audience, natsClient, app, stOutLogger)

	stOutLogger.Info().Msgf("main : Started : Routing support")
	stOutLogger.Info().Msgf(fmt.Sprintf("main : Started : Application version %q", build))

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, errInternalServer)

	case sig := <-shutdown:
		if sig == syscall.SIGSTOP {
			return errors.New(errShutdown)
		}

		stOutLogger.Info().Msgf(fmt.Sprintf("main : %v : Start shutdown", sig))

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)

		// Asking listener to shutdown and load shed.
		err := httpServer.Shutdown(ctx)
		if err != nil {
			cancel()
			return errors.Wrap(err, errGracefulShutdown)
		}
		err = <-errGroup
		return err
	}
}
