package mimereader

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// MailWrapper around headers and parts
type MailWrapper struct {
	// https://golang.org/src/net/http/header.go?s=1542:1582#L48
	Header textproto.MIMEHeader
	Parts  []*Part
}

func (m MailWrapper) Close() (err error) {
	for _, p := range m.Parts {
		err = p.Close()
		if err != nil {
			return
		}
	}
	return nil
}

// Part is a copyable representation of a multipart.Part
type Part struct {
	Header textproto.MIMEHeader
	Body   io.Reader
	Closer io.ReadCloser
	File   *os.File
}

func (p Part) Close() error {
	return p.Closer.Close()
}

// type TrimReader2 struct{ io.Reader }
//
// var trailingws = regexp.MustCompile(` *\r?\n`)
//
// func (tr TrimReader2) Read(bs []byte) (int, error) {
// 	// b := make([]byte, len(bs))
//
// 	// Perform the requested read on the given reader.
// 	n, err := tr.Reader.Read(bs)
// 	if err != nil {
// 		return n, err
// 	}
//
// 	// bs = bytes.TrimRight(b, "\r\n")
// 	// return len(bs), nil
//
// 	// Remove trailing whitespace from each line.
// 	lines := string(bs[:n])
// 	trimmed := []byte(trailingws.ReplaceAllString(lines, ""))
// 	copy(bs, trimmed)
// 	return len(trimmed), nil
// }

// trimReader is a custom io.Reader that will trim any leading
// whitespace, as this can cause email imports to fail.
// type trimReader struct {
// 	rd io.Reader
// }
//
// // Read trims off any unicode whitespace from the originating reader
// func (tr trimReader) Read(buf []byte) (int, error) {
// 	n, err := tr.rd.Read(buf)
// 	t := bytes.TrimLeftFunc(buf[:n], unicode.IsSpace)
// 	n = copy(buf, t)
// 	return n, err
// }

// NewEmailFromReader reads a stream of bytes from an io.Reader, r,
// and returns an email struct containing the parsed data.
// This function expects the data in RFC 5322 format.
func NewEmailFromReader(r io.Reader, dir string) (mw MailWrapper, err error) {
	s := r
	// s := trimReader{rd: r}
	// s := TrimReader2{trimReader{rd: r}}
	tp := textproto.NewReader(bufio.NewReader(s))

	// io.CopyN(os.Stdout, s, int64(2000))

	mw.Header, err = tp.ReadMIMEHeader()
	if err != nil {
		err = errors.Wrap(err, "NewEmailFromReader")
		return
	}

	// Recursively parse the MIME parts
	mw.Parts, err = parseMIMEParts(mw.Header, tp.R, dir)

	// if err != nil {
	// 	panic(err)
	// }
	return
}

// parseMIMEParts will recursively walk a MIME entity and return a []mime.Part containing
// each (flattened) mime.Part found.
// It is important to note that there are no limits to the number of recursions, so be
// careful when parsing unknown MIME structures!
func parseMIMEParts(hs textproto.MIMEHeader, b io.Reader, dir string) (parts []*Part, err error) {

	ct, params, err := mime.ParseMediaType(hs.Get("Content-Type"))
	if err != nil {
		err = errors.Wrap(err, "Error parsing MediaType")
		return
	}

	// If it's a multipart email, recursively parse the parts
	if strings.HasPrefix(ct, "multipart/") {

		if _, ok := params["boundary"]; !ok {
			return parts, errors.New("Missing boundary")
		}

		// Readers are buffered https://golang.org/src/mime/multipart/multipart.go#L99
		mr := multipart.NewReader(b, params["boundary"])

		// For checking if multipart.Reader{nl == [13 10] == '\r\n'}
		// fmt.Printf("MR: %+v\n", mr)

		var p *multipart.Part
		for {

			// Decodes quotedprintable: https://golang.org/src/mime/multipart/multipart.go#L128
			// Closes last part reader: https://golang.org/src/mime/multipart/multipart.go#L302
			p, err = mr.NextPart()
			if err == io.EOF {
				fmt.Println("reached EOF")
				err = nil
				break
			}

			// For checking if multipart.Reader{nl == [13 10] == '\r\n'}
			// fmt.Printf("MR: %+v\n", mr)

			/////////////////////
			// TODO
			// There is an error in this section because we're not correctly padding
			// the messages with newlines causing the check above to fail ^
			// https://golang.org/src/mime/multipart/multipart.go#L313
			if err != nil && err.Error() == "multipart: NextPart: EOF" {
				fmt.Printf("Type: %T\n", err)
				// err = nil
				// return
			}
			/////////////////////

			if err != nil {
				err = errors.Wrap(err, "Error fetching next part")
				return
			}

			// Correctly decode the body bytes
			body := contentDecoderReader(p.Header, p)

			// https://golang.org/ref/spec#Type_assertions
			// http.Header and textproto.MIMEHeader are both just a map[string][]string
			// httpHeader := http.Header(p.Header)
			// httpHeader := p.Header.(map[string][]string)
			// httpHeader := (*map[string][]string).(p.Header)
			fmt.Printf("Header: %#v\n", p.Header)

			var subct string
			subct, _, err = mime.ParseMediaType(p.Header.Get("Content-Type"))

			if strings.HasPrefix(subct, "multipart/") {
				fmt.Println("\tparsing multipart?", subct)

				var subparts []*Part
				subparts, err = parseMIMEParts(p.Header, body, dir)
				if err != nil {
					err = errors.Wrap(err, "Error parsing subparts")
					return
				}
				parts = append(parts, subparts...)

			} else {
				fmt.Println("\tparsing plain?", subct)

				// v1: in-memory
				// parts = append(parts, &Part{Body: body, Header: p.Header, Closer: p})

				// v2: disk-cache
				var tmpFile *os.File
				tmpFile, err = ioutil.TempFile(dir, "mime")
				if err != nil {
					err = errors.Wrap(err, "Error creating email temp file")
					return
				}

				_, err = io.Copy(tmpFile, body) // Save body disk
				if err != nil {
					err = errors.Wrap(err, "Error saving to email temp file")
					return
				}

				// Rewind for reading
				tmpFile.Seek(0, 0)

				parts = append(parts, &Part{Body: tmpFile, Closer: tmpFile, File: tmpFile, Header: p.Header})

			}
		}
	} else {
		// If it is not a multipart email, parse the body content as a single "part"
		parts = append(parts, &Part{Body: contentDecoderReader(hs, b), Header: hs})
	}

	return parts, nil
}

// func headersToReader(headers map[string][]string) io.Reader {
// 	var buffer bytes.Buffer // Go 1.10+ can use strings.Builder
// 	for name, values := range headers {
// 		for _, value := range values {
// 			buffer.WriteString(fmt.Sprintf("%s: %s\n", name, value))
// 		}
// 	}
// 	return &buffer
// }

// func headerToReader(header http.Header) io.Reader {
// 	header.Write
// }

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
