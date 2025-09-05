package main

/*
#cgo LDFLAGS: -ldl

*/
import "C"
import "fmt"

//export getMonitorName
func getMonitorName(siteid *C.char, mid int) *C.char {
	return C.CString(fmt.Sprintf("%d", mid))
}

func main() {
	DLOPEN()
}
