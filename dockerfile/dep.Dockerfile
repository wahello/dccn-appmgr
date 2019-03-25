FROM golang:1.11.4-alpine as builder

RUN apk add --no-cache git dep openssh-client

WORKDIR /go/src/github.com/Ankr-network/dccn-hub
COPY . .

RUN dep ensure -vendor-only