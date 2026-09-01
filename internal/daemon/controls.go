package daemon

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/aayush/torrcli/internal/state"
)

func (d *Daemon) pauseTorrent(ctx context.Context, id model.TorrentID) (model.TorrentSnapshot, error) {
	if err := d.engine.Pause(ctx, id); err != nil {
		return model.TorrentSnapshot{}, err
	}
	return d.finishControl(ctx, id, func(record *model.TorrentRecord) { record.DesiredState = model.TorrentStatePaused })
}

func (d *Daemon) resumeTorrent(ctx context.Context, id model.TorrentID) (model.TorrentSnapshot, error) {
	if err := d.engine.Resume(ctx, id); err != nil {
		return model.TorrentSnapshot{}, err
	}
	return d.finishControl(ctx, id, func(record *model.TorrentRecord) { record.DesiredState = model.TorrentStateDownloading })
}

func (d *Daemon) setFilePriority(ctx context.Context, params rpc.SetFilePriorityParams) (model.TorrentSnapshot, error) {
	if err := d.engine.SetFilePriority(ctx, params.ID, params.FileIndex, params.Priority); err != nil {
		return model.TorrentSnapshot{}, err
	}
	return d.finishControl(ctx, params.ID, func(record *model.TorrentRecord) {
		if record.FilePriorities == nil {
			record.FilePriorities = make(map[string]model.FilePriority)
		}
		record.FilePriorities[strconv.Itoa(params.FileIndex)] = params.Priority
	})
}

func (d *Daemon) finishControl(ctx context.Context, id model.TorrentID, update func(*model.TorrentRecord)) (model.TorrentSnapshot, error) {
	torrent, err := d.engine.Snapshot(ctx, id)
	if err != nil {
		return model.TorrentSnapshot{}, err
	}
	if err := d.updateRecord(id, update); err != nil {
		return model.TorrentSnapshot{}, err
	}
	if err := d.torrents.put(ctx, torrent); err != nil {
		return model.TorrentSnapshot{}, err
	}
	return torrent, nil
}

func (d *Daemon) removeTorrent(ctx context.Context, params rpc.RemoveTorrentParams) error {
	if err := d.engine.Remove(ctx, params.ID, params.DeleteData); err != nil {
		return err
	}
	if err := d.removeRecord(params.ID); err != nil {
		return err
	}
	return d.torrents.remove(ctx, params.ID)
}

func (d *Daemon) updateRecord(id model.TorrentID, update func(*model.TorrentRecord)) error {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session := cloneSession(d.session)
	record, ok := session.Torrents[id]
	if !ok {
		return fmt.Errorf("torrent %q not found", id)
	}
	update(&record)
	session.Torrents[id] = record
	if err := state.SaveSession(d.paths.SessionFile, session); err != nil {
		return err
	}
	d.session = session
	return nil
}

func (d *Daemon) removeRecord(id model.TorrentID) error {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session := cloneSession(d.session)
	if _, ok := session.Torrents[id]; !ok {
		return fmt.Errorf("torrent %q not found", id)
	}
	delete(session.Torrents, id)
	for index, currentID := range session.Order {
		if currentID == id {
			session.Order = append(session.Order[:index], session.Order[index+1:]...)
			break
		}
	}
	if err := state.SaveSession(d.paths.SessionFile, session); err != nil {
		return err
	}
	d.session = session
	return nil
}

func validFilePriority(priority model.FilePriority) bool {
	return priority == model.FilePrioritySkip || priority == model.FilePriorityNormal || priority == model.FilePriorityHigh
}
