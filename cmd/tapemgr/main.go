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

	status, err := sch.ReadElementStatus(0, element.ELEMENT_TYPE_ALL, 0, 100, true, false, true)
	if err != nil {
		log.Fatalf("Failed to read element status: %v", err)
	}

	for _, elem := range status {
		log.Printf("==================")
		log.Printf("Address: %d", elem.Address())
		log.Printf("Element Type: %d", elem.ElementType())
	}
}
