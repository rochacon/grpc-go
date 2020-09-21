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
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "gRPC server endpoint.")
	concurrency := flag.Int("concurrency", 3, "Number of concurrent workers.")
	sleep := flag.Duration("sleep", time.Duration(0), "Duration to sleep between calls.")
	tls := flag.Bool("tls", false, "Use TLS to connect to server")
	flag.Parse()

	// metrics server
	grpc_prometheus.EnableClientHandlingTimeHistogram()
	http.Handle("/metrics", promhttp.Handler())
	log.Println("metrics server listening on port :9090")
	go http.ListenAndServe(":9090", nil)

	// Set up a connection to the server.
	log.Println("connecting to gRPC server", *addr)
	clientOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	if *tls {
		clientOpts = append(clientOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		clientOpts = append(clientOpts, grpc.WithInsecure())
	}
	dialTimeout, dtCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dtCancel()
	conn, err := grpc.DialContext(dialTimeout, *addr, clientOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := "gRPC"
	if flag.NArg() > 1 {
		name = strings.Join(flag.Args(), " ")
	}
	log.Println("name:", name)
	log.Printf("sleeping %s between requests", *sleep)
	log.Println("launching", *concurrency, "workers")
	wg := &sync.WaitGroup{}
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, sleep time.Duration) {
			defer wg.Done()
			for {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				start := time.Now()
				reply, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
				cancel()
				if err != nil {
					log.Println("error on SayHello call:", err)
				} else {
					log.Printf("HelloReply.Message=%q in %s", reply.Message, time.Now().Sub(start))
				}
				if sleep > 0 {
					time.Sleep(sleep)
				}
			}
		}(wg, *sleep)
	}
	wg.Wait()
	log.Println("shutting down")
}
