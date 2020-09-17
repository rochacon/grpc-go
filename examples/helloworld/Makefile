default: client server docker

client:
	GOOS=linux go build -v -ldflags '--extldflags "--static"' -o client ./greeter_client/

server:
	GOOS=linux go build -v -ldflags '--extldflags "--static"' -o server ./greeter_server/

docker: client server
	# docker build -f examples/helloworld/Dockerfile.build -t quay.io/rochacon/grpc-hello-world:build ../../../
	# docker create --name tmp quay.io/rochacon/grpc-hello-world
	# docker cp tmp:/go/bin/greeter_client ./client
	# docker cp tmp:/go/bin/greeter_server ./server
	docker build -t quay.io/rochacon/grpc-hello-world:${VERSION} .

docker_push:
	docker push quay.io/rochacon/grpc-hello-world:${VERSION} 
