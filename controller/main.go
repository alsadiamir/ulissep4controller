package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

const (
	defaultPort     = 50050
	defaultAddr     = "127.0.0.1"
	defaultWait     = 250 * time.Millisecond
	packetCounter   = "MyIngress.port_packets_in"
	packetCountWarn = 20
	packetCheckRate = 5 * time.Second
	p4topology      = "../config/topology.json"
)

func main() {
	var nDevices int
	flag.IntVar(&nDevices, "n", 1, "Number of devices")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var trace bool
	flag.BoolVar(&trace, "trace", false, "Enable trace mode with log messages")
	var configName string
	flag.StringVar(&configName, "config", "../config/config.json", "Program name")
	var configNameAlt string
	flag.StringVar(&configNameAlt, "config-alt", "", "Alternative config name")
	flag.Parse()

	if configNameAlt == "" {
		configNameAlt = configName
	}
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if trace {
		log.SetLevel(log.TraceLevel)
	}
	log.Infof("Starting %d devices", nDevices)

	ctx, cancel := context.WithCancel(context.Background())
	switchs := make([]*GrpcSwitch, 0, nDevices)
	for i := 0; i < nDevices; i++ {
		sw := createSwitch(uint64(i+1), configName, 3)
		if err := sw.runSwitch(ctx); err != nil {
			sw.log.Errorf("Cannot start")
			log.Errorf("%v", err)
		} else {
			switchs = append(switchs, sw)
		}
	}
	if len(switchs) == 0 {
		log.Info("No switches started")
		return
	}

	buff := make([]byte, 10)
	n, _ := os.Stdin.Read(buff)
	currentConfig := configName
	for n > 0 {
		if currentConfig == configName {
			currentConfig = configNameAlt
		} else {
			currentConfig = configName
		}
		log.Infof("Changing switch config to %s", currentConfig)
		for _, sw := range switchs {
			go changeConfig(ctx, sw, currentConfig)
		}
		log.Info("Press enter to change switch config or EOF to terminate")
		n, _ = os.Stdin.Read(buff)
	}

	fmt.Println()
	cancel()
	time.Sleep(defaultWait)
}

func changeConfig(ctx context.Context, sw *GrpcSwitch, configName string) {
	if err := sw.ChangeConfig(configName); err != nil {
		if status.Convert(err).Code() == codes.Canceled {
			sw.log.Warn("Failed to update config, restarting")
			if err := sw.runSwitch(ctx); err != nil {
				sw.log.Errorf("Cannot start")
				log.Errorf("%v", err)
			}
		} else {
			sw.log.Errorf("Error updating swConfig: %v", err)
		}
		return
	}
	sw.log.Tracef("Config updated to %s", configName)
}
