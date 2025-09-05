package dl

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"

	"obsessiontech/common/config"
	myCtx "obsessiontech/common/context"
)

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
*/
import "C"

var Config struct {
	LDLibKeepaliveHour time.Duration
}

func init() {
	if err := config.GetConfig("config.yaml", &Config); err != nil {
		panic(err)
	}
}

var e_lib_lock_timeout = errors.New("lib timeout")
var e_lib_expired = errors.New("lib expired")

var lock sync.RWMutex
var dls = make(map[string]*libpack)

type libpack struct {
	Ptr   unsafe.Pointer
	Reset chan byte
	L     sync.RWMutex
}

func (l *libpack) rlock(cancelChan chan<- func()) {
	l.L.RLock()
	defer l.L.RUnlock()

	ctx, cancel := myCtx.GetContext()

	select {
	case cancelChan <- cancel:
	case <-time.After(10 * time.Second):
		log.Println("error send cancel func to chan: timeout")
		cancel()
		return
	}

	<-ctx.Done()
}

func (l *libpack) use(symbol string) (unsafe.Pointer, func(), error) {
	cancelChan := make(chan func())

	go l.rlock(cancelChan)
	var cancel func()
	select {
	case cancel = <-cancelChan:
	case <-time.After(10 * time.Second):
		log.Println("error fail to request lock: timeout")
		return nil, nil, e_lib_lock_timeout
	}

	if l.Reset != nil {
		select {
		case l.Reset <- 1:
		case <-time.After(10 * time.Second):
			log.Println("error fail to reset lib timer. probably expired")
			cancel()
			return nil, nil, e_lib_expired
		}
	}

	cSymbol := C.CString(symbol)
	defer C.free(unsafe.Pointer(cSymbol))

	ptr := C.dlsym(l.Ptr, cSymbol)

	if ptr == nil {
		var err error
		msg := C.dlerror()
		if msg != nil {
			defer C.free(unsafe.Pointer(msg))
			goMsg := C.GoString(msg)
			log.Println("error dl get symbol: ", symbol, goMsg)
			err = errors.New(goMsg)
		} else {
			err = fmt.Errorf("error dl get symbol: nil %s", symbol)
		}
		return nil, nil, err
	}

	return ptr, cancel, nil
}

func OpenDL(libpath, symbol string) (ptr unsafe.Pointer, cancel func(), err error) {

	log.Println("open dl lib: ", libpath, symbol)

	defer func() {
		if e := recover(); e != nil {
			log.Println("fatal error open dl panic: ", libpath, symbol, e)
			err = errors.New("fatal error open dl")
		}
	}()

	defer func() {
		if ptr == nil {
			lock.Lock()
			defer lock.Unlock()

			lib, exists := dls[libpath]
			if exists {
				ptr, cancel, err = lib.use(symbol)
				return
			}

			cLibpath := C.CString(libpath)
			defer C.free(unsafe.Pointer(cLibpath))

			libPtr := C.dlopen(cLibpath, C.RTLD_LAZY)

			if libPtr == nil {
				msg := C.dlerror()
				if msg != nil {
					defer C.free(unsafe.Pointer(msg))
					goMsg := C.GoString(msg)
					log.Println("error dl open: ", libpath, goMsg)
					err = errors.New(goMsg)
				} else {
					err = fmt.Errorf("error dlopen return nil: %s", libpath)
				}
				return
			}

			lib = new(libpack)

			lib.Ptr = libPtr

			if Config.LDLibKeepaliveHour > 0 {
				lib.Reset = make(chan byte)
				go func() {
					t := time.NewTimer(Config.LDLibKeepaliveHour * time.Hour)
					for {
						select {
						case <-lib.Reset:
							if !t.Stop() {
								select {
								case <-t.C:
								default:
								}
							}
							t.Reset(Config.LDLibKeepaliveHour * time.Hour)
						case <-t.C:
							log.Println("lib keepalive expired: ", libpath)

							lib.L.Lock()
							defer lib.L.Unlock()

							lock.Lock()
							defer lock.Unlock()

							if current, exists := dls[libpath]; !exists || current != lib {
								return
							}

							C.dlclose(lib.Ptr)

							delete(dls, libpath)

							return
						}
					}
				}()
			}

			dls[libpath] = lib

			ptr, cancel, err = lib.use(symbol)
		}
	}()

	lock.RLock()
	defer lock.RUnlock()

	lib, exists := dls[libpath]
	if exists {
		ptr, cancel, err = lib.use(symbol)
		return
	}

	return nil, nil, nil
}
