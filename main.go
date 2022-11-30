package main

import (
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/libp2p/go-reuseport"
	"net"
	"os"
	"time"
)

func sendUdpPackets(conn net.Conn, sendBytesNumChan chan uint64) {
	var accumulated uint64 = 0
	payload := make([]byte, 1453)

	for {
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
	var dstIpPort, srcIpPort string

	flag.IntVar(&threadNum, "thread", -1, "How many thread to use for sending UDP packets")
	flag.StringVar(&dstIpPort, "dst", "", "Destination IP:Port")
	flag.StringVar(&srcIpPort, "src", "", "Bind source IP:Port")

	flag.Parse()

	if threadNum <= 0 || dstIpPort == "" || srcIpPort == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Incorrect parameters.\n")
		flag.Usage()

		os.Exit(1)
		return
	}

	sendBytesNumChan := make(chan uint64, threadNum*8)

	// Set timer to print the sending speed every 1 second
	ticker := time.NewTicker(time.Second)

	var accumulatedBytesNum uint64 = 0
	var lastTimestampMs int64 = -1

	fmt.Printf("Sending packets from %s to %s with %d threads...\n", srcIpPort, dstIpPort, threadNum)

	for i := 0; i < threadNum; i++ {
		conn, err := reuseport.Dial("udp", srcIpPort, dstIpPort)
		if err != nil {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"Error while creating connection #%d (total %d) from %s to %s: %s\n", i+1, threadNum, srcIpPort, dstIpPort, err.Error(),
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
