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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	zapcloudlogging "github.com/zchee/zap-cloudlogging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/zchee/go-googlecloud-samples/run/grpc-ping/pkg/api/v1"
)

type pingService struct {
	pb.UnimplementedPingServiceServer
}

func (s *pingService) Send(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	logger := zapcloudlogging.FromContext(ctx)

	logger.Info("sending ping response")

	return &pb.Response{
		Pong: &pb.Pong{
			Index:      1,
			Message:    req.GetMessage(),
			ReceivedOn: timestamppb.Now(),
		},
	}, nil
}

func (s *pingService) SendUpstream(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	logger := zapcloudlogging.FromContext(ctx)

	if conn == nil {
		return nil, fmt.Errorf("no upstream connection configured")
	}

	p := &pb.Request{
		Message: req.GetMessage() + " (relayed)",
	}

	hostWithoutPort := strings.Split(os.Getenv("GRPC_PING_HOST"), ":")[0]
	tokenAudience := "https://" + hostWithoutPort
	resp, err := PingRequest(conn, p, tokenAudience, os.Getenv("GRPC_PING_UNAUTHENTICATED") == "")
	if err != nil {
		logger.Error("PingRequest", zap.Error(err))
		c := status.Code(err)
		return nil, status.Errorf(c, "Could not reach ping service: %s", status.Convert(err).Message())
	}

	logger.Info("received upstream pong")
	return &pb.Response{
		Pong: resp.Pong,
	}, nil
}

// UnaryServerInterceptor is a gRPC server-side interceptor that provides reporting for Unary RPCs.
func UnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Info("gRPC info", zap.Any("req", req), zap.Any("info", info))

		ctx = zapcloudlogging.NewContext(ctx, logger)

		return handler(ctx, req)
	}
}
