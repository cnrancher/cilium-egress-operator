package nodes

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/cnrancher/cilium-egress-operator/pkg/controller/wrangler"
	ciliumcontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/cilium.io/v2"
	corecontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/core/v1"
	"github.com/cnrancher/cilium-egress-operator/pkg/internal/gateway"
	"github.com/cnrancher/cilium-egress-operator/pkg/utils"
)

const (
	handlerName = "cilium-egress-operator-node"

	providedNodeIPAnnotationkey = "alpha.kubernetes.io/provided-node-ip"
	kubeVIPPodLabelKey          = "app.kubernetes.io/name"
	kubeVIPPodLabelValue        = "kube-vip"
)

var (
	initWg sync.WaitGroup
)

func init() {
	initWg.Add(1)
}

type handler struct {
	nodeClient corecontroller.NodeClient
	nodeCache  corecontroller.NodeCache
	podCache   corecontroller.PodCache
	cegpClient ciliumcontroller.CiliumEgressGatewayPolicyClient
	cegpCache  ciliumcontroller.CiliumEgressGatewayPolicyCache

	nodeEnqueueAfter func(string, time.Duration)
	nodeEnqueue      func(string)
}

func Register(
	ctx context.Context,
	wctx *wrangler.Context,
) {
	h := &handler{
		nodeClient: wctx.Core.Node(),
		nodeCache:  wctx.Core.Node().Cache(),
		podCache:   wctx.Core.Pod().Cache(),
		cegpClient: wctx.Cilium.CiliumEgressGatewayPolicy(),
		cegpCache:  wctx.Cilium.CiliumEgressGatewayPolicy().Cache(),

		nodeEnqueueAfter: wctx.Core.Node().EnqueueAfter,
		nodeEnqueue:      wctx.Core.Node().Enqueue,
	}

	wctx.Core.Node().OnChange(ctx, handlerName, h.handleError(h.sync))
}

func InitNodeIP(wctx *wrangler.Context) error {
	defer initWg.Done()

	nodes, err := wctx.Core.Node().Cache().List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to init Node IP: failed to list node from cache: %w", err)
	}
	if len(nodes) == 0 {
		return nil
	}
	for _, node := range nodes {
		if node == nil || len(node.Annotations) == 0 || node.DeletionTimestamp != nil {
			continue
		}

		if ok, _ := isKubeVIPControlPlaneNode(node, wctx.Core.Pod().Cache()); !ok {
			continue
		}
		if isNodeActive(node) {
			logrus.WithFields(fieldsNode(node)).
				Debugf("Initialize node IP to available record")
			gateway.RecordNodeIP(nodeIP(node), true)
		}
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		logrus.Debugf("Initialized available node IP: %v",
			utils.Print(gateway.GetAvailableIPs()))
	}

	return nil
}

func (h *handler) handleError(
	sync func(string, *corev1.Node) (*corev1.Node, error),
) func(string, *corev1.Node) (*corev1.Node, error) {
	return func(s string, node *corev1.Node) (*corev1.Node, error) {
		initWg.Wait()

		nodeSynced, err := sync(s, node)
		if err != nil {
			logrus.WithFields(fieldsNode(node)).Error(err)
			return node, err
		}
		return nodeSynced, nil
	}
}

func (h *handler) sync(_ string, node *corev1.Node) (*corev1.Node, error) {
	// Skip non kube-vip master nodes
	ok, err := isKubeVIPControlPlaneNode(node, h.podCache)
	if err != nil {
		return node, err
	}
	if !ok {
		return node, nil
	}
	logrus.WithFields(fieldsNode(node)).
		Debugf("Node is kube-vip master node")

	if node.DeletionTimestamp != nil {
		logrus.WithFields(fieldsNode(node)).Infof("Node %q is being deleted", node.Name)
		gateway.RecordNodeIP(nodeIP(node), false)

		// Update cilium egress gateway on node delete
		if err := h.ensureEgressGatewayAvailable(); err != nil {
			return node, err
		}
		return node, nil
	}

	if isNodeActive(node) {
		// Node condition is ready
		logrus.WithFields(fieldsNode(node)).Debugf("Node %q is ready", node.Name)
		gateway.RecordNodeIP(nodeIP(node), true)
	} else {
		logrus.WithFields(fieldsNode(node)).Infof("Node %q is not ready", node.Name)
		gateway.RecordNodeIP(nodeIP(node), false)
	}

	// Ensure all cilium egress gateways available
	if err := h.ensureEgressGatewayAvailable(); err != nil {
		return node, err
	}

	return node, nil
}

func isKubeVIPControlPlaneNode(node *corev1.Node, podCache corecontroller.PodCache) (bool, error) {
	if node == nil || len(node.Annotations) == 0 || node.Name == "" {
		return false, nil
	}

	pods, err := podCache.List("", labels.Everything())
	if err != nil {
		return false, fmt.Errorf("failed to list pod from cache: %w", err)
	}
	if len(pods) == 0 {
		return false, nil
	}
	for _, p := range pods {
		if p.DeletionTimestamp != nil {
			continue
		}
		if p.Spec.NodeName != node.Name {
			continue
		}
		if len(p.Labels) == 0 {
			continue
		}
		if p.Labels[kubeVIPPodLabelKey] == kubeVIPPodLabelValue {
			return true, nil
		}
	}
	return false, nil
}

func isNodeActive(node *corev1.Node) bool {
	if len(node.Status.Conditions) == 0 {
		return false
	}
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status != corev1.ConditionTrue {
				logrus.WithFields(fieldsNode(node)).
					Infof("Node is not ready: %v", cond.Reason)
				return false
			}
		}
	}
	return true
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

func fieldsNode(node *corev1.Node) logrus.Fields {
	if node == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"Node": fmt.Sprintf("%v", node.Name),
		"IP":   nodeIP(node),
	}
}
