package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"

	"github.com/Xeoncross/mimestream"
)

func main() {

	parts := mimestream.Parts{
		mimestream.MixedPart{
			mimestream.Part{
				Source: mimestream.TextPart{
					Text: "This is the text that goes in the plain part. It will need to be wrapped to 76 characters and quoted.",
				},
			},
			mimestream.Part{
				ContentType: mimestream.TextHTML,
				Source: mimestream.TextPart{
					Text: "<p>This is the text that goes in the plain part. It will need to be wrapped to 76 characters and quoted.</p>",
				},
			},
		},
		mimestream.Part{
			Source: mimestream.File{
				Name:   "filename-2 שלום.txt",
				Reader: strings.NewReader("Filename text content"),
			},
		},
	}

	// Throw away
	// out := ioutil.Discard
	out := os.Stdout

	// Save to test file
	// out, err := os.OpenFile("output.txt", os.O_RDWR|os.O_CREATE, 0755)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer out.Close()

	mw := multipart.NewWriter(out)

	err := parts.Into(mw)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(mw.FormDataContentType())
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
