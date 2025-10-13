package wrangler

import (
	"context"
	"fmt"
	"sync"

	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/leader"
	"github.com/rancher/wrangler/v3/pkg/start"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/cilium.io"
	ciliumv2 "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/cilium.io/v2"
	"github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/core"
	corecontroller "github.com/cnrancher/cilium-egress-operator/pkg/generated/controllers/core/v1"
)

const (
	controllerName      = "cilium-egress-operator"
	controllerNamespace = "kube-system"
)

type Context struct {
	RESTConfig        *rest.Config
	Kubernetes        kubernetes.Interface
	ControllerFactory controller.SharedControllerFactory

	Core   corecontroller.Interface
	Cilium ciliumv2.Interface

	leadership *leader.Manager
	starters   []start.Starter

	controllerLock sync.Mutex
}

func NewContext(restCfg *rest.Config) (*Context, error) {
	core, err := core.NewFactoryFromConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("core factory: %w", err)
	}
	cilium, err := cilium.NewFactoryFromConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("cilium factory: %w", err)
	}

	controllerFactory, err := controller.NewSharedControllerFactoryFromConfig(restCfg, runtime.NewScheme())
	if err != nil {
		return nil, fmt.Errorf("failed to build shared controller factory: %w", err)
	}

	k8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes.NewForConfig: %w", err)
	}
	leadership := leader.NewManager(controllerNamespace, controllerName, k8s)
	c := &Context{
		RESTConfig:        restCfg,
		Kubernetes:        k8s,
		ControllerFactory: controllerFactory,

		Core:   core.Core().V1(),
		Cilium: cilium.Cilium().V2(),

		leadership: leadership,
	}
	c.starters = append(c.starters,
		core, cilium)

	return c, nil
}

func (c *Context) OnLeader(f func(ctx context.Context) error) {
	c.leadership.OnLeader(f)
}

func (c *Context) WaitForCacheSync(ctx context.Context) error {
	if err := c.ControllerFactory.SharedCacheFactory().Start(ctx); err != nil {
		return fmt.Errorf("failed to start shared cache factory: %w", err)
	}
	ok := c.ControllerFactory.SharedCacheFactory().WaitForCacheSync(ctx)
	succeed := true
	for k, v := range ok {
		if !v {
			logrus.Errorf("Failed to wait for [%v] cache sync", k)
			succeed = false
		}
	}
	if !succeed {
		return fmt.Errorf("failed to wait for cache sync")
	}
	logrus.Infof("Informer cache synced")
	return nil
}

// Run starts the leader-election process and block.
func (c *Context) Run(ctx context.Context) {
	c.controllerLock.Lock()
	c.leadership.Start(ctx)
	c.controllerLock.Unlock()

	logrus.Infof("Waiting for pod becomes leader")
}

func (c *Context) StartHandler(ctx context.Context, worker int) error {
	c.controllerLock.Lock()
	defer c.controllerLock.Unlock()

	return start.All(ctx, worker, c.starters...)
}
