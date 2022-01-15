package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"

	"controller/pkg/client"
	"controller/pkg/util/conversion"

	"github.com/antoninbas/p4runtime-go-client/pkg/signals"
)

const (
	defaultPort = 50050
	defaultAddr = "127.0.0.1"
)

// var (
// 	defaultIp = net.ParseIP(defaultAddr).To4()
// )

func handleStreamMessages(p4RtC *client.Client, messageCh <-chan *p4_v1.StreamMessageResponse) {
	for message := range messageCh {
		switch message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			log.Debugf("Received Packetin")
		case *p4_v1.StreamMessageResponse_Digest:
			log.Debugf("Received DigestList")
		case *p4_v1.StreamMessageResponse_IdleTimeoutNotification:
			log.Debugf("Received IdleTimeoutNotification")
		case *p4_v1.StreamMessageResponse_Error:
			log.Errorf("Received StreamError")
		default:
			log.Errorf("Received unknown stream message")
		}
	}
}

func addTableEntry(p4RtC *client.Client, ip string, port int) {
	p, _ := conversion.UInt32ToBinaryCompressed(uint32(port))
	ipv4 := net.ParseIP(ip).To4()
	entry := p4RtC.NewTableEntry(
		"MyIngress.ipv4_lpm",
		[]client.MatchInterface{&client.LpmMatch{
			Value: ipv4,
			PLen:  32,
		}},
		p4RtC.NewTableActionDirect("MyIngress.ipv4_forward", [][]byte{p}),
		nil,
	)
	if err := p4RtC.InsertTableEntry(entry); err != nil {
		log.Errorf("Cannot insert entry :%v", err)
	} else {
		log.Debugf("Added table entry to device")
	}
}

func addConfig(p4RtC *client.Client, deviceID uint64) {
	switch deviceID {
	case 1:
		addTableEntry(p4RtC, "10.0.1.2", 2)
		addTableEntry(p4RtC, "10.0.1.1", 1)
	case 2:
		addTableEntry(p4RtC, "10.0.1.2", 1)
		addTableEntry(p4RtC, "10.0.1.1", 2)
	}
}

func startSwitch(wg *sync.WaitGroup, deviceID uint64, binBytes []byte, p4infoBytes []byte) {
	defer wg.Done()

	addr := fmt.Sprintf("%s:%d", defaultAddr, defaultPort+deviceID)
	log.Infof("Connecting to server at %s", addr)

	creds, err := credentials.NewClientTLSFromFile("/tmp/cert.pem", "")
	if err != nil {
		log.Fatalf("Cannot create credentials: %v", err)
	}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Cannot connect to server: %v", err)
	}
	defer conn.Close()

	c := p4_v1.NewP4RuntimeClient(conn)
	resp, err := c.Capabilities(context.Background(), &p4_v1.CapabilitiesRequest{})
	if err != nil {
		log.Fatalf("Error in Capabilities RPC: %v", err)
	}
	log.Infof("Connected to %s, runtime version: %s", addr, resp.P4RuntimeApiVersion)

	// create channels
	electionID := p4_v1.Uint128{High: 0, Low: 1}
	stopCh := signals.RegisterSignalHandlers()
	arbitrationCh := make(chan bool)
	messageCh := make(chan *p4_v1.StreamMessageResponse, 1000)
	defer close(messageCh)

	// create the p4runtime client
	p4RtC := client.NewClient(c, deviceID, electionID)
	go p4RtC.Run(stopCh, arbitrationCh, messageCh)

	time.Sleep(500 * time.Millisecond)
	log.Info("Setting forwarding pipe")
	if _, err := p4RtC.SetFwdPipeFromBytes(binBytes, p4infoBytes, 0); err != nil {
		log.Fatalf("Error when setting forwarding pipe: %v", err)
	}

	// start handling packet i/o
	addConfig(p4RtC, deviceID)
	handleStreamMessages(p4RtC, messageCh)
}

func main() {
	var wg sync.WaitGroup
	var nDevices int
	flag.IntVar(&nDevices, "n", 1, "Number of devices")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var binPath string
	flag.StringVar(&binPath, "bin", "", "Path to P4 bin (not needed for bmv2 simple_switch_grpc)")
	var p4infoPath string
	flag.StringVar(&p4infoPath, "p4info", "", "Path to P4Info (not needed for bmv2 simple_switch_grpc)")
	flag.Parse()

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

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

	var i uint64
	for i = 1; i <= uint64(nDevices); i++ {
		wg.Add(1)
		go startSwitch(&wg, i, binBytes, p4infoBytes)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("Ctrl+C Pressed Exiting")
		os.Exit(0)
	}()

	wg.Wait()
}
