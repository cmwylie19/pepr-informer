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
go run main.go --server-address=":8080" --nats-url="nats://localhost:4222" --in-cluster=false
```

```bash
curl -X POST http://localhost:8080/watch -H "Content-Type: application/json" -d '{
  "group": "",
  "version": "v1",
  "resource": "pods",
  "namespace": "default"
}'

curl -X POST http://localhost:8080/watch -H "Content-Type: application/json" -d '{
  "group": "",
  "version": "v1",
  "resource": "pods",
  "namespace": ""
}'

curl -X POST http://localhost:8080/watch -H "Content-Type: application/json" -d '{
  "group": "apps",
  "version": "v1",
  "resource": "deployments",
  "namespace": "default"
}'

nats sub "k8s.apps.v1.deployments.default"
nats sub "k8s.v1.pods.default"
nats sub "k8s.v1.pods"
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

```yaml
        - name: informer
          image: pepr-informer:dev
          command: ["./pepr-informer", "--server-address=:8080", "--nats-url=nats://nats-headless:4222"]
          resources:
            requests:
              memory: 128Mi
              cpu: 100m
            limits:
              memory: 128Mi
              cpu: 200m
```

```ts
import {
  Capability,
  K8s,
  a,
} from "pepr";

/**
 *  The HelloPepr Capability is an example capability to demonstrate some general concepts of Pepr.
 *  To test this capability you run `pepr dev`and then run the following command:
 *  `kubectl apply -f capabilities/hello-pepr.samples.yaml`
 */
export const HelloPepr = new Capability({
  name: "hello-pepr",
  description: "A simple example capability to show how things work."
});

// Use the 'When' function to create a new action, use 'Store' to persist data
const { When } = HelloPepr;
const deletePod = async (name: string) => {
  await K8s(a.Pod).InNamespace("pepr-demo").Delete(name);
};

When(a.Pod)
  .IsCreatedOrUpdated()
  .InNamespace("pepr-demo")
  .Reconcile(async instance => {
    await deletePod(instance.metadata.name);
  });
```
