package anacrolix

import (
	"context"

	"github.com/aayush/torrcli/internal/model"
	"github.com/anacrolix/torrent"
)

func (e *Engine) SetSequential(ctx context.Context, id model.TorrentID, enabled bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return err
	}
	entry.sequential = enabled
	updateSequential(&entry)
	e.torrents[id] = entry
	return nil
}

// Boost a bounded window of unfinished, wanted pieces. File priorities remain
// intact so disabling sequential mode or skipping a file restores normal policy.
func updateSequential(entry *torrentEntry) {
	t := entry.torrent
	info := t.Info()
	if info == nil {
		return
	}
	next := make(map[int]torrent.PiecePriority)
	if entry.sequential {
		remaining := max(1, int((16<<20)/info.PieceLength))
		for _, file := range t.Files() {
			if file.Priority() == torrent.PiecePriorityNone {
				continue
			}
			for i := file.BeginPieceIndex(); i < file.EndPieceIndex() && remaining > 0; i++ {
				if _, exists := next[i]; exists || t.Piece(i).State().Complete {
					continue
				}
				priority := torrent.PiecePriorityReadahead
				if len(next) == 0 {
					priority = torrent.PiecePriorityNow
				}
				next[i] = priority
				remaining--
			}
			if remaining == 0 {
				break
			}
		}
	}
	for i := range entry.boosted {
		if _, keep := next[i]; !keep {
			t.Piece(i).SetPriority(torrent.PiecePriorityNone)
		}
	}
	for i, priority := range next {
		if entry.boosted[i] != priority {
			t.Piece(i).SetPriority(priority)
		}
	}
	entry.boosted = next
}
