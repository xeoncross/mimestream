# mimestream

Performant, low-memory multipart/mime library for building and parsing email bodies in Go.

Status: *Work In Progress*

## Writer Usage

Simple create a struct containing the parts as needed. Attach an io.Reader
and file contents (or a network connection) will be streamed without requiring
everything to be loaded into memory.

    // Assuming this is some type of io.Reader or io.ReadCloser
    var openfilehandle io.ReadCloser

    parts := mimestream.Parts{
      mimestream.Alternative{
        Parts: []mimestream.Part{
          mimestream.Text{
            Text: "This is the text that goes in the plain part.",
          },
          mimestream.Text{
            ContentType: mimestream.TextHTML,
            Text:        "<p>This is the text that goes in the plain part.</p>",
          },
        },
      },
      mimestream.File{
        Name:   "filename.jpg",
        Inline: true,
        Reader: openfilehandle,
        Closer: openfilehandle,
      },
      mimestream.File{
        Name:   "payload.json",
        Reader: strings.NewReader(`{"one":1,"two":2}`),
      },
    }

    // Destination could be file or network connection, here we just throw it away
    mr := multipart.NewWriter(ioutil.Discard)

    err = parts.Into(mr)
    if err != nil {
      log.Fatal(err)
    }

## Reader Usage

Reading emails is done with a simple callback that provides a place to stream
each part to a destination. In the example below we simply read everything.

    // Assume mailreader is a network connection or email file
    var mailreader io.Reader

    err = HandleEmailFromReader(mailreader, func(header textproto.MIMEHeader, body io.Reader) (err error) {
      maxbytes := 1024 * 1024

      var b []byte
      b, err = ioutil.ReadAll(io.LimitReader(body, maxbytes))
      if err != nil {
        return
      }

      fmt.Printf("Part Header: %v\n", header)
      fmt.Printf("Part body: %q\n", b)

      return
    })

## TODO

- More Tests
- Support writing nested parts
- Multipart mixed and alternative


# Why

Most multipart/mime handling by both email and MIME libraries in Go requires everything to be loaded into memory. I wanted to be able to create mime bodies without needing much memory.

Here are some of the following problems this package addresses:

1. Requires memory equal to mime content (+ encoding)
2. Uses string building to construct envelope (instead of using `encoding/mime`)

- https://github.com/jhillyerd/enmime: [1](https://github.com/jhillyerd/enmime/blob/874cc30e023f36bd1df525716196887b0f04851b/encode.go#L32) [2](https://github.com/jhillyerd/enmime/blob/874cc30e023f36bd1df525716196887b0f04851b/encode.go#L50)

- https://github.com/domodwyer/mailyak: [1](https://github.com/domodwyer/mailyak/blob/89444b05799b115121931b3b6bd05e820e69dc8b/mime.go#L152) [2](https://github.com/domodwyer/mailyak/blob/89444b05799b115121931b3b6bd05e820e69dc8b/mime.go#L57)

- https://github.com/sloonz/go-mime-message: [2](https://github.com/sloonz/go-mime-message/blob/cf50e17d2410fee25cdb89485ab0d5996f2d3bfc/multipart.go#L54)

- https://github.com/lavab/mailer: [1](https://github.com/lavab/mailer/blob/a0901ff739bb9a5599f40133deaadb250ec834db/outbound/send.go#L595)

Similar libraries

- https://github.com/emersion/go-message
- https://github.com/philippfranke/multipart-related

# Resources

- https://tools.ietf.org/html/rfc2046
- @emersion has many [Go packages for Email and related things](https://github.com/emersion?utf8=%E2%9C%93&tab=repositories&q=&type=&language=go).
- Collection of [Go projects that deal with email, SMTP, IMAP, and other related tasks](https://gist.github.com/Xeoncross/4ef91d6a47bc33b85d8250772a0622e1)
