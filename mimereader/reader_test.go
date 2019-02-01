package mimereader

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Xeoncross/mimestream"
)

func Test(t *testing.T) {

	var err error

	var tmpfile *os.File
	tmpfile, err = ioutil.TempFile("", "mimestream")
	if err != nil {
		t.Error(err)
	}

	defer os.Remove(tmpfile.Name())
	// d, _ := os.Getwd()
	// log.Println(d)
	// os.RemoveAll("./tmp")

	parts := mimestream.Parts{
		mimestream.Part{
			ContentType: mimestream.TextPlain,
			Source: mimestream.TextPart{
				Text: "Hello World",
			},
		},
		// mimestream.Part{
		// 	Name: "image/jpeg",
		// 	Source: mimestream.File{
		// 		Name:   "filename.jpg",
		// 		Reader: mockDataSrc(1024 * 1024 * 0), // in MB
		// 	},
		// },
		// mimestream.Part{
		// 	ContentType: mimestream.TextPlain,
		// 	Source: mimestream.File{
		// 		Name:   "filename-2 שלום.txt",
		// 		Inline: true,
		// 		Reader: tmpfile,
		// 		Closer: tmpfile,
		// 	},
		// },
		mimestream.Part{
			ContentType: "application/json",
			Source: mimestream.File{
				Name:   "payload.json",
				Reader: strings.NewReader(`{"one":1,"two":2}`),
			},
		},
	}

	// To pipe to reader
	pr, pw := io.Pipe()

	// Log everything read
	sow := &StdoutWriter{}

	// Log how much data passed through
	wc := &WriteCounter{}

	// One write to rule-them-all
	forkedwriter := io.MultiWriter(wc, pw, sow)

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

	///////////////////////////
	// Debugging here

	// io.CopyN(os.Stdout, mailreader, int64(2000))

	// var tmpFile *os.File
	// tmpFile, err = os.Create("mime")
	// if err != nil {
	// 	err = errors.Wrap(err, "Error creating email temp file")
	// 	return
	// }
	//
	// _, err = io.Copy(tmpFile, mailreader) // Save body disk
	// if true {
	// 	return
	// }

	///////////////////////////

	message, err := NewEmailFromReader(mailreader, "tmp")

	if err != nil {
		t.Error(err)
	}

	fmt.Println(len(message.Parts), "parts found")

	fmt.Println(message.Close())

	for _, p := range message.Parts {

		b, err := ioutil.ReadAll(p.Body)
		// _, err = io.Copy(os.Stdout, p.Body)
		if err != nil {
			log.Fatalf("ioutil.ReadAll: %s\n", err)
		}

		fmt.Printf("Part body: %q\n", b)

		p.Close()
		// os.Remove(p.File.Name())
	}

}
