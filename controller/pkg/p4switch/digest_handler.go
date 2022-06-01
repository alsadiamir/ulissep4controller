package p4switch

import (
	"controller/pkg/util/conversion"
	"fmt"
	"net"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

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
	flow uint32
}

func (sw *GrpcSwitch) handleDigest(digestList *p4_v1.DigestList) {
	for _, digestData := range digestList.Data {
		digestStruct := parseDigestData(digestData.GetStruct())
		sw.log.Debugf("FLOW SUSPECT(hash:%d) %s -> %s", digestStruct.flow, digestStruct.srcAddr, digestStruct.dstAddr)
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
	flow := conversion.BinaryCompressedToUint32(str.Members[4].GetBitstring())
	return digest_t{
		srcAddr:  srcAddr,
		dstAddr:  dstAddr,
		srcPort:  int(srcPort),
		dstPort:  int(dstPort),
		flow: flow,
	}
}
