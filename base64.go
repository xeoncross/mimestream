package mimestream

import (
	"encoding/base64"
	"io"

	textwrapper "github.com/emersion/go-textwrapper"
)

func NewMimeBase64Writer(w io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, textwrapper.NewRFC822(w))
}
