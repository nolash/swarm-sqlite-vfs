package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"

	"./hello"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
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
	buf := bytes.NewBufferString("foobar")
	swg := &sync.WaitGroup{}
	wwg := &sync.WaitGroup{}
	key, err := dpa.Store(buf, int64(len(buf.Bytes())), swg, wwg)
	if err != nil {
		log.Crit(err.Error())
	}
	if !hello.Init() {
		log.Crit("init fail")
	}
	log.Debug("open", "result", hello.Open(key.String()))
}
