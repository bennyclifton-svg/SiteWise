package correspondence

import (
	"context"
	"net"
	"net/smtp"
	"sync"
)

// Sender hands one built message to the mail service. "Sent" means the
// service accepted it (delivery tracking is out of scope).
type Sender interface {
	Send(ctx context.Context, envelopeFrom string, to []string, msg []byte) error
}

// SMTP sends through the configured relay (SITEWISE_SMTP_*: Mailgun on 587
// in production). net/smtp upgrades to TLS when the server offers it, and
// refuses to send a password over a plain connection to another host.
type SMTP struct {
	Addr, User, Password string
}

// Send delivers msg. The context bounds the dial only; net/smtp has no
// per-call deadline, so the outbox runs each send in its own goroutine.
func (s SMTP) Send(ctx context.Context, from string, to []string, msg []byte) error {
	var a smtp.Auth
	if s.User != "" {
		host, _, err := net.SplitHostPort(s.Addr)
		if err != nil {
			return err
		}
		a = smtp.PlainAuth("", s.User, s.Password, host)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return smtp.SendMail(s.Addr, a, from, to, msg)
}

// Sink keeps what it was given, for tests and the bench (as auth.Sink).
type Sink struct {
	mu   sync.Mutex
	Sent []Sent
	// Fail, when set, is returned instead of keeping the message.
	Fail error
}

// Sent is one message a Sink kept.
type Sent struct {
	From string
	To   []string
	Msg  []byte
}

func (s *Sink) Send(_ context.Context, from string, to []string, msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Fail != nil {
		return s.Fail
	}
	s.Sent = append(s.Sent, Sent{From: from, To: append([]string(nil), to...), Msg: append([]byte(nil), msg...)})
	return nil
}

// Kept is a copy of what the sink holds.
func (s *Sink) Kept() []Sent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Sent(nil), s.Sent...)
}

// SetFail changes whether the sink refuses.
func (s *Sink) SetFail(err error) {
	s.mu.Lock()
	s.Fail = err
	s.mu.Unlock()
}
