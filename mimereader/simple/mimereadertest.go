package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"

	"github.com/Xeoncross/mimestream"
)

func main() {
	// msg := &mail.Message{
	// 	Header: map[string][]string{
	// 		"Content-Type": {"multipart/mixed; boundary=foo"},
	// 	},
	// 	Body: strings.NewReader(
	// 		"--foo\r\nFoo: one\r\n\r\nA section\r\n" +
	// 			"--foo\r\nFoo: two\r\n\r\nAnd another\r\n" +
	// 			"--foo--\r\n"),
	// }

	parts := mimestream.Parts{
		mimestream.Part{
			ContentType: mimestream.TextPlain,
			Source: mimestream.TextPart{
				Text: "Hello World",
			},
		},
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

	mw := multipart.NewWriter(pw)

	// writing without a reader will deadlock so write in a goroutine
	go func() {
		// Start the pipeline
		err := parts.Into(mw)
		if err != nil {
			log.Fatal(err)
		}

		pw.Close()
	}()

	headers := strings.Join([]string{
		"From: John <john@example.com>",
		"Mime-Version: 1.0 (1.0)",
		"Date: Thu, 10 Jan 2002 11:12:00 -0700",
		"Subject: My Temp Message",
		"Message-Id: <1234567890>",
		"To: <user@example.com>",
		"Content-Type: " + mw.FormDataContentType()}, "\r\n") + "\r\n\r\n"

	mailreader := io.MultiReader(strings.NewReader(headers), pr)

	msg, err := mail.ReadMessage(mailreader)
	if err != nil {
		log.Fatal(err)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatal(err)
			}
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Part %q: %q\n", p.Header, slurp)
		}
	}
	// Output:
	// Part "one": "A section"
	// Part "two": "And another"
}
