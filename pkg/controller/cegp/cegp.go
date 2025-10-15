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

const (
	handlerName = "cilium-egress-operator-cegp"

	hostnameLabelKey   = "kubernetes.io/hostname"
	defaultEnqueueTime = time.Minute * 3
)

type handler struct {
	cegpCache  ciliumcontroller.CiliumEgressGatewayPolicyCache
	cegpClient ciliumcontroller.CiliumEgressGatewayPolicyClient

	cegpEnqueueAfter func(string, time.Duration)
	cegpEnqueue      func(string)

	opts Options
}

type Options struct {
	SetPolicyEgressIPToNodeIP bool
	SetPolicyNodeSelector     bool
}

func Register(
	ctx context.Context,
	wctx *wrangler.Context,
	opts Options,
) {
	logrus.Debugf("CiliumEgressGatewayPolicy Handler Options: %v", utils.DebugPrint(opts))
	h := &handler{
		cegpCache:  wctx.Cilium.CiliumEgressGatewayPolicy().Cache(),
		cegpClient: wctx.Cilium.CiliumEgressGatewayPolicy(),

		cegpEnqueueAfter: wctx.Cilium.CiliumEgressGatewayPolicy().EnqueueAfter,
		cegpEnqueue:      wctx.Cilium.CiliumEgressGatewayPolicy().Enqueue,

		opts: opts,
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
	if p.Spec.EgressGateway == nil {
		return nil
	}
	ip := getPolicyIP(p)
	hostname := getPolicyHostname(p)

	desiredPolicy, needUpdate := h.policyNeedUpdate(p)
	if !needUpdate {
		logrus.WithFields(fieldEgressPolicy(p)).
			Debugf("Policy EgressIP [%v] HostName [%v] is available", ip, hostname)
		return nil
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pp, err := h.cegpClient.Get(p.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		pp = pp.DeepCopy()
		pp.Spec.EgressGateway.EgressIP = desiredPolicy.Spec.EgressGateway.EgressIP
		if pp.Spec.EgressGateway.NodeSelector == nil {
			pp.Spec.EgressGateway.NodeSelector = &slimv1.LabelSelector{}
		}
		if pp.Spec.EgressGateway.NodeSelector.MatchLabels == nil {
			pp.Spec.EgressGateway.NodeSelector.MatchLabels = make(map[string]slimv1.MatchLabelsValue)
		}
		pp.Spec.EgressGateway.NodeSelector = desiredPolicy.Spec.EgressGateway.NodeSelector
		_, err = h.cegpClient.Update(pp)
		return err
	}); err != nil {
		return fmt.Errorf("failed to sync CiliumEgressGatewayIP %q: %w",
			p.Name, err)
	}

	return nil
}

func (h *handler) policyNeedUpdate(p *ciliumv2.CiliumEgressGatewayPolicy) (*ciliumv2.CiliumEgressGatewayPolicy, bool) {
	if p == nil || p.Spec.EgressGateway == nil {
		return nil, false
	}

	desiredIP := gateway.LeaderNodeIP()
	desiredHostname := gateway.LeaderNode()

	needUpdate := false
	pp := p.DeepCopy()
	if h.opts.SetPolicyEgressIPToNodeIP && desiredIP != "" {
		ip := getPolicyIP(p)
		if ip != desiredIP {
			needUpdate = true
			pp.Spec.EgressGateway.EgressIP = desiredIP
			logrus.WithFields(fieldEgressPolicy(p)).
				Infof("Policy egressIP [%v] is not available, set to [%v]",
					ip, desiredIP)
		}
	}
	if h.opts.SetPolicyNodeSelector && desiredHostname != "" {
		hostname := getPolicyHostname(p)
		if hostname != desiredHostname {
			needUpdate = true
			if pp.Spec.EgressGateway.NodeSelector == nil {
				pp.Spec.EgressGateway.NodeSelector = &slimv1.LabelSelector{}
			}
			if pp.Spec.EgressGateway.NodeSelector.MatchLabels == nil {
				pp.Spec.EgressGateway.NodeSelector.MatchLabels = make(map[string]slimv1.MatchLabelsValue)
			}
			pp.Spec.EgressGateway.NodeSelector.MatchLabels[hostnameLabelKey] = desiredHostname
			logrus.WithFields(fieldEgressPolicy(p)).
				Infof("Policy node hostname [%v] is not available, set to [%v]",
					hostname, desiredHostname)
		}
	}

	return pp, needUpdate
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
