package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"sync"
	"time"

	"sitewise/internal/store"
)

// Mailer delivers an invite token to a person. Implementations must not log the token.
type Mailer interface {
	SendInvite(ctx context.Context, to, body string) error
}

// Message is one invite captured by Sink.
type Message struct {
	To   string
	Body string
}

// Sink is the test and local mail delivery. It keeps the message instead of sending it.
type Sink struct {
	mu       sync.Mutex
	Messages []Message
}

// SendInvite records the message.
func (s *Sink) SendInvite(_ context.Context, to, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, Message{To: to, Body: body})
	return nil
}

// SMTP delivers invite mail. Addr is host:port.
type SMTP struct {
	Addr string
	From string
}

// SendInvite sends a plain-text invite. The token is in the message body only.
func (s SMTP) SendInvite(_ context.Context, to, body string) error {
	msg := "Subject: SiteWise invite\r\n\r\n" + body
	return smtp.SendMail(s.Addr, nil, s.From, []string{to}, []byte(msg))
}

// CreateInvite stores a hashed single-use token and hands the raw token to mail.
func CreateInvite(ctx context.Context, st *store.Store, mail Mailer, orgID, email, role string, expires time.Time, logw io.Writer) (string, error) {
	raw, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	if err := st.CreateInvite(ctx, orgID, store.Invite{
		ID:        newID(),
		Email:     email,
		TokenHash: hash,
		Role:      role,
		ExpiresAt: expires,
	}); err != nil {
		return "", err
	}
	if err := mail.SendInvite(ctx, email, "Sign in to SiteWise with this token:\n"+raw+"\n"); err != nil {
		return "", err
	}
	logger(logw).Printf("invite created org=%s email=%s", orgID, email)
	return raw, nil
}

// Bootstrap creates an org and a single owner invite. The raw token is returned
// to the caller so a local operator can sign in. It is not written to logw.
func Bootstrap(ctx context.Context, st *store.Store, mail Mailer, orgName, email string, logw io.Writer) (string, string, error) {
	orgID := newID()
	if err := st.CreateOrg(ctx, orgID, orgName); err != nil {
		return "", "", err
	}
	raw, err := CreateInvite(ctx, st, mail, orgID, email, "owner", time.Now().Add(24*time.Hour), logw)
	if err != nil {
		return "", "", err
	}
	return raw, orgID, nil
}

func logger(w io.Writer) *log.Logger {
	if w == nil {
		w = io.Discard
	}
	return log.New(w, "", 0)
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
