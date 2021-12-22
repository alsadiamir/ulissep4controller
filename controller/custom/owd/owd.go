package owd

import (
	//"github.com/google/gopacket/layers"
	"encoding/binary"
	"fmt"
	"net"
)

type OWD struct {
	SrcMAC, DstMAC                                   net.HardwareAddr
	EthernetType                                     uint16
	Pid, Dst_id, Nhop                                uint16
	Ts_ing1, Ts_eg1, Ts_is2, Ts_es2, Ts_ing2, Ts_eg2 uint64
}

func Deserialize(owdbytes []byte) *OWD {
	tmp := make([]byte, 8)
	tmp[0] = 0x00
	tmp[1] = 0x00

	owdpack := new(OWD)
	owdpack.SrcMAC = net.HardwareAddr(owdbytes[0:6])
	owdpack.DstMAC = net.HardwareAddr(owdbytes[6:12])
	owdpack.EthernetType = binary.BigEndian.Uint16(owdbytes[12:14])
	owdpack.Pid = binary.BigEndian.Uint16(owdbytes[14:16])
	owdpack.Dst_id = binary.BigEndian.Uint16(owdbytes[16:18])
	owdpack.Nhop = binary.BigEndian.Uint16(owdbytes[18:20])
	copy(tmp[2:], owdbytes[20:26])
	owdpack.Ts_ing1 = binary.BigEndian.Uint64(tmp)
	copy(tmp[2:], owdbytes[26:32])
	owdpack.Ts_eg1 = binary.BigEndian.Uint64(tmp)
	copy(tmp[2:], owdbytes[32:38])
	owdpack.Ts_is2 = binary.BigEndian.Uint64(tmp)
	copy(tmp[2:], owdbytes[38:44])
	owdpack.Ts_es2 = binary.BigEndian.Uint64(tmp)
	copy(tmp[2:], owdbytes[44:50])
	owdpack.Ts_ing2 = binary.BigEndian.Uint64(tmp)
	copy(tmp[2:], owdbytes[50:56])
	owdpack.Ts_eg2 = binary.BigEndian.Uint64(tmp)
	return owdpack
}

func (owdpack OWD) String() string {
	return fmt.Sprintf("dst=%s, src=%s, ethType=%d, Pid=%d, Dst_id=%d, Nhop=%d,Ts_ing1=%d, Ts_eg1=%d, Ts_is2=%d, Ts_es2=%d, Ts_ing2=%d, Ts_eg2=%d \n",
		owdpack.DstMAC,
		owdpack.SrcMAC,
		owdpack.EthernetType,
		owdpack.Pid,
		owdpack.Dst_id,
		owdpack.Nhop,
		owdpack.Ts_ing1,
		owdpack.Ts_eg1,
		owdpack.Ts_is2,
		owdpack.Ts_es2,
		owdpack.Ts_ing2,
		owdpack.Ts_eg2)
}

func Serialize(packet OWD) []byte {

	tmp := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	owdbytes := make([]byte, 56)
	copy(owdbytes[0:6], packet.DstMAC)
	copy(owdbytes[6:12], packet.SrcMAC)
	binary.LittleEndian.PutUint16(owdbytes[12:14], packet.EthernetType)
	binary.LittleEndian.PutUint16(owdbytes[14:16], packet.Pid)
	binary.LittleEndian.PutUint16(owdbytes[16:18], packet.Dst_id)

	binary.LittleEndian.PutUint16(tmp, packet.Nhop)
	copy(owdbytes[18:20], tmp[2:])
	binary.LittleEndian.PutUint64(tmp, packet.Ts_ing1)
	copy(owdbytes[20:26], tmp[2:])
	binary.LittleEndian.PutUint64(tmp, packet.Ts_eg1)
	copy(owdbytes[26:32], tmp[2:])
	binary.LittleEndian.PutUint64(tmp, packet.Ts_is2)
	copy(owdbytes[32:38], tmp[2:])
	binary.LittleEndian.PutUint64(tmp, packet.Ts_es2)
	copy(owdbytes[38:44], tmp[2:])
	binary.LittleEndian.PutUint64(tmp, packet.Ts_ing2)
	copy(owdbytes[44:50], tmp[2:])
	binary.LittleEndian.PutUint64(tmp, packet.Ts_eg2)
	copy(owdbytes[50:56], tmp[2:])

	return owdbytes
}
