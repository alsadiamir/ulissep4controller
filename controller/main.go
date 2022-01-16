package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
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
	defaultPort      = 50050
	defaultAddr      = "127.0.0.1"
	defaultWait      = 250 * time.Millisecond
	reconnectTimeout = 5 * time.Second
	maxRetry         = 5
	packetCounter    = "MyIngress.port_packets_in"
	packetCountWarn  = 20
	packetCheckRate  = 5 * time.Second
)

type swError struct {
	err error  // errors
	id  uint64 // deviceId
}

type swState struct {
	//id       uint64
	ok       bool
	nRestart int
}

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

func readCounter(p4RtC *client.Client, ports []int) error {
	deviceID := p4RtC.GetDeviceId()
	for _, port := range ports {
		counter, err := p4RtC.ReadCounterEntry(packetCounter, int64(port))
		if err != nil {
			log.WithFields(log.Fields{"ID": deviceID, "Port": port}).Error("Failed to read counter")
			return err
		}

		lFields := log.WithFields(log.Fields{"ID": deviceID, "Port": port})
		if counter.GetPacketCount() > packetCountWarn {
			lFields.Warnf("Packet count %d", counter.GetPacketCount())
		} else {
			lFields.Debugf("Packet count %d", counter.GetPacketCount())
		}
		err = p4RtC.ModifyCounterEntry(
			packetCounter,
			int64(port),
			&p4_v1.CounterData{PacketCount: 0},
		)
		if err != nil {
			log.WithFields(log.Fields{"ID": deviceID, "Port": port}).Error("Failed to set counter")
			return err
		}
	}
	return nil
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
		log.WithField("ID", p4RtC.GetDeviceId()).Errorf("Cannot insert entry :%v", err)
	} else {
		log.WithField("ID", p4RtC.GetDeviceId()).Debugf("Added table entry to device")
	}
}

func addConfig(p4RtC *client.Client) {
	deviceID := p4RtC.GetDeviceId()
	addTableEntry(p4RtC, "10.0.1."+strconv.FormatUint(deviceID, 10), 1)
	switch deviceID {
	case 1:
		addTableEntry(p4RtC, "10.0.1.2", 2)
		addTableEntry(p4RtC, "10.0.1.4", 2)
	case 2:
		addTableEntry(p4RtC, "10.0.1.1", 2)
		addTableEntry(p4RtC, "10.0.1.4", 3)
	case 4:
		addTableEntry(p4RtC, "10.0.1.2", 3)
		addTableEntry(p4RtC, "10.0.1.1", 3)
	}
}

func runSwitch(deviceID uint64, binBytes []byte, p4infoBytes []byte, stateCh chan swError) {
	addr := fmt.Sprintf("%s:%d", defaultAddr, defaultPort+deviceID)
	logF := log.WithField("ID", deviceID)
	logF.Infof("Connecting to server at %s", addr)

	creds, err := credentials.NewClientTLSFromFile("/tmp/cert.pem", "")
	if err != nil {
		stateCh <- swError{err, deviceID}
		return
	}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		stateCh <- swError{err, deviceID}
		return
	}
	defer conn.Close()

	c := p4_v1.NewP4RuntimeClient(conn)
	resp, err := c.Capabilities(context.Background(), &p4_v1.CapabilitiesRequest{})
	if err != nil {
		stateCh <- swError{err, deviceID}
		return
	}
	logF.Infof("Connected to %s, runtime version: %s", addr, resp.P4RuntimeApiVersion)

	// create channels
	electionID := p4_v1.Uint128{High: 0, Low: 1}
	arbitrationCh := make(chan bool)
	stopCh := make(chan struct{})
	messageCh := make(chan *p4_v1.StreamMessageResponse, 100)
	defer close(messageCh)

	// create the p4runtime client
	p4RtC := client.NewClient(c, deviceID, electionID)
	go p4RtC.Run(stopCh, arbitrationCh, messageCh)

	time.Sleep(defaultWait)
	if _, err := p4RtC.SetFwdPipeFromBytes(binBytes, p4infoBytes, 0); err != nil {
		stateCh <- swError{err, deviceID}
		stopCh <- struct{}{}
		return
	}
	logF.Debug("Setted forwarding pipe")

	// add default switch config
	addConfig(p4RtC)

	// start handling packet i/o
	go handleStreamMessages(p4RtC, messageCh)

	// handle ticker and disconnession
	ticker := time.NewTicker(packetCheckRate)
	for {
		select {
		case <-ticker.C:
			if err := readCounter(p4RtC, []int{1, 2, 3}); err != nil {
				stateCh <- swError{err, deviceID}
				stopCh <- struct{}{}
				return
			}
		case <-stateCh:
			stopCh <- struct{}{}
			logF.Debug("Stopped client")
			return
		}
	}
}

func main() {
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

	var i uint64
	stateCh := make(chan swError)
	states := make([]swState, nDevices+1)
	for i = 1; i <= uint64(nDevices); i++ {
		states[i] = swState{true, 0}
		go runSwitch(i, binBytes, p4infoBytes, stateCh)
	}

	// clean exit
	signalCh := signals.RegisterSignalHandlers()
	time.Sleep(defaultWait)
	log.Info("Do Ctrl-C to quit")
	for {
		select {
		case <-signalCh:
			log.Debug("Signal to stop")
			// stop only the active switches
			for _, state := range states {
				if state.ok {
					stateCh <- swError{}
				}
			}
			time.Sleep(defaultWait)
			return
		case state := <-stateCh:
			// register error of a specific switch
			log.WithField("ID", state.id).Errorf("Error %v", state.err)
			states[state.id].ok = false
			if states[state.id].nRestart == maxRetry {
				break
			}
			states[state.id].nRestart += 1
			time.AfterFunc(reconnectTimeout, func() {
				states[state.id].ok = true
				log.WithField("ID", state.id).Infof("Tring to reconnect, attempt %d", states[state.id].nRestart)
				go runSwitch(uint64(state.id), binBytes, p4infoBytes, stateCh)
			})
		}
	}
}
