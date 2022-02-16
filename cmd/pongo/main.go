package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
)

func main() {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		log.Println(err)
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	if err != nil {
		log.Println(err)
	}
	addr := syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{0, 0, 0, 0},
	}
	file := os.NewFile(uintptr(fd), "")
	for {
		buf := make([]byte, 84)
		_, err := file.Read(buf)
		if err != nil {
			log.Println(err)
		}
		if bytes.Equal(buf[20:21], []byte{8}) {
			packet := forgedPacket(buf)
			err = syscall.Sendto(fd, packet, 0, &addr)
			if err != nil {
				log.Fatal("Sendto:", err)
			}
		}
	}
}

func forgedPacket(buf []byte) []byte {
	var icmpTime int64
	forgedTime := make([]byte, 4)
	now := time.Now().Unix()
	icmpTime = int64(binary.LittleEndian.Uint32(buf[28:32]))
	fakeTime := icmpTime + (now - icmpTime)
	fmt.Println(fakeTime)
	binary.LittleEndian.PutUint32(forgedTime, uint32(fakeTime))
	copy(buf[20:21], []byte{0})    //reply
	copy(buf[22:24], []byte{0, 0}) //zeroing checksum
	copy(buf[28:32], forgedTime)   // timestamp
	cs := csum(buf[20:])
	copy(buf[22:24], intToBytes(cs)) //insert checksum
	return buf
}

func csum(b []byte) uint16 {
	var s uint32
	for i := 0; i < len(b); i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
	}
	// add back the carry
	s = s>>16 + s&0xffff
	s = s + s>>16
	return uint16(^s)
}
func intToBytes(i uint16) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, i)
	return buf.Bytes()
}
