package mimestream

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"

	"github.com/pkg/errors"
)

// Alternative multipart/mime part
type Alternative struct {
	Parts Parts
}

func (p Alternative) Add(w *multipart.Writer) (err error) {

	if len(p.Parts) == 0 {
		return
	}

	buf := &bytes.Buffer{}
	w2 := multipart.NewWriter(buf)

	for _, p := range p.Parts {
		err = p.Add(w2)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to add %T part %v", p, p))
			return
		}
	}

	err = w2.Close()
	if err != nil {
		return
	}

	contentType := fmt.Sprintf("%s; boundary=%s", MultipartAlternative, w2.Boundary())

	header := textproto.MIMEHeader{
		"Content-Type": []string{contentType},
	}

	var part io.Writer
	part, err = w.CreatePart(header)
	if err != nil {
		return err
	}

	// How much data are we going to copy?
	bl := int64(buf.Len())

	var n int64
	n, err = io.Copy(part, buf)
	if err != nil {
		return
	}

	if n != bl {
		return ErrPartialWrite
	}

	return
}
