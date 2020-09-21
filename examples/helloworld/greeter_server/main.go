/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

// sayHello implements helloworld.GreeterServer.SayHello
func sayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func defaultAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":50051"
}

func main() {
	addr := flag.String("addr", defaultAddr(), "gRPC server port (defaults to :$PORT or :50051)")
	metricsAddr := flag.String("metrics-addr", ":9090", "gRPC metrics server port (defaults to :9090)")
	tlsCrt := flag.String("tls-crt", "", "gRPC TLS Certificate")
	useGRPCGoServe := flag.Bool("use-grpc-go-serve", false, "use grpc-go Serve method, disabling CORS and HTTP compatibility")
	tlsKey := flag.String("tls-key", "", "gRPC TLS Private Key")
	flag.Parse()
	tlsEnabled := *tlsCrt != "" && *tlsKey != ""

	// metrics server setup
	log.Println("listening for metrics at", *metricsAddr)
	grpc_prometheus.EnableHandlingTimeHistogram()
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(*metricsAddr, nil)

	// gRPC server setup
	var grpcServerOptions = []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}
	if tlsEnabled && *useGRPCGoServe {
		log.Println("loading certificates...")
		creds, err := credentials.NewServerTLSFromFile(*tlsCrt, *tlsKey)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		grpcServerOptions = append(grpcServerOptions, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(grpcServerOptions...)
	grpc_prometheus.DefaultServerMetrics.InitializeMetrics(grpcServer)
	pb.RegisterGreeterService(grpcServer, &pb.GreeterService{
		SayHello: sayHello,
	})
	reflection.Register(grpcServer)

	log.Println("listening for gRPC at", *addr)
	if *useGRPCGoServe {
		log.Printf("using grpc.Server.Serve")
		lis, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %s", err)
		}
	} else {
		var err error
		corsH := cors.New(cors.Options{
			AllowCredentials: true,
			AllowedHeaders: []string{
				"Accept",
				"Accept-Encoding",
				"Authorization",
				"Content-Length",
				"Content-Type",
				"XMLHttpRequest",
				"grpc-message",
				"grpc-status",
				"x-grpc-web",
				"x-user-agent",
			},
			AllowedMethods: []string{http.MethodOptions, http.MethodPost},
			AllowedOrigins: []string{"*"},
			ExposedHeaders: []string{"grpc-status", "grpc-message"},
		})
		h1s := &http.Server{
			Addr:    *addr,
			Handler: corsH.Handler(grpcweb.WrapServer(grpcServer)),
		}
		if tlsEnabled {
			log.Printf("using net/http.ListenAndServeTLS")
			err = h1s.ListenAndServeTLS(*tlsCrt, *tlsKey)
		} else {
			log.Printf("using net/http.ListenAndServe plus h2c support")
			h2s := &http2.Server{}
			h1s.Handler = h2c.NewHandler(h1s.Handler, h2s)
			err = h1s.ListenAndServe()
		}
		if err != nil {
			log.Fatalf("failed to serve: %s", err)
		}
	}
	log.Println("Shutting down")
}
