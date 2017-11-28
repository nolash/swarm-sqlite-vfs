package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	sys "golang.org/x/sys/unix"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	sqlite3 "github.com/mattn/go-sqlite3"

	"./hello"
)

const (
	dbDriver  = "sqlite"
	dataCount = 100
	dataSize  = 1024 * 1024
)

var (
	dbFile  = "hello.db"
	diskReq = uint64(((dataSize + 16) * dataCount * 2) + (1024 * 1024)) // allowing 16 bytes pr row extra for sqlite (plenty) plus 1MB for leveldb stuff
)

// If arg is a directory, a new database will be created in that directory
// If arg is a file, it will be used as database file
// If no arg, a new database will be created in the system tmp directory
func main() {

	var err error

	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))

	// check if we have arg, and if it's file or dir
	havedb := false
	var stat sys.Statfs_t
	var datadir string
	var dataurl string
	if len(os.Args) > 1 {
		fi, err := os.Stat(os.Args[1])
		if err != nil {
			log.Crit(err.Error())
		}
		if fi.IsDir() {
			datadir, err = ioutil.TempDir(os.Args[1], "bzzvfs-hello")
			if err != nil {
				log.Crit(err.Error())
			}
			dataurl = filepath.Join(datadir, dbFile)
			defer os.RemoveAll(datadir)
		} else {
			havedb = true
			datadir = filepath.Dir(os.Args[1])
			dataurl = os.Args[1]
			defer os.RemoveAll(filepath.Join(datadir, "chunks"))
		}
	} else {
		// create data dir
		datadir, err = ioutil.TempDir("", "bzzvfs-hello")
		if err != nil {
			log.Crit(err.Error())
		}
		defer os.RemoveAll(datadir)
		dataurl = filepath.Join(datadir, dbFile)
	}

	// check for enough space if we're creating db
	// we need enough for db AND chunks + a little more
	if !havedb {
		sys.Statfs(datadir, &stat)
		diskfree := uint64(stat.Bsize) * uint64(stat.Bavail)
		if diskfree < diskReq {
			log.Crit("Not enough disk space", "datadir", datadir, "need", diskReq, "have", diskfree)
		}
	}

	log.Debug("paths", "datadir", datadir, "db url", dataurl)

	if !havedb {
		// create a database and add some values
		sql.Register(dbDriver, &sqlite3.SQLiteDriver{})
		db, err := sql.Open(dbDriver, dataurl)
		if err != nil {
			log.Crit(err.Error())
		}

		_, err = db.Exec(`CREATE TABLE hello (
			id INT UNSIGNED NOT NULL,
			val BLOB NOT NULL
		);`)
		if err != nil {
			log.Crit(err.Error())
		}

		for i := 0; i < dataCount; i++ {
			data := make([]byte, dataSize)
			c, err := rand.Read(data)
			if err != nil {
				log.Crit(err.Error())
			} else if c < dataSize {
				log.Crit("shortread")
			}
			_, err = db.Exec(`INSERT INTO hello (id, val) VALUES 
				($1, $2)`,
				i,
				data)
			if err != nil {
				log.Crit(err.Error())
			}
			log.Trace("insert", "id", i, "data", fmt.Sprintf("%x", data[:8]))
		}
		err = db.Close()
		if err != nil {
			log.Crit(err.Error())
		}
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
	r, err := os.Open(dataurl)
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
