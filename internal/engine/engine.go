package engine

import (
	"context"

	"github.com/aayush/torrcli/internal/model"
)

type Engine interface {
	Add(context.Context, model.AddInput) (model.TorrentID, bool, error)
	Pause(context.Context, model.TorrentID) error
	Resume(context.Context, model.TorrentID) error
	Remove(context.Context, model.TorrentID, bool) error
	SetFilePriority(context.Context, model.TorrentID, int, model.FilePriority) error
	Snapshot(context.Context, model.TorrentID) (model.TorrentSnapshot, error)
	Details(context.Context, model.TorrentID) (model.TorrentDetails, error)
	Events() <-chan model.EngineEvent
	Close() error
}
