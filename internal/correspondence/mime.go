package correspondence

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// Outgoing is one message as it leaves: every header value comes from Go
// or from the PM's dialog, checked here so no value can add a header line.
type Outgoing struct {
	From      mail.Address
	ReplyTo   []mail.Address
	To, Cc    []mail.Address
	Subject   string
	Body      string
	MessageID string // "<uuid@domain>"
	InReplyTo string // "<id>" or ""
	// References is the thread's ids, oldest first.
	References []string
	Date       time.Time
	Files      []File
}

// File is one attachment: the transmittal sheet, then the documents.
type File struct {
	Name, ContentType string
	Body              []byte
}

// ErrHeader is a value that would break a header: a line break or a control
// character. The send is refused before anything is stored.
var ErrHeader = errors.New("a line break or control character is not allowed in a header")

// CleanHeader refuses a value with a line break or a control character.
func CleanHeader(v string) error {
	for _, r := range v {
		if r == '\r' || r == '\n' || r == 0 || (r < 0x20 && r != '\t') || r == 0x7f || r == ' ' || r == ' ' {
			return ErrHeader
		}
	}
	return nil
}

// ParseRecipients reads typed addresses (one or several, comma-separated)
// with net/mail. A line break anywhere is refused, never parsed.
func ParseRecipients(values []string) ([]mail.Address, error) {
	var out []mail.Address
	for _, v := range values {
		if err := CleanHeader(v); err != nil {
			return nil, err
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		list, err := mail.ParseAddressList(v)
		if err != nil {
			return nil, fmt.Errorf("%q is not an email address", v)
		}
		for _, a := range list {
			a.Address = strings.ToLower(a.Address)
			if !strings.Contains(a.Address, "@") || len(a.Address) > 254 {
				return nil, fmt.Errorf("%q is not an email address", a.Address)
			}
			out = append(out, *a)
		}
	}
	return out, nil
}

func addressHeader(list []mail.Address) (string, error) {
	parts := make([]string, len(list))
	for i, a := range list {
		if err := CleanHeader(a.Name + a.Address); err != nil {
			return "", err
		}
		parts[i] = a.String()
	}
	return strings.Join(parts, ", "), nil
}

// Build writes the message as multipart/mixed: the note as quoted-printable
// text, then each file as base64. Loop guard: X-Auto-Response-Suppress asks
// recipients' servers not to answer automatically; SiteWise itself never
// answers mail and drops automatic replies it receives (mailin.Dropped).
func Build(m Outgoing) ([]byte, error) {
	for _, v := range append([]string{m.Subject, m.MessageID, m.InReplyTo}, m.References...) {
		if err := CleanHeader(v); err != nil {
			return nil, err
		}
	}
	if len(m.To) == 0 {
		return nil, errors.New("no recipient")
	}
	var b bytes.Buffer
	h := func(name, value string) { fmt.Fprintf(&b, "%s: %s\r\n", name, value) }
	from, err := addressHeader([]mail.Address{m.From})
	if err != nil {
		return nil, err
	}
	h("From", from)
	for _, x := range []struct {
		name string
		list []mail.Address
	}{{"To", m.To}, {"Cc", m.Cc}, {"Reply-To", m.ReplyTo}} {
		if len(x.list) == 0 {
			continue
		}
		v, err := addressHeader(x.list)
		if err != nil {
			return nil, err
		}
		h(x.name, v)
	}
	h("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	h("Date", m.Date.Format(time.RFC1123Z))
	h("Message-ID", m.MessageID)
	if m.InReplyTo != "" {
		h("In-Reply-To", m.InReplyTo)
	}
	if len(m.References) > 0 {
		h("References", strings.Join(m.References, " "))
	}
	h("MIME-Version", "1.0")
	h("X-Auto-Response-Suppress", "All")
	w := multipart.NewWriter(&b)
	h("Content-Type", mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": w.Boundary()}))
	b.WriteString("\r\n")
	text, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return nil, err
	}
	qp := quotedprintable.NewWriter(text)
	if _, err := qp.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	for _, f := range m.Files {
		if err := CleanHeader(f.Name); err != nil {
			return nil, err
		}
		ctype := f.ContentType
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {mime.FormatMediaType(ctype, map[string]string{"name": f.Name})},
			"Content-Disposition":       {mime.FormatMediaType("attachment", map[string]string{"filename": f.Name})},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return nil, err
		}
		enc := base64.StdEncoding.EncodeToString(f.Body)
		for len(enc) > 76 {
			part.Write([]byte(enc[:76] + "\r\n"))
			enc = enc[76:]
		}
		part.Write([]byte(enc + "\r\n"))
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Recipients are the envelope recipients: To and Cc, once each.
func Recipients(m Outgoing) []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range append(append([]mail.Address{}, m.To...), m.Cc...) {
		if !seen[a.Address] {
			seen[a.Address] = true
			out = append(out, a.Address)
		}
	}
	return out
}
