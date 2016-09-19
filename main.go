package main

import "fmt"

/*
#cgo CFLAGS: -I./ext/drafter/src/ -I./ext/drafter/ext/snowcrash/src/
#cgo LDFLAGS: -L"./ext/drafter/build/out/Release/" -ldrafter -lsos -lsnowcrash -lmarkdownparser -lsundown -lc++
#include <stdlib.h>
#include <stdio.h>
#include "drafter.h"
*/
import "C"

func main() {
	v := C.drafter_version_string()
	s := C.GoString(v)

	fmt.Printf("Drafter version: %s\n", s)
}
