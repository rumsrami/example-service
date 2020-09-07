"use strict";
const pulumi = require("@pulumi/pulumi");
const aws = require("@pulumi/aws");
const awsx = require("@pulumi/awsx");

// STEP - 1

// Allocate a new VPC with a smaller CIDR range:
const vpc = new awsx.ec2.Vpc("vpc", {
    cidrBlock: "192.16.0.0/22",
});

// security group for the cluster
const clusterSecurityGroup = new awsx.ec2.SecurityGroup("ecs-core-cluster", { vpc });

// outbound TCP traffic on any port to anywhere
clusterSecurityGroup.createEgressRule("sg-egress-alltraffic-ecs-core-cluster", {
    location: new awsx.ec2.AnyIPv4Location(),
    ports: new awsx.ec2.AllTraffic(),
    description: "allow outbound access to anywhere",
});

// Create an ECS cluster explicitly, and give it a name tag.
const appCoreCluster = new awsx.ecs.Cluster("ecs-core-cluster", {
    securityGroups: [clusterSecurityGroup],
    vpc,
    tags: {
        "Name": "ecs-core-cluster",
    },
    settings: [
        {
            name: "containerInsights",
            value: "enabled"
        }
    ],
});

// Create ALB Security Group
const albSecurityGroup = new awsx.ec2.SecurityGroup("sg-alb-ecs-core", { vpc });

// outbound TCP traffic on any port to anywhere
albSecurityGroup.createEgressRule("sg-egress-alb-all", {
    location: new awsx.ec2.AnyIPv4Location(),
    ports: new awsx.ec2.AllTraffic(),
    description: "allow outbound access to anywhere",
});

// Create ELB
const alb = new awsx.lb.ApplicationLoadBalancer("alb-ecs-core", {
    vpc: vpc,
    idleTimeout: 4000,
    subnets: vpc.publicSubnetIds,
    enableCrossZoneLoadBalancing: true,
    securityGroups: [albSecurityGroup.id]
});

// targetGroup
const targetGroup = alb.createTargetGroup("tg-alb-http", {
    vpc: vpc,
    loadBalancer: alb,
    port: 80,
    protocol: "HTTP",
    healthCheck: {
        path: "/_ah/warmup",
        protocol: "HTTP",
    },
})

// STEP - 2

// creat alb http listener
const listener = alb.createListener("ls-alb-https", {
    port: 443,
    vpc: vpc,
    protocol: "HTTPS",
    sslPolicy: "ELBSecurityPolicy-TLS-1-2-2017-01",
    certificateArn: "arn:aws:acm:us-east-1:07900:certificate",
    defaultActions: [
        {
            type: "forward",
            targetGroupArn: targetGroup.targetGroup.arn,
        }
    ],
});

// STEP 3

//Get iam role
const ecsTaskRole = pulumi.output(aws.iam.getRole({
    name: "ecsTaskExecutionRole",
}, { async: true }));

// Create Task Definition
const appCoreTask = new awsx.ecs.FargateTaskDefinition("ecs-core-task", {
    vpc: vpc,
    taskRole: ecsTaskRole,
    executionRole: ecsTaskRole,
    cpu: 256,
    memory: 512,
    containers: {
        core: {
            image: "079002453034.dkr.ecr.us-east-1.amazonaws.com",
            environment: [
                {
                    "name": "AUTH_AUDIENCE",
                    "value": "<auth-domain>"
                },
                {
                    "name": "APP_ENV",
                    "value": "staging"
                }
            ],
            logConfiguration: {
                logDriver: "awslogs",
                options: {
                  "awslogs-group": "ecs-cluster-staging",
                  "awslogs-region": "us-east-1",
                  "awslogs-stream-prefix": "core"
                }
            },
            privileged: false,
            portMappings: [
                {
                    "protocol": "tcp",
                    "containerPort": 9000
                }
            ]
        }
    }
})

// Create Task Definition
const appNatsTask = new awsx.ecs.FargateTaskDefinition("ecs-nats-task", {
    vpc: vpc,
    taskRole: ecsTaskRole,
    executionRole: ecsTaskRole,
    cpu: 256,
    memory: 512,
    containers: {
        nats: {
            image: "nats:alpine",
            privileged: false,
            portMappings: [
                {
                    "protocol": "tcp",
                    "containerPort": 4222
                },
                {
                    "protocol": "tcp",
                    "containerPort": 8222
                },
                {
                    "protocol": "tcp",
                    "containerPort": 6222
                }
            ],
            logConfiguration: {
                logDriver: "awslogs",
                options: {
                  "awslogs-group": "ecs-cluster-staging",
                  "awslogs-region": "us-east-1",
                  "awslogs-stream-prefix": "nats"
                }
            }
        }
    }
})

// add ingress rule for the cluster security group
// --> Note the [0] dangerous
clusterSecurityGroup.createIngressRule("sg-ingress-ecs-core-only-alb", {
    location: { sourceSecurityGroupId: alb.securityGroups[0].id },
    ports: new awsx.ec2.AllTcpPorts(),
    description: "allow app-elb access",
});

clusterSecurityGroup.createIngressRule("sg-ingress-ecs-core-tcp-to-nats", {
    location: { sourceSecurityGroupId: clusterSecurityGroup.id },
    ports: new awsx.ec2.AllTcpPorts(),
    description: "allow app-nats access",
});

// STEP 4

// // Create the Core ECS Service
const ecsService = new awsx.ecs.FargateService("ecs-core-service", {
    cluster: appCoreCluster,
    taskDefinition: appCoreTask,
    desiredCount: 2,
    assignPublicIp: false,
    healthCheckGracePeriodSeconds: 30,
    loadBalancers: [{
        targetGroupArn: targetGroup.targetGroup.arn,
        containerName: "core",
        containerPort: "9000"
    }],
    deploymentMaximumPercent: 200,
    deploymentMinimumHealthyPercent: 100,
})

// Create the service discovery private namespace
const appNS = new aws.servicediscovery.PrivateDnsNamespace("app-namespace", {
    description: "namespace for private app services",
    name: "chatapp.com",
    vpc: vpc.id
})

// Create service discovery
const natsDiscoveryService = new aws.servicediscovery.Service("app-discovery-nats", {
    description: "discovery service for nats in chatapp",
    name: "nats",
    dnsConfig: {
        namespaceId: appNS.id,
        dnsRecords : [{
            ttl: 10,
            type: "A",
        }],
        routingPolicy: "MULTIVALUE"
    },
    healthCheckCustomConfig: {
        failureThreshold: 5,
    },
})

// Create the NATS ECS Service
const ecsNatsService = new awsx.ecs.FargateService("ecs-nats-service", {
    cluster: appCoreCluster,
    taskDefinition: appNatsTask,
    desiredCount: 1,
    assignPublicIp: false,
    deploymentMaximumPercent: 200,
    deploymentMinimumHealthyPercent: 100,
    serviceRegistries: {
        registryArn: natsDiscoveryService.arn
    },
})

// STEP 4
const us_east_1_Table = new aws.dynamodb.Table("dynamodb-prod-us-east-1", {
    hashKey: "pK",
    rangeKey: "sK",
    streamEnabled: true,
    streamViewType: "NEW_AND_OLD_IMAGES",
    billingMode: "PAY_PER_REQUEST",
    attributes: [
        {
            name: "pK",
            type: "S"
        },
        {
            name: "sK",
            type: "S"
        },
        {
            name: "gSI1PK",
            type: "S"
        },
        {
            name: "gSI1SK",
            type: "S"
        },
        {
            name: "gSI2PK",
            type: "S"
        },
        {
            name: "gSI2SK",
            type: "S"
        },
        {
            name: "email",
            type: "S"
        },
        {
            name: "gSI4PK",
            type: "S"
        },
    ],
    replicas: [
        {
            regionName: "ap-southeast-2"
        }
    ],
    globalSecondaryIndexes: [
        {
            name: "INVERTED",
            hashKey: "sK",
            rangeKey: "pK",
            projectionType: "ALL"
        }
    ]
})