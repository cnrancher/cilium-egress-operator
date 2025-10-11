package signal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownHandler      chan os.Signal
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
	wg                   = sync.WaitGroup{}
)

// SetupSignalContext is same as SetupSignalHandler, but a context.Context is returned.
// Only one of SetupSignalContext and SetupSignalHandler should be called, and only can
// be called once.
func SetupSignalContext() context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	shutdownHandler = make(chan os.Signal, 2)

	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(shutdownHandler, shutdownSignals...)
	go func() {
		<-shutdownHandler
		wg.Add(1)
		cancel()
		fmt.Println()
		logrus.Infof("Stopping Cilium Egress Operator.")
		wg.Done()
		<-shutdownHandler

		// second signal. Exit directly.
		logrus.Warnf("forced to stop.")
		os.Exit(130)
	}()
	return ctx
}

func Flush() {
	wg.Wait()
}
