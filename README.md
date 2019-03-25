# Taskmgr Service

This is the Taskmgr service

Generated with

```bash
micro new github.com/Ankr-network/refactor/app_dccn_taskmgr --namespace=go.micro --alias=taskmgr --type=srv
```

## Getting Started

- publisher: publish the v1's task info to "topic.task.new, topic.task.cancel, topic.task.update"
- subscriber: subscribe tasks's result from "topic.task.result"
- handler: request handler

- [Configuration](#configuration)
- [Dependencies](#dependencies)
- [Usage](#usage)

## Configuration

- FQDN: go.micro.srv.v1
- Type: srv
- Alias: v1

## Dependencies

`brew install consul`
`brew install rabbitmq`
`go get -u -v github.com/micro/micro`
`go get -u -v github.com/micro/go-micro`
`go get -u -v github.com/micro/examples`
Micro services depend on service discovery. The default is consul.

## Usage

A Makefile is included for convenience

* Build the binary

    $ make build

* Run the service in local

    $ make local

* Build a docker image

```bash
make docker
```
