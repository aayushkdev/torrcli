package rpc

import (
	"encoding/json"
	"time"

	"github.com/aayush/torrcli/internal/model"
)

const Version = "2.0"

const (
	MethodDaemonPing             = "daemon.ping"
	MethodDaemonInfo             = "daemon.info"
	MethodTorrentAdd             = "torrent.add"
	MethodTorrentList            = "torrent.list"
	MethodTorrentPause           = "torrent.pause"
	MethodTorrentResume          = "torrent.resume"
	MethodTorrentRemove          = "torrent.remove"
	MethodTorrentSetFilePriority = "torrent.set_file_priority"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

type PingResult struct {
	ProtocolVersion string `json:"protocol_version"`
}

type DaemonInfo struct {
	DaemonVersion   string    `json:"daemon_version"`
	ProtocolVersion string    `json:"protocol_version"`
	StartedAt       time.Time `json:"started_at"`
	ConfigFile      string    `json:"config_file"`
	SessionFile     string    `json:"session_file"`
	SocketPath      string    `json:"socket_path"`
}

type AddTorrentParams struct {
	Source   string `json:"source"`
	SavePath string `json:"save_path"`
}

type AddTorrentResult struct {
	Torrent model.TorrentSnapshot `json:"torrent"`
}

type ListTorrentsResult struct {
	Torrents []model.TorrentSnapshot `json:"torrents"`
}

type TorrentParams struct {
	ID model.TorrentID `json:"id"`
}
type RemoveTorrentParams struct {
	ID         model.TorrentID `json:"id"`
	DeleteData bool            `json:"delete_data"`
}
type SetFilePriorityParams struct {
	ID        model.TorrentID    `json:"id"`
	FileIndex int                `json:"file_index"`
	Priority  model.FilePriority `json:"priority"`
}
type TorrentResult struct {
	Torrent model.TorrentSnapshot `json:"torrent"`
}
