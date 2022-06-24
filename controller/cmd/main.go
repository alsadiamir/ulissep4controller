package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
	"io/ioutil"

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

func GetSwitchInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
		
    vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }

    json.NewEncoder(w).Encode(switch_list[i].GetSwitchInfo())
}

func GetSwitchSuspectFlows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
		
    vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }

    json.NewEncoder(w).Encode(switch_list[i].GetFlows())
}

func GetSwitchDroppedFlows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
		
    vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }

    json.NewEncoder(w).Encode(switch_list[i].GetDroppedFlows())
}

func GetDigestsLUCID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
		
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

func contains(flow p4switch.Flow, flows []p4switch.Flow) bool {
    for _, f := range flows {
        if f.GetAttacker().Equal(flow.GetAttacker()) && f.GetVictim().Equal(flow.GetVictim()) {
            return true
        }
    }
    return false
}

func DropFlowLUCID(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
    var flow p4switch.Flow 

    json.Unmarshal(reqBody, &flow)

	vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }

    if contains(flow, switch_list[i].GetDroppedFlows()) == false {
    	switch_list[i].AddDroppedFlow(flow)
    	switch_list[i].DropFlow(flow)
    } 

    json.NewEncoder(w).Encode(flow)
}

func UpdateDDoSLUCID(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
    var flow p4switch.Flow 

    json.Unmarshal(reqBody, &flow)

	vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }

    if contains(flow, switch_list[i].GetFlows()) == true {
    	switch_list[i].UpdateSuspectFlow(flow)
    } 
    if contains(flow, switch_list[i].GetDroppedFlows()) == true {
    	switch_list[i].UpdateDroppedFlow(flow)
    }

    json.NewEncoder(w).Encode(flow)
}

func RemoveDropFlowLUCID(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
    var flow p4switch.Flow 

    json.Unmarshal(reqBody, &flow)

	vars := mux.Vars(r)
    n := vars["n"]

    i, err := strconv.Atoi(n)
    if err != nil {
        // handle error
        fmt.Println(err)
        os.Exit(2)
    }

    if contains(flow, switch_list[i].GetDroppedFlows()) == true {
    	switch_list[i].RemoveDroppedFlow(flow)
    	switch_list[i].RemoveDropFlow(flow)
    } 

    json.NewEncoder(w).Encode(flow)
}


func ListenToLUCIDRequests(nDevices int){
    myRouter := mux.NewRouter().StrictSlash(true)

    myRouter.HandleFunc("/digests/{n}", GetDigestsLUCID) 
    myRouter.HandleFunc("/info/{n}", GetSwitchInfo) 
    myRouter.HandleFunc("/suspect/{n}", GetSwitchSuspectFlows) 
    myRouter.HandleFunc("/dropped/{n}", GetSwitchDroppedFlows) 
    myRouter.HandleFunc("/digests/drop/{n}", DropFlowLUCID).Methods("POST")  
	myRouter.HandleFunc("/digests/removedrop/{n}", RemoveDropFlowLUCID).Methods("POST") 
	myRouter.HandleFunc("/digests/updateflows/{n}", UpdateDDoSLUCID).Methods("POST") 

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

}
