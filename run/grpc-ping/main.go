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
	"os"

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
	if os.Getenv("GRPC_PING_HOST") != "" {
		var err error
		conn, err = NewConn(os.Getenv("GRPC_PING_HOST"), os.Getenv("GRPC_PING_INSECURE") != "")
		if err != nil {
			logger.Fatal("failed to NewConn", zap.Error(err))
		}
	} else {
		logger.Info("Starting without support for SendUpstream: configure with 'GRPC_PING_HOST' environment variable. E.g., example.com:443")
	}
}

func main() {
	logger.Info("grpc-ping: starting server...")

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
		UnaryServerInterceptor(ctx)),
	)
	pb.RegisterPingServiceServer(gsrv, &pingService{})
	if err = gsrv.Serve(listener); err != nil {
		logger.Fatal("could not serve", zap.Error(err))
	}
}
