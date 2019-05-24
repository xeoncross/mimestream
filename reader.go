package mimestream

import (
	"bufio"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/pkg/errors"
)

// Most emails will never contain more than 3 levels of nested multipart bodies
var MaximumMultipartDepth = 10

var ErrMaximumMultipartDepth = errors.New("Mimestream: Maximum multipart/mime nesting level reached")

// Most emails will never have more than a dozen attachments / text parts total
var MaximumPartsPerMultipart = 50

var ErrMaximumPartsPerMultipart = errors.New("Mimestream: Maximum number of multipart/mime parts reached")

// ErrMissingBoundary for multipart bodies without a boundary header
var ErrMissingBoundary = errors.New("Missing boundary")

// The format of the callback for handling MIME emails
type partHandler func(textproto.MIMEHeader, io.Reader) error

// NewEmailFromReader reads a stream of bytes from an io.Reader, r,
// and returns an email struct containing the parsed data.
// This function expects the data in RFC 5322 format.
func HandleEmailFromReader(r io.Reader, h partHandler) (err error) {
	tp := textproto.NewReader(bufioReader(r))

	var header textproto.MIMEHeader
	header, err = tp.ReadMIMEHeader()
	if err != nil {
		return
	}

	// FYI: http.Header and textproto.MIMEHeader are both map[string][]string
	// http.Header(header)
	// header.(map[string][]string)
	// (*map[string][]string).(header)

	// Recursively parse the MIME parts
	err = parseMIMEParts(header, tp.R, h, 0)
	return
}

// parseMIMEParts will recursively walk a MIME entity calling the handler
func parseMIMEParts(hs textproto.MIMEHeader, body io.Reader, handler partHandler, level int) (err error) {

	// Protect against bad actors
	if level > MaximumMultipartDepth {
		return ErrMaximumMultipartDepth
	}

	ct, params, err := mime.ParseMediaType(hs.Get("Content-Type"))
	if err != nil {
		return
	}

	// Either a leaf node, or not a multipart email
	if !strings.HasPrefix(ct, "multipart/") {
		err = handler(hs, contentDecoderReader(hs, body))
		return
	}

	// Should we allow this?
	if _, ok := params["boundary"]; !ok {
		return ErrMissingBoundary
	}

	// Readers are buffered https://golang.org/src/mime/multipart/multipart.go#L99
	mr := multipart.NewReader(body, params["boundary"])

	var partsCounter int
	var p *multipart.Part
	for {

		// Decodes quotedprintable: https://golang.org/src/mime/multipart/multipart.go#L128
		// Closes last part reader: https://golang.org/src/mime/multipart/multipart.go#L302
		p, err = mr.NextPart()
		if err == io.EOF {
			err = nil
			break
		}

		if err != nil {
			return
		}

		// Protect against bad actors
		partsCounter++
		if partsCounter > MaximumPartsPerMultipart {
			return ErrMaximumPartsPerMultipart
		}

		// Correctly decode the body bytes
		body := contentDecoderReader(p.Header, p)

		var subct string
		subct, _, err = mime.ParseMediaType(p.Header.Get("Content-Type"))

		// Nested multipart
		if strings.HasPrefix(subct, "multipart/") {
			err = parseMIMEParts(p.Header, body, handler, level+1)
			if err != nil {
				return
			}

		} else {
			// Leaf node
			err = handler(p.Header, body)
			if err != nil {
				return
			}
		}
	}

	return
}

// contentDecoderReader
func contentDecoderReader(headers textproto.MIMEHeader, bodyReader io.Reader) *bufio.Reader {
	// Already handled by textproto
	// if headers.Get("Content-Transfer-Encoding") == "quoted-printable" {
	// 	return bufioReader(quotedprintable.NewReader(bodyReader))
	// }
	if headers.Get("Content-Transfer-Encoding") == "base64" {
		return bufioReader(base64.NewDecoder(base64.StdEncoding, bodyReader))
	}
	return bufioReader(bodyReader)
}

// bufioReader ...
func bufioReader(r io.Reader) *bufio.Reader {
	if bufferedReader, ok := r.(*bufio.Reader); ok {
		return bufferedReader
	}
	return bufio.NewReader(r)
}
