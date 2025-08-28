package inventory

import "time"

type FileInfo struct {
	ModifiedTime time.Time `json:"modified_time"`
	Size         int64     `json:"size"`

	tape *Tape
	name string
}

func (i *Inventory) GetBestFiles() map[string]*FileInfo {
	files := make(map[string]*FileInfo)
	for _, tape := range i.tapes {
		for name, info := range tape.Files {
			oldInfo, ok := files[name]
			if !ok || info.IsBetterThan(oldInfo) {
				files[name] = info
			}
		}
	}

	for name, file := range files {
		if file.IsTombstone() {
			delete(files, name)
		}
	}

	return files
}

func (i *Inventory) GetBestFilesOn(barcode string) map[string]*FileInfo {
	files := i.GetBestFiles()
	for name, info := range files {
		if info.tape.Barcode != barcode {
			delete(files, name)
		}
	}
	return files
}

func (i *Inventory) GetFile(name string) *FileInfo {
	var best *FileInfo
	for _, tape := range i.tapes {
		if info, ok := tape.Files[name]; ok {
			if best == nil || info.IsBetterThan(best) {
				best = info
			}
		}
	}

	if best == nil || best.IsTombstone() {
		return nil
	}
	return best
}

func (i *FileInfo) IsTombstone() bool {
	return i.Size <= 0
}

func (f *FileInfo) IsBetterThan(other *FileInfo) bool {
	return f.ModifiedTime.After(other.ModifiedTime)
}
