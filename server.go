package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type errors []error

func (e errors) Error() string {
	var errs []string
	for _, err := range e {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, ";")
}

// Server handles the client connections.
type Server struct {
	mutex     *sync.RWMutex
	ticker    *time.Ticker
	waitgroup *sync.WaitGroup

	clients []*Client
}

func NewServer(reap time.Duration) *Server {
	s := &Server{
		mutex:     &sync.RWMutex{},
		ticker:    time.NewTicker(reap),
		waitgroup: &sync.WaitGroup{},
	}

	go func() {
		for range s.ticker.C {
			s.Reap()
		}
	}()

	return s
}

func (s *Server) Register(client *Client) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, c := range s.clients {
		if c.String() == client.String() {
			return fmt.Errorf("client already registered")
		}
	}

	s.clients = append(s.clients, client)

	return nil
}

func (s *Server) Reap() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i := 0; i < len(s.clients); i++ {
		if s.clients[i].err == nil {
			continue
		}
		s.clients, s.clients[len(s.clients)-1] = append(s.clients[:i], s.clients[i+1:]...), nil
	}
}

func (s *Server) Send(msg []byte) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	res := make(chan error)
	for _, c := range s.clients {
		s.waitgroup.Add(1)
		go func(client *Client) {
			defer s.waitgroup.Done()
			res <- c.Send(msg)
		}(c)
	}

	var errs []error
	go func() {
		for err := range res {
			errs = append(errs, err)
		}
	}()

	s.waitgroup.Wait()

	close(res)

	if len(errs) > 0 {
		return errors(errs)
	}

	return nil
}

func (s *Server) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ticker.Stop()
	for _, c := range s.clients {
		c.conn.Close()
	}
}
