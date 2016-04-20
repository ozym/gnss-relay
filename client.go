package main

/*
 * Connect to a remote RTCM streaming service and take a feed.
 *
 * At the same time allow client connections which each receiver a
 * copy of the incoming data feed.
 *
 */

import (
	"net"
)

// Client represents a downstream listener interested the data feed.
type Client struct {
	conn net.Conn
	err  error
}

func (c *Client) String() string {
	return c.conn.RemoteAddr().String()
}

func (c *Client) Send(msg []byte) error {
	if c.err != nil {
		return c.err
	}

	var err error
	if _, err = c.conn.Write(msg); err != nil {
		c.conn.Close()
		c.err = err
	}

	return err
}
