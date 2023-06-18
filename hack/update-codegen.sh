#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/..; pwd)
cd ${SCRIPT_ROOT}

# download code-generator locally
rm -rf .gopath
mkdir -p .gopath/src/github.com/qiankunli
ln -s ${SCRIPT_ROOT} ${SCRIPT_ROOT}/.gopath/src/github.com/qiankunli/workflow
export GOPATH=${SCRIPT_ROOT}/.gopath

# -d means just clone code-generator, no need to install
go get -d k8s.io/code-generator@v0.22.3
CODEGEN_PKG=".gopath/pkg/mod/k8s.io/code-generator@v0.22.3"

bash "${CODEGEN_PKG}"/generate-groups.sh \
  all \
  github.com/qiankunli/workflow/pkg/generated \
  github.com/qiankunli/workflow/pkg/apis \
  "workflow:v1alpha1" \
  --go-header-file ./hack/boilerplate/boilerplate.go.txt