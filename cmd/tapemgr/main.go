package main

import (
	"log"

	"github.com/FoxDenHome/tapemgr/scsi"
	"github.com/FoxDenHome/tapemgr/scsi/element"
)

func main() {
	log.Printf("Hello from tapemgr!")

	sch, err := scsi.NewSCSIDevice("/dev/sch0")
	if err != nil {
		log.Fatalf("Failed to open SCSI device: %v", err)
	}

	ready, err := sch.TestUnitReady()
	if err != nil {
		log.Fatalf("Failed to check if SCSI device is ready: %v", err)
	}
	log.Printf("SCSI device is ready: %v", ready)

	status, err := sch.ReadElementStatus(element.ELEMENT_TYPE_ALL, 0, 0xFFFF, true, false, false)
	if err != nil {
		log.Fatalf("Failed to read element status: %v", err)
	}

	for _, elem := range status {
		log.Printf("==================")
		log.Printf("Address: %d", elem.Address)
		log.Printf("Element Type: %s", elem.ElementType)
	}
}
