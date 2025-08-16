package storage

import "time"

type File struct {
	ModifiedTime time.Time `json:"modified_time"`
	Partition    string    `json:"partition"`
	Size         uint64    `json:"size"`
	StartBlock   uint64    `json:"start_block"`
}

type Tape struct {
	Barcode string `json:"barcode"`
	Files   []File `json:"files"`
	Size    uint64 `json:"size"`
	Free    uint64 `json:"free"`
}
