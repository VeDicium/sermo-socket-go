package sermo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Client struct {
	ID uuid.UUID
	net.Conn

	Routes   []Route
	routines *sync.WaitGroup
}
type Clients []Client

func (c Client) Printf(format string, v ...interface{}) {
	log.Printf("Client [%s]: %s", c.ID, fmt.Sprintf(format, v...))
}

func (c Client) Authenticate() error {
	return nil
}

func (c Client) Listen() {
	c.Printf("Listening to new client\n")

	// Authentication
	err := c.Authenticate()
	if err != nil {
		c.Printf("Authentication error: %s\n", err)
		c.disconnect()
		return
	}

	c.Printf("Authenticated client")

	for {
		request, err := c.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Printf("Read error: %s", err)
			continue
		}

		c.handleRequest(request)
	}

	defer c.disconnect()
}

func (c Client) handleRequest(r *Request) {
	c.routines.Add(1)
	go func(wg *sync.WaitGroup) {
		// Set done on end
		defer wg.Done()

		// Print request
		c.Printf("%+v", r)

		// Get routes
		foundRoute := false
		for _, route := range c.Routes {
			if strings.ToLower(r.Method) == strings.ToLower(route.Method) && strings.ToLower(r.URL) == strings.ToLower(route.URL) {
				route.RouteFunction(*r, Response{
					Type:      "request",
					URL:       route.URL,
					RequestID: r.RequestID,
					Client:    c,
				})
				foundRoute = true
			}
		}

		// Did not find route, so send 404
		if foundRoute == false {
			c.Write(Response{
				Type:      "request",
				URL:       r.URL,
				RequestID: r.RequestID,
				Code:      404,
				Data: map[string]interface{}{
					"error": "NotFound",
				},
			})
		}
	}(c.routines)
}

func (c Client) Read() (*Request, error) {
	reader := bufio.NewReader(c.Conn)
	var bytes []byte
	var request Request
	for {
		// Read until 0x0A (\n)
		ba, isPrefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}

		// Append bytes
		bytes = append(bytes, ba...)

		// 0x0A (\n) has occured, so stop reading
		if !isPrefix {
			err = json.Unmarshal(bytes, &request)
			if err != nil {
				return nil, err
			}

			break
		}
	}

	return &request, nil
}

func (c Client) Write(r Response) (n int, err error) {
	// Marshal JSON
	bytes, err := json.Marshal(r)
	if err != nil {
		return 0, nil
	}
	bytes = append(bytes, 4)

	return c.Conn.Write(bytes)
}

func (c Client) disconnect() {
	c.Printf("Disconnect")
	c.Close()
}
