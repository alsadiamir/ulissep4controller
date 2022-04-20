package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/antoninbas/p4runtime-go-client/pkg/signals"
)

const (
	defaultPort      = 50050
	defaultAddr      = "127.0.0.1"
	defaultWait      = 250 * time.Millisecond
	reconnectTimeout = 5 * time.Second
	packetCounter    = "MyIngress.port_packets_in"
	packetCountWarn  = 20
	packetCheckRate  = 5 * time.Second
	digestName       = "digest_t"
)

var (
	maxRetry int
)

func main() {
	var nDevices int
	flag.IntVar(&nDevices, "n", 1, "Number of devices")
	flag.IntVar(&maxRetry, "retry", 0, "Number of times retry to connect")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var trace bool
	flag.BoolVar(&trace, "trace", false, "Enable trace mode with log messages")
	var programName string
	flag.StringVar(&programName, "program", "simple", "Program name")
	//var programNameAlt string
	//flag.StringVar(&programNameAlt, "program-alt", "", "Alternative program name")
	flag.Parse()


	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if trace {
		log.SetLevel(log.TraceLevel)
	}
	log.Infof("Starting %d devices with the program %s", nDevices, programName)

	switchs := make([]*GrpcSwitch, nDevices)
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < nDevices; i++ {
		sw := createSwitch(ctx, uint64(i+1), programName, 3)
		if err := sw.runSwitch();err != nil {
			sw.log.Errorf("Cannot start")
			log.Errorf("%v", err)
		}
		switchs[i] = sw
	}

	// clean exit
	signalCh := signals.RegisterSignalHandlers()
	log.Info("Do Ctrl-C to quit")
	<-signalCh


	fmt.Println()
	cancel()
	time.Sleep(defaultWait)
}
