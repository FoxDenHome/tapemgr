package mapper

import (
	"crypto/rand"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

func (m *FileMapper) BackupRecursive(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(target)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			err = m.BackupRecursive(filepath.Join(target, entry.Name()))
			if err != nil {
				return err
			}
		}
	} else {
		err = m.Encrypt(target)
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

func (m *FileMapper) RestoreByFilter(filter FilterFunc, target string) error {
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
			log.Printf("Copying file %s sb=%d part=%s from tape %s", fileInfo.decryptedPath, fileInfo.StartBlock, fileInfo.Partition, barcode)
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
