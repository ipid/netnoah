package main

import (
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"net"
	"net/netip"
	"os"
	"time"
)

func sendUdpPackets(conn *net.UDPConn, sendBytesNumChan chan uint64) {
	var accumulated uint64 = 0
	payload := make([]byte, 1403)

	for {
		payload[0] = payload[0] + 1

		bytesWritten, err := conn.Write(payload)
		if err != nil {
			panic(err)
		}

		accumulated += uint64(bytesWritten)

		if accumulated > 1000000 {
			sendBytesNumChan <- accumulated
			accumulated = 0
		}
	}
}

func main() {
	var threadNum int
	var dstIpPort, bindSrcIp string

	flag.IntVar(&threadNum, "thread", -1, "How many thread to use for sending UDP packets")
	flag.StringVar(&dstIpPort, "dst", "", "Destination IP:Port")
	flag.StringVar(&bindSrcIp, "src", "", "Bind source IP address")

	flag.Parse()

	if threadNum <= 0 || dstIpPort == "" || bindSrcIp == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Incorrect parameters.\n")
		flag.Usage()

		os.Exit(1)
		return
	}

	bindSrcAddr := net.ParseIP(bindSrcIp)
	if bindSrcAddr == nil {
		_, _ = fmt.Fprintf(os.Stderr, "Invalid bind udpSrc addr: %s\n", bindSrcIp)
		os.Exit(1)
		return
	}

	dstAddrPort, err := netip.ParseAddrPort(dstIpPort)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Invalid target ip: %s\nError while parsing destination IP addr:port: %s\n", dstIpPort, err.Error())
		os.Exit(1)
		return
	}

	sendBytesNumChan := make(chan uint64, threadNum*8)

	// Set timer to print the sending speed every 1 second
	ticker := time.NewTicker(time.Second)

	var accumulatedBytesNum uint64 = 0
	var lastTimestampMs int64 = -1

	fmt.Printf("Sending packets from %s to %s with %d threads...\n", bindSrcIp, dstIpPort, threadNum)

	udpSrc := &net.UDPAddr{
		IP:   bindSrcAddr,
		Port: 0,
	}
	udpDst := net.UDPAddrFromAddrPort(dstAddrPort)

	for i := 0; i < threadNum; i++ {
		conn, err := net.DialUDP("udp", udpSrc, udpDst)
		if err != nil {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"Error while creating connection #%d (total %d) from %s to %s: %s\n", i+1, threadNum, bindSrcIp, dstIpPort, err.Error(),
			)
			os.Exit(1)
			return
		}

		go sendUdpPackets(conn, sendBytesNumChan)
	}

	for {
		select {
		case <-ticker.C:
			if lastTimestampMs > 0 {
				fmt.Printf("Sending speed: %s/s\n", humanize.Bytes(accumulatedBytesNum))
			}

			accumulatedBytesNum = 0
			lastTimestampMs = time.Now().UnixMilli()

		case n := <-sendBytesNumChan:
			accumulatedBytesNum += n
		}
	}
}
