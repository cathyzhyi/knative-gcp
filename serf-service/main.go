package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/hashicorp/serf/serf"
)

func main() {
	_, cancel := context.WithCancel(context.Background())

	// start the serf instance
	eventCh := make(chan serf.Event, 64)
	serfConfig := serf.DefaultConfig()
	serflib, err := startSerfInstance(serfConfig, eventCh)
	if err != nil {
		fmt.Printf("error creating serflib\n")
		serflib = nil
	}
	fmt.Printf("yizhang started serf service\n")

	go processReceivedEvent(eventCh)

	// handle ctrl-C
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	select {
	case <-signalCh:
		cancel()
		serflib.Shutdown()
	}
}

func startSerfInstance(serfConfig *serf.Config, eventCh chan serf.Event) (*serf.Serf, error) {
	serfConfig.EventCh = eventCh
	bindAddr := os.Getenv("MY_POD_IP")
	serfConfig.NodeName = os.Getenv("MY_POD_NAME")
	serfConfig.MemberlistConfig.BindAddr = bindAddr
	return serf.Create(serfConfig)
}

func processReceivedEvent(eventCh chan serf.Event) {
	for {
		select {
		case event := <-eventCh:
			fmt.Printf("SUCCESS!! received user Event: %s\n", event.String())
		}
	}
}
