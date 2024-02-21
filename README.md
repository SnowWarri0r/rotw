# rotw(File Rotation Writer)

## What is it?

A file rotation writer which can write file split by time span

## Features

- [x] Rotate by time span
- [x] Multiple time span selection
- [x] Max Keep files
- [x] Compatible with zapcore.WriteSyncer
- [x] Customizable rotate rule
- [x] Support Windows/Linux/macOS
- [ ] Customize file write strategy
- [ ] ...

## Quick Start

classic style:

```go
package main

import (
	"fmt"
	"time"

	"github.com/SnowWarri0r/rotw"
)

func main() {
	// rotate writer configuration
	rwo := &rotw.RotateWriterConfig{
		// max keep files
		KeepFiles: 2,
		// log file path
		LogPath: "log/test.log",
		// rotate rule
		Rule: "1min",
		// check file opened span
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

optional style:

```go
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
```