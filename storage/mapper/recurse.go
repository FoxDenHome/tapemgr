package mapper

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

type recursionInfo interface {
	IsDir() bool
	Name() string
}

func (m *FileMapper) Backup(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return m.backupFile(target)
	}

	err = m.backupDir(target)
	if err != nil {
		return err
	}

	return m.tombstonePath(target)
}

func (m *FileMapper) backupDir(target string) error {
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		subTarget := filepath.Join(target, entry.Name())
		if entry.IsDir() {
			err = m.backupDir(subTarget)
		} else {
			err = m.backupFile(subTarget)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

type FilterFunc func(path string, info *inventory.FileInfo) bool

type restoreFile struct {
	*inventory.ExtendedFileInfo
	decryptedPath string
}

func (m *FileMapper) Restore(filter FilterFunc, target string) error {
	if !filepath.IsAbs(target) {
		return fmt.Errorf("target path %s is not absolute", target)
	}

	allFileMap := make(map[string]map[string]*inventory.FileInfo)

	allFiles := m.inventory.GetBestFiles()
	for path, file := range allFiles {
		if file.IsTombstone() {
			continue
		}
		decryptedPath := m.path.Decrypt(path)
		if !filter(decryptedPath, file) {
			continue
		}
		tape := file.GetTape()
		if _, ok := allFileMap[tape.Barcode]; !ok {
			allFileMap[tape.Barcode] = make(map[string]*inventory.FileInfo)
		}
		allFileMap[tape.Barcode][decryptedPath] = file
	}

	for barcode, filesMap := range allFileMap {
		log.Printf("Copying from tape %s", barcode)
		tape := m.inventory.GetOrCreateTape(barcode)
		err := m.loadAndMount(tape)
		if err != nil {
			return err
		}

		fileInfos := make([]*restoreFile, 0, len(filesMap))
		for decryptedPath, file := range filesMap {
			var fileInfo *inventory.ExtendedFileInfo
			if DryRun {
				sb, _ := rand.Int(rand.Reader, big.NewInt(1<<32-1))
				sb64 := sb.Int64()
				part := "a"
				if sb64%2 == 1 {
					part = "b"
				}
				fileInfo = &inventory.ExtendedFileInfo{
					Path:       file.GetPath(),
					Partition:  part,
					StartBlock: int(sb64),
				}
			} else {
				fileInfo, err = file.GetExtended(m.drive)
				if err != nil {
					return err
				}
			}
			fileInfos = append(fileInfos, &restoreFile{
				ExtendedFileInfo: fileInfo,
				decryptedPath:    decryptedPath,
			})
		}

		slices.SortFunc(fileInfos, func(a, b *restoreFile) int {
			partitionCmp := strings.Compare(a.Partition, b.Partition)
			if partitionCmp != 0 {
				return partitionCmp
			}
			return a.StartBlock - b.StartBlock
		})

		for _, fileInfo := range fileInfos {
			log.Printf("Copying file path=%s sb=%d part=%s from tape %s", fileInfo.decryptedPath, fileInfo.StartBlock, fileInfo.Partition, barcode)
			if DryRun {
				continue
			}
			err := m.file.DecryptMkdirAll(filepath.Join(m.drive.MountPoint(), fileInfo.Path), filepath.Join(target, fileInfo.decryptedPath))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
