package main

import (
	"encoding/base64"
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

var fileMapper *mapper.FileMapper

func main() {
	configFile := os.Getenv("TAPEMGR_CONFIG")
	if configFile == "" {
		configFile = "/etc/tapemgr/config.yml"
	}
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config %s: %v", configFile, err)
	}

	loaderDeviceStr := flag.String("loader-device", config.LoaderDevice, "Path to the SCSI tape loader device")
	driveDeviceStr := flag.String("drive-device", config.DriveDevice, "Path to the SCSI tape drive device")
	tapeMount := flag.String("tape-mount", config.TapeMount, "Path to the tape mount point")
	tapesPath := flag.String("tapes-path", config.TapesPath, "Path to the tapes directory")
	cmdMode := flag.String("mode", "help", "Mode to run in (scan, statistics, backup, restore-tape, restore-file, mount, format)")
	dryRun := flag.Bool("dry-run", config.DryRun, "Dry run mode (do not perform any write operations)")
	flag.Parse()
	mapper.DryRun = *dryRun

	log.Printf("tapemgr (version %s / git %s) starting up", util.GetVersion(), util.GetGitRev())

	fileCryptor, err := encryption.NewFileCryptor(config.TapeFileKey)
	if err != nil {
		log.Fatalf("Failed to create file cryptor: %v", err)
	}

	b64decoded, err := base64.StdEncoding.DecodeString(config.TapePathKey)
	if err != nil {
		log.Fatalf("Failed to base64 decode tape path key: %v", err)
	}

	nameCryptor, err := encryption.NewPathCryptor(b64decoded)
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

	inv, err := inventory.New(*tapesPath)
	if err != nil {
		log.Fatalf("Failed to create inventory: %v", err)
	}

	fileMapper, err = mapper.New(fileCryptor, nameCryptor, inv, loaderDevice, driveDevice)
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
			log.Printf("Backing up target %v", target)
			err = fileMapper.Backup(target)
			if err != nil {
				log.Fatalf("Failed to backup %v: %v", target, err)
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

		target := flag.Arg(0)
		tapes := flag.Args()[1:]
		tapesMap := make(map[string]bool)
		for _, tape := range tapes {
			tapesMap[tape] = true
		}

		err := fileMapper.Restore(func(path string, info *inventory.FileInfo) bool {
			return tapesMap[info.GetTape().Barcode]
		}, target)
		if err != nil {
			log.Fatalf("Failed to restore tapes: %v", err)
		}

	case "restore-file":
		defer putLibraryToIdle()

		target := flag.Arg(0)
		files := flag.Args()[1:]
		filesMap := make(map[string]bool)
		for _, file := range files {
			file, _ = util.StripLeadingSlashes(file)
			filesMap[file] = true
		}

		err := fileMapper.Restore(func(path string, info *inventory.FileInfo) bool {
			return filesMap[path]
		}, target)
		if err != nil {
			log.Fatalf("Failed to restore files: %v", err)
		}

	case "help":
		flag.Usage()
		return
	default:
		log.Printf("Unknown mode: %v", *cmdMode)
		flag.Usage()
		return
	}

	log.Printf("tapemgr command done, shutting down")
}

func putLibraryToIdle() {
	err := fileMapper.UnmountAndUnload()
	if err != nil {
		log.Printf("Error unmounting and unloading tape: %v", err)
	}
}
