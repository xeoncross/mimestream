package mimereader

import (
	"io"
	"sync/atomic"
	"time"
)

type devZero byte

func (z devZero) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = byte(z)
	}
	return len(b), nil
}

type SlowReader struct {
	Reader io.Reader
	Speed  time.Duration
}

func (s *SlowReader) Read(b []byte) (int, error) {
	time.Sleep(s.Speed)
	return s.Reader.Read(b)
}

func mockDataSrc(size int64) io.Reader {
	// fmt.Printf("dev/zero of size %d (%d MB)\n", size, size/1024/1024)
	var z devZero
	// return &SlowReader{Reader: io.LimitReader(z, size), Speed: time.Microsecond * 1}
	return io.LimitReader(z, size)
}

// WriteCounter counts the number of bytes written to it.
type WriteCounter struct {
	total  int64 // Total # of bytes transferred
	recent int64 // Used for per-second/minute/etc.. reports
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	atomic.AddInt64(&wc.total, int64(n))
	atomic.AddInt64(&wc.recent, int64(n))
	return n, nil
}

func (wc *WriteCounter) Total() int64 {
	return atomic.LoadInt64(&wc.total)
}

func (wc *WriteCounter) Recent() (n int64) {
	n = atomic.LoadInt64(&wc.recent)
	atomic.StoreInt64(&wc.recent, int64(0))
	return n
}
