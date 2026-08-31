package daemon_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/aayush/torrcli/internal/daemon"
	"github.com/aayush/torrcli/internal/platform"
	"github.com/aayush/torrcli/internal/rpc"
)

func TestDaemonServesPing(t *testing.T) {
	directory := t.TempDir()
	paths := platform.Paths{
		ConfigDir:   filepath.Join(directory, "config"),
		StateDir:    filepath.Join(directory, "state"),
		RuntimeDir:  filepath.Join(directory, "runtime"),
		ConfigFile:  filepath.Join(directory, "config", "config.json"),
		SessionFile: filepath.Join(directory, "state", "session.json"),
		SocketPath:  filepath.Join(directory, "runtime", "torrd.sock"),
		LockFile:    filepath.Join(directory, "runtime", "torrd.lock"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runResult := make(chan error, 1)
	go func() { runResult <- daemon.New(paths).Run(ctx) }()

	connection := waitForSocket(t, paths.SocketPath, runResult)
	defer connection.Close()
	if _, err := connection.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"daemon.ping"}` + "\n")); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	var response rpc.Response
	if err := json.NewDecoder(bufio.NewReader(connection)).Decode(&response); err != nil {
		t.Fatalf("read ping response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected RPC error: %#v", response.Error)
	}

	cancel()
	if err := <-runResult; err != nil {
		t.Fatalf("run daemon: %v", err)
	}
}

func waitForSocket(t *testing.T, socketPath string, runResult <-chan error) net.Conn {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-runResult:
			if errors.Is(err, syscall.EPERM) {
				t.Skip("Unix sockets are unavailable in this test environment")
			}
			t.Fatalf("daemon exited before listening: %v", err)
		default:
		}
		connection, err := net.DialTimeout("unix", socketPath, 50*time.Millisecond)
		if err == nil {
			return connection
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("daemon socket %q did not become ready", socketPath)
	return nil
}
