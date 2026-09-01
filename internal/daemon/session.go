package daemon

import (
	"context"
	"fmt"
	"sort"

	"github.com/aayush/torrcli/internal/model"
)

type torrentSession struct {
	commands chan sessionCommand
}

type sessionCommand struct {
	apply func(map[model.TorrentID]model.TorrentSnapshot) error
	done  chan error
}

func newTorrentSession(ctx context.Context) *torrentSession {
	session := &torrentSession{commands: make(chan sessionCommand)}
	go func() {
		torrents := make(map[model.TorrentID]model.TorrentSnapshot)
		for {
			select {
			case <-ctx.Done():
				return
			case command := <-session.commands:
				command.done <- command.apply(torrents)
			}
		}
	}()
	return session
}

func (s *torrentSession) put(ctx context.Context, torrent model.TorrentSnapshot) error {
	return s.run(ctx, func(torrents map[model.TorrentID]model.TorrentSnapshot) error {
		torrents[torrent.ID] = torrent
		return nil
	})
}

func (s *torrentSession) get(ctx context.Context, id model.TorrentID) (model.TorrentSnapshot, error) {
	var torrent model.TorrentSnapshot
	err := s.run(ctx, func(torrents map[model.TorrentID]model.TorrentSnapshot) error {
		var ok bool
		torrent, ok = torrents[id]
		if !ok {
			return fmt.Errorf("torrent %q not found", id)
		}
		return nil
	})
	return torrent, err
}

func (s *torrentSession) list(ctx context.Context) ([]model.TorrentSnapshot, error) {
	var result []model.TorrentSnapshot
	err := s.run(ctx, func(torrents map[model.TorrentID]model.TorrentSnapshot) error {
		result = make([]model.TorrentSnapshot, 0, len(torrents))
		for _, torrent := range torrents {
			result = append(result, torrent)
		}
		sort.Slice(result, func(i, j int) bool {
			if result[i].AddedAt.Equal(result[j].AddedAt) {
				return result[i].ID < result[j].ID
			}
			return result[i].AddedAt.Before(result[j].AddedAt)
		})
		return nil
	})
	return result, err
}

func (s *torrentSession) run(ctx context.Context, apply func(map[model.TorrentID]model.TorrentSnapshot) error) error {
	command := sessionCommand{apply: apply, done: make(chan error, 1)}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.commands <- command:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-command.done:
		return err
	}
}
