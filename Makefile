.DEFAULT_GOAL := docker-image
CI_IMAGE ?= pepr-informer:ci
IMAGE ?= pepr-informer:dev
PROD_IMAGE ?= cmwylie19/pepr-informer:prod

build-ci-image:
	docker build -t $(CI_IMAGE) -f Dockerfile .
build-push-prod-image:
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(PROD_IMAGE) -f Dockerfile.amd .

build-dev-image: 
	docker build -t $(IMAGE) -f Dockerfile .

build-prod-image:
	docker build -t $(IMAGE) -f Dockerfile.amd .

build-push-arm-image: 
	docker buildx build --push -t $(IMAGE) -f Dockerfile .

build-push-amd-image: 
	docker buildx build --push -t $(IMAGE) -f Dockerfile.amd .

unit-test:
	go test -v ./... -tags='!e2e'

e2e-test:
	ginkgo -v --tags='e2e' ./e2e

deploy-dev:
	kind create cluster
	docker build -t pepr-informer:dev . -f Dockerfile
	kind load docker-image pepr-informer:dev
	kubectl apply -k kustomize/overlays/dev

curl-dev:
	docker build -t curler:ci -f hack/Dockerfile hack/
	kind load docker-image curler:ci
	kubectl apply -f hack/
	kubectl wait --for=condition=ready pod -n pepr-informer -l app=curler
	kubectl exec -it curler -n pepr-informer -- grpcurl -plaintext -d '{"group": "", "version": "v1", "resource": "pod", "namespace": "pepr-informer"}' pepr-informer.pepr-informer.svc.cluster.local:50051 api.WatchService.Watch | jq

clean-dev:
	kind delete cluster --name kind
	docker system prune -a -f

check-logs:
	kubectl logs -n pepr-informer -l app=pepr-informer -f | jq
