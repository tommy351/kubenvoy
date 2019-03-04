#!/bin/bash

set -euxo pipefail

kind_cluster_name=kubenvoy

if ! kind get clusters | grep -q $kind_cluster_name; then
  kind create cluster --name=$kind_cluster_name --image=kindest/node:v1.13.3@sha256:d1af504f20f3450ccb7aed63b67ec61c156f9ed3e8b0d973b3dee3c95991753c
fi

export KUBECONFIG=$(kind get kubeconfig-path --name=$kind_cluster_name)

goreleaser release --snapshot --skip-publish --rm-dist
kind load docker-image --name=$kind_cluster_name tommy351/kubenvoy
kubectl delete -f test/kubernetes
kubectl apply -f test/kubernetes
go test -v ./test/e2e
