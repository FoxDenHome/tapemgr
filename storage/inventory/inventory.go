package inventory

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"

	"github.com/FoxDenHome/tapemgr/storage/encryption"
)

type Inventory struct {
	path string
	db   *sql.DB
}

func New(path string) (*Inventory, error) {
	db, err := sql.Open("duckdb", filepath.Join(path, "inventory.duckdb"))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS files (path VARCHAR, barcode VARCHAR, size BIGINT, modified_time TIMESTAMP WITH TIME ZONE, PRIMARY KEY(path, barcode))")
	if err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS tapes (barcode VARCHAR PRIMARY KEY, size BIGINT, free BIGINT)")
	if err != nil {
		db.Close()
		return nil, err
	}

	inv := &Inventory{
		path: path,
		db:   db,
	}
	return inv, inv.Reload()
}

func (i *Inventory) loadTapeList(suffix string, files []os.DirEntry, loader func(i *Inventory, filename string) error) {
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		barcode := strings.TrimSuffix(name, suffix)
		if barcode == "" {
			continue
		}

		log.Printf("Loading tape inventory file with suffix %s: %s", suffix, name)
		err := loader(i, name)
		if err != nil {
			log.Fatalf("Failed to load tape inventory from %s: %v", name, err)
		}
	}
}

func (i *Inventory) Reload() error {
	files, err := os.ReadDir(i.path)
	if err != nil {
		return err
	}

	i.loadTapeList(".proto", files, loadFromFileProto)

	return nil
}

func (i *Inventory) GetOrCreateTape(barcode string) *Tape {
	tape := &Tape{
		inventory: i,
	}
	row := i.db.QueryRow("SELECT barcode, size, free FROM tapes WHERE barcode = ?", barcode)
	err := row.Scan(&tape.Barcode, &tape.Size, &tape.Free)
	if errors.Is(err, sql.ErrNoRows) {
		tape.Barcode = barcode
		tape.Size = 0
		tape.Free = 0
	} else if err != nil {
		log.Fatalf("failed to scan tape row: %v", err)
	}
	return tape
}

func (i *Inventory) getTapesByQuery(query string, args ...any) []*Tape {
	tapes := make([]*Tape, 0)

	dbTapes, err := i.db.Query(query, args...)
	if err != nil {
		log.Fatalf("failed to query tapes from database: %v", err)
	}

	var barcode string
	var size, free int64
	for dbTapes.Next() {
		if err := dbTapes.Scan(&barcode, &size, &free); err != nil {
			log.Fatalf("failed to scan tape row: %v", err)
		}
		tapes = append(tapes, &Tape{
			inventory: i,
			Barcode:   barcode,
			Size:      size,
			Free:      free,
		})
	}

	return tapes
}

func (i *Inventory) HasTape(barcode string) bool {
	row := i.db.QueryRow("SELECT COUNT(barcode) FROM tapes WHERE barcode = ?", barcode)
	var count int
	err := row.Scan(&count)
	if err != nil {
		log.Fatalf("failed to scan tape count: %v", err)
	}
	return count > 0
}

func (i *Inventory) TapeCount() int {
	row := i.db.QueryRow("SELECT COUNT(barcode) FROM tapes")
	var count int
	err := row.Scan(&count)
	if err != nil {
		log.Fatalf("failed to scan tape count: %v", err)
	}
	return count
}

func (i *Inventory) GetTapesSortByFreeDesc() []*Tape {
	return i.getTapesByQuery("SELECT barcode, size, free FROM tapes ORDER BY free DESC")
}

func (i *Inventory) GetBestFiles(pathCryptor *encryption.PathCryptor) map[string]*File {
	files := make(map[string]*File)

	dbFiles, err := i.db.Query("SELECT path, barcode, size, modified_time FROM files")
	if err != nil {
		log.Fatalf("failed to query files from database: %v", err)
	}

	var path, barcode string
	var size int64
	var modifiedTime time.Time
	for dbFiles.Next() {
		if err := dbFiles.Scan(&path, &barcode, &size, &modifiedTime); err != nil {
			log.Fatalf("failed to scan file row: %v", err)
		}
		clearName, err := pathCryptor.Decrypt(path)
		if err != nil {
			log.Printf("failed to decrypt path %q: %v", path, err)
			continue
		}

		info := &File{
			barcode:      barcode,
			inv:          i,
			path:         path,
			Size:         size,
			ModifiedTime: modifiedTime,
		}

		oldInfo, ok := files[clearName]
		if !ok || info.IsBetterThan(oldInfo) {
			files[clearName] = info
		}
	}

	for name, file := range files {
		if file.isTombstone() {
			delete(files, name)
		}
	}

	return files
}
