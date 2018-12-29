package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Xeoncross/mimestream"
)

func main() {

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
				Reader: strings.NewReader("Filename text content"),
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

	buf := &bytes.Buffer{}

	header, _ := parts.Into(buf)

	fmt.Println(header)
	fmt.Println(buf)
}
