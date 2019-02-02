package mimestream

import (
	"encoding/base64"
	"io"
	"unicode"

	textwrapper "github.com/emersion/go-textwrapper"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

// NewMimeBase64Writer base64 encodes content and wraps to 76 characters
func NewMimeBase64Writer(w io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, textwrapper.NewRFC822(w))
}

// Based on https://github.com/jhillyerd/enmime/blob/master/internal/stringutil/unicode.go

// ToASCII converts unicode to ASCII by stripping accents and converting some special characters
// into their ASCII approximations.  Anything else will be replaced with an underscore.
func ToASCII(s string) string {

	// Replace runes higher than allowed by ASCII
	underscore := runes.Map(func(r rune) rune {
		// ASCII 126 (tilde)
		if r > 0x7e {
			return '_'
		}
		return r
	})

	// convert full width characters
	// https://godoc.org/golang.org/x/text/width#Transformer
	// https://stackoverflow.com/a/37646059/99923
	s = width.Narrow.String(s)

	// unicode.Mn: nonspacing marks
	tr := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), underscore, norm.NFC)
	r, _, _ := transform.String(tr, s)
	return r
}
