package rotw

import (
	"math/rand"
	"testing"
	"time"
)

func Test_main(t *testing.T) {
	rg, err := NewRotateGenerator("1min", "log/abc.log")
	if err != nil {
		t.Fatalf(">> %v\n", err)
	}
	rwo := &RotateWriterOption{
		KeepFiles: 2,
		Rig:       rg,
		CheckSpan: time.Second,
	}
	rw, err := NewRotateWriter(rwo)
	if err != nil {
		t.Fatalf(">> %v\n", err)
	}
	for i := 0; i < 4; i++ {
		t.Logf("[%v] begin write\n", nowFunc().Format(time.StampMicro))
		for j := 0; j < 10000; j++ {
			_, _ = rw.Write([]byte("hello world\n"))
			//t.Logf(">> %v, %v\n", n, errWrite)
		}
		t.Logf("[%v] end write\n", nowFunc().Format(time.StampMicro))
		time.Sleep(36*time.Second + time.Duration(rand.Intn(500))*time.Millisecond)
	}
	err = rw.Close()
	if err != nil {
		t.Fatalf(">> %v\n", err)
	}
}
