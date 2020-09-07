package rpc

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/rumsrami/example-service/internal/platform/broker"
	"github.com/rumsrami/example-service/internal/proto"
	"github.com/rumsrami/example-service/internal/db"
)

const (
	packageNameKey = "package"
	packageName    = "rpc"
	tableName      = "chat"
	// RPC errors
	dataErr               = "data error"
	brokerErr             = "broker error"
	internalErr           = "internal error"
	reqValidationErr      = "invalid request body"
	chatTopicPrefix       = "users.chat."
	publishChatMessageErr = "cannot publish chat message after creation"
)

// Shutdowner ....
type Shutdowner interface {
	SignalShutdown()
}

// Chat represents an RPC server
type Chat struct {
	app   Shutdowner
	build string
	db    *db.Database
	rlog  zerolog.Logger
	Val   *validator.Validate
	mb    broker.MessageBroker
}

// NewChat ...
func NewChat(app Shutdowner, build string, db *db.Database, appLog zerolog.Logger, val *validator.Validate, mb broker.MessageBroker) *Chat {
	rpcLogger := appLog.With().Str(packageNameKey, packageName).Logger()

	return &Chat{
		app:   app,
		build: build,
		db:    db,
		rlog:  rpcLogger,
		Val:   val,
		mb:    mb,
	}
}

// Ping is a health check that returns an empty message.
func (d *Chat) Ping(ctx context.Context) (bool, error) {
	return true, nil
}

// Version returns service version details
func (d *Chat) Version(ctx context.Context) (*proto.Version, error) {
	return &proto.Version{
		WebrpcVersion: proto.WebRPCVersion(),
		SchemaVersion: proto.WebRPCSchemaVersion(),
		SchemaHash:    proto.WebRPCSchemaHash(),
		AppVersion:    d.build,
	}, nil
}

// CreateChatMessage ...
func (d *Chat) CreateChatMessage(ctx context.Context, req *proto.ChatMessage) (bool, error) {
	// 1 - Add the chat message to dynamo db
	// 2 - publish to topic
	/*
	err = d.mb.Pub(fmt.Sprintf("%s%s", chatTopicPrefix, req.ToEmail), byteMessage)
	if err != nil {
		c.rlog.Err(err).Msg(publishChatMessageErr)
		return false, proto.WrapError(proto.ErrInternal, err, internalErr)
	}
	
	err = d.mb.Pub(fmt.Sprintf("%s%s", chatTopicPrefix, req.FromEmail), byteMessage)
	if err != nil {
		c.rlog.Err(err).Msg(publishChatMessageErr)
		return false, proto.WrapError(proto.ErrInternal, err, internalErr)
	}
	
	*/
	return false, nil
}
