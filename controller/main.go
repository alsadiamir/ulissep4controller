package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
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
	defaultDeviceID = 0
	mgrp            = 0xab
	macTimeout      = 10 * time.Second
	defaultPorts    = "0,1,2,3,4,5,6,7"
)

var (
	defaultAddr = fmt.Sprintf("127.0.0.1:%d", client.P4RuntimePort)
)

func portsToSlice(ports string) ([]uint32, error) {
	p := strings.Split(ports, ",")
	res := make([]uint32, len(p))
	for idx, vStr := range p {
		v, err := strconv.Atoi(vStr)
		if err != nil {
			return nil, err
		}
		res[idx] = uint32(v)
	}
	return res, nil
}

func handleStreamMessages(p4RtC *client.Client, messageCh <-chan *p4_v1.StreamMessageResponse) {
	for message := range messageCh {
		switch m := message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			log.Debug("Recived packet in")
			for _, metadata := range m.Packet.GetMetadata() {
				if metadata.GetMetadataId() == 1 {
					destAddr := net.HardwareAddr(metadata.GetValue())
					log.Debugf("Recived packet in: destAddr %d", destAddr)
					outPort, _ := conversion.UInt32ToBinaryCompressed(2)
					outMetadata := []*p4_v1.PacketMetadata{
						{
							MetadataId: 1,
							Value:      outPort,
						},
					}
					p4RtC.SendPacketOut(m.Packet.Payload, outMetadata)
				}
			}
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

func main() {
	var addr string
	flag.StringVar(&addr, "addr", defaultAddr, "P4Runtime server socket")
	var deviceID uint64
	flag.Uint64Var(&deviceID, "device-id", defaultDeviceID, "Device id")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var binPath string
	flag.StringVar(&binPath, "bin", "", "Path to P4 bin (not needed for bmv2 simple_switch_grpc)")
	var p4infoPath string
	flag.StringVar(&p4infoPath, "p4info", "", "Path to P4Info (not needed for bmv2 simple_switch_grpc)")
	var switchPorts string
	flag.StringVar(&switchPorts, "ports", defaultPorts, "List of switch ports - required for configuring multicast group for broadcast")

	flag.Parse()

	if verbose {
		// log.WithFields(log.Fields{
		// 	"addr":   addr,
		// 	"id":     deviceID,
		// 	"bin":    binPath,
		// 	"p4info": p4infoPath,
		// 	"ports":  switchPorts,
		// }).Info("Set log level to debug")
		log.SetLevel(log.DebugLevel)
	}

	_, err := portsToSlice(switchPorts)
	if err != nil {
		log.Fatalf("Cannot parse port list: %v", err)
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

	log.Infof("Connecting to server at %s", addr)
	creds, _ := credentials.NewClientTLSFromFile("/tmp/cert.pem", "")
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
	log.Infof("P4Runtime server version is %s", resp.P4RuntimeApiVersion)

	// create channels
	electionID := p4_v1.Uint128{High: 0, Low: 1}
	stopCh := signals.RegisterSignalHandlers()
	arbitrationCh := make(chan bool)
	waitCh := make(chan struct{})
	messageCh := make(chan *p4_v1.StreamMessageResponse, 1000)
	defer close(messageCh)

	// create the p4runtime client
	p4RtC := client.NewClient(c, deviceID, electionID)
	go p4RtC.Run(stopCh, arbitrationCh, messageCh)

	// check if we are the primary client so we can do packet I/O
	go func() {
		for isPrimary := range arbitrationCh {
			if isPrimary {
				log.Infof("We are the primary client!")
				waitCh <- struct{}{}
				break
			} else {
				log.Infof("We are not the primary client!")
			}
		}
	}()

	// wait for 5 seconds to become the primary client
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		log.Fatalf("Could not become the primary client within %v", timeout)
	case <-waitCh:
	}

	log.Info("Setting forwarding pipe")
	if _, err := p4RtC.SetFwdPipeFromBytes(binBytes, p4infoBytes, 0); err != nil {
		log.Fatalf("Error when setting forwarding pipe: %v", err)
	}

	// start handling packet i/o
	go handleStreamMessages(p4RtC, messageCh)

	log.Info("Do Ctrl-C to quit")
	<-stopCh
	log.Info("Stopping client")
}
