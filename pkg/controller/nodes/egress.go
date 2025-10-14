package nodes

import (
	"fmt"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slimv1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cnrancher/cilium-egress-operator/pkg/internal/gateway"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

const (
	watchAnnotationPrefix = "egress.cilium.pandaria.io/monitored"
	watchAnnotationValue  = "true"

	hostnameLabelKey = "kubernetes.io/hostname"
)

func (h *handler) ensureEgressGatewayAvailable() error {
	policies, err := h.cegpCache.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list cilium egress gateway policies from cache: %w", err)
	}
	if len(policies) == 0 {
		return nil
	}
	availableIP, availableHostname := gateway.AvailableNode()
	if availableIP == "" || availableHostname == "" {
		logrus.Warnf("Skip check EgressPolicy EgressIP: no available master node IP")
		return nil
	}
	for _, p := range policies {
		if len(p.Annotations) == 0 || p.Annotations[watchAnnotationPrefix] != watchAnnotationValue {
			continue
		}
		if p.Spec.EgressGateway != nil && p.Spec.EgressGateway.EgressIP != "" {
			ip := getPolicyIP(p)
			hostname := getPolicyHostname(p)
			if gateway.NodeAvailable(ip, hostname) {
				logrus.WithFields(fieldEgressPolicy(p)).
					Debugf("Policy EgressIP [%v] HostName [%v] is available", ip, hostname)
			} else {
				logrus.WithFields(fieldEgressPolicy(p)).
					Infof("Egress IP [%v] HostName [%v] is not available",
						ip, hostname)
				if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
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
			}
		}
	}

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
