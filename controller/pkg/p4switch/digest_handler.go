package p4switch

import (
	"controller/pkg/util/conversion"
	"fmt"
	"net"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

type Flow struct {
	Attacker net.IP `json:"attacker"`
	Victim net.IP `json:"victim"`
}

func (flow *Flow) GetAttacker() net.IP {
	return flow.Attacker
}

func (flow *Flow) GetVictim() net.IP {
	return flow.Victim
}


type Digest struct {
    Ingress_timestamp  uint64    `json:"ingress_timestamp"`
    Packet_length   int `json:"packet_length"`
    Ip_flags    int `json:"ip_flags"`
    Tcp_len int `json:"tcp_len"`
    Tcp_ack int `json:"tcp_ack"`
    Tcp_flags int `json:"tcp_flags"`
    Tcp_window_size int `json:"tcp_window_size"`
    Udp_len int `json:"udp_len"`
    Icmp_type int `json:"icmp_type"`

    SrcPort int `json:"srcPort"`
    DstPort int `json:"dstPort"`
    Src_ip  net.IP `json:"src_ip"`
    Dst_ip  net.IP `json:"dst_ip"`
    Ip_upper_protocol int `json:"ip_upper_protocol"`   
}


var digestConfig p4_v1.DigestEntry_Config = p4_v1.DigestEntry_Config{
	MaxTimeoutNs: 0,
	MaxListSize:  1,
	AckTimeoutNs: time.Second.Nanoseconds() * 1000,
}

func (sw *GrpcSwitch) enableDigest() error {
	digestName := sw.getDigests()
	for _, digest := range digestName {
		if digest == "" {
			continue
		}
		if err := sw.p4RtC.EnableDigest(digest, &digestConfig); err != nil {
			return fmt.Errorf("cannot enable digest %s", digest)
		}
		sw.log.Debugf("Enabled digest %s", digest)
	}
	return nil
}

type digest_t struct {
	srcAddr  net.IP
	dstAddr  net.IP
	srcPort  int
	dstPort  int
//	flow uint32
	swap int
}

type digestL_t struct {
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
        swap int
}

func (sw *GrpcSwitch) pruneDigests() {
	var size = len(sw.digests)
	if size > 0 {
		var pruneDigests = []Digest{}

		var now_time = sw.digests[size-1].Ingress_timestamp

		for i:=size-1; i > 0; i-- {
			if now_time - sw.digests[i].Ingress_timestamp < 30000000 { //if the timestamp is within 30 seconds from the last tracked timestamp
				pruneDigests = append(pruneDigests, sw.digests[i])
			}
		}
		sw.digests = pruneDigests
	}
}

func (sw *GrpcSwitch) handleDigest(digestList *p4_v1.DigestList) {
	sw.pruneDigests()
	for _, digestData := range digestList.Data {		
		str := digestData.GetStruct()
		mode := int(conversion.BinaryCompressedToUint16(str.Members[0].GetBitstring()))	
		//sw.log.Debugf("mode=%d", mode)		
		if mode == 0 && sw.GetConf() == 0{	//normal
			digestStruct := parseDigestData(str)
			//sw.log.Debugf("FLOW SUSPECT NOTIFICATION swap=%d", digestStruct.swap)
			if(digestStruct.swap == 0){
				sw.log.Debugf("FLOW SUSPECT %s -> %s", digestStruct.srcAddr, digestStruct.dstAddr)
				sw.suspect_flows=append(sw.suspect_flows,Flow{
					digestStruct.srcAddr,
					digestStruct.dstAddr,
				})
			}else{
				changeConfig(sw.ctx,sw,sw.configNameAlt)
				sw.suspect_flows = []Flow{}	
			}
		}
		if mode == 1 && sw.GetConf() == 1{	//alt
			digestStruct := parseDigestDataL(str)
			//sw.log.Debugf("LUCID NOTIFICATION flow -> (%s,%s)", digestStruct.src_ip.String(), digestStruct.dst_ip.String())
			if(digestStruct.swap == 0){
				//sw.log.Debugf("LUCID NOTIFICATION at %d (swap=%d)", digestStruct.ingress_timestamp, digestStruct.swap)
				if(digestStruct.ingress_timestamp == 0){
					sw.log.Debugf("FLOW SUSPECT %s -> %s", digestStruct.src_ip, digestStruct.dst_ip)
					sw.suspect_flows=append(sw.suspect_flows,Flow{
						digestStruct.src_ip,
						digestStruct.dst_ip,
					})
				} else {
					sw.digests=append(sw.digests, Digest{
						Ingress_timestamp: digestStruct.ingress_timestamp,
						Packet_length: digestStruct.packet_length,
						Ip_flags: digestStruct.ip_flags,
						Tcp_len: digestStruct.tcp_len,
						Tcp_ack: digestStruct.tcp_ack,
						Tcp_flags: digestStruct.tcp_flags,
						Tcp_window_size: digestStruct.tcp_window_size,
						Udp_len: digestStruct.udp_len,
						Icmp_type: digestStruct.icmp_type,
						SrcPort: digestStruct.srcPort,
						DstPort: digestStruct.dstPort,
						Src_ip: digestStruct.src_ip,
						Dst_ip: digestStruct.dst_ip,
						Ip_upper_protocol: digestStruct.ip_upper_protocol,
					})
				}
			}else{	
				changeConfig(sw.ctx,sw,sw.configName)
				sw.digests = []Digest{}
			}
		}
	}
	if err := sw.p4RtC.AckDigestList(digestList); err != nil {
		sw.errCh <- err
	}
}

func parseDigestData(str *p4_v1.P4StructLike) digest_t {
	srcAddrByte := str.Members[1].GetBitstring()
	dstAddrByte := str.Members[2].GetBitstring()
	srcAddr := conversion.BinaryToIpv4(srcAddrByte)
	dstAddr := conversion.BinaryToIpv4(dstAddrByte)
	srcPort := conversion.BinaryCompressedToUint16(str.Members[3].GetBitstring())
	dstPort := conversion.BinaryCompressedToUint16(str.Members[4].GetBitstring())
//	flow := conversion.BinaryCompressedToUint32(str.Members[5].GetBitstring())
	swap := conversion.BinaryCompressedToUint16(str.Members[5].GetBitstring())
	
	return digest_t{
		srcAddr:  srcAddr,
		dstAddr:  dstAddr,
		srcPort:  int(srcPort),
		dstPort:  int(dstPort),
//		flow: flow,
		swap: int(swap),
	}
}

func parseDigestDataL(str *p4_v1.P4StructLike) digestL_t {
        ingress_timestamp := conversion.BinaryCompressedToUint64(str.Members[1].GetBitstring())
        packet_length := conversion.BinaryCompressedToUint16(str.Members[2].GetBitstring())
        ip_flags := conversion.BinaryCompressedToUint16(str.Members[3].GetBitstring())
        tcp_len := conversion.BinaryCompressedToUint16(str.Members[4].GetBitstring())
        tcp_ack := conversion.BinaryCompressedToUint16(str.Members[5].GetBitstring())
        tcp_flags := conversion.BinaryCompressedToUint16(str.Members[6].GetBitstring())
        tcp_window_size := conversion.BinaryCompressedToUint16(str.Members[7].GetBitstring())
        udp_len := conversion.BinaryCompressedToUint16(str.Members[8].GetBitstring())
        icmp_type := conversion.BinaryCompressedToUint16(str.Members[9].GetBitstring())

        srcPort := conversion.BinaryCompressedToUint16(str.Members[10].GetBitstring())
        dstPort := conversion.BinaryCompressedToUint16(str.Members[11].GetBitstring())
        src_ip := conversion.BinaryToIpv4(str.Members[12].GetBitstring())
        dst_ip := conversion.BinaryToIpv4(str.Members[13].GetBitstring())
        ip_upper_protocol := conversion.BinaryCompressedToUint16(str.Members[14].GetBitstring())
        swap := conversion.BinaryCompressedToUint16(str.Members[15].GetBitstring())
        //swap := int(str.Members[14].GetBitstring())

        return digestL_t{
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
                swap: int(swap),
        }
}

