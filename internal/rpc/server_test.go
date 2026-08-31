package rpc_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/aayush/torrcli/internal/rpc"
)

func TestServeConnPing(t *testing.T) {
	t.Parallel()

	serverConnection, clientConnection := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- rpc.ServeConn(context.Background(), serverConnection, func(_ context.Context, request rpc.Request) (any, *rpc.Error) {
			if request.Method != rpc.MethodDaemonPing {
				return nil, rpc.MethodNotFound(request.Method)
			}
			return rpc.PingResult{ProtocolVersion: rpc.Version}, nil
		})
	}()

	if _, err := clientConnection.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"daemon.ping"}` + "\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var response rpc.Response
	if err := json.NewDecoder(bufio.NewReader(clientConnection)).Decode(&response); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected RPC error: %#v", response.Error)
	}

	result, err := json.Marshal(response.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var ping rpc.PingResult
	if err := json.Unmarshal(result, &ping); err != nil {
		t.Fatalf("decode ping result: %v", err)
	}
	if ping.ProtocolVersion != rpc.Version {
		t.Fatalf("protocol version = %q, want %q", ping.ProtocolVersion, rpc.Version)
	}

	if err := clientConnection.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("serve connection: %v", err)
	}
}
