package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aayush/torrcli/internal/rpc"
	"github.com/aayush/torrcli/internal/transport"
)

type Client struct {
	socketPath string
}

func New(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

func (c *Client) Ping(ctx context.Context) (rpc.PingResult, error) {
	var result rpc.PingResult
	if err := c.call(ctx, rpc.MethodDaemonPing, nil, &result); err != nil {
		return rpc.PingResult{}, err
	}
	return result, nil
}

func (c *Client) Info(ctx context.Context) (rpc.DaemonInfo, error) {
	var result rpc.DaemonInfo
	if err := c.call(ctx, rpc.MethodDaemonInfo, nil, &result); err != nil {
		return rpc.DaemonInfo{}, err
	}
	return result, nil
}

func (c *Client) Add(ctx context.Context, params rpc.AddTorrentParams) (rpc.AddTorrentResult, error) {
	var result rpc.AddTorrentResult
	if err := c.call(ctx, rpc.MethodTorrentAdd, params, &result); err != nil {
		return rpc.AddTorrentResult{}, err
	}
	return result, nil
}

func (c *Client) List(ctx context.Context) (rpc.ListTorrentsResult, error) {
	var result rpc.ListTorrentsResult
	if err := c.call(ctx, rpc.MethodTorrentList, nil, &result); err != nil {
		return rpc.ListTorrentsResult{}, err
	}
	return result, nil
}

func (c *Client) Details(ctx context.Context, params rpc.TorrentParams) (rpc.TorrentDetailsResult, error) {
	var result rpc.TorrentDetailsResult
	if err := c.call(ctx, rpc.MethodTorrentGet, params, &result); err != nil {
		return rpc.TorrentDetailsResult{}, err
	}
	return result, nil
}

func (c *Client) Move(ctx context.Context, params rpc.MoveTorrentParams) (rpc.ListTorrentsResult, error) {
	var result rpc.ListTorrentsResult
	if err := c.call(ctx, rpc.MethodTorrentMove, params, &result); err != nil {
		return rpc.ListTorrentsResult{}, err
	}
	return result, nil
}

func (c *Client) Pause(ctx context.Context, params rpc.TorrentParams) (rpc.TorrentResult, error) {
	return c.torrentCall(ctx, rpc.MethodTorrentPause, params)
}
func (c *Client) Resume(ctx context.Context, params rpc.TorrentParams) (rpc.TorrentResult, error) {
	return c.torrentCall(ctx, rpc.MethodTorrentResume, params)
}
func (c *Client) SetFilePriority(ctx context.Context, params rpc.SetFilePriorityParams) (rpc.TorrentResult, error) {
	return c.torrentCall(ctx, rpc.MethodTorrentSetFilePriority, params)
}
func (c *Client) Remove(ctx context.Context, params rpc.RemoveTorrentParams) error {
	var result struct{}
	return c.call(ctx, rpc.MethodTorrentRemove, params, &result)
}
func (c *Client) torrentCall(ctx context.Context, method string, params any) (rpc.TorrentResult, error) {
	var result rpc.TorrentResult
	if err := c.call(ctx, method, params, &result); err != nil {
		return rpc.TorrentResult{}, err
	}
	return result, nil
}

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	connection, err := transport.DialContext(ctx, c.socketPath)
	if err != nil {
		return fmt.Errorf("connect to torrd: %w", err)
	}
	defer connection.Close()

	request := rpc.Request{JSONRPC: rpc.Version, ID: json.RawMessage("1"), Method: method}
	if params != nil {
		encoded, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("encode %s params: %w", method, err)
		}
		request.Params = encoded
	}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return fmt.Errorf("write %s: %w", method, err)
	}

	var response rpc.Response
	if err := json.NewDecoder(bufio.NewReader(connection)).Decode(&response); err != nil {
		return fmt.Errorf("read %s response: %w", method, err)
	}
	if response.Error != nil {
		return fmt.Errorf("%s", response.Error.Message)
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
