// +build windows

package main

import (
	"fmt"
	"io"
	"log"
)

func LogErrorf(err error, format string, v ...interface{}) {
	if err != nil {
		if err == io.EOF {
			return
		}
		log.Println("ERROR", fmt.Sprintf(format, v...), err)
	}
}
