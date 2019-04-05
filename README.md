# Appmgr Service

This is the Appmgr service

Generated with

```bash
micro new github.com/Ankr-network/refactor/app_dccn_appmgr --namespace=go.micro --alias=appmgr --type=srv
```

## Getting Started

- publisher: publish the v1's app info to "topic.app.new, topic.app.cancel, topic.app.update"
- subscriber: subscribe apps's result from "topic.app.result"
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
