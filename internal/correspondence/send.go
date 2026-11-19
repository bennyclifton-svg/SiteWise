package correspondence

import (
	"context"
	"crypto/tls"
	"net"
	"net/smtp"
	"sync"
	"time"
)

// Sender hands one built message to the mail service. "Sent" means the
// service accepted it (delivery tracking is out of scope).
type Sender interface {
	Send(ctx context.Context, envelopeFrom string, to []string, msg []byte) error
}

// SMTP sends through the configured relay (SITEWISE_SMTP_*: Mailgun on 587
// in production). Every step is bounded: the dial, and the whole exchange
// by a deadline on the connection (mail review L3), so a hung relay fails
// the attempt, which the outbox retries. TLS is taken when the server
// offers it; a password is never sent without it.
type SMTP struct {
	Addr, User, Password string
	// Timeout bounds one send; zero is SendTimeout.
	Timeout time.Duration
}

// SendTimeout bounds one message's whole SMTP exchange.
const SendTimeout = 2 * time.Minute

// Send delivers msg.
func (s SMTP) Send(ctx context.Context, from string, to []string, msg []byte) error {
	host, _, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return err
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = SendTimeout
	}
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	dialer := net.Dialer{Deadline: deadline}
	conn, err := dialer.DialContext(ctx, "tcp", s.Addr)
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if s.User != "" {
		if err := c.Auth(smtp.PlainAuth("", s.User, s.Password, host)); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, r := range to {
		if err := c.Rcpt(r); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
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
