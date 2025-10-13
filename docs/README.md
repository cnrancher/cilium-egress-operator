# Cilium Egress Operator

## Requirements

- Cilium Version >= `1.17`
- Kubernetes Version >= `v1.30`
- [Cilium Egress Gateway](https://docs.cilium.io/en/stable/network/egress-gateway/egress-gateway/) Enabled.

## Usage

1. Download Helm Chart from the [GitHub Release Assets](https://github.com/cnrancher/cilium-egress-operator/releases).
1. Install the Cilium Egress Operator Helm Chart.

    ```sh
    helm upgrade --install \
        -n kube-system \
        --set global.cattle.systemDefaultRegistry='registry.rancher.cn' \
        --set operator.debug=true \
        --set operator.image.pullPolicy=IfNotPresent \
        cilium-egress-operator \
        ./cilium-egress-operator-*.tgz
    ```

1. Create the example `CiliumEgressGatewayPolicy` with annotation `egress.cilium.pandaria.io/monitored=true`:

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
        nodeSelector: {}
      selectors:
        - podSelector:
            matchLabels:
              io.kubernetes.pod.namespace: default
    ```

1. Poweroff the Master node corresponding to the above policy and check the operator log.  
    After the node becomes unavailable, the `egressGateway.egressIP` will be automatically updated to another available master node.

    ```log
    [08:00:05] [DEBU] [IP:192.168.0.10] [Node:cilium-vip-1-master-hmwtd-rjvf2] Node is kube-vip master node
    [08:00:05] [INFO] [IP:192.168.0.10] [Node:cilium-vip-1-master-hmwtd-rjvf2] Node is not ready: NodeStatusUnknown
    [08:00:05] [INFO] [IP:192.168.0.10] [Node:cilium-vip-1-master-hmwtd-rjvf2] Node "cilium-vip-1-master-hmwtd-rjvf2" is not ready
    [08:00:05] [INFO] [EGP:test-policy] Egress IP "192.168.0.10" is not available
    [08:00:05] [INFO] [EGP:test-policy] Update CiliumEgressGatewayPolicy "test-policy" egressGateway.egressIP to "192.168.0.11"
    [08:00:05] [DEBU] [IP:192.168.0.10] [Node:cilium-vip-1-master-hmwtd-rjvf2] Node is kube-vip master node
    [08:00:05] [INFO] [IP:192.168.0.10] [Node:cilium-vip-1-master-hmwtd-rjvf2] Node is not ready: NodeStatusUnknown
    [08:00:05] [INFO] [IP:192.168.0.10] [Node:cilium-vip-1-master-hmwtd-rjvf2] Node "cilium-vip-1-master-hmwtd-rjvf2" is not ready
    [08:00:05] [DEBU] [EGP:test-policy] Policy EgressIP "192.168.0.11" is available
    ```