package rotw

import (
	"math/rand"
	"testing"
	"time"
)

func Benchmark_main(t *testing.B) {
	rwo := &RotateWriterConfig{
		KeepFiles: 2,
		LogPath:   "log/test.log",
		Rule:      "1min",
	}
	rw, err := NewRotateWriter(rwo)
	if err != nil {
		t.Fatalf(">> %v\n", err)
	}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, _ = rw.Write([]byte("hello world\n"))
	}
	t.StopTimer()
	err = rw.Close()
	if err != nil {
		t.Fatalf(">> %v\n", err)
	}
}

func Test_classic(t *testing.T) {
	rwo := &RotateWriterConfig{
		KeepFiles: 2,
		LogPath:   "log/test.log",
		Rule:      "1min",
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

func Test_optional(t *testing.T) {
	rw, err := NewRotateWriterWithOpt("log/test.log", WithKeepFiles(2), WithRule("1min"), WithCheckSpan(time.Second))
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
	}
	err = rw.Close()
	if err != nil {
		t.Fatalf(">> %v\n", err)
	}
}
