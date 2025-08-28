package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/scsi/loader"
	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/FoxDenHome/tapemgr/storage/inventory"
	"github.com/FoxDenHome/tapemgr/storage/mapper"
	"github.com/FoxDenHome/tapemgr/util"
)

var loaderDeviceStr = flag.String("loader-device", "/dev/sch0", "Path to the SCSI tape loader device")
var driveDeviceStr = flag.String("drive-device", "/dev/nst0", "Path to the SCSI tape drive device")
var tapeMount = flag.String("tape-mount", "/mnt/tape", "Path to the tape mount point")
var tapeFileKey = flag.String("tape-file-key", "tapes/file.key", "Path to the tape file key")
var tapePathKey = flag.String("tape-path-key", "tapes/path.key", "Path to the tape path key")
var cmdMode = flag.String("mode", "help", "Mode to run in (scan, statistics, backup, restore-tape, restore-file, mount, format)")
var dryRun = flag.Bool("dry-run", false, "Dry run mode (do not perform any write operations)")

var fileMapper *mapper.FileMapper
var inv *inventory.Inventory

func main() {
	flag.Parse()
	mapper.DryRun = *dryRun

	log.Printf("tapemgr starting up")

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

	loaderDevice, err := loader.NewTapeLoader(*loaderDeviceStr)
	if err != nil {
		log.Fatalf("Failed to create tape loader: %v", err)
	}

	driveDevice, err := drive.NewTapeDrive(*driveDeviceStr, *tapeMount)
	if err != nil {
		log.Fatalf("Failed to create tape drive: %v", err)
	}

	inv, err = inventory.New()
	if err != nil {
		log.Fatalf("Failed to create inventory: %v", err)
	}

	fileMapper, err = mapper.New(fileCryptor, nameCryptor, inv, loaderDevice, driveDevice, "/")
	if err != nil {
		log.Fatalf("Failed to create mapper: %v", err)
	}

	log.Printf("tapemgr startup done, parsing command")

	switch strings.ToLower(*cmdMode) {
	case "scan":
		defer putLibraryToIdle()

		barcode := flag.Arg(0)
		if barcode == "" {
			log.Fatalf("No barcode provided for scan")
		}

		err := fileMapper.ScanTape(barcode)
		if err != nil {
			log.Fatalf("Failed to scan tape %s: %v", barcode, err)
		}

	case "statistics":
		for _, tape := range inv.GetTapesSortByFreeDesc() {
			log.Printf(
				"Tape: %s, Free: %s / %s (%d%% full)",
				tape.Barcode,
				util.FormatSize(tape.Free),
				util.FormatSize(tape.Size),
				(100*(tape.Size-tape.Free))/tape.Size,
			)
		}

	case "backup":
		defer putLibraryToIdle()

		targets := flag.Args()
		for _, target := range targets {
			err = fileMapper.EncryptRecursive(target)
			if err != nil {
				log.Fatalf("Failed to store %v: %v", target, err)
			}
			err = fileMapper.TombstonePath(target)
			if err != nil {
				log.Fatalf("Failed to create tombstones for %v: %v", target, err)
			}
		}

	case "mount":
		defer putLibraryToIdle()

		barcode := flag.Arg(0)
		if barcode == "" {
			log.Fatalf("No barcode provided for scan")
		}

		err := fileMapper.MountTapeWait(barcode)
		if err != nil {
			log.Fatalf("Failed to mount tape %s: %v", barcode, err)
		}

	case "format":
		defer putLibraryToIdle()

		barcode := flag.Arg(0)
		if barcode == "" {
			log.Fatalf("No barcode provided for format")
		}

		err := fileMapper.FormatTape(barcode)
		if err != nil {
			log.Fatalf("Failed to format tape %s: %v", barcode, err)
		}

	case "restore-tape":
		defer putLibraryToIdle()

		log.Printf("restore-tape TODO")

	case "restore-file":
		defer putLibraryToIdle()

		log.Printf("restore-file TODO")

	case "help":
		flag.Usage()
	default:
		log.Printf("Unknown mode: %v", *cmdMode)
		flag.Usage()
	}
}

func putLibraryToIdle() {
	err := fileMapper.UnmountAndUnload()
	if err != nil {
		log.Printf("Error unmounting and unloading tape: %v", err)
	}
}
