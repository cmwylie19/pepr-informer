k3d cluster delete --all
k3d cluster create;
docker build -t curler:ci -f hack/Dockerfile .;
k3d image import curler:ci -c k3s-default;
k apply -f hack


kubectl exec -it curler -- grpcurl -plaintext -d '{"group": "", "version": "v1", "resource": "pod", "namespace": "default"}' pepr-informer.pepr-informer.svc.cluster.local:50051 api.WatchService.Watch
