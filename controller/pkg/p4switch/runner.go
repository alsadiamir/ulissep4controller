package p4switch

import (
	"context"
	"controller/pkg/client"
	"fmt"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultPort = 50050
	defaultAddr = "127.0.0.1"
	defaultWait = 250 * time.Millisecond
)

func CreateSwitch(deviceID uint64, configName string, configNameAlt string, ports int, certFile string) *GrpcSwitch {
	return &GrpcSwitch{
		id:         deviceID,
		conf: 0,
		configName: configName,
		configNameAlt: configNameAlt,
		ports:      ports,
		addr:       fmt.Sprintf("%s:%d", defaultAddr, defaultPort+deviceID),
		log:        log.WithField("ID", deviceID),
		certFile:   certFile,
		suspect_flows: []Flow{},
		digests: []Digest{},
	}
}

func (sw *GrpcSwitch) RunSwitch(ct context.Context) error {
	ctx, cancel := context.WithCancel(ct)
	sw.ctx = ctx
	sw.cancel = cancel
	sw.log.Infof("Connecting to server at %s", sw.addr)
	var creds credentials.TransportCredentials
	if sw.certFile != "" {
		var err error
		if creds, err = credentials.NewClientTLSFromFile(sw.certFile, ""); err != nil {
			return err
		}
	} else {
		creds = insecure.NewCredentials()
	}
	conn, err := grpc.Dial(sw.addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return err
	}
	// checking runtime
	c := p4_v1.NewP4RuntimeClient(conn)
	resp, err := c.Capabilities(sw.ctx, &p4_v1.CapabilitiesRequest{})
	if err != nil {
		return err
	}
	sw.log.Debugf("Connected, runtime version: %s", resp.P4RuntimeApiVersion)
	// create runtime client
	electionID := p4_v1.Uint128{High: 0, Low: 1}
	sw.messageCh = make(chan *p4_v1.StreamMessageResponse, 1000)
	arbitrationCh := make(chan bool)
	sw.p4RtC = client.NewClient(c, sw.id, electionID)
	go sw.p4RtC.Run(ctx, conn, arbitrationCh, sw.messageCh)
	// check primary
	for isPrimary := range arbitrationCh {
		if isPrimary {
			log.Trace("we are the primary client")
			break
		} else {
			return fmt.Errorf("we are not the primary client")
		}
	}
	// set pipeline config
	time.Sleep(defaultWait)
	if _, err := sw.p4RtC.SetFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.log.Debug("Setted forwarding pipe")
	//
	sw.errCh = make(chan error, 1)
	go sw.handleStreamMessages()
	go sw.startRunner()
	//
	sw.addRules()
	sw.enableDigest()
	//
	sw.log.Info("Switch started")
	return nil
}

func (sw *GrpcSwitch) startRunner() {
	defer func() {
		close(sw.messageCh)
		sw.cancel()
		sw.log.Info("Stopping")
	}()
	for {
		select {
		case err := <-sw.errCh:
			sw.log.Errorf("%v", err)
			sw.cancel()
		case <-sw.ctx.Done():
			return
		}
	}
}

func (sw *GrpcSwitch) handleStreamMessages() {
	for message := range sw.messageCh {
		switch m := message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			sw.log.Debug("Received Packetin")
		case *p4_v1.StreamMessageResponse_Digest:
			//sw.log.Trace("Received DigestList")
			sw.handleDigest(m.Digest)
		case *p4_v1.StreamMessageResponse_IdleTimeoutNotification:
			sw.log.Trace("Received IdleTimeoutNotification")
		case *p4_v1.StreamMessageResponse_Error:
			sw.log.Trace("Received StreamError")
			sw.errCh <- fmt.Errorf("StreamError: %v", m.Error)
		default:
			sw.log.Debug("Received unknown stream message")
		}
	}
	sw.log.Trace("Closed message channel")
}
