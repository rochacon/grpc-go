FROM golang:1.15 AS build

WORKDIR /app
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN go build -v -ldflags '--extldflags "--static"' -o /client ./greeter_client
RUN go build -v -ldflags '--extldflags "--static"' -o /server ./greeter_server

FROM alpine:3.11
RUN apk add --no-cache ca-certificates
COPY --from=build /client /client
COPY --from=build /server /server
