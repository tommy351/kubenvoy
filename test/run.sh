#!/bin/sh

set -e

export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"

goreleaser release --snapshot --skip-publish --rm-dist
kind load docker-image tommy351/kubenvoy:latest
kubectl apply -f test/kubernetes
