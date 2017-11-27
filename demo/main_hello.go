package main

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	sqlite3 "github.com/mattn/go-sqlite3"

	"./hello"
)

const (
	dbFile   = "hello.db"
	dbDriver = "sqlite"
)

func main() {

	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))

	// create data dir
	datadir, err := ioutil.TempDir("", "bzzvfs-hello")
	if err != nil {
		log.Crit(err.Error())
	}
	defer os.RemoveAll(datadir)
	log.Debug("datadir", "path", datadir)

	// create a database and add some values
	sql.Register(dbDriver, &sqlite3.SQLiteDriver{})
	db, err := sql.Open(dbDriver, filepath.Join(datadir, dbFile))
	if err != nil {
		log.Crit(err.Error())
	}
	_, err = db.Exec(`CREATE TABLE hello (
		id INT UNSIGNED NOT NULL,
		val TEXT NOT NULL
	);`)
	if err != nil {
		log.Crit(err.Error())
	}
	_, err = db.Exec(`INSERT INTO hello (id, val) VALUES 
		(42, "foo"),
		(666, "bar")
	`)
	if err != nil {
		log.Crit(err.Error())
	}
	err = db.Close()
	if err != nil {
		log.Crit(err.Error())
	}

	// create the chunkstore and start dpa on it
	dpa, err := storage.NewLocalDPA(filepath.Join(datadir, "chunks"), "SHA3")
	if err != nil {
		log.Crit(err.Error())
	}
	dpa.Start()
	defer dpa.Stop()
	if !hello.Init(dpa) {
		log.Crit("init fail")
	}

	// stick the database in the chunker, muahahaa
	r, err := os.Open(filepath.Join(datadir, dbFile))
	if err != nil {
		log.Crit(err.Error())
	}
	fi, err := r.Stat()
	if err != nil {
		log.Crit(err.Error())
	}
	swg := &sync.WaitGroup{}
	wwg := &sync.WaitGroup{}
	key, err := dpa.Store(r, fi.Size(), swg, wwg)
	if err != nil {
		log.Crit(err.Error())
	}
	log.Debug("store", "key", key)

	// sometimes the dpa in doesn't keep up in the C backend and retrieves 0x0-key,
	// a small delay helps
	time.Sleep(time.Millisecond * 250)

	// test the sqlite_vfs bzz backend:

	// open the database
	err = hello.Open(key)
	if err != nil {
		log.Crit("open fail", "err", err)
	}

	// execute query
	err = hello.Exec("SELECT * FROM hello")
	if err != nil {
		log.Crit("exec fail", "err", err)
	}
}
