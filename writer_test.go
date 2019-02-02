package mimestream

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriter(t *testing.T) {

	var err error

	var tmpfile *os.File
	tmpfile, err = ioutil.TempFile("", "mimestream")
	if err != nil {
		t.Error(err)
	}

	parts := Parts{
		Alternative{
			Parts: []Part{
				Text{
					Text: "This is the text that goes in the plain part. It will need to be wrapped to 76 characters and quoted.",
				},
				Text{
					ContentType: TextHTML,
					Text:        "<p>This is the text that goes in the plain part. It will need to be wrapped to 76 characters and quoted.</p>",
				},
			},
		},
		File{
			Name:   "filename.jpg",
			Reader: mockDataSrc(1024 * 1024 * 10), // in MB
		},
		File{
			Name:   "filename-2 שלום.txt",
			Inline: true,
			Reader: tmpfile,
			Closer: tmpfile,
		},
		File{
			Name:   "payload.json",
			Reader: strings.NewReader(`{"one":1,"two":2}`),
		},
	}

	// 1: Throw away
	out := ioutil.Discard

	// 2: Save
	// out, err := os.OpenFile("output.txt", os.O_RDWR|os.O_CREATE, 0755)
	// if err != nil {
	// 	t.Error(err)
	// }
	// defer out.Close()

	// Log how much data passed through
	wc := &WriteCounter{}
	finalwriter := io.MultiWriter(wc, out)

	finalwriter2 := multipart.NewWriter(finalwriter)

	// Report
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				fmt.Printf("Encoded and sent %10d MB/s (%10d MB Total) using %7d MB of RAM\n", wc.Recent()/1024/1024, wc.Total()/1024/1024, m.Alloc/1024/1024)
			}
		}
	}()

	// Start the pipeline
	err = parts.Into(finalwriter2)
	if err != nil {
		t.Error(err)
	}

}
