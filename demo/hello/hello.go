package hello

import (
	"fmt"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

/*
#cgo LDFLAGS: -lsqlite3
#include <stdlib.h>
#include <sqlite3.h>
#include "hello.h"
*/
import "C"

var (
	vfs        *C.struct_sqlite3_vfs
	chunkFiles []*chunkFile
	dpa        *storage.DPA
)

type chunkFile struct {
	keys []*storage.Key
	size int64
}

//export GoBzzOpen
func GoBzzOpen(name *C.char, fd *C.int) C.int {
	hash := common.HexToHash(C.GoString(name))
	key := storage.Key(hash[:])
	log.Debug("retrieve", "key", key, "name", name)
	r := dpa.Retrieve(key)

	sz, err := r.Size(nil)
	if err != nil {
		return 1
	}
	chunkfile := &chunkFile{
		size: sz,
	}
	chunkFiles = append(chunkFiles, chunkfile)
	*fd = C.int(len(chunkFiles) - 1)
	return 0
}

//export GoBzzFileSize
func GoBzzFileSize(c_fd C.int) C.longlong {
	c := int(c_fd)
	if !isValidFD(c) {
		return -1
	}
	return C.longlong(chunkFiles[c].size)
}

//export GoBzzRead
func GoBzzRead(c_fd C.int, p unsafe.Pointer, amount C.int, offset C.longlong) int64 {
	c := int(c_fd)
	if !isValidFD(c) {
		return -1
	}
	// mockdata
	data := []byte("foobar")
	pp := (*[]byte)(p)
	copy(*pp, data)
	// fill rest of buffers with zeros if count mismatch
	for i := len(data); i < int(amount); i++ {
		(*pp) = append((*pp), 0x0)
	}
	return int64(len(data))
}

func Open(key storage.Key) error {
	bzzhash := C.CString(key.String())
	defer C.free(unsafe.Pointer(bzzhash))
	r := C.bzzvfs_open(bzzhash)
	if r != C.SQLITE_OK {
		fmt.Errorf("sqlite open fail")
	}
	return nil
}

func Init(newdpa *storage.DPA) bool {
	if C.bzzvfs_register() != C.SQLITE_OK {
		return false
	}
	dpa = newdpa
	return true
}

func isValidFD(fd int) bool {
	if fd > len(chunkFiles) {
		return false
	}
	return true
}
