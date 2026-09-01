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
			_ = d.engine.Pause(ctx, torrent.ID)
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
			_ = d.engine.Pause(ctx, torrent.ID)
			torrent.State = model.TorrentStateQueued
			_ = d.torrents.put(ctx, torrent)
			continue
		}
		_ = d.engine.Resume(ctx, torrent.ID)
		if updated, snapshotErr := d.engine.Snapshot(ctx, torrent.ID); snapshotErr == nil {
			torrent = updated
		}
		(*active)++
		_ = d.torrents.put(ctx, torrent)
	}
}
