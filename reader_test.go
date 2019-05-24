package mimestream

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReader(t *testing.T) {

	var err error

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
			// ContentType: "image/jpeg", // Optional
			Name:   "filename.jpg",
			Reader: mockDataSrc(32), // in bytes
		},
		// Part{
		// 	ContentType: TextPlain,
		// 	Source: File{
		// 		Name:   "filename-2 שלום.txt",
		// 		Inline: true,
		// 		Reader: tmpfile,
		// 		Closer: tmpfile,
		// 	},
		// },
		File{
			ContentType: "application/json", // Optional
			Name:        "payload.json",
			Reader:      strings.NewReader(`{"one":1,"two":2}`),
		},
	}

	// To pipe to reader
	pr, pw := io.Pipe()

	// Log everything we read
	// sow := &StdoutWriter{}

	// Log how much data passed through
	wc := &WriteCounter{}

	// One write to rule-them-all
	forkedwriter := io.MultiWriter(wc, pw) //, sow)

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

	mw := multipart.NewWriter(forkedwriter)

	// writing without a reader will deadlock so write in a goroutine
	go func() {
		// Start the pipeline
		err = parts.Into(mw)
		if err != nil {
			t.Error(err)
		}

		pw.Close()
	}()

	// ioutil.ReadAll(pr)

	headers := strings.Join([]string{
		"From: John <john@example.com>",
		"Mime-Version: 1.0 (1.0)",
		"Date: Thu, 10 Jan 2002 11:12:00 -0700",
		"Subject: My Temp Message",
		"Message-Id: <1234567890>",
		"To: <user@example.com>",
		"Content-Type: " + mw.FormDataContentType()}, "\r\n") + "\r\n\r\n"

	mailreader := io.MultiReader(strings.NewReader(headers), pr)

	var partCounter int

	err = HandleEmailFromReader(mailreader, func(header textproto.MIMEHeader, body io.Reader) (err error) {
		partCounter++

		// var b []byte
		_, err = ioutil.ReadAll(io.LimitReader(body, 1024))
		if err != nil {
			return err
		}

		// fmt.Printf("Part Header: %v\n", header)
		// fmt.Printf("Part body: %q\n", b)

		return
	})

	if err != nil {
		t.Error(err)
	}

	want := 4

	if partCounter != want {
		t.Errorf("Invalid number of parts found:\n\tGot:%d\n\tWant:%d\n", partCounter, want)
	}
}
