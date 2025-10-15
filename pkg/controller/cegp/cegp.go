package cegp

import (
	"context"
	"fmt"
	"time"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slimv1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cnrancher/cilium-egress-operator/pkg/controller/wrangler"
	ciliumcontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/cilium.io/v2"
	"github.com/cnrancher/cilium-egress-operator/pkg/internal/gateway"
	"github.com/cnrancher/cilium-egress-operator/pkg/utils"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type handler struct {
	cegpCache  ciliumcontroller.CiliumEgressGatewayPolicyCache
	cegpClient ciliumcontroller.CiliumEgressGatewayPolicyClient

	cegpEnqueueAfter func(string, time.Duration)
	cegpEnqueue      func(string)
}

const (
	handlerName = "cilium-egress-operator-cegp"

	hostnameLabelKey   = "kubernetes.io/hostname"
	defaultEnqueueTime = time.Minute * 3
)

func Register(
	ctx context.Context,
	wctx *wrangler.Context,
) {
	h := &handler{
		cegpCache:  wctx.Cilium.CiliumEgressGatewayPolicy().Cache(),
		cegpClient: wctx.Cilium.CiliumEgressGatewayPolicy(),

		cegpEnqueueAfter: wctx.Cilium.CiliumEgressGatewayPolicy().EnqueueAfter,
		cegpEnqueue:      wctx.Cilium.CiliumEgressGatewayPolicy().Enqueue,
	}

	wctx.Cilium.CiliumEgressGatewayPolicy().OnChange(ctx, handlerName, h.handleError(h.sync))
}

func (h *handler) handleError(
	sync func(string, *ciliumv2.CiliumEgressGatewayPolicy) (*ciliumv2.CiliumEgressGatewayPolicy, error),
) func(string, *ciliumv2.CiliumEgressGatewayPolicy) (*ciliumv2.CiliumEgressGatewayPolicy, error) {
	return func(s string, policy *ciliumv2.CiliumEgressGatewayPolicy) (*ciliumv2.CiliumEgressGatewayPolicy, error) {
		policySynced, err := sync(s, policy)
		if err != nil {
			logrus.WithFields(fieldEgressPolicy(policy)).Error(err)
			return policy, err
		}
		return policySynced, nil
	}
}

func (h *handler) sync(_ string, policy *ciliumv2.CiliumEgressGatewayPolicy) (*ciliumv2.CiliumEgressGatewayPolicy, error) {
	if policy == nil || policy.DeletionTimestamp != nil {
		return policy, nil
	}
	if len(policy.Annotations) == 0 || policy.Annotations[utils.WatchAnnotationPrefix] != utils.WatchAnnotationValue {
		return policy, nil
	}
	if err := h.ensurePolicyAvailable(policy); err != nil {
		return policy, err
	}
	h.cegpEnqueueAfter(policy.Name, defaultEnqueueTime)
	return policy, nil
}

func (h *handler) ensurePolicyAvailable(p *ciliumv2.CiliumEgressGatewayPolicy) error {
	availableIP := gateway.LeaderNodeIP()
	availableHostname := gateway.LeaderNode()
	if availableIP == "" || availableHostname == "" {
		return nil
	}
	if p.Spec.EgressGateway == nil {
		return nil
	}
	ip := getPolicyIP(p)
	hostname := getPolicyHostname(p)
	if ip == availableIP && hostname == availableHostname {
		logrus.WithFields(fieldEgressPolicy(p)).
			Debugf("Policy EgressIP [%v] HostName [%v] is available", ip, hostname)
		return nil
	}

	logrus.WithFields(fieldEgressPolicy(p)).
		Infof("Egress IP [%v] HostName [%v] is not available",
			ip, hostname)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pp, err := h.cegpClient.Get(p.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		pp = pp.DeepCopy()
		pp.Spec.EgressGateway.EgressIP = availableIP
		if pp.Spec.EgressGateway.NodeSelector == nil {
			pp.Spec.EgressGateway.NodeSelector = &slimv1.LabelSelector{}
		}
		if pp.Spec.EgressGateway.NodeSelector.MatchLabels == nil {
			pp.Spec.EgressGateway.NodeSelector.MatchLabels = make(map[string]slimv1.MatchLabelsValue)
		}
		pp.Spec.EgressGateway.NodeSelector.MatchLabels[hostnameLabelKey] = availableHostname
		_, err = h.cegpClient.Update(pp)
		return err
	}); err != nil {
		return fmt.Errorf("failed to update %q egressGateway.egressIP to %q: %v",
			p.Name, availableIP, err)
	}
	logrus.WithFields(fieldEgressPolicy(p)).
		Infof("Update CiliumEgressGatewayPolicy [%v] egressGateway.egressIP to [%v] with node hostname [%v]",
			p.Name, availableIP, availableHostname)

	return nil
}

func getPolicyIP(p *ciliumv2.CiliumEgressGatewayPolicy) string {
	if p == nil || p.Spec.EgressGateway == nil {
		return ""
	}
	return p.Spec.EgressGateway.EgressIP
}

func getPolicyHostname(p *ciliumv2.CiliumEgressGatewayPolicy) string {
	if p == nil || p.Spec.EgressGateway == nil || p.Spec.EgressGateway.NodeSelector == nil ||
		p.Spec.EgressGateway.NodeSelector.MatchLabels == nil {
		return ""
	}
	return p.Spec.EgressGateway.NodeSelector.MatchLabels[hostnameLabelKey]
}

func fieldEgressPolicy(p *ciliumv2.CiliumEgressGatewayPolicy) logrus.Fields {
	if p == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"EGP": fmt.Sprintf("%v", p.Name),
	}
}
