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
	"strings"

	"github.com/gorilla/handlers"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// metrics server
	log.Println("listening for metrics at", *metricsAddr)
	grpc_prometheus.EnableHandlingTimeHistogram()
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(*metricsAddr, nil)

	// gRPC server
	var srv *grpc.Server
	var srvOpts = []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}
	if *useGRPCGoServe && *tlsCrt != "" && *tlsKey != "" {
		log.Println("loading certificates...")
		creds, err := credentials.NewServerTLSFromFile(*tlsCrt, *tlsKey)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		srvOpts = append(srvOpts, grpc.Creds(creds))
	}

	srv = grpc.NewServer(srvOpts...)
	grpc_prometheus.DefaultServerMetrics.InitializeMetrics(srv)
	pb.RegisterGreeterService(srv, &pb.GreeterService{
		SayHello: sayHello,
	})
	reflection.Register(srv)

	log.Println("listening for gRPC at", *addr)
	if *useGRPCGoServe {
		log.Printf("using grpc.Server.Serve")
		lis, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		defer lis.Close()
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %s", err)
		}
	} else {
		var err error
		handler := grpcHandlerFunc(srv)
		if *tlsCrt != "" && *tlsKey != "" {
			log.Printf("using net/http.ListenAndServeTLS")
			err = http.ListenAndServeTLS(*addr, *tlsCrt, *tlsKey, handler)
		} else {
			log.Printf("using net/http.ListenAndServe plus h2c support")
			cors := handlers.CORS(
				handlers.AllowCredentials(),
				handlers.AllowedHeaders([]string{
					"Accept",
					"Accept-Encoding",
					"Authorization",
					"Content-Length",
					"Content-Type",
					"X-CSRF-Token",
					"XMLHttpRequest",
					"grpc-message",
					"grpc-status",
					"x-grpc-web",
					"x-user-agent",
				}),
				handlers.AllowedMethods([]string{"DELETE", "GET", "OPTIONS", "POST", "PUT"}),
				handlers.AllowedOrigins([]string{"*"}),
				handlers.ExposedHeaders([]string{"grpc-status", "grpc-message"}),
			)
			handler = cors(h2c.NewHandler(handler, &http2.Server{}))
			err = http.ListenAndServe(*addr, handler)
		}
		if err != nil {
			log.Fatalf("failed to serve: %s", err)
		}
	}
	log.Println("Shutting down")
}

func grpcHandlerFunc(grpcServer *grpc.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			http.DefaultServeMux.ServeHTTP(w, r)
		}
	})
}
