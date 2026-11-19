// Package mailintest builds raw test messages and signs them with DKIM, with
// a stub DNS lookup, so tests prove the sender check with real signatures
// and never touch the network. Dummy content only.
package mailintest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"
	"sync"

	"github.com/emersion/go-msgauth/dkim"
)

// File is one attachment of a test message.
type File struct {
	Name, Type string
	Body       []byte
	// Inline with a ContentID makes it a part the body shows.
	Inline    bool
	ContentID string
}

// Message is a test message.
type Message struct {
	From, To, Cc, Subject, MessageID, InReplyTo, References, Date, Body string
	Headers                                                             [][2]string
	Files                                                               []File
}

// Build writes m as RFC 5322 with CRLF line ends.
func (m Message) Build() []byte {
	var b bytes.Buffer
	h := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%s: %s\r\n", k, v)
		}
	}
	h("From", m.From)
	h("To", m.To)
	h("Cc", m.Cc)
	h("Subject", m.Subject)
	h("Message-ID", m.MessageID)
	h("In-Reply-To", m.InReplyTo)
	h("References", m.References)
	date := m.Date
	if date == "" {
		date = "Fri, 14 Nov 2026 09:15:00 +1100"
	}
	h("Date", date)
	for _, x := range m.Headers {
		h(x[0], x[1])
	}
	h("MIME-Version", "1.0")
	if len(m.Files) == 0 {
		h("Content-Type", "text/plain; charset=utf-8")
		b.WriteString("\r\n" + strings.ReplaceAll(m.Body, "\n", "\r\n") + "\r\n")
		return b.Bytes()
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	h("Content-Type", "multipart/mixed; boundary="+w.Boundary())
	b.WriteString("\r\n")
	tp, _ := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}})
	tp.Write([]byte(strings.ReplaceAll(m.Body, "\n", "\r\n")))
	for _, f := range m.Files {
		hd := textproto.MIMEHeader{"Content-Type": {f.Type}, "Content-Transfer-Encoding": {"base64"}}
		disp := "attachment"
		if f.Inline {
			disp = "inline"
		}
		hd.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disp, f.Name))
		if f.ContentID != "" {
			hd.Set("Content-ID", "<"+f.ContentID+">")
		}
		p, _ := w.CreatePart(hd)
		enc := base64.StdEncoding.EncodeToString(f.Body)
		for len(enc) > 76 {
			p.Write([]byte(enc[:76] + "\r\n"))
			enc = enc[76:]
		}
		p.Write([]byte(enc + "\r\n"))
	}
	w.Close()
	b.Write(body.Bytes())
	return b.Bytes()
}

// Keys is a stub DNS of DKIM keys: domain → key.
type Keys struct {
	mu   sync.Mutex
	keys map[string]ed25519.PrivateKey
}

// NewKeys is an empty stub DNS.
func NewKeys() *Keys { return &Keys{keys: map[string]ed25519.PrivateKey{}} }

func (k *Keys) key(domain string) ed25519.PrivateKey {
	k.mu.Lock()
	defer k.mu.Unlock()
	if p, ok := k.keys[domain]; ok {
		return p
	}
	_, p, _ := ed25519.GenerateKey(rand.Reader)
	k.keys[domain] = p
	return p
}

// Signed is the header list Sign signs, as a mail service would.
var Signed = []string{"From", "To", "Cc", "Subject", "Date", "Message-ID", "In-Reply-To", "References", "MIME-Version", "Content-Type"}

// Sign signs raw as domain (selector "t"), publishing the key in the stub.
func (k *Keys) Sign(raw []byte, domain string) []byte {
	return k.SignHeaders(raw, domain, Signed)
}

// SignHeaders signs raw as domain over the named headers only.
func (k *Keys) SignHeaders(raw []byte, domain string, headers []string) []byte {
	var out bytes.Buffer
	err := dkim.Sign(&out, bytes.NewReader(raw), &dkim.SignOptions{Domain: domain, Selector: "t", Signer: k.key(domain),
		HeaderKeys: headers})
	if err != nil {
		panic(err)
	}
	return out.Bytes()
}

// Lookup is the stub resolver: t._domainkey.<domain> for every signed domain.
func (k *Keys) Lookup(_ context.Context, name string) ([]string, error) {
	domain, ok := strings.CutPrefix(name, "t._domainkey.")
	if !ok {
		return nil, fmt.Errorf("no TXT for %s", name)
	}
	k.mu.Lock()
	p, found := k.keys[domain]
	k.mu.Unlock()
	if !found {
		return nil, fmt.Errorf("no TXT for %s", name)
	}
	pub := p.Public().(ed25519.PublicKey)
	return []string{"v=DKIM1; k=ed25519; p=" + base64.StdEncoding.EncodeToString(pub)}, nil
}
