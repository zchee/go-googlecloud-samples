// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Sample grpc-ping acts as an intermediary to the ping service.
package main

import (
	"context"
	"net"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	zapcloudlogging "github.com/zchee/zap-cloudlogging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	pb "github.com/zchee/go-googlecloud-samples/run/grpc-ping/pkg/api/v1"
)

// conn holds an open connection to the ping service.
var conn *grpc.ClientConn

var logger = zap.New(zapcloudlogging.NewCore(zapcore.Lock(os.Stdout), zap.NewAtomicLevelAt(zapcore.DebugLevel).Level()))

func init() {
	ctx := context.Background()
	ctx = zapcloudlogging.NewContext(ctx, logger)
	if os.Getenv("GRPC_PING_HOST") != "" {
		var err error
		conn, err = NewConn(ctx, os.Getenv("GRPC_PING_HOST"), os.Getenv("GRPC_PING_INSECURE") != "")
		if err != nil {
			logger.Fatal("failed to NewConn", zap.Error(err))
		}
	} else {
		logger.Info("Starting without support for SendUpstream: configure with 'GRPC_PING_HOST' environment variable. E.g., example.com:443")
	}
}

const (
	// Project ID of the project the Cloud Run service or job belongs to
	projectProjectID = "project/project-id"

	// Project number of the project the Cloud Run service or job belongs to
	projectNumericProjectID = "project/numeric-project-id"

	// Region of this Cloud Run service or job, returns projects/PROJECT-NUMBER/regions/REGION
	instanceRegion = "instance/region"

	// Unique identifier of the container instance (also available in logs).
	instanceID = "instance/id"

	// Email for the runtime service account of this Cloud Run service or job.
	instanceSADefaultEmail = "instance/service-accounts/default/email"

	// Generates an OAuth2 access token for the service account of this Cloud Run service or job. The Cloud Run service agent is used to fetch a token. This endpoint will return a JSON response with an access_token attribute. Read more about how to extract and use this access token.
	instanceSADefaultToken = "instance/service-accounts/default/token"
)

func fetchMetadata(mdc *metadata.Client, logger *zap.Logger) {
	projectID, err := mdc.Get(projectProjectID)
	if err != nil {
		logger.Fatal("could not get project id from /computeMetadata/v1/project/project-id endpoint", zap.Error(err))
	}

	numericProjectID, err := mdc.Get(projectNumericProjectID)
	if err != nil {
		logger.Fatal("could not get numeric project id from /computeMetadata/v1/project/numeric-project-id endpoint", zap.Error(err))
	}

	region, err := mdc.Get(instanceRegion)
	if err != nil {
		logger.Fatal("could not get project id from /computeMetadata/v1/project/project-id endpoint", zap.Error(err))
	}

	id, err := mdc.Get(instanceID)
	if err != nil {
		logger.Fatal("could not get project id from /computeMetadata/v1/project/project-id endpoint", zap.Error(err))
	}

	saDefaultEmail, err := mdc.Get(instanceSADefaultEmail)
	if err != nil {
		logger.Fatal("could not get project id from /computeMetadata/v1/project/project-id endpoint", zap.Error(err))
	}

	saDefaultToken, err := mdc.Get(instanceSADefaultToken)
	if err != nil {
		logger.Fatal("could not get project id from /computeMetadata/v1/project/project-id endpoint", zap.Error(err))
	}

	logger.Info("metadata",
		zap.String(projectProjectID, projectID),
		zap.String(projectNumericProjectID, numericProjectID),
		zap.String(instanceRegion, region),
		zap.String(instanceID, id),
		zap.String(instanceSADefaultEmail, saDefaultEmail),
		zap.String(instanceSADefaultToken, saDefaultToken),
	)
}

func main() {
	logger.Info("grpc-ping: starting server...")

	logger.Info("get metadata from metadata server")
	mdc := metadata.NewClient(http.DefaultClient)
	fetchMetadata(mdc, logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		logger.Info("Defaulting to port", zap.String("port", port))
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatal("net.Listen", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = zapcloudlogging.NewContext(ctx, logger)

	gsrv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		UnaryServerInterceptor(logger)),
	)
	pb.RegisterPingServiceServer(gsrv, &pingService{})
	if err = gsrv.Serve(listener); err != nil {
		logger.Fatal("could not serve", zap.Error(err))
	}
}
