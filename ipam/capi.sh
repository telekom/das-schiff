#!/bin/bash
# Copyright 2021 Deutsche Telekom AG
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

kubectl rollout status -n capv-system deployment/capv-controller-manager
if [[ "$?" -ne 0 ]]; then
    clusterctl init --infrastructure=docker,vsphere --config tmp.yaml
fi