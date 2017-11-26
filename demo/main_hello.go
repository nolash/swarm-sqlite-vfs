package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"

	"./hello"
)

func main() {

	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))

	datadir, err := ioutil.TempDir("", "bzzvfs-hello")
	if err != nil {
		log.Crit(err.Error())
	}
	defer os.RemoveAll(datadir)

	dpa, err := storage.NewLocalDPA(datadir, "SHA3")
	if err != nil {
		log.Crit(err.Error())
	}
	dpa.Start()
	defer dpa.Stop()
	if !hello.Init(dpa) {
		log.Crit("init fail")
	}

	buf := bytes.NewBufferString("foobar")
	swg := &sync.WaitGroup{}
	wwg := &sync.WaitGroup{}
	key, err := dpa.Store(buf, int64(len(buf.Bytes())), swg, wwg)
	if err != nil {
		log.Crit(err.Error())
	}
	log.Debug("store", "key", key)

	err = hello.Open(key)
	if err != nil {
		log.Crit("open fail", "err", err)
	}
}
