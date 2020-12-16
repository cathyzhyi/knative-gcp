package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
	isController := strings.Contains(serfConfig.NodeName, "controller")
	if isController {
		serfConfig.Tags["role"] = "controller"
	} else {
		serfConfig.Tags["role"] = "dataplane"
	}

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
	go ProcessReceivedEvent(ctx, eventCh)

	// broadcast messages
	go func() {
		var bstr strings.Builder
		bstr.WriteString("ping all from ")
		bstr.WriteString(serfConfig.NodeName)

		var qstr strings.Builder
		qstr.WriteString("query brokercell pods name from ")
		qstr.WriteString(serfConfig.NodeName)
		for {
			time.Sleep(10 * time.Second)

			/*
				err = serflib.UserEvent(bstr.String(), []byte("tests"), false)
				if err != nil {
					logging.FromContext(ctx).Error("Failed to send events")
				}
			*/

			if !isController {
				continue
			}
			params := serflib.DefaultQueryParams()
			params.FilterTags = map[string]string{
				"role": "dataplane",
			}

			params.FilterNodes = make([]string, 0)
			params.FilterNodes = append(params.FilterNodes, "default-brokercell-fanout-6b6478fdfd-4zgvk")
			params.FilterNodes = append(params.FilterNodes, "default-brokercell-retry-78fcd97f78-hz72m")
			var payload strings.Builder
			payload.WriteString("tests query")
			payload.WriteString(time.Now().Format("15:04:05"))
			resp, err := serflib.Query(qstr.String(), []byte(payload.String()), params)
			respCh := resp.ResponseCh()
			for r := range respCh {
				s := string(r.Payload)
				logging.FromContext(ctx).Info("dump Response from ", zap.String("From: ", r.From), zap.String("Response: ", s))
			}

			if err != nil {
				logging.FromContext(ctx).Error("Failed to send events")
			}
			logging.FromContext(ctx).Info("dump query response ", zap.Any("response", resp))
		}
	}()
	return serflib
}

func startSerfInstance(serfConfig *serf.Config, eventCh chan serf.Event) (*serf.Serf, error) {
	serfConfig.EventCh = eventCh
	bindAddr := os.Getenv("POD_IP")
	serfConfig.NodeName = os.Getenv("POD_NAME")
	serfConfig.MemberlistConfig.BindAddr = bindAddr
	return serf.Create(serfConfig)
}

func ProcessReceivedEvent(ctx context.Context, eventCh chan serf.Event) {
	for {
		select {
		case event := <-eventCh:
			switch event.EventType() {
			case serf.EventQuery:
				q := event.(*serf.Query)
				nodeName := os.Getenv("POD_NAME")
				var rstr strings.Builder
				rstr.WriteString("response query: ")
				rstr.WriteString(nodeName)
				rstr.WriteString(" for time: ")
				rstr.WriteString(string(q.Payload))
				err := q.Respond([]byte(rstr.String()))
				if err == nil {
					fmt.Printf("SUCCESS!! received user query: %s\n", event.String())
				} else {
					logging.FromContext(ctx).Error("FAILD: reply to: ", zap.Error(err))
				}
			case serf.EventUser:
				fmt.Printf("SUCCESS!! received user Event: %s\n", event.String())
			}
		}
	}
}
