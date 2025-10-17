<div align="center">
  <h1>Cilium Egress Operator</h1>
  <p>
    <a href="https://github.com/cnrancher/cilium-egress-operator/actions/workflows/ci.yaml"><img alt="CI" src="https://github.com/cnrancher/cilium-egress-operator/actions/workflows/ci.yaml/badge.svg"></a>
    <a href="https://goreportcard.com/report/github.com/cnrancher/cilium-egress-operator"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cnrancher/cilium-egress-operator"></a>
    <a href="https://github.com/cnrancher/cilium-egress-operator/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/cnrancher/cilium-egress-operator?color=default&label=release&logo=github"></a>
    <a href="https://github.com/cnrancher/cilium-egress-operator/releases"><img alt="GitHub pre-release" src="https://img.shields.io/github/v/release/cnrancher/cilium-egress-operator?include_prereleases&label=pre-release&logo=github"></a>
    <img alt="License" src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
  </p>
</div>

Operator to automatically manage [Cilium](https://docs.cilium.io/en/stable/) `CiliumEgressGatewayPolicy` egress and gateway policies in [kube-vip](https://kube-vip.io) cluster, eliminating kube-vip master node changes and gateway node failover issues, ensuring highly available and reliable pod egress network traffic.

## Documents

Documents are available in the [docs](./docs/README.md) directory.

## License

Copyright 2025 SUSE Rancher

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.