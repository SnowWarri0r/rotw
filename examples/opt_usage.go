//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"time"

	"github.com/SnowWarri0r/rotw"
)

func main() {
	// create a rotate writer
	rw, err := rotw.NewRotateWriterWithOpt("log/test.log", rotw.WithRule("1min"), rotw.WithKeepFiles(2), rotw.WithCheckSpan(time.Second))
	if err != nil {
		panic(err)
	}
	// defer close the rotate writer
	defer func() {
		errClose := rw.Close()
		fmt.Printf("close err=%v\n", errClose)
	}()
	// write some data
	for i := 0; i < 100; i++ {
		_, err = rw.Write([]byte(fmt.Sprintf("hello world %d\n", i)))
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
	}
}
