package main

import (
	"log"
	"os"

	"github.com/FoxDenHome/tapemgr/storage/encryption"
)

func main() {
	log.Printf("Hello from tapemgr!")

	filenameKey, err := os.ReadFile("tapes/filename.key")
	if err != nil {
		log.Fatalf("Failed to read tape filename key: %v", err)
	}

	nameCryptor, err := encryption.NewPathCryptor(filenameKey)
	if err != nil {
		log.Fatalf("Failed to create name cryptor: %v", err)
	}

	mapper, err := encryption.NewMappedCryptor(nil, nameCryptor, "/rootfs", "/mnt/tape")
	if err != nil {
		log.Fatalf("Failed to create mapper: %v", err)
	}

	err = mapper.Encrypt("/rootfs/source/prefix/file.txt")
	if err != nil {
		log.Fatalf("Failed to encrypt file: %v", err)
	}

	err = mapper.Decrypt("/mnt/tape/wN4DUcoWiADikO_-AZIYmA==/bHUrmjmlyltlvQPx2ahjbw==/xmf6scNuarkMoV0sNUudmQ==")
	if err != nil {
		log.Fatalf("Failed to decrypt file: %v", err)
	}
}
