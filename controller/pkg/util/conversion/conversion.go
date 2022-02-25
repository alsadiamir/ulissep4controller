package conversion

import (
	"encoding/binary"
	"fmt"
	"net"
)

func IpToBinary(ipStr string) ([]byte, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("not a valid IP: %s", ipStr)
	}
	return []byte(ip.To4()), nil
}

func MacToBinary(macStr string) ([]byte, error) {
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return nil, err
	}
	return []byte(mac), nil
}

func UInt32ToBinary(i uint32, numBytes int) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b[numBytes:], nil
}

func UInt32ToBinaryCompressed(i uint32) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	for idx := 0; idx < 4; idx++ {
		if b[idx] != 0 {
			return b[idx:], nil
		}
	}
	return []byte{'\x00'}, nil
}

func UInt64ToBinaryCompressed(i uint64) ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	for idx := 0; idx < 8; idx++ {
		if b[idx] != 0 {
			return b[idx:], nil
		}
	}
	return []byte{'\x00'}, nil
}

func ToCanonicalBytestring(bytes []byte) []byte {
	if len(bytes) == 0 {
		return bytes
	}
	i := 0
	for _, b := range bytes {
		if b != 0 {
			break
		}
		i++
	}
	if i == len(bytes) {
		return bytes[:1]
	}
	return bytes[i:]
}

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
