package mimestream

import (
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
)

// Text is a text/HTML/other content body part
type Text struct {
	ContentType string
	Text        string
}

// Add implements the Source interface.
func (p Text) Add(w *multipart.Writer) error {

	contentType := p.ContentType

	// Default to text plain
	if contentType == "" {
		contentType = TextPlain
	}

	quotedPart, err := CreateQuotedPart(w, contentType)
	if err != nil {
		return err
	}

	var n int
	n, err = quotedPart.Write([]byte(p.Text))
	if err != nil {
		return err
	}

	if n != len(p.Text) {
		return ErrPartialWrite
	}

	// Need to close after writing
	// https://golang.org/pkg/mime/quotedprintable/#Writer.Close
	quotedPart.Close()

	return err
}

// CreateQuotedPart creates a quoted-printable, wrapped, mime part
func CreateQuotedPart(writer *multipart.Writer, contentType string) (w *quotedprintable.Writer, err error) {
	header := textproto.MIMEHeader{
		"Content-Type":              []string{contentType},
		"Content-Transfer-Encoding": []string{"quoted-printable"},
	}

	var part io.Writer
	part, err = writer.CreatePart(header)
	if err != nil {
		return
	}

	w = quotedprintable.NewWriter(part)
	return
}
