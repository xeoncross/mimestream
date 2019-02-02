package mimestream

import (
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
)

// File is a multipart implementation for files read from an io.Reader.
type File struct {
	// Name is the basename name of the file
	Name string

	// Optional, will be detected by File.Name extension
	ContentType string

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
func (f File) Add(w *multipart.Writer) (err error) {

	// Valid Attachment-Headers:
	//
	//  - Content-Disposition: attachment; filename="frog.jpg"
	//  - Content-Disposition: inline; filename="frog.jpg"
	//  - Content-Type: attachment; filename="frog.jpg"

	if f.Charset == "" {
		f.Charset = "utf-8"
	}

	fName := filepath.Base(f.Name)

	contentType := f.ContentType

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
