package lease

import (
	"context"
	"fmt"
	"net"

	"github.com/cnrancher/cilium-egress-operator/pkg/controller/wrangler"
	ciliumcontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/cilium.io/v2"
	coordinationcontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/coordination.k8s.io/v1"
	corecontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/core/v1"
	"github.com/cnrancher/cilium-egress-operator/pkg/internal/gateway"
	"github.com/cnrancher/cilium-egress-operator/pkg/utils"
	"github.com/sirupsen/logrus"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	handlerName = "cilium-egress-operator-lease"

	providedNodeIPAnnotationkey = "alpha.kubernetes.io/provided-node-ip"
	hostnameLabelKey            = "kubernetes.io/hostname"

	kubeVIPLeaseName      = "plndr-svcs-lock"
	kubeVIPLeaseNamespace = "kube-system"
)

type handler struct {
	nodeCache  corecontroller.NodeCache
	leaseCache coordinationcontroller.LeaseCache
	cegpCache  ciliumcontroller.CiliumEgressGatewayPolicyCache

	cegpEnqueue func(string)
}

func Register(
	ctx context.Context,
	wctx *wrangler.Context,
) {
	h := &handler{
		nodeCache:  wctx.Core.Node().Cache(),
		leaseCache: wctx.Coordination.Lease().Cache(),
		cegpCache:  wctx.Cilium.CiliumEgressGatewayPolicy().Cache(),

		cegpEnqueue: wctx.Cilium.CiliumEgressGatewayPolicy().Enqueue,
	}

	wctx.Coordination.Lease().OnChange(ctx, handlerName, h.handleError(h.sync))
}

func (h *handler) handleError(
	sync func(string, *coordinationv1.Lease) (*coordinationv1.Lease, error),
) func(string, *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	return func(s string, lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
		leaseSynced, err := sync(s, lease)
		if err != nil {
			logrus.WithFields(fieldsLease(lease)).Error(err)
			return lease, err
		}
		return leaseSynced, nil
	}
}

func (h *handler) sync(_ string, lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	if lease == nil || lease.DeletionTimestamp != nil || lease.Name != kubeVIPLeaseName || lease.Namespace != kubeVIPLeaseNamespace {
		return lease, nil
	}
	if lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity == "" {
		return lease, nil
	}
	nodeName := *lease.Spec.HolderIdentity
	if gateway.LeaderNode() == nodeName {
		return lease, nil
	}

	node, err := h.nodeCache.Get(nodeName)
	if err != nil {
		return lease, fmt.Errorf("failed to get node from cache: %w", err)
	}
	ip := nodeIP(node)
	hostname := nodeHostname(node)
	if ip == "" || hostname == "" {
		logrus.WithFields(fieldsLease(lease)).Warnf("Failed to get IP/hostname from node %q", nodeName)
		return lease, nil
	}
	logrus.WithFields(fieldsLease(lease)).Infof("Node [%v] IP [%v] is KubeVIP Leader Node", nodeName, ip)
	gateway.SetLeaderNode(ip, hostname)

	if err := h.enqueueAllPolicies(); err != nil {
		return lease, err
	}
	return lease, nil
}

func (h *handler) enqueueAllPolicies() error {
	policies, err := h.cegpCache.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list CiliumEgressgatewayPolicy from cache: %w", err)
	}
	if len(policies) == 0 {
		return nil
	}
	for _, p := range policies {
		if p == nil || p.DeletionTimestamp != nil || len(p.Annotations) == 0 {
			continue
		}
		if p.Annotations[utils.WatchAnnotationPrefix] != utils.WatchAnnotationValue {
			continue
		}
		h.cegpEnqueue(p.Name)
	}

	return nil
}

func nodeIP(node *corev1.Node) string {
	if node == nil || len(node.Annotations) == 0 {
		return ""
	}
	ip := node.Annotations[providedNodeIPAnnotationkey]
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

func nodeHostname(node *corev1.Node) string {
	if node == nil || len(node.Labels) == 0 {
		return ""
	}
	return node.Labels[hostnameLabelKey]
}

func fieldsLease(lease *coordinationv1.Lease) logrus.Fields {
	if lease == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"Lease": fmt.Sprintf("%v", lease.Name),
		"Node":  utils.Value(lease.Spec.HolderIdentity),
	}
}
