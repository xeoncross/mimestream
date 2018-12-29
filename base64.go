package mimestream

import (
	"encoding/base64"
	"io"
)

// MimeWrapWriter adds a newline pair (CRLF) every 76 bytes
type MimeWrapWriter struct {
	Out io.Writer
}

func (w MimeWrapWriter) Write(b []byte) (written int, err error) {
	// https://tools.ietf.org/html/rfc2822#section-2.1.1
	stride := 76
	var n int

	for left := 0; left < len(b); left += stride {
		// Some lines will be less than 76 characters. This is not a problem.
		right := left + stride
		if right > len(b) {
			right = len(b)
		}

		n, err = w.Out.Write(b[left:right])
		if err != nil {
			return
		}
		written += n

		// The newlines are not a part of the provide slice. Do not count.
		_, err = w.Out.Write([]byte("\r\n"))
		if err != nil {
			return
		}
	}

	return
}

func NewMimeBase64Writer(w io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, MimeWrapWriter{w})
}

// TODO: found a similar library: https://github.com/emersion/go-textwrapper/blob/master/wrapper.go
