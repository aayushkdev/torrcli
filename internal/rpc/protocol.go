// Package rpc implements the local newline-delimited JSON-RPC protocol.
package rpc

import (
	"encoding/json"
	"time"
)

const Version = "2.0"

const (
	MethodDaemonPing = "daemon.ping"
	MethodDaemonInfo = "daemon.info"
)

// Request is a JSON-RPC request sent by a client.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response sent by the daemon.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a stable JSON-RPC error payload.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInternalError  = -32603
)

// PingResult confirms that a daemon is ready to accept requests.
type PingResult struct {
	ProtocolVersion string `json:"protocol_version"`
}

// DaemonInfo describes a running daemon instance.
type DaemonInfo struct {
	DaemonVersion   string    `json:"daemon_version"`
	ProtocolVersion string    `json:"protocol_version"`
	StartedAt       time.Time `json:"started_at"`
	ConfigFile      string    `json:"config_file"`
	SessionFile     string    `json:"session_file"`
	SocketPath      string    `json:"socket_path"`
}
