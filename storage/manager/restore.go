package manager

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

type FilterFunc func(path string, info inventory.File) bool

type restoreFile struct {
	file          inventory.File
	info          *inventory.FileLTFSInfo
	decryptedPath string
}

func (m *Manager) Restore(filter FilterFunc, target string) error {
	if !filepath.IsAbs(target) {
		return fmt.Errorf("target path %s is not absolute", target)
	}

	allFileMap := make(map[string]map[string]inventory.File)

	allFiles := m.inventory.GetBestFiles(m.path)
	for decryptedPath, file := range allFiles {
		if !filter(decryptedPath, file) {
			continue
		}
		tape := file.GetTape()
		barcode := tape.GetBarcode()
		if _, ok := allFileMap[barcode]; !ok {
			allFileMap[barcode] = make(map[string]inventory.File)
		}
		allFileMap[barcode][decryptedPath] = file
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
			var fileInfo *inventory.FileLTFSInfo
			if DryRun {
				sb, _ := rand.Int(rand.Reader, big.NewInt(1<<32-1))
				sb64 := sb.Int64()
				part := "a"
				if sb64%2 == 1 {
					part = "b"
				}
				fileInfo = &inventory.FileLTFSInfo{
					Partition:  part,
					StartBlock: int(sb64),
				}
			} else {
				fileInfo, err = file.GetLTFSInfo(m.drive)
				if err != nil {
					return err
				}
			}
			fileInfos = append(fileInfos, &restoreFile{
				info:          fileInfo,
				file:          file,
				decryptedPath: decryptedPath,
			})
		}

		slices.SortFunc(fileInfos, func(a, b *restoreFile) int {
			partitionCmp := strings.Compare(a.info.Partition, b.info.Partition)
			if partitionCmp != 0 {
				return partitionCmp
			}
			return a.info.StartBlock - b.info.StartBlock
		})

		for _, fileInfo := range fileInfos {
			filePath := filepath.Join(m.drive.MountPoint(), fileInfo.file.GetPath())
			log.Printf("[COPY] %s", filePath)
			if DryRun {
				continue
			}
			err := m.file.DecryptMkdirAll(filePath, filepath.Join(target, fileInfo.decryptedPath))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
