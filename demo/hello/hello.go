package hello

import (
	"fmt"
	"io"
	"reflect"
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
	reader io.ReadSeeker
	size   int64
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
		reader: r,
		size:   sz,
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
	log.Debug(fmt.Sprintf("reporting filesize: %d", chunkFiles[c].size))
	return C.longlong(chunkFiles[c].size)
}

//export GoBzzRead
func GoBzzRead(c_fd C.int, p unsafe.Pointer, amount C.int, offset C.longlong) int64 {

	// check if we have this reader
	c := int(c_fd)
	if !isValidFD(c) {
		return -1
	}

	// seek and retrieve from dpa
	file := chunkFiles[c]
	file.reader.Seek(int64(offset), 0)
	data := make([]byte, amount)
	c, err := file.reader.Read(data)
	if err != nil && err != io.EOF {
		log.Warn("read error", "err", err, "read", c)
		return -1
	}

	// not sure about this pointer handling, looks risky
	var pp []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&pp))
	hdr.Len = int(amount)
	hdr.Data = uintptr(p)
	copy(pp, data[:amount])

	log.Trace(fmt.Sprintf("returning data: '%x'...'%x'", data[:16], data[amount-16:amount]))
	return int64(len(data))
}

// open database with using dpa
func Open(key storage.Key) error {
	bzzhash := C.CString(key.String())
	defer C.free(unsafe.Pointer(bzzhash))
	r := C.bzzvfs_open(bzzhash)
	if r != C.SQLITE_OK {
		fmt.Errorf("sqlite open fail")
	}
	return nil
}

// execute query on database using dpa
func Exec(sql string) error {
	csql := C.CString(sql)
	defer C.free(unsafe.Pointer(csql))
	res := make([]byte, 1024)
	cres := C.CString(string(res))
	defer C.free(unsafe.Pointer(cres))
	r := C.bzzvfs_exec(C.int(len(sql)), csql, 1024, cres)
	if r != C.SQLITE_OK {
		return fmt.Errorf("sqlite exec fail (%d): %s", r, C.GoString(cres))
	}
	return nil
}

// register bzz vfs
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
