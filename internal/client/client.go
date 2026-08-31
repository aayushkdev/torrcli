// Package client provides a typed client for the local torrd RPC API.
package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/aayush/torrcli/internal/rpc"
)

// Client communicates with one local torrd endpoint.
type Client struct {
	socketPath string
}

// New returns a client for socketPath.
func New(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// Ping confirms that torrd is ready to handle requests.
func (c *Client) Ping(ctx context.Context) (rpc.PingResult, error) {
	var result rpc.PingResult
	if err := c.call(ctx, rpc.MethodDaemonPing, &result); err != nil {
		return rpc.PingResult{}, err
	}
	return result, nil
}

// Info returns the current daemon metadata.
func (c *Client) Info(ctx context.Context) (rpc.DaemonInfo, error) {
	var result rpc.DaemonInfo
	if err := c.call(ctx, rpc.MethodDaemonInfo, &result); err != nil {
		return rpc.DaemonInfo{}, err
	}
	return result, nil
}

func (c *Client) call(ctx context.Context, method string, result any) error {
	connection, err := (&net.Dialer{}).DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("connect to torrd: %w", err)
	}
	defer connection.Close()

	request := rpc.Request{JSONRPC: rpc.Version, ID: json.RawMessage("1"), Method: method}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return fmt.Errorf("write %s: %w", method, err)
	}

	var response rpc.Response
	if err := json.NewDecoder(bufio.NewReader(connection)).Decode(&response); err != nil {
		return fmt.Errorf("read %s response: %w", method, err)
	}
	if response.Error != nil {
		return fmt.Errorf("%s: %s", method, response.Error.Message)
	}
	encoded, err := json.Marshal(response.Result)
	if err != nil {
		return fmt.Errorf("encode %s result: %w", method, err)
	}
	if err := json.Unmarshal(encoded, result); err != nil {
		return fmt.Errorf("decode %s result: %w", method, err)
	}
	return nil
}
