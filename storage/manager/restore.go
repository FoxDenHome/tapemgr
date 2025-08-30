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

type FilterFunc func(path string, info *inventory.File) bool

type restoreFile struct {
	*inventory.ExtendedFile
	decryptedPath string
}

func (m *Manager) Restore(filter FilterFunc, target string) error {
	if !filepath.IsAbs(target) {
		return fmt.Errorf("target path %s is not absolute", target)
	}

	allFileMap := make(map[string]map[string]*inventory.File)

	allFiles := m.inventory.GetBestFiles(m.path)
	for decryptedPath, file := range allFiles {
		if !filter(decryptedPath, file) {
			continue
		}
		tape := file.GetTape()
		if _, ok := allFileMap[tape.Barcode]; !ok {
			allFileMap[tape.Barcode] = make(map[string]*inventory.File)
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
			var fileInfo *inventory.ExtendedFile
			if DryRun {
				sb, _ := rand.Int(rand.Reader, big.NewInt(1<<32-1))
				sb64 := sb.Int64()
				part := "a"
				if sb64%2 == 1 {
					part = "b"
				}
				fileInfo = &inventory.ExtendedFile{
					File:       file,
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
				ExtendedFile:  fileInfo,
				decryptedPath: decryptedPath,
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
			filePath := filepath.Join(m.drive.MountPoint(), fileInfo.GetPath())
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
