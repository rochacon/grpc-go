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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "world"
)

func main() {
	address := flag.String("address", "localhost:50051", "gRPC server endpoint.")
	concurrency := flag.Int("concurrency", 3, "Number of concurrent workers.")
	sleep := flag.Duration("sleep", time.Duration(0), "Duration to sleep between calls.")
	flag.Parse()

	// metrics server
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Metric server listening on port :9090")
	go http.ListenAndServe(":9090", nil)

	// Set up a connection to the server.
	log.Println("Connecting to gRPC server", *address)
	conn, err := grpc.Dial(*address, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if flag.NArg() > 1 {
		name = strings.Join(flag.Args(), " ")
	}
	log.Println("Name:", name)

	log.Println("Launching", *concurrency, "workers")
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
					log.Println("Sleeping for", sleep, "seconds")
					time.Sleep(sleep)
				}
			}
		}(wg, *sleep)
	}
	wg.Wait()
	log.Println("Shutting down")
}
