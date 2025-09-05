package main

/*
#include <dlfcn.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#ifndef GO_ENV_DATATYPE_H
typedef enum  DATATYPE {
    RealTime,
    Minutely,
    Hourly,
    Daily
} DATATYPE;
typedef DATATYPE (*GET_DATATYPE) ();
#endif

#ifndef GO_ENV_MONITOR_H
typedef struct MONITOR {
    int id;
    char* name;
    char* unit;
} MONITOR;
#endif

#ifndef GO_ENV_DATA_H
typedef struct DATA {
    DATATYPE data_type;
    time_t data_time;
    int station_id;
    int monitor_id;
    double data;
    char* flag;
} DATA;
#endif

#ifndef GO_ENV_ENTITY_H
typedef struct ENTITY {
    int id;
    char* name;
    char* address;
} ENTITY;
#endif

#ifndef GO_ENV_STATION_H
typedef struct STATION {
    int id;
    int entity_id;
    char* name;
} STATION;
#endif

#ifndef GO_ENV_PUSH_H
typedef char* (*GET_MONITOR_NAME) (char*, int);
#endif

typedef int (*TEST_ENTITYLIST) (ENTITY [], int, char ** [], GET_MONITOR_NAME gmn);

int TestListCall(void* f, ENTITY d[], int size, char ** names [], GET_MONITOR_NAME gmn) {
	return ((TEST_ENTITYLIST) f)(d, size, names, gmn);
}

typedef int (*HELLO)();
int HelloCall(void* f) {
	return ((HELLO) f)();
}

extern char* getMonitorName(char*, int i);
*/
import "C"
import (
	"log"
	"runtime"
	"unsafe"
)

type CharList []*C.char

func DLOPEN() {

	log.Println(runtime.GOARCH)

	sopath := C.CString("/Users/limbo/GIT/ob_server/c/push/environment/keqin/keqin_station_push.so")
	defer C.free(unsafe.Pointer(sopath))

	handle := C.dlopen(sopath, C.RTLD_LAZY)
	defer C.dlclose(unsafe.Pointer(handle))

	if handle == nil {
		err := C.dlerror()
		log.Println("error handle nil")
		C.puts(err)
		return
	}

	// funcName := C.CString("TestEntityList")
	funcName := C.CString("hello_world")
	defer C.free(unsafe.Pointer(funcName))

	funcP := C.dlsym(handle, funcName)

	if handle == nil {
		log.Println("error funcP nil")
		err := C.dlerror()
		C.puts(err)
		return
	}

	// entityList := make([]C.ENTITY, 5)

	// entity := C.ENTITY{}
	// entity.id = C.int(15)
	// entity.name = C.CString("152")

	// entityList[0] = entity
	// entityList[1] = entity
	// entityList[2] = entity
	// entityList[3] = entity
	// entityList[4] = entity

	// var names **C.char

	// log.Println("names: ", &names, names)

	// result, err := C.TestListCall(funcP, (*C.ENTITY)(unsafe.Pointer(&entityList[0])), 5, &names, C.GET_MONITOR_NAME(C.getMonitorName))
	// if err != nil {
	// 	log.Println("error:", err)
	// 	return
	// }

	// log.Println("names: ", &names, names)

	// size := int(result)
	// log.Println(size)

	// var slice []string
	// for i, s := range (*[1 << 28]*C.char)(unsafe.Pointer(names))[:size:size] {
	// 	goStr := C.GoString(s)
	// 	slice = append(slice, goStr)
	// 	log.Println(i, goStr)
	// 	C.free(unsafe.Pointer(s))
	// }

	result, err := C.HelloCall(funcP)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}
