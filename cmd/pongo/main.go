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

	file := os.NewFile(uintptr(fd), "")
	log.Println("Listening to IPv4 ICMP Packets:")
	for {
		buf := make([]byte, 84)
		_, err := file.Read(buf)
		if err != nil {
			log.Println(err)
		}
		if bytes.Equal(buf[20:21], []byte{8}) {
			packet, date := forgedPacket(buf)
			addr := syscall.SockaddrInet4{
				Port: 0,
				Addr: [4]byte{packet[16], packet[17], packet[18], packet[19]},
			}
			err = syscall.Sendto(fd, packet, 0, &addr)
			if err != nil {
				log.Fatal("Sendto:", err)
			}
			log.Printf("Replied to host %d.%d.%d.%d with date %s", int16(packet[16]), int16(packet[17]), int16(packet[18]), int16(packet[19]), date)
		}
	}
}

func forgedPacket(buf []byte) ([]byte, string) {
	forgedTime := make([]byte, 4)
	src := []byte{buf[12], buf[13], buf[14], buf[15]}
	dst := []byte{buf[16], buf[17], buf[18], buf[19]}
	fakeTime := int64(binary.LittleEndian.Uint32(buf[28:32])) * 1000
	year, month, day := time.Unix(fakeTime, 0).Date()
	date := fmt.Sprintf("%d-%s-%d", day, month, year)
	binary.LittleEndian.PutUint32(forgedTime, uint32(fakeTime))
	copy(buf[12:16], dst)            //destination for ipv4
	copy(buf[16:20], src)            //source for ipv4
	copy(buf[20:21], []byte{0})      //reply
	copy(buf[22:24], []byte{0, 0})   //zeroing checksum
	copy(buf[28:32], forgedTime)     //timestamp
	cs := csum(buf[20:])             //calculate checksum
	copy(buf[22:24], intToBytes(cs)) //insert checksum
	return buf, date
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
