package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/knative-gcp/pkg/logging"
	"github.com/hashicorp/serf/serf"
	"go.uber.org/zap"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/pkg/system"
)

func Start(ctx context.Context) *serf.Serf {

	// start the serf instance
	eventCh := make(chan serf.Event, 64)
	serfConfig := serf.DefaultConfig()
	serflib, err := startSerfInstance(serfConfig, eventCh)
	if err != nil {
		logging.FromContext(ctx).Error("error creating serflib", zap.Error(err))
		serflib = nil
	}

	// join other cluster
	otherGroup := []string{"serf-service"}
	if err != nil {
		logging.FromContext(ctx).Error("getting serf-service failed yizhang", zap.Error(err))
		if apierrs.IsNotFound(err) {
			logging.FromContext(ctx).Error("didn't find service in ", zap.String("my-namespace", system.Namespace()))
		}
	}

	logging.FromContext(ctx).Error("yizhang after reading ip")
	_, err = serflib.Join(otherGroup, false)
	if err != nil {
		logging.FromContext(ctx).Error("join existing cluster failed", zap.String("serf-service-cluster IP", otherGroup[0]), zap.Error(err))
	}

	logging.FromContext(ctx).Info("join existing cluster succeed", zap.String("serf-service-cluster IP", otherGroup[0]))
	go ProcessReceivedEvent(eventCh)

	// broadcast messages
	var b strings.Builder
	b.WriteString("ping all from ")
	b.WriteString(serfConfig.NodeName)
	err = serflib.UserEvent(b.String(), []byte("tests"), false)
	if err != nil {
		logging.FromContext(ctx).Error("Failed to send events")
	}
	return serflib
}

func startSerfInstance(serfConfig *serf.Config, eventCh chan serf.Event) (*serf.Serf, error) {
	serfConfig.EventCh = eventCh
	bindAddr := os.Getenv("MY_POD_IP")
	serfConfig.NodeName = os.Getenv("MY_POD_NAME")
	serfConfig.MemberlistConfig.BindAddr = bindAddr
	return serf.Create(serfConfig)
}

func ProcessReceivedEvent(eventCh chan serf.Event) {
	for {
		select {
		case event := <-eventCh:
			fmt.Printf("SUCCESS!! received user Event: %s\n", event.String())
		}
	}
}
