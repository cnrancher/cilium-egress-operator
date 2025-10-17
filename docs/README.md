# Cilium Egress Operator

Operator to automatically manage [Cilium](https://docs.cilium.io/en/stable/) `CiliumEgressGatewayPolicy` egress and gateway policies in [kube-vip](https://kube-vip.io) clusters, eliminating kube-vip master node drift and gateway node failover issues, ensuring highly available and reliable pod egress network traffic.

## Requirements

- Kubernetes >= `v1.30`
- [Cilium](https://docs.cilium.io/en/stable/) >= `v1.17`
- [kube-vip](https://kube-vip.io) >= `v0.8`
- [Cilium Egress Gateway](https://docs.cilium.io/en/stable/network/egress-gateway/egress-gateway/) Enabled.

## Usage

1. Download Helm Chart from the [GitHub Release Assets](https://github.com/cnrancher/cilium-egress-operator/releases).
1. Install the Cilium Egress Operator Helm Chart.

    ```sh
    helm upgrade --install \
        -n kube-system \
        --set global.cattle.systemDefaultRegistry='registry.rancher.cn' \
        --set operator.replicas=2 \
        --set operator.debug=false \
        --set operator.image.pullPolicy=IfNotPresent \
        --set operator.setNodeIP=false \
        --set operator.setNodeLabelSelector=true \
        cilium-egress-operator \
        ./cilium-egress-operator-*.tgz
    ```

    Helm Chart Options:

    | Option | Description | Default Value |
    |--------|-------------|---------------|
    | `global.cattle.systemDefaultRegistry` | Operator pod image registry                               | `registry.rancher.cn` |
    | `operator.image.pullPolicy`           | Operator pod image pullPolicy                             | `IfNotPresent` |
    | `operator.replicas`                   | Operator pod replicas                                     | `2` |
    | `operator.debug`                      | Enable operator pod debug output                          | `false` |
    | `operator.setNodeIP`                  | Update policy egressIP to nodeIP, set to `false` to manually manage egressIP | `false` |
    | `operator.setNodeLabelSelector`       | Update policy node labelSelector to desired node hostname | `true` |

1. Create the following example `CiliumEgressGatewayPolicy` with annotation `egress.cilium.pandaria.io/monitored=true`.

    ```yaml
    apiVersion: cilium.io/v2
    kind: CiliumEgressGatewayPolicy
    metadata:
      annotations:
        egress.cilium.pandaria.io/monitored: 'true'
      name: test-policy
    spec:
      destinationCIDRs:
        - 0.0.0.0/0
      egressGateway:
        egressIP: 192.168.0.10 # Change to Master NodeIP
        nodeSelector:
          matchLabels:
            kubernetes.io/hostname: NODE_HOSTNAME # Change to Master Node HostName
      selectors:
        - podSelector:
            matchLabels:
              io.kubernetes.pod.namespace: default # Match pods in the default namespace
    ```

1. Poweroff the Master node corresponding to the above policy and check the operator log.  
    After the node becomes unavailable, the `egressGateway.egressIP` and `egressGateway.nodeSelector.matchLabels` will be automatically updated to another available master node.

    ```log
    [08:00:00] [INFO] [Lease:plndr-svcs-lock] [Node:cilium-master-hmwtd-d8n7q] Node [cilium-master-hmwtd-d8n7q] IP [192.168.0.46] is KubeVIP Leader Node
    [08:00:00] [INFO] [EGP:test-policy] Policy node hostname [cilium-master-hmwtd-dn4m5] is not available, set to [cilium-master-hmwtd-d8n7q]
    [08:00:00] [DEBU] [EGP:test-policy] Policy EgressIP [192.168.0.10] HostName [cilium-master-hmwtd-d8n7q] is available
    ```

1. Run `cilium-dbg` command in the Cilium DaemonSet Pod to ensure the pod gateway updated to the kube-vip master node IP.

    ```console
    $ kubectl -n kube-system exec -it cilium-xxxx -- bash
    root@cilium-worker-vgvbc-k8pqx:/home/cilium# cilium-dbg bpf egress list
    Source IP     Destination CIDR   Egress IP        Gateway IP
    10.42.0.87    0.0.0.0/0          192.168.0.10     192.168.0.46
    10.42.2.178   0.0.0.0/0          192.168.0.10     192.168.0.46
    10.42.3.47    0.0.0.0/0          192.168.0.10     192.168.0.46
    10.42.4.231   0.0.0.0/0          192.168.0.10     192.168.0.46
    ```
