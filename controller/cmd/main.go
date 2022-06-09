package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

//	"google.golang.org/grpc/codes"
//	"google.golang.org/grpc/status"

	"controller/pkg/p4switch"
//	"controller/pkg/restapi"
	"controller/pkg/signals"
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
	"strconv"
//	"gopkg.in/yaml.v3"

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

var switch_list = []*p4switch.GrpcSwitch{}

func GetDigestsLUCID(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Endpoint Hit: getDigests")
    vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }


    json.NewEncoder(w).Encode(switch_list[i].GetDigests())
}

func ListenToLUCIDRequests(nDevices int){
    myRouter := mux.NewRouter().StrictSlash(true)
    //for i := 0; i < nDevices; i++ {
    myRouter.HandleFunc("/digests/{n}", GetDigestsLUCID)   
    //}
    log.Fatal(http.ListenAndServe(":10000", myRouter))
}

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
	var certFile string
	flag.StringVar(&certFile, "cert-file", "", "Certificate file for tls")
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
	switchs := make([]*p4switch.GrpcSwitch, 0, nDevices)
	for i := 0; i < nDevices; i++ {
		sw := p4switch.CreateSwitch(uint64(i+1), configName, configNameAlt, 3, certFile)
		if err := sw.RunSwitch(ctx); err != nil {
			sw.GetLogger().Errorf("Cannot start")
			log.Errorf("%v", err)
		} else {
			switchs = append(switchs, sw)
		}
	}
	if len(switchs) == 0 {
		log.Info("No switches started")
		return
	}
	switch_list = switchs

	ListenToLUCIDRequests(nDevices)

	// clean exit
	signalCh := signals.RegisterSignalHandlers()
	log.Info("Do Ctrl-C to quit")
	<-signalCh


	fmt.Println()
	cancel()
	time.Sleep(defaultWait)

/*
    var config *p4switch.SwitchConfig =  p4switch.ParseSwConfig("s1", "../config/singlesw-config.yml")

    config.Rules = append(config.Rules, p4switch.Rule{
    	Table:       "MyIngress.ipv4_lpm",
		Key:         []string{"10.0.1.1","10.0.1.2"},
		Type:        "exact",
		Action:      "",
		ActionParam: []string{},
	})



    fmt.Println(" --- YAML with maps and arrays ---")
    fmt.Println("+%v",config)

    file, err := os.OpenFile("test.yml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        log.Fatalf("error opening/creating file: %v", err)
    }
    defer file.Close()

    enc := yaml.NewEncoder(file)

    err = enc.Encode(config)
    if err != nil {
        log.Fatalf("error encoding: %v", err)
    }
*/
}
