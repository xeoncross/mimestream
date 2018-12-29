package mimestream

import (
	"bytes"
	"encoding/json"
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

// Content Types
var (
	// Unformatted Text
	TextPlain = "text/plain; charset=utf-8"

	// HTML
	TextHTML = "text/html; charset=utf-8"

	// Markdown
	TextMarkdown = "text/markdown; charset=utf-8"
)

// Parts is a collection of parts of a multipart message.
type Parts []Part

// Into creates a multipart message into the given target from the provided
// parts.
func (p Parts) Into(target io.Writer) (formDataContentType string, err error) {
	w := multipart.NewWriter(target)
	for _, part := range p {
		err = part.Source.Add(part.Name, w)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to add %T part %v", part, part))
			return
		}
	}
	formDataContentType = w.FormDataContentType()
	return formDataContentType, w.Close()
}

// Part defines a named part inside of a multipart message.
type Part struct {
	Name string
	Source
}

// Source is a data source that can add itself to a mime/multipart.Writer.
type Source interface {
	// Name can be a field name or content type. It is the part Content-Type
	Add(name string, w *multipart.Writer) error
}

// JSON is a Source implementation that handles marshaling a value to JSON
type JSON struct {
	Value interface{}
}

// Add implements the Source interface.
func (j JSON) Add(name string, w *multipart.Writer) error {
	jsonBytes, err := json.Marshal(j.Value)
	if err != nil {
		return err
	}
	part, err := w.CreateFormField(name)
	if err != nil {
		return err
	}
	jsonBuffer := bytes.NewBuffer(jsonBytes)
	_, err = io.Copy(part, jsonBuffer)
	return err
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
func (f File) Add(name string, w *multipart.Writer) (err error) {

	// Valid Attachment-Headers:
	//
	//  - Content-Disposition: attachment; filename="frog.jpg"
	//  - Content-Disposition: inline; filename="frog.jpg"
	//  - Content-Type: attachment; filename="frog.jpg"

	if f.Charset == "" {
		f.Charset = "utf-8"
	}

	fName := filepath.Base(f.Name)
	contentType := mime.TypeByExtension(filepath.Ext(fName))

	param := map[string]string{
		"charset": f.Charset,
		"name":    ToASCII(fName),
	}
	cType := mime.FormatMediaType(contentType, param)
	if cType != "" {
		contentType = cType
	}

	fmt.Println("contentType", contentType)

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
	return nil
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
func (p TextPart) Add(name string, w *multipart.Writer) error {
	quotedPart, err := CreateQuotedPart(w, name)
	if err != nil {
		return err
	}

	var n int
	n, err = quotedPart.Write([]byte(p.Text))
	if err != nil {
		return err
	}

	if n != len(p.Text) {
		fmt.Println("Didn't write enough!")
	}

	// Need to close after writing
	// https://golang.org/pkg/mime/quotedprintable/#Writer.Close
	quotedPart.Close()

	return err
}
