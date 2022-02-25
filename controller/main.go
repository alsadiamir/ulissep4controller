package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/antoninbas/p4runtime-go-client/pkg/signals"
)

const (
	defaultPort      = 50050
	defaultAddr      = "127.0.0.1"
	defaultWait      = 250 * time.Millisecond
	reconnectTimeout = 5 * time.Second
	maxRetry         = 3
	packetCounter    = "MyIngress.port_packets_in"
	packetCountWarn  = 20
	packetCheckRate  = 5 * time.Second
	digestName       = "digest_t"
)

func main() {
	var nDevices int
	flag.IntVar(&nDevices, "n", 1, "Number of devices")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var trace bool
	flag.BoolVar(&trace, "trace", false, "Enable trace mode with log messages")
	var binPath string
	flag.StringVar(&binPath, "bin", "", "Path to P4 bin (not needed for bmv2 simple_switch_grpc)")
	var p4infoPath string
	flag.StringVar(&p4infoPath, "p4info", "", "Path to P4Info (not needed for bmv2 simple_switch_grpc)")
	flag.Parse()

	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if trace {
		log.SetLevel(log.TraceLevel)
	}
	log.Infof("Starting %d devices", nDevices)

	binBytes := []byte("per")
	if binPath != "" {
		var err error
		if binBytes, err = ioutil.ReadFile(binPath); err != nil {
			log.Fatalf("Error when reading binary config from '%s': %v", binPath, err)
		}
	}

	p4infoBytes := []byte("per")
	if p4infoPath != "" {
		var err error
		if p4infoBytes, err = ioutil.ReadFile(p4infoPath); err != nil {
			log.Fatalf("Error when reading P4Info text file '%s': %v", p4infoPath, err)
		}
	}

	switchs := make([]*GrpcSwitch, nDevices)
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < nDevices; i++ {
		sw := createSwitch(ctx, uint64(i+1), binBytes, p4infoBytes, 3)
		if err := sw.runSwitch(); err != nil {
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
	time.Sleep(defaultWait * time.Duration(nDevices))
}
