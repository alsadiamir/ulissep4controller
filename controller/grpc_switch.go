package main

import (
	"controller/pkg/client"
	"controller/pkg/util/conversion"
	"fmt"
	"net"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	ipv4_lpm_table  = "MyIngress.ipv4_lpm"
	ipv4_drop_table = "MyIngress.ipv4_drop"
	ipv4_forward    = "MyIngress.ipv4_forward"
	ipv4_drop       = "MyIngress.drop"
	tableTimeout    = 10 * time.Second
)

type digest_t struct {
	srcAddr net.IP
	dstAddr net.IP
	srcPort int
	dstPort int
}

func (sw *GrpcSwitch) handleDigest(digestList *p4_v1.DigestList) {
	for _, digestData := range digestList.Data {
		digestData := parseDigestData(digestData.GetStruct())
		sw.log.WithFields(log.Fields{
			"srcAddr": digestData.srcAddr,
			"srcPort": digestData.srcPort,
			"dstAddr": digestData.dstAddr,
			"dstPort": digestData.dstPort,
		}).Trace()
		sw.addIpv4Drop(digestData.srcAddr.To4())
	}
	if err := sw.p4RtC.AckDigestList(digestList); err != nil {
		sw.errCh <- err
	}
	sw.log.Trace("Ack digest list")
}

func parseDigestData(str *p4_v1.P4StructLike) digest_t {
	srcAddrByte := str.Members[0].GetBitstring()
	dstAddrByte := str.Members[1].GetBitstring()
	srcAddr := conversion.BinaryToIpv4(srcAddrByte)
	dstAddr := conversion.BinaryToIpv4(dstAddrByte)
	srcPort := conversion.BinaryCompressedToUint16(str.Members[2].GetBitstring())
	dstPort := conversion.BinaryCompressedToUint16(str.Members[3].GetBitstring())
	return digest_t{
		srcAddr: srcAddr,
		dstAddr: dstAddr,
		srcPort: int(srcPort),
		dstPort: int(dstPort),
	}
}

func (sw *GrpcSwitch) addIpv4Drop(ip []byte) {
	entry := sw.p4RtC.NewTableEntry(
		ipv4_drop_table,
		[]client.MatchInterface{&client.ExactMatch{
			Value: ip,
		}},
		sw.p4RtC.NewTableActionDirect(ipv4_drop, [][]byte{}),
		&client.TableEntryOptions{IdleTimeout: tableTimeout},
	)
	if err := sw.p4RtC.SafeInsertTableEntry(entry); err != nil {
		sw.errCh <- err
		return
	}
	sw.log.Warnf("Added ipv4_drop entry: %d", ip)
}

func (sw *GrpcSwitch) handleIdleTimeout(notification *p4_v1.IdleTimeoutNotification) {
	for _, entry := range notification.TableEntry {
		// handle drop table id
		if entry.TableId != sw.p4RtC.TableId(ipv4_drop_table) {
			return
		}
		if err := sw.p4RtC.DeleteTableEntry(entry); err != nil {
			sw.errCh <- err
			return
		}
		sw.log.Infof("Remvd ipv4_drop entry: %d", entry.Match[0].GetExact().Value)
	}
}

func (sw *GrpcSwitch) addIpv4Lpm(ip []byte, mac []byte, port []byte) {
	entry := sw.p4RtC.NewTableEntry(
		ipv4_lpm_table,
		[]client.MatchInterface{&client.LpmMatch{
			Value: ip,
			PLen:  32,
		}},
		sw.p4RtC.NewTableActionDirect(ipv4_forward, [][]byte{mac, port}),
		nil,
	)
	if err := sw.p4RtC.InsertTableEntry(entry); err != nil {
		sw.errCh <- err
		return
	}
	sw.log.Debugf("Added ipv4_lpm entry: %d", ip)
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
	time.Sleep(defaultWait)
}

func (sw *GrpcSwitch) readCounter() {
	sw.log.Debug("Reading counter")
	for port := 1; port <= sw.ports; port++ {
		lFields := log.WithFields(log.Fields{"ID": sw.id, "Port": port})
		// read counter
		counter, err := sw.p4RtC.ReadCounterEntry(packetCounter, int64(port))
		if err != nil {
			sw.errCh <- err
			return
		}
		// log counter
		if counter.GetPacketCount() > packetCountWarn {
			lFields.Warnf("Packet count %d", counter.GetPacketCount())
		} else {
			lFields.Debugf("Packet count %d", counter.GetPacketCount())
		}
		// reset counter
		if err = sw.p4RtC.ModifyCounterEntry(
			packetCounter,
			int64(port),
			&p4_v1.CounterData{PacketCount: 0},
		); err != nil {
			sw.errCh <- err
			return
		}
	}
}
