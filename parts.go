package mimestream

import (
	"fmt"
	"mime/multipart"

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

	// File attachments
	MultipartMixed = "multipart/mixed"

	// Text and HTML content
	MultipartAlternative = "multipart/alternative"
)

// Parts is a collection of parts of a multipart message.
type Parts []Part

// Into the given multipart.Writer
func (p Parts) Into(w *multipart.Writer) (err error) {
	for _, part := range p {
		err = part.Add(w)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to add %T part %v", part, part))
			return
		}
	}
	return w.Close()
}

// Part defines a named part inside of a multipart message.
// Part is a data source that can add itself to a mime/multipart.Writer.
type Part interface {
	// Name can be a field name or content type. It is the part Content-Type
	Add(w *multipart.Writer) error
}
