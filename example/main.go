package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Xeoncross/mimestream"
)

func main() {

	go func() {
		for {
			// https://golang.org/pkg/runtime/#MemStats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			log.Printf("Alloc = %10d HeapAlloc = %10d Sys = %10d NumGC = %10d\n", m.Alloc/1024, m.HeapAlloc/1024, m.Sys/1024, m.NumGC)
			time.Sleep(1 * time.Second)
		}
	}()

	parts := mimestream.Parts{
		mimestream.Part{
			Name: mimestream.TextPlain,
			Source: mimestream.TextPart{
				Text: "This is the text that goes in the plain part. It will need to be wrapped to 76 characters and quoted.",
			},
		},
		mimestream.Part{
			Name: "filepart1",
			Source: mimestream.File{
				Name:   "filename.jpg",
				Reader: mockDataSrc(1024 * 1024 * 1000), // in MB
			},
		},
		mimestream.Part{
			Name: "filepart1",
			Source: mimestream.File{
				Name:   "filename-2 שלום.txt",
				Inline: true,
				Reader: strings.NewReader("Filename text content"),
			},
		},
		mimestream.Part{
			Name: "jsonpart1",
			Source: mimestream.JSON{
				Value: map[string]int{"one": 1, "two": 2},
			},
		},
	}

	// Throw away
	// out := ioutil.Discard

	// Save to test file
	out, err := os.OpenFile("output.txt", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	header, err := parts.Into(out)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(header)
}

type devZero byte

func (z devZero) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = byte(z)
	}
	return len(b), nil
}

func mockDataSrc(size int64) io.Reader {
	var z devZero
	return io.LimitReader(z, size)
}
