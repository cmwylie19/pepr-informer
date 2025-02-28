# Pepr Informer

- [Pepr Informer](#pepr-informer)
  - [Usage](#usage)
  - [Test](#test)
  - [Generic Usage](#generic-usage)
  - [Generate the Protocol Buffers](#generate-the-protocol-buffers)
  - [Generate Mocks](#generate-mocks)

A simple gRPC server that watches for Kubernetes resources and streams events to clients. It can be run in or out of cluster (for pepr dev).

```bash
Starts the pepr-informer

Usage:
  pepr-informer [flags]

Flags:
  -h, --help               help for pepr-informer
      --in-cluster         Use in-cluster configuration (default true)
  -l, --log-level string   Log level (debug, info, error) (default "info")
```

## Prereqs

- [Nats Server](https://github.com/nats-io/nats-server/releases/tag/v2.10.26)


## Local Development

```bash
./nats-server&
go run main.go --server-address ":8080" --nats-url "nats://localhost:4222" --in-cluster=false
```

```bash
curl -X POST http://localhost:8080/watch -H "Content-Type: application/json" -d '{
  "group": "",
  "version": "v1",
  "resource": "pods",
  "namespace": "default"
}'

curl -X POST http://localhost:8080/watch -H "Content-Type: application/json" -d '{
  "group": "apps",
  "version": "v1",
  "resource": "deployments",
  "namespace": "default"
}'
```

subscribe through nats cli 

```bash
nats context add localhost --description "Localhost"
nats ctx select localhost
nats sub "k8s.ADD"
```

## Usage

Bring up a dev cluster with application deployed  
```bash
make deploy-dev
```

Get Events

```bash
make curl-dev
```


## Test 

unit 
```bash
make unit test
```

e2e
```bash
make e2e test
```
