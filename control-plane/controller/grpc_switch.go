package main

import (
	"controller/pkg/client"
	"controller/pkg/util/conversion"
	"net"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
)

const (
	ipv4_drop_table = "MyIngress.ipv4_drop"
	ipv4_drop       = "MyIngress.drop"
	tableTimeout    = 10 * time.Second
)

type digest_t struct {
	ingress_timestamp uint64
	packet_length int
	ip_flags int
	tcp_len int
	tcp_ack int
	tcp_flags int
	tcp_window_size int
	udp_len int
	icmp_type int
	srcPort int
	dstPort int
	src_ip net.IP
	dst_ip net.IP
	ip_upper_protocol int
}

func (sw *GrpcSwitch) handleDigest(digestList *p4_v1.DigestList) {
	for _, digestData := range digestList.Data {
		digestData := parseDigestData(digestData.GetStruct())
		sw.log.WithFields(log.Fields{
			"ingress_timestamp": digestData.ingress_timestamp,
			"packet_length": digestData.packet_length,
			"ip_flags": digestData.ip_flags,
			"tcp_len": digestData.tcp_len,
			"tcp_ack": digestData.tcp_ack,
			"tcp_flags": digestData.tcp_flags,
			"tcp_window_size": digestData.tcp_window_size,
			"udp_len": digestData.udp_len,
			"icmp_type": digestData.icmp_type,
			"srcPort": digestData.srcPort,
			"dstPort": digestData.dstPort,
			"src_ip": digestData.src_ip,
			"dst_ip": digestData.dst_ip,
			"ip_upper_protocol": digestData.ip_upper_protocol,
		}).Trace()
		//sw.addIpv4Drop(digestData.srcAddr.To4())
	}
	if err := sw.p4RtC.AckDigestList(digestList); err != nil {
		sw.errCh <- err
	}
	sw.log.Trace("Ack digest list")
}

func parseDigestData(str *p4_v1.P4StructLike) digest_t {
	ingress_timestamp := conversion.BinaryCompressedToUint64(str.Members[0].GetBitstring())
	packet_length := conversion.BinaryCompressedToUint16(str.Members[1].GetBitstring())
	ip_flags := conversion.BinaryCompressedToUint16(str.Members[2].GetBitstring())
	tcp_len := conversion.BinaryCompressedToUint16(str.Members[3].GetBitstring())
	tcp_ack := conversion.BinaryCompressedToUint16(str.Members[4].GetBitstring())
	tcp_flags := conversion.BinaryCompressedToUint16(str.Members[5].GetBitstring())
	tcp_window_size := conversion.BinaryCompressedToUint16(str.Members[6].GetBitstring())
	udp_len := conversion.BinaryCompressedToUint16(str.Members[7].GetBitstring())
	icmp_type := conversion.BinaryCompressedToUint16(str.Members[8].GetBitstring())

	srcPort := conversion.BinaryCompressedToUint16(str.Members[9].GetBitstring())
	dstPort := conversion.BinaryCompressedToUint16(str.Members[10].GetBitstring())
	src_ip := conversion.BinaryToIpv4(str.Members[11].GetBitstring())
	dst_ip := conversion.BinaryToIpv4(str.Members[12].GetBitstring())
	ip_upper_protocol := conversion.BinaryCompressedToUint16(str.Members[13].GetBitstring())



	return digest_t{
		ingress_timestamp: ingress_timestamp,
		packet_length: int(packet_length),
		ip_flags: int(ip_flags),
		tcp_len: int(tcp_len),
		tcp_ack: int(tcp_ack),
		tcp_flags: int(tcp_flags),
		tcp_window_size: int(tcp_window_size),
		udp_len: int(udp_len),
		icmp_type: int(icmp_type),
		srcPort: int(srcPort),
		dstPort: int(dstPort),
		src_ip: src_ip,
		dst_ip: dst_ip,
		ip_upper_protocol: int(ip_upper_protocol),
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
