package daemon

import (
	"context"
	"fmt"

	"github.com/aayush/torrcli/internal/model"
)

type torrentSession struct {
	commands chan sessionCommand
}

type sessionCommand struct {
	apply func(*sessionState) error
	done  chan error
}

type sessionState struct {
	torrents map[model.TorrentID]model.TorrentSnapshot
	order    []model.TorrentID
}

func newTorrentSession(ctx context.Context) *torrentSession {
	session := &torrentSession{commands: make(chan sessionCommand)}
	go func() {
		state := sessionState{torrents: make(map[model.TorrentID]model.TorrentSnapshot)}
		for {
			select {
			case <-ctx.Done():
				return
			case command := <-session.commands:
				command.done <- command.apply(&state)
			}
		}
	}()
	return session
}

func (s *torrentSession) put(ctx context.Context, torrent model.TorrentSnapshot) error {
	return s.run(ctx, func(state *sessionState) error {
		if _, ok := state.torrents[torrent.ID]; !ok {
			state.order = append(state.order, torrent.ID)
		}
		state.torrents[torrent.ID] = torrent
		return nil
	})
}

func (s *torrentSession) get(ctx context.Context, id model.TorrentID) (model.TorrentSnapshot, error) {
	var torrent model.TorrentSnapshot
	err := s.run(ctx, func(state *sessionState) error {
		var ok bool
		torrent, ok = state.torrents[id]
		if !ok {
			return fmt.Errorf("torrent %q not found", id)
		}
		return nil
	})
	return torrent, err
}

func (s *torrentSession) list(ctx context.Context) ([]model.TorrentSnapshot, error) {
	var result []model.TorrentSnapshot
	err := s.run(ctx, func(state *sessionState) error {
		result = make([]model.TorrentSnapshot, 0, len(state.order))
		for _, id := range state.order {
			result = append(result, state.torrents[id])
		}
		return nil
	})
	return result, err
}

func (s *torrentSession) remove(ctx context.Context, id model.TorrentID) error {
	return s.run(ctx, func(state *sessionState) error {
		if _, ok := state.torrents[id]; !ok {
			return fmt.Errorf("torrent %q not found", id)
		}
		delete(state.torrents, id)
		for index, currentID := range state.order {
			if currentID == id {
				state.order = append(state.order[:index], state.order[index+1:]...)
				break
			}
		}
		return nil
	})
}

func (s *torrentSession) move(ctx context.Context, id model.TorrentID, offset int) error {
	return s.run(ctx, func(state *sessionState) error {
		index := -1
		for i, current := range state.order {
			if current == id {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("torrent %q not found", id)
		}
		target := index + offset
		if target < 0 || target >= len(state.order) {
			return nil
		}
		state.order[index], state.order[target] = state.order[target], state.order[index]
		return nil
	})
}

func (s *torrentSession) run(ctx context.Context, apply func(*sessionState) error) error {
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
