/*
 *
 * Copyright 2015, Google Inc.
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *     * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *     * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *     * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
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
	conn, err := grpc.Dial(*address, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
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
				start := time.Now()
				reply, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
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
