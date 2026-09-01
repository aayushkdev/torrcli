package daemon

import (
	"context"

	"github.com/aayush/torrcli/internal/model"
)

func (d *Daemon) reconcileSchedule(ctx context.Context) {
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	torrents, err := d.torrents.list(ctx)
	if err != nil {
		return
	}
	d.sessionMu.Lock()
	session := cloneSession(d.session)
	d.sessionMu.Unlock()
	downloads, seeds := 0, 0
	for _, torrent := range torrents {
		record, ok := session.Torrents[torrent.ID]
		if !ok {
			continue
		}
		if record.DesiredState == model.TorrentStatePaused {
			_ = d.setScheduled(ctx, torrent.ID, false)
			if updated, snapshotErr := d.engine.Snapshot(ctx, torrent.ID); snapshotErr == nil {
				torrent = updated
			}
			torrent.State = model.TorrentStatePaused
			_ = d.torrents.put(ctx, torrent)
			continue
		}
		isSeed := torrent.Progress >= 1
		limit, active := d.config.MaxActiveDownloads, &downloads
		if isSeed {
			limit, active = d.config.MaxActiveSeeds, &seeds
		}
		if limit > 0 && *active >= limit {
			_ = d.setScheduled(ctx, torrent.ID, false)
			if updated, snapshotErr := d.engine.Snapshot(ctx, torrent.ID); snapshotErr == nil {
				torrent = updated
			}
			torrent.State = model.TorrentStateQueued
			_ = d.torrents.put(ctx, torrent)
			continue
		}
		if d.setScheduled(ctx, torrent.ID, true) != nil {
			continue
		}
		if updated, snapshotErr := d.engine.Snapshot(ctx, torrent.ID); snapshotErr == nil {
			torrent = updated
		}
		(*active)++
		_ = d.torrents.put(ctx, torrent)
	}
}

func (d *Daemon) setScheduled(ctx context.Context, id model.TorrentID, running bool) error {
	if current, ok := d.scheduled[id]; ok && current == running {
		return nil
	}
	var err error
	if running {
		err = d.engine.Resume(ctx, id)
	} else {
		err = d.engine.Pause(ctx, id)
	}
	if err == nil {
		d.scheduled[id] = running
	}
	return err
}
