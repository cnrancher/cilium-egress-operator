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

1. Create the following example `CiliumEgressGatewayPolicy` with annotation `egress.cilium.pandaria.io/monitored=true`:

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
    [10:39:44] [DEBU] [IP:192.168.0.46] [Node:cilium-master-hmwtd-d8n7q] Node is kube-vip master node
    [10:39:44] [INFO] [IP:192.168.0.46] [Node:cilium-master-hmwtd-d8n7q] Node is not ready: NodeStatusUnknown
    [10:39:44] [INFO] [IP:192.168.0.46] [Node:cilium-master-hmwtd-d8n7q] Node "cilium-master-hmwtd-d8n7q" is not ready
    [10:39:44] [INFO] [EGP:test-policy] Egress IP [192.168.0.46] HostName [cilium-master-hmwtd-d8n7q] is not available
    [10:39:45] [INFO] [EGP:test-policy] Update CiliumEgressGatewayPolicy [test-policy] egressGateway.egressIP to [192.168.0.57] with node hostname [cilium-master-hmwtd-5t82d]
    [10:39:54] [DEBU] [IP:192.168.0.57] [Node:cilium-master-hmwtd-5t82d] Node is kube-vip master node
    [10:39:54] [DEBU] [IP:192.168.0.57] [Node:cilium-master-hmwtd-5t82d] Node "cilium-master-hmwtd-5t82d" is ready
    [10:39:54] [DEBU] [EGP:test-policy] Policy EgressIP [192.168.0.57] HostName [cilium-master-hmwtd-5t82d] is available
    ```

1. Run `cilium-dbg` command in the Cilium DaemonSet Pod to ensure the pod gateway updated to the expected Node IP.

    ```console
    $ kubectl -n kube-system exec -it cilium-xxxx -- bash
    root@cilium-worker-vgvbc-k8pqx:/home/cilium# cilium-dbg bpf egress list
    Source IP     Destination CIDR   Egress IP   Gateway IP
    10.42.0.87    0.0.0.0/0          0.0.0.0     192.168.0.57
    10.42.2.178   0.0.0.0/0          0.0.0.0     192.168.0.57
    10.42.3.47    0.0.0.0/0          0.0.0.0     192.168.0.57
    10.42.4.231   0.0.0.0/0          0.0.0.0     192.168.0.57
    ```