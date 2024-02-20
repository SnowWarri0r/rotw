# rotw(File Rotation Writer)  
## What is it?  
A file rotation writer which can write file split by time span  

## Features  
- [x] Rotate by time span
- [x] Max Keep files
- [x] Compatible with zapcore.WriteSyncer
- [x] Customizable rotate rule
- [x] Support Windows/Linux
- [ ] Customize file write strategy
- [ ] ...

## Quick Start

```go
package main

import (
	"fmt"
	"time"

	"github.com/SnowWarri0r/rotw"
)

func main() {
	// create a rotate info generator
	rig, err := rotw.NewRotateInfoGenerator("1min", "log/abc.log")
	if err != nil {
		panic(err)
	}
	// configure the rotate writer
	rwo := &rotw.RotateWriterOption{
		// 2 files will be kept
		KeepFiles: 2,
		// rotate info generator
		Rig:        rig,
		// check file opened every 1 second
		CheckSpan: time.Second,
	}
	// create a rotate writer
	rw, err := rotw.NewRotateWriter(rwo)
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
```