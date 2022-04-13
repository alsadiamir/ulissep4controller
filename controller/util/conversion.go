package util

import (
	"encoding/binary"
	"net"
)

func BinaryCompressedToUint64(bytes []byte) uint64 {
	buff := make([]byte, 8)
	offset := 8 - len(bytes)
	for idx, b := range bytes {
		buff[offset+idx] = b
	}
	return binary.BigEndian.Uint64(buff)
}

func BinaryToIpv4(bytes []byte) net.IP {
	if len(bytes) != 4 {
		return net.IPv4(0, 0, 0, 0).To4()
	}
	return net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
}

func BinaryCompressedToUint16(bytes []byte) uint16 {
	if len(bytes) == 2 {
		return binary.BigEndian.Uint16(bytes)
	}
	var zero byte = 0
	return binary.BigEndian.Uint16([]byte{zero, bytes[0]})
}
