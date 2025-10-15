package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/cnrancher/cilium-egress-operator/pkg/controller/cegp"
	"github.com/cnrancher/cilium-egress-operator/pkg/controller/lease"
	"github.com/cnrancher/cilium-egress-operator/pkg/controller/wrangler"
	"github.com/cnrancher/cilium-egress-operator/pkg/signal"
	"github.com/cnrancher/cilium-egress-operator/pkg/utils"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
)

var (
	masterURL            string
	kubeconfigFile       string
	worker               int
	version              bool
	versionString        string
	setNodeIP            bool
	setNodeLabelSelector bool
	profileServer        bool
	profileServerAddr    string
	debug                bool
)

func init() {
	if utils.GitCommit != "" {
		versionString = fmt.Sprintf("%v - %v", utils.Version, utils.GitCommit)
	} else {
		versionString = utils.Version
	}
}

func main() {
	utils.SetupLogrus(false)
	ctx := signal.SetupSignalContext()

	flag.StringVar(&kubeconfigFile, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.IntVar(&worker, "worker", 10, "Number of controller worker threads (1-50).")
	flag.BoolVar(&version, "version", false, "Show version.")
	flag.BoolVar(&setNodeIP, "set-node-ip", false, "Set CiliumEgressGatewayPolicy EgressIP to NodeIP.")
	flag.BoolVar(&setNodeLabelSelector, "set-node-label-selector", true, "Set CiliumEgressGatewayPolicy NodeSelector to desired Node.")
	flag.BoolVar(&profileServer, "profile-server", false, "Enable the Go pprof profiling HTTP server.")
	flag.StringVar(&profileServerAddr, "profile-server-addr", "127.0.0.1:6060", "Profiling server listen address.")
	flag.BoolVar(&debug, "debug", false, "Enable the debug output.")
	flag.Parse()

	if version {
		fmt.Printf("cilium-egress-operator %v\n", versionString)
		return
	}
	if debug || os.Getenv("CATTLE_DEV_MODE") != "" {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("Debug output enabled")
	}
	if worker > 50 || worker < 1 {
		logrus.Warnf("Invalid worker: %v, should be 1-50, set to default: 10", worker)
		worker = 10
	}
	if profileServer {
		go func() {
			logrus.Infof("Go pprof server listen on: http://%v", profileServerAddr)
			if err := http.ListenAndServe(profileServerAddr, nil); err != nil {
				logrus.Errorf("Failed to start pprof server: %v", err)
			}
		}()
	}

	// This will load the kubeconfig file in a style the same as kubectl
	cfg, err := kubeconfig.GetNonInteractiveClientConfig(kubeconfigFile).ClientConfig()
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %v", err)
	}

	wctx, err := wrangler.NewContext(cfg)
	if err != nil {
		logrus.Fatalf("Failed to build wrangler context: %v", err)
	}
	if err = wctx.WaitForCacheSync(ctx); err != nil {
		logrus.Fatalf("Failed to wait for cache synced: %v", err)
	}

	lease.Register(ctx, wctx)
	cegp.Register(ctx, wctx, cegp.Options{
		SetPolicyEgressIPToNodeIP: setNodeIP,
		SetPolicyNodeSelector:     setNodeLabelSelector,
	})
	wctx.OnLeader(func(ctx context.Context) error {
		logrus.Infof("Pod [%v] is leader, starting handlers", utils.Hostname())

		// Start controller when this pod becomes leader.
		if err := wctx.StartHandler(ctx, worker); err != nil {
			return err
		}
		return nil
	})
	wctx.Run(ctx)

	select {}
}
