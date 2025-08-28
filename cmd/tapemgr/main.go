package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/scsi/loader"
	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/FoxDenHome/tapemgr/storage/inventory"
	"github.com/FoxDenHome/tapemgr/storage/mapper"
)

var loaderDeviceStr = flag.String("loader-device", "/dev/sch0", "Path to the SCSI tape loader device")
var driveDeviceStr = flag.String("drive-device", "/dev/nst0", "Path to the SCSI tape drive device")
var tapeMount = flag.String("tape-mount", "/mnt/tape", "Path to the tape mount point")
var tapeFileKey = flag.String("tape-file-key", "tapes/file.key", "Path to the tape file key")
var tapePathKey = flag.String("tape-path-key", "tapes/path.key", "Path to the tape path key")
var cmdMode = flag.String("mode", "help", "Mode to run in (inventory, statistics, store, copyback)")

var encMapper *mapper.FileMapper
var inv *inventory.Inventory

func main() {
	flag.Parse()

	log.Printf("Hello from tapemgr!")

	driveDevice, err := drive.NewTapeDrive(*driveDeviceStr, *tapeMount)
	if err != nil {
		log.Fatalf("Failed to create tape drive: %v", err)
	}

	loaderDevice, err := loader.NewTapeLoader(*loaderDeviceStr)
	if err != nil {
		log.Fatalf("Failed to create tape loader: %v", err)
	}

	identity, err := os.ReadFile(*tapeFileKey)
	if err != nil {
		log.Fatalf("Failed to read tape file key: %v", err)
	}

	fileCryptor, err := encryption.NewFileCryptor(strings.Trim(string(identity), "\r\t\n "))
	if err != nil {
		log.Fatalf("Failed to create file cryptor: %v", err)
	}

	filenameKey, err := os.ReadFile(*tapePathKey)
	if err != nil {
		log.Fatalf("Failed to read tape path key: %v", err)
	}

	nameCryptor, err := encryption.NewPathCryptor(filenameKey)
	if err != nil {
		log.Fatalf("Failed to create path cryptor: %v", err)
	}

	inv, err = inventory.New()
	if err != nil {
		log.Fatalf("Failed to create inventory: %v", err)
	}

	encMapper, err = mapper.New(fileCryptor, nameCryptor, inv, loaderDevice, driveDevice, "/")
	if err != nil {
		log.Fatalf("Failed to create mapper: %v", err)
	}

	switch strings.ToLower(*cmdMode) {
	case "inventory":
		log.Printf("Inventory loaded with %v", inv.GetBestFilesOn("P0003SL6"))
	case "statistics":
		log.Printf("Statistics TODO")
	case "store":
		targets := flag.Args()
		for _, target := range targets {
			err = storeRecursive(target)
			if err != nil {
				log.Fatalf("Failed to store %v: %v", target, err)
			}
			err = encMapper.TombstonePath(target)
			if err != nil {
				log.Fatalf("Failed to create tombstones for %v: %v", target, err)
			}
		}
	case "copyback":
		log.Printf("Copyback TODO")
	case "help":
		flag.Usage()
	default:
		log.Printf("Unknown mode: %v", *cmdMode)
		flag.Usage()
	}
}

func storeRecursive(target string) error {
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
			err = storeRecursive(filepath.Join(target, entry.Name()))
			if err != nil {
				return err
			}
		}
	} else {
		err = encMapper.Encrypt(target)
		if err != nil {
			return err
		}
	}

	return nil
}
