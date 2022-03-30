package main

import (
	"context"
	"controller/pkg/client"
	"fmt"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GrpcSwitch struct {
	id         uint64
	configName string
	ports      int
	addr       string
	restarts   int
	log        *log.Entry
	errCh      chan error
	ctx        context.Context
	p4RtC      *client.Client
	messageCh  chan *p4_v1.StreamMessageResponse
}

func createSwitch(ctx context.Context, deviceID uint64, configName string, ports int) *GrpcSwitch {
	return &GrpcSwitch{
		id:         deviceID,
		configName: configName,
		ports:      ports,
		addr:       fmt.Sprintf("%s:%d", defaultAddr, defaultPort+deviceID),
		log:        log.WithField("ID", deviceID),
		ctx:        ctx,
	}
}

func (sw *GrpcSwitch) runSwitch() error {
	sw.log.Infof("Connecting to server at %s", sw.addr)
	creds, err := credentials.NewClientTLSFromFile("/tmp/cert.pem", "")
	if err != nil {
		return err
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
	sw.log.Infof("Connected, runtime version: %s", resp.P4RuntimeApiVersion)
	// create runtime client
	electionID := p4_v1.Uint128{High: 0, Low: 1}
	sw.messageCh = make(chan *p4_v1.StreamMessageResponse, 1000)
	arbitrationCh := make(chan bool)
	sw.p4RtC = client.NewClient(c, sw.id, electionID)
	go sw.p4RtC.Run(sw.ctx, arbitrationCh, sw.messageCh)
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
	sw.enableDigest(digestName)

	sw.errCh = make(chan error, 1)
	sw.addRoutes()
	go sw.handleStreamMessages(conn)
	go sw.startRunner()

	sw.log.Debug("Switch configured")
	return nil
}

func (sw *GrpcSwitch) startRunner() {
	defer func() {
		close(sw.messageCh)
		sw.log.Info("Stopping")
	}()
	for {
		select {
		case err := <-sw.errCh:
			sw.log.Errorf("%v", err)
			go sw.reconnect()
			return
		case <-sw.ctx.Done():
			return
		}
	}
}

func (sw *GrpcSwitch) reconnect() {
	if sw.restarts >= maxRetry {
		sw.log.Errorf("Max retry attempt, killing")
		return
	}
	sw.restarts++
	sw.log.Infof("Reconnect attempt n. %d", sw.restarts)
	if err := sw.runSwitch(); err != nil {
		sw.log.Errorf("%v", err)
		time.Sleep(reconnectTimeout)
		sw.reconnect()
	} else {
		// reset retries
		sw.restarts = 0
	}
}

func (sw *GrpcSwitch) handleStreamMessages(conn *grpc.ClientConn) {
	defer conn.Close()
	for message := range sw.messageCh {
		switch m := message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			sw.log.Debug("Received Packetin")
		case *p4_v1.StreamMessageResponse_Digest:
			sw.log.Trace("Received DigestList")
			sw.handleDigest(m.Digest)
		case *p4_v1.StreamMessageResponse_IdleTimeoutNotification:
			sw.log.Trace("Received IdleTimeoutNotification")
			sw.handleIdleTimeout(m.IdleTimeoutNotification)
		case *p4_v1.StreamMessageResponse_Error:
			sw.log.Trace("Received StreamError")
			sw.errCh <- fmt.Errorf("StreamError: %v", m.Error)
		default:
			sw.log.Debug("Received unknown stream message")
		}
	}
	sw.log.Trace("Closed message channel")
}
