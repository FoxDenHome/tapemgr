package main

import (
	"log"
	"os"

	"github.com/FoxDenHome/tapemgr/storage/encryption"
)

func main() {
	log.Printf("Hello from tapemgr!")

	filenameKey, err := os.ReadFile("tapes/tape-filename.key")
	if err != nil {
		log.Fatalf("Failed to read tape filename key: %v", err)
	}

	nameCryptor, err := encryption.NewNameCryptor(filenameKey)
	if err != nil {
		log.Fatalf("Failed to create name cryptor: %v", err)
	}

	decryptedName := nameCryptor.Decrypt(os.Args[1])
	reEncryptedName := nameCryptor.Encrypt(decryptedName)
	log.Printf("DEC = |%s|", decryptedName)
	log.Printf("ENC = |%s| (%v)", reEncryptedName, reEncryptedName == os.Args[1])
}
