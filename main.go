package main

/*
 * Connect to a remote RTCM streaming service and take a feed.
 *
 * At the same time allow client connections which each receiver a
 * copy of the incoming data feed.
 *
 */

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Simple GNSS relay \n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s [options] <command> [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Extra details")

	var receiver string
	flag.StringVar(&receiver, "receiver", "192.168.59.22:8855", "GNSS receiver end-point")

	var listener string
	flag.StringVar(&listener, "listener", ":8855", "GNSS local listening address")

	var reap time.Duration
	flag.DurationVar(&reap, "reap", time.Minute, "How often stale connections are purged")

	flag.Parse()

	server := NewServer(reap)
	defer server.Close()

	local, err := net.Listen("tcp", listener)
	if err != nil {
		log.Fatalf("Unable to bind to local address: %v", err)
	}
	defer local.Close()

	log.Printf("Listening for connections on: %s", listener)

	go func() {
		for {
			conn, err := local.Accept()
			if err != nil {
				log.Fatalf("Unable to accept local connection: %v", err)
			}
			client := &Client{
				conn: conn,
			}

			log.Printf("Connection from: %s", client.String())
			if err := server.Register(client); err != nil {
				log.Printf("Unable to register connection from: %s", client.String())
				conn.Close()
			}
		}
	}()

	tcpAddress, err := net.ResolveTCPAddr("tcp", receiver)
	if err != nil {
		log.Fatalf("Unable to resolve receiver address: %v", err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddress)
	if err != nil {
		log.Fatalf("Unable to connect to receiver: %v", err)
	}
	defer conn.Close()

	log.Printf("Connected to: %s", receiver)

	var buf []byte
	for {
		blk := make([]byte, 81920)

		// read a packet
		n, err := conn.Read(blk)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error from the receiver: %v", err)
			}
			break
		}

		// append new bytes to existing buffer
		buf = append(buf, blk[0:n]...)

		// check we have the start of a packet
		for len(buf) > 0 {
			if buf[0] == 0xD3 {
				break
			}
			buf = buf[1:]
		}

		// need enough space for the header and crc
		if len(buf) < 6 {
			continue
		}

		var l uint16
		// decode the message length
		if err := binary.Read(bytes.NewReader([]byte{buf[1] & 0x3, buf[2]}), binary.BigEndian, &l); err != nil {
			log.Printf("Unable to decode message length: %v", err)
			buf = buf[1:]
			continue
		}

		// check the available space
		if len(buf) < int(l+6) {
			continue
		}

		var u uint32
		// decode the message checksum
		if err := binary.Read(bytes.NewReader(append([]byte{0}, buf[l+3:l+6]...)), binary.BigEndian, &u); err != nil {
			log.Printf("Unable to decode message checksum: %v", err)
			buf = buf[1:]
			continue
		}

		// check they match ...
		if crcCalc(buf[0:l+3]) != u {
			log.Printf("Invalid message checksum")
			buf = buf[1:]
			continue
		}

		if verbose {
			log.Printf("Message: %08x (%3d)", u, l)
		}

		if err := server.Send(buf[0 : l+6]); err != nil {
			log.Printf("Unable to send some or all messages: %v", err)
		}

		buf = buf[l+6:]
	}

	log.Printf("Terminating")
}
