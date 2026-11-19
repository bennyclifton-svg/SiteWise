package mailin

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
)

// RawField is the form field Mailgun's raw-MIME route posts the whole
// message in: the route's forward URL ends in "mime" (Task 0 confirms the
// name, and that the bytes are exact enough for DKIM to verify). The
// webhook reads it here only.
const RawField = "body-mime"

// MaxHeaders: a message with more headers is held, never cut and trusted.
const MaxHeaders = 500

// Limits on walking a raw message's parts.
const (
	maxDepth = 6
	maxParts = 120
)

var (
	// ErrTooManyHeaders: more than MaxHeaders headers.
	ErrTooManyHeaders = errors.New("too many headers")
	// ErrFromCount: not exactly one From header.
	ErrFromCount = errors.New("not exactly one From header")
	// ErrDuplicateHeader: a header the record or MIME reads appears twice.
	ErrDuplicateHeader = errors.New("a header appears twice")
)

// singleHeaders appear at most once, or the message is held: DKIM signs
// the last instance of a header, a reader may take the first, and a sender
// could prepend one above a genuinely signed message (review HIGH-1).
var singleHeaders = []string{"From", "Subject", "Message-Id", "In-Reply-To", "References", "To", "Cc", "Date", "Content-Type", "Mime-Version", "Reply-To"}

// Attachment is one file of a message: a form file of Mailgun's parsed post,
// or a part of the raw message.
type Attachment struct {
	Field       string // the form field ("attachment-1"), or "" from raw MIME
	Name        string
	ContentType string
	ContentID   string
	Inline      bool
	Size        int64
	Open        func() (io.ReadCloser, error)
}

// FileAttachment wraps a parsed post's form file.
func FileAttachment(field string, fh *multipart.FileHeader) Attachment {
	return Attachment{Field: field, Name: fh.Filename, ContentType: fh.Header.Get("Content-Type"), Size: fh.Size,
		Open: func() (io.ReadCloser, error) { return fh.Open() }}
}

// ParseRaw reads the raw message: its headers (at most MaxHeaders, exactly
// one From, and no second instance of a header the record or MIME reads),
// the text body, and its attachments. The webhook records what this reads
// from a raw post; which of those headers count is Trust's to decide, by
// what the verifying signature signed.
func ParseRaw(raw []byte) (Message, []Attachment, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return Message{}, nil, ErrPost
	}
	count := 0
	for _, vs := range msg.Header {
		count += len(vs)
	}
	if count > MaxHeaders {
		return Message{}, nil, ErrTooManyHeaders
	}
	if len(msg.Header["From"]) != 1 {
		return Message{}, nil, ErrFromCount
	}
	for _, h := range singleHeaders {
		if len(msg.Header[h]) > 1 {
			return Message{}, nil, ErrDuplicateHeader
		}
	}
	dec := new(mime.WordDecoder)
	decode := func(v string) string {
		if s, err := dec.DecodeHeader(v); err == nil {
			return s
		}
		return v
	}
	m := Message{InlineIDs: map[string]bool{}}
	// Header order as written, for the drop rules and threading.
	for _, l := range strings.Split(strings.ReplaceAll(string(headBytes(raw)), "\r\n", "\n"), "\n") {
		if l == "" || l[0] == ' ' || l[0] == '\t' {
			continue
		}
		if k, _, ok := strings.Cut(l, ":"); ok {
			k = textproto(k)
			for _, v := range msg.Header[k] {
				if !hasHeader(m.Headers, k, v) {
					m.Headers = append(m.Headers, [2]string{k, strings.TrimSpace(v)})
				}
			}
		}
	}
	addr, name, err := SenderAddress(msg.Header.Get("From"))
	if err != nil {
		return Message{}, nil, err
	}
	m.FromAddress, m.FromName = addr, name
	m.Subject = OneLine(decode(msg.Header.Get("Subject")), 998)
	m.To = OneLine(decode(msg.Header.Get("To")), 4000)
	m.Cc = OneLine(decode(msg.Header.Get("Cc")), 4000)
	m.MessageID = OneLine(msg.Header.Get("Message-Id"), 998)
	m.InReplyTo, m.References = ThreadHeaders(m)
	m.DateText = OneLine(msg.Header.Get("Date"), 200)
	if d, err := mail.ParseDate(m.DateText); err == nil {
		m.Date = d
	}
	w := &walker{}
	w.part(textproto2(msg.Header), msg.Body, 0)
	m.BodyPlain = cutBody(PlainBody(w.plain, w.html))
	for _, a := range w.files {
		if a.ContentID != "" && a.Inline {
			m.InlineIDs[a.ContentID] = true
		}
	}
	return m, w.files, nil
}

func hasHeader(h [][2]string, k, v string) bool {
	v = strings.TrimSpace(v)
	for _, x := range h {
		if x[0] == k && x[1] == v {
			return true
		}
	}
	return false
}

// textproto is a header name in canonical form ("message-id" → "Message-Id").
func textproto(k string) string {
	parts := strings.Split(strings.TrimSpace(k), "-")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, "-")
}

type header interface{ Get(string) string }

type mapHeader map[string][]string

func (h mapHeader) Get(k string) string {
	if v := h[textproto(k)]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func textproto2(h mail.Header) header { return mapHeader(h) }

type walker struct {
	plain, html string
	files       []Attachment
	n           int
}

func (w *walker) part(h header, body io.Reader, depth int) {
	w.n++
	if w.n > maxParts || depth > maxDepth {
		return
	}
	ctype, params, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil || ctype == "" {
		ctype = "text/plain"
	}
	if strings.HasPrefix(ctype, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			p, err := mr.NextRawPart()
			if err != nil {
				return
			}
			w.part(p.Header, p, depth+1)
		}
	}
	data, err := io.ReadAll(io.LimitReader(decoder(h.Get("Content-Transfer-Encoding"), body), MaxPostBytes))
	if err != nil {
		return
	}
	disp, dparams, _ := mime.ParseMediaType(h.Get("Content-Disposition"))
	name := dparams["filename"]
	if name == "" {
		name = params["name"]
	}
	if name == "" && disp != "attachment" && (ctype == "text/plain" || ctype == "text/html") {
		if ctype == "text/plain" && w.plain == "" {
			w.plain = string(data)
		} else if ctype == "text/html" && w.html == "" {
			w.html = string(data)
		}
		return
	}
	if ctype == "message/rfc822" && name == "" {
		name = "message.eml"
	}
	cid := strings.Trim(strings.TrimSpace(h.Get("Content-Id")), "<>")
	a := Attachment{Name: name, ContentType: ctype, ContentID: cid, Inline: disp == "inline" || (disp == "" && cid != ""), Size: int64(len(data)),
		Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }}
	w.files = append(w.files, a)
}

func decoder(cte string, r io.Reader) io.Reader {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, &stripSpace{r: r})
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	}
	return r
}

// stripSpace drops line breaks and spaces inside base64.
type stripSpace struct{ r io.Reader }

func (s *stripSpace) Read(p []byte) (int, error) {
	buf := make([]byte, len(p))
	for {
		n, err := s.r.Read(buf)
		k := 0
		for _, b := range buf[:n] {
			if b != '\r' && b != '\n' && b != ' ' && b != '\t' {
				p[k] = b
				k++
			}
		}
		if k > 0 || err != nil {
			return k, err
		}
	}
}
