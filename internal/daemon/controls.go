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
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	torrent, err := d.torrents.get(ctx, id)
	if err != nil {
		return model.TorrentSnapshot{}, err
	}
	if err = d.engine.Pause(ctx, id); err != nil {
		return model.TorrentSnapshot{}, err
	}
	delete(d.scheduled, id)
	if err = d.updateRecord(context.WithoutCancel(ctx), id, func(record *model.TorrentRecord) { record.DesiredState = model.TorrentStatePaused }); err != nil {
		_ = d.engine.Resume(context.WithoutCancel(ctx), id)
		return model.TorrentSnapshot{}, err
	}
	torrent.State = model.TorrentStatePaused
	_ = d.torrents.put(context.WithoutCancel(ctx), torrent)
	return torrent, nil
}

func (d *Daemon) resumeTorrent(ctx context.Context, id model.TorrentID) (model.TorrentSnapshot, error) {
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	torrent, err := d.torrents.get(ctx, id)
	if err != nil {
		return model.TorrentSnapshot{}, err
	}
	if err = d.engine.Resume(ctx, id); err != nil {
		return model.TorrentSnapshot{}, err
	}
	delete(d.scheduled, id)
	if err = d.updateRecord(context.WithoutCancel(ctx), id, func(record *model.TorrentRecord) { record.DesiredState = model.TorrentStateDownloading }); err != nil {
		_ = d.engine.Pause(context.WithoutCancel(ctx), id)
		return model.TorrentSnapshot{}, err
	}
	torrent.State = model.TorrentStateDownloading
	_ = d.torrents.put(context.WithoutCancel(ctx), torrent)
	return torrent, nil
}

func (d *Daemon) setFilePriority(ctx context.Context, params rpc.SetFilePriorityParams) (model.TorrentSnapshot, error) {
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	torrent, err := d.torrents.get(ctx, params.ID)
	if err != nil {
		return model.TorrentSnapshot{}, err
	}
	previous, err := d.filePriority(params.ID, params.FileIndex)
	if err != nil {
		return model.TorrentSnapshot{}, err
	}
	if err = d.engine.SetFilePriority(ctx, params.ID, params.FileIndex, params.Priority); err != nil {
		return model.TorrentSnapshot{}, err
	}
	if err = d.updateRecord(context.WithoutCancel(ctx), params.ID, func(record *model.TorrentRecord) {
		if record.FilePriorities == nil {
			record.FilePriorities = make(map[string]model.FilePriority)
		}
		record.FilePriorities[strconv.Itoa(params.FileIndex)] = params.Priority
	}); err != nil {
		_ = d.engine.SetFilePriority(context.WithoutCancel(ctx), params.ID, params.FileIndex, previous)
		return model.TorrentSnapshot{}, err
	}
	return torrent, nil
}

func (d *Daemon) removeTorrent(ctx context.Context, params rpc.RemoveTorrentParams) error {
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	previous, err := d.removeRecord(context.WithoutCancel(ctx), params.ID)
	if err != nil {
		return err
	}
	if err = d.engine.Remove(ctx, params.ID, params.DeleteData); err != nil {
		_ = d.replaceSession(context.WithoutCancel(ctx), previous)
		return err
	}
	delete(d.scheduled, params.ID)
	return d.torrents.remove(context.WithoutCancel(ctx), params.ID)
}

func (d *Daemon) moveTorrent(ctx context.Context, id model.TorrentID, offset int) ([]model.TorrentSnapshot, error) {
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	d.sessionMu.Lock()
	previous := cloneSession(d.session)
	session := cloneSession(d.session)
	index := -1
	for i, current := range session.Order {
		if current == id {
			index = i
			break
		}
	}
	if index < 0 {
		d.sessionMu.Unlock()
		return nil, fmt.Errorf("torrent %q not found", id)
	}
	target := index + offset
	if target >= 0 && target < len(session.Order) {
		session.Order[index], session.Order[target] = session.Order[target], session.Order[index]
	}
	if err := state.SaveSession(d.paths.SessionFile, session); err != nil {
		d.sessionMu.Unlock()
		return nil, err
	}
	d.session = session
	d.sessionMu.Unlock()
	if err := d.torrents.move(context.WithoutCancel(ctx), id, offset); err != nil {
		_ = d.replaceSession(context.WithoutCancel(ctx), previous)
		return nil, err
	}
	return d.torrents.list(ctx)
}

func (d *Daemon) filePriority(id model.TorrentID, index int) (model.FilePriority, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	record, ok := d.session.Torrents[id]
	if !ok {
		return "", fmt.Errorf("torrent %q not found", id)
	}
	if priority, ok := record.FilePriorities[strconv.Itoa(index)]; ok {
		return priority, nil
	}
	return model.FilePriorityNormal, nil
}

func (d *Daemon) updateRecord(_ context.Context, id model.TorrentID, update func(*model.TorrentRecord)) error {
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

func (d *Daemon) removeRecord(_ context.Context, id model.TorrentID) (model.Session, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	previous := cloneSession(d.session)
	session := cloneSession(d.session)
	if _, ok := session.Torrents[id]; !ok {
		return model.Session{}, fmt.Errorf("torrent %q not found", id)
	}
	delete(session.Torrents, id)
	for i, current := range session.Order {
		if current == id {
			session.Order = append(session.Order[:i], session.Order[i+1:]...)
			break
		}
	}
	if err := state.SaveSession(d.paths.SessionFile, session); err != nil {
		return model.Session{}, err
	}
	d.session = session
	return previous, nil
}

func (d *Daemon) replaceSession(_ context.Context, session model.Session) error {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	if err := state.SaveSession(d.paths.SessionFile, session); err != nil {
		return err
	}
	d.session = session
	return nil
}

func validFilePriority(priority model.FilePriority) bool {
	return priority == model.FilePrioritySkip || priority == model.FilePriorityNormal || priority == model.FilePriorityHigh
}
