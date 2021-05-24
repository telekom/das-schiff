#!/bin/bash
kubectl rollout status -n capv-system deployment/capv-controller-manager
if [[ "$?" -ne 0 ]]; then
    clusterctl init --infrastructure=docker,vsphere --config tmp.yaml
fi