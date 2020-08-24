package sermo

import (
	"log"
	"net"
	"sync"
	"syscall"

	"github.com/google/uuid"
)

var wg sync.WaitGroup

// Server
type Server struct {
	Network string
	Address string

	Clients Clients

	Routines sync.WaitGroup

	Router Routes

	net.Listener
}

// Start server
func (s *Server) Start() error {
	// Unlink address, so we start with a clean slate
	syscall.Unlink(s.Address)

	// Listen to network and address
	server, err := net.Listen(s.Network, s.Address)
	if err != nil {
		return err
	}
	s.Listener = server
	defer s.Close()

	log.Printf("SOCKET: Opened on %s socket at %s\n", s.Network, s.Address)
	for {
		// Accept new connections, dispatching them to echoServer
		// in a goroutine.
		client, err := s.Accept()
		if err != nil {
			log.Printf("Accept error: %s\n", err)
			client.Close()
			break
		}

		_, err = s.Connect(client)
		if err != nil {
			log.Printf("Connection error: %s\n", err)
			client.Close()
			break
		}
	}

	return nil
}

func (s *Server) Connect(connection net.Conn) (*Client, error) {
	Client := Client{
		ID:     uuid.New(),
		Conn:   connection,
		Routes: s.Router,
	}
	s.Clients = append(s.Clients, Client)

	s.Routines.Add(1)
	go func(wg *sync.WaitGroup) {
		// Listen to Client
		Client.Listen()

		// Disconnect client
		err := s.Disconnect(Client)
		if err != nil {
			log.Printf("Remove error: %s", err)
		}

		// Set WaitGroup to done
		wg.Done()
	}(&s.Routines)

	return &Client, nil
}

func (s *Server) Broadcast(r Response) (err error) {
	for _, client := range s.Clients {
		_, err := client.Write(r)
		if err != nil {
			client.Printf("Broadcast error %+v\n", err)
		}
	}

	return nil
}

func (s *Server) Disconnect(c Client) error {
	for idx, client := range s.Clients {
		if client.ID == c.ID {
			s.Clients = append(s.Clients[:idx], s.Clients[idx+1:]...)
			break
		}
	}

	return nil
}
