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
	"time"
)

// Client represents a downstream listener interested the data feed.
type Client struct {
	conn    net.Conn
	timeout time.Duration

	err error
}

func (c *Client) String() string {
	return c.conn.RemoteAddr().String()
}

func (c *Client) Send(msg []byte) error {
	if c.err != nil {
		return c.err
	}

	var err error
	if err = c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		c.conn.Close()
		c.err = err
	}

	if _, err = c.conn.Write(msg); err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			return err
		}
		c.conn.Close()
		c.err = err
	}

	return err
}
