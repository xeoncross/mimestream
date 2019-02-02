package mimestream

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"path/filepath"

	"github.com/pkg/errors"
)

// Based on: https://github.com/skillian/mparthelp/
// (with help from https://github.com/philippfranke/multipart-related/)
// TODO: ShiftJIS: https://gist.github.com/hyamamoto/db03c03fd624881d4b84

// ErrPartialWrite happens when the full body can't be written
var ErrPartialWrite = errors.New("Failed to write full body")

// Content Types
var (
	// Unformatted Text
	TextPlain = "text/plain; charset=utf-8"

	// HTML
	TextHTML = "text/html; charset=utf-8"

	// Markdown
	TextMarkdown = "text/markdown; charset=utf-8"

	// Don't need...?
	MultipartMixed = "multipart/mixed"
)

// Parts is a collection of parts of a multipart message.
type Parts []Part

// Into the given multipart.Writer
func (p Parts) Into(w *multipart.Writer) (err error) {
	for _, part := range p {

		// if len(part.Parts) > 0 {
		// 	buf := &bytes.Buffer{}
		// 	w2 := multipart.NewWriter(buf)
		// 	part.Parts.Into(w2)
		//
		// 	continue
		// }

		err = part.Source.Add(part.ContentType, w)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to add %T part %v", part, part))
			return
		}
	}
	return w.Close()
}

// Part defines a named part inside of a multipart message.
type Part struct {
	ContentType string
	Source
}

// Source is a data source that can add itself to a mime/multipart.Writer.
type Source interface {
	// Name can be a field name or content type. It is the part Content-Type
	Add(contentType string, w *multipart.Writer) error
}

type MixedPart struct {
	ContentType string
	Parts       Parts
}

func (p *MixedPart) Add(contentType string, w *multipart.Writer) (err error) {

	if len(p.Parts) == 0 {
		return
	}

	buf := &bytes.Buffer{}
	w2 := multipart.NewWriter(buf)

	for _, p := range p.Parts {

		err = p.Source.Add(p.ContentType, w2)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to add %T part %v", p, p))
			return
		}
	}

	err = w2.Close()
	if err != nil {
		return
	}

	header := textproto.MIMEHeader{
		"Content-Type": []string{w2.FormDataContentType()},
	}

	var part io.Writer
	part, err = w.CreatePart(header)
	if err != nil {
		return err
	}

	var n int64
	n, err = io.Copy(part, buf)
	if err != nil {
		return
	}

	if n != int64(buf.Len()) {
		return ErrPartialWrite
	}

	return
}

// File is a Source implementation for files read from an io.Reader.
type File struct {
	// Name is the name of the file, not to be confused with the name of the Part.
	Name string

	// Include Inline, or as an Attachment (default)?
	Inline bool

	// Character set to use (defaults to utf-8)
	Charset string

	// Reader is the data source that the part is populated from.
	io.Reader

	// Closer is an optional io.Closer that is called after reading the Reader
	io.Closer
}

// Add implements the Source interface.
func (f File) Add(contentType string, w *multipart.Writer) (err error) {

	// Valid Attachment-Headers:
	//
	//  - Content-Disposition: attachment; filename="frog.jpg"
	//  - Content-Disposition: inline; filename="frog.jpg"
	//  - Content-Type: attachment; filename="frog.jpg"

	if f.Charset == "" {
		f.Charset = "utf-8"
	}

	fName := filepath.Base(f.Name)

	// If a Content Type is not provided, detect it
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(fName))
	}

	param := map[string]string{
		"charset": f.Charset,
		"name":    ToASCII(fName),
	}
	cType := mime.FormatMediaType(contentType, param)
	if cType != "" {
		contentType = cType
	}

	// fmt.Println("contentType", contentType)
	// mt := mime.FormatMediaType(p.ContentType, param)

	header := textproto.MIMEHeader{
		"Content-Type":              []string{contentType},
		"Content-Disposition":       []string{"attachment"},
		"Content-Transfer-Encoding": []string{"base64"},
	}

	if f.Inline {
		header["Content-Disposition"] = []string{"Inline"}
	}

	var part io.Writer
	part, err = w.CreatePart(header)
	if err != nil {
		return err
	}

	// Base64 encode + Mime Wrap to 76 characters
	base64Encoder := NewMimeBase64Writer(part)

	// Copy everything into the base64 encoder
	// TODO we should be checking bytes written here to prevent partial sends
	_, err = io.Copy(base64Encoder, f.Reader)
	if err != nil {
		return err
	}

	// Must close the encoder
	base64Encoder.Close()

	// Close the source stream (if needed)
	if f.Closer != nil {
		return f.Closer.Close()
	}

	return
}

// // FormFile is a Source implementation for files read from an io.Reader.
// type FormFile struct {
// 	// Name is the name of the file, not to be confused with the name of the
// 	// Part.
// 	Name string
//
// 	// Reader is the data source that the part is populated from.
// 	io.Reader
//
// 	// Closer is an optional io.Closer that is called after reading the Reader
// 	io.Closer
// }
//
// // Add implements the Source interface.
// func (f FormFile) Add(name string, w *multipart.Writer) error {
// 	part, err := w.CreateFormFile(name, f.Name)
// 	if err != nil {
// 		return err
// 	}
// 	_, err = io.Copy(part, f.Reader)
// 	if err != nil {
// 		return err
// 	}
// 	if f.Closer != nil {
// 		return f.Closer.Close()
// 	}
// 	return nil
// }

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

type TextPart struct {
	Text string
}

// Add implements the Source interface.
func (p TextPart) Add(contentType string, w *multipart.Writer) error {

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
