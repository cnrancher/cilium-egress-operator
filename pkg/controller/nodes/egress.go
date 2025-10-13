package nodes

import (
	"fmt"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cnrancher/cilium-egress-operator/pkg/internal/gateway"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

const (
	watchAnnotationPrefix = "egress.cilium.pandaria.io/monitored"
	watchAnnotationValue  = "true"
)

func (h *handler) ensureEgressGatewayAvailable() error {
	policies, err := h.cegpCache.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list cilium egress gateway policies from cache: %w", err)
	}
	if len(policies) == 0 {
		return nil
	}
	availableIPs := gateway.GetAvailableIPs()
	if len(availableIPs) == 0 {
		logrus.Warnf("Skip check EgressPolicy EgressIP: no available node IP")
		return nil
	}
	for _, p := range policies {
		if len(p.Annotations) == 0 || p.Annotations[watchAnnotationPrefix] != watchAnnotationValue {
			continue
		}
		if p.Spec.EgressGateway != nil && p.Spec.EgressGateway.EgressIP != "" {
			ip := p.Spec.EgressGateway.EgressIP
			if gateway.NodeAvailable(ip) {
				logrus.WithFields(fieldEgressPolicy(p)).
					Debugf("Policy EgressIP %q is available", ip)
			} else {
				logrus.WithFields(fieldEgressPolicy(p)).
					Infof("Egress IP %q is not available", ip)
				if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
					pp, err := h.cegpClient.Get(p.Name, metav1.GetOptions{})
					if err != nil {
						return err
					}
					pp = pp.DeepCopy()
					pp.Spec.EgressGateway.EgressIP = availableIPs[0]
					_, err = h.cegpClient.Update(pp)
					return err
				}); err != nil {
					return fmt.Errorf("failed to update %q egressGateway.egressIP to %q: %v",
						p.Name, ip, err)
				}
				logrus.WithFields(fieldEgressPolicy(p)).
					Infof("Update CiliumEgressGatewayPolicy %q egressGateway.egressIP to %q",
						p.Name, availableIPs[0])
			}
		}

		// TODO: Cilium 1.18+ supports egressGateways configuration
		// if len(p.Spec.EgressGateways) > 0 {
		// 	gws := p.DeepCopy().Spec.EgressGateways
		// 	availableGWs := make([]ciliumv2.EgressGateway, 0, len(gws))
		// 	for _, gw := range gws {
		// 		if gw.EgressIP == "" {
		// 			continue
		// 		}
		// 		if gateway.NodeAvailable(gw.EgressIP) {
		// 			availableGWs = append(availableGWs, gw)
		// 		} else {
		// 			logrus.WithFields(fieldEgressPolicy(p)).
		// 				Infof("Egress IP %q is not available", gw.EgressIP)
		// 		}
		// 	}
		// 	if len(availableGWs) == len(gws) {
		// 		logrus.WithFields(fieldEgressPolicy(p)).
		// 			Debugf("Policy EgressGateways are available, skip update")
		// 	} else {
		// 		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 			pp, err := h.cegpClient.Get(p.Name, metav1.GetOptions{})
		// 			if err != nil {
		// 				return err
		// 			}
		// 			pp = pp.DeepCopy()
		// 			pp.Spec.EgressGateways = availableGWs
		// 			_, err = h.cegpClient.Update(pp)
		// 			return err
		// 		}); err != nil {
		// 			return fmt.Errorf("failed to update %q egressGateways: %w",
		// 				p.Name, err)
		// 		}
		// 		logrus.WithFields(fieldEgressPolicy(p)).
		// 			Infof("Update CiliumEgressGatewayPolicy %q egressGateways",
		// 				p.Name)
		// 	}
		// }
	}

	return nil
}

func fieldEgressPolicy(p *ciliumv2.CiliumEgressGatewayPolicy) logrus.Fields {
	if p == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"EGP": fmt.Sprintf("%v", p.Name),
	}
}
