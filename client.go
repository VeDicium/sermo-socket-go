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

	Routes []Route

	reader   *bufio.Reader
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

	// Create Reader and WaitGroup
	c.reader = bufio.NewReader(c.Conn)
	c.routines = &sync.WaitGroup{}

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

func (c Client) handleRequest(request *Request) {
	c.routines.Add(1)
	go func(wg *sync.WaitGroup) {
		// Set done on end
		defer wg.Done()

		// Print request
		c.Printf("%+v", request)

		// Get routes
		route := c.matchRoute(*request)

		// 404 when route not found
		if route == nil {
			c.Write(Response{
				Type:      "request",
				URL:       request.URL,
				RequestID: request.RequestID,
				Code:      404,
				Data: map[string]interface{}{
					"error": "NotFound",
				},
			})
		} else {
			// Add params
			if len(route.Params) > 0 {
				request.Params = map[string]interface{}{}
				matches := route.urlRegex().FindSubmatch([]byte(request.URL))
				for idx, param := range route.Params {
					if len(matches) >= idx {
						request.Params[param] = string(matches[idx+1])
					}
				}
			}

			route.RouteFunction(*request, Response{
				Type:      "request",
				URL:       route.URL,
				RequestID: request.RequestID,
				Client:    c,
			})
		}
	}(c.routines)
}

func (c Client) Read() (*Request, error) {
	var bytes []byte
	var request Request
	for {
		// Read until 0x0A (\n)
		ba, isPrefix, err := c.reader.ReadLine()
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
	bytes = append(bytes, 0x0A)

	return c.Conn.Write(bytes)
}

func (c Client) disconnect() {
	c.Printf("Disconnect")
	c.Close()
}

func (c Client) matchRoute(request Request) *Route {
	for _, route := range c.Routes {
		// Check if method match
		if strings.ToLower(route.Method) != strings.ToLower(request.Method) {
			continue
		}

		if route.urlRegex().MatchString(strings.ToLower(request.URL)) == false {
			continue
		}

		return &route
	}

	return nil
}
