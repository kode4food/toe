package view

import (
	"os"
	"time"
)

type diskSnapshot struct {
	modTime time.Time
	size    int64
	exists  bool
}

func (d *Document) refreshDiskSnapshot() {
	d.file.snapshot = snapshotDisk(d.Path())
	d.file.external = ExternalStateClean
}

func (d *Document) diskChanged() (diskSnapshot, bool) {
	snap := snapshotDisk(d.Path())
	return snap, snap != d.file.snapshot
}

func snapshotDisk(path string) diskSnapshot {
	if path == "" {
		return diskSnapshot{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return diskSnapshot{}
	}
	return diskSnapshot{
		modTime: info.ModTime(),
		size:    info.Size(),
		exists:  true,
	}
}
