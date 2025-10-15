package main

import (
	"os"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
)

func main() {
	os.Unsetenv("GOPATH")

	controllergen.Run(args.Options{
		OutputPackage: "github.com/cnrancher/cilium-egress-operator/pkg/generated",
		Boilerplate:   "pkg/codegen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"": {
				Types: []any{
					corev1.Pod{},
					corev1.Node{},
					corev1.Secret{},
				},
			},
			coordinationv1.GroupName: {
				Types: []any{
					coordinationv1.Lease{},
				},
			},
			"cilium.io": {
				Types: []any{
					ciliumv2.CiliumEgressGatewayPolicy{},
				},
			},
		},
	})
}
