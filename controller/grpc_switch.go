package main

import (
	"controller/pkg/client"
	"controller/pkg/util/conversion"
	"net"
	"strconv"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

const (
	ipv4_drop_table = "MyIngress.ipv4_drop"
	ipv4_drop       = "MyIngress.drop"
	tableTimeout    = 2 * time.Second
)

type digest_t struct {
	srcAddr  net.IP
	dstAddr  net.IP
	srcPort  int
	dstPort  int
	pktCount uint64
}

func (sw *GrpcSwitch) handleDigest(digestList *p4_v1.DigestList) {
	for _, digestData := range digestList.Data {
		digestStruct := parseDigestData(digestData.GetStruct())
		sw.log.Debugf("%s P%d -> %s P%d pkt %d", digestStruct.srcAddr, digestStruct.srcPort, digestStruct.dstAddr, digestStruct.dstPort, digestStruct.pktCount)
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
	pktCount := conversion.BinaryCompressedToUint64(str.Members[4].GetBitstring())
	return digest_t{
		srcAddr:  srcAddr,
		dstAddr:  dstAddr,
		srcPort:  int(srcPort),
		dstPort:  int(dstPort),
		pktCount: pktCount,
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

func (sw *GrpcSwitch) GetName() string {
	return "s" + strconv.FormatUint(sw.id, 10)
}
