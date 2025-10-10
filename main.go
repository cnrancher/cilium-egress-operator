package main

import (
	"flag"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/cilium-egress-operator/pkg/utils"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/rancher/wrangler/v3/pkg/start"
	"github.com/sirupsen/logrus"
)

var (
	masterURL      string
	kubeconfigFile string
	version        bool
	debug          bool
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		TimestampFormat: "2006-01-02 15:04:05",
		FieldsOrder:     []string{"cluster", "phase"},
	})

	flag.StringVar(&kubeconfigFile, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&version, "version", false, "Show version.")
	flag.BoolVar(&debug, "debug", false, "Enable the debug output.")
	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("debug output enabled")
	}
	if version {
		if utils.GitCommit != "" {
			logrus.Infof("cilium-egress-operator %v - %v", utils.Version, utils.GitCommit)
		} else {
			logrus.Infof("cilium-egress-operator %v", utils.Version)
		}
		os.Exit(0)
	}
}

func main() {
	// set up signals so we handle the first shutdown signal gracefully
	ctx := signals.SetupSignalContext()

	// This will load the kubeconfig file in a style the same as kubectl
	cfg, err := kubeconfig.GetNonInteractiveClientConfig(kubeconfigFile).ClientConfig()
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %v", err)
	}

	// Generated apps controller
	core := core.NewFactoryFromConfigOrDie(cfg)

	// // Generated controller
	// cce, err := ccev1.NewFactoryFromConfig(cfg)
	// if err != nil {
	// 	logrus.Fatalf("Error building cce factory: %v", err)
	// }

	// The typical pattern is to build all your controller/clients then just pass to each handler
	// the bare minimum of what they need.  This will eventually help with writing tests.  So
	// don't pass in something like kubeClient, apps, or sample
	// controller.Register(ctx,
	// 	core.Core().V1().Secret(),
	// )

	// Start all the controllers
	if err := start.All(ctx, 2, core); err != nil {
		logrus.Fatalf("Error starting cce controller: %v", err)
	}

	<-ctx.Done()
	logrus.Infof("Cilium Helper Operator stopped gracefully")
}
