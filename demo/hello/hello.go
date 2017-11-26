package hello

import (
	"unsafe"
)

/*
#cgo LDFLAGS: -lsqlite3
#include <stdlib.h>
#include <sqlite3.h>
#include "hello.h"
*/
import "C"

var (
	vfs *C.struct_sqlite3_vfs
)

func Init() bool {
	if C.bzzvfs_register() != C.SQLITE_OK {
		return false
	}
	return true
}

func Open(hash string) bool {
	bzzhash := C.CString(hash)
	defer C.free(unsafe.Pointer(bzzhash))
	r := C.bzzvfs_open(bzzhash)
	if r != C.SQLITE_OK {
		return false
	}
	return true
}
