// Package llm is the one client for the one LLM that AGENTS.md rule 1
// allows. It carries what it is given and returns what the model wrote. It
// knows nothing about tenders or chat.
package llm

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

var (
	// ErrOff: no key is configured.
	ErrOff = errors.New("llm: off")
	// ErrBusy: too many calls have just failed; try later.
	ErrBusy = errors.New("llm: unavailable")
	// ErrRefused: the model declined the request.
	ErrRefused = errors.New("llm: declined")
)

// Part is one block of the user message: text, or a JPEG.
type Part struct {
	Text      string
	ImageJPEG string // base64, no newlines; when set, Text is ignored
}

// Request is one call. It has no field for a user, an org or a project, so
// none can be sent.
type Request struct {
	Name      string // for the log, e.g. "tender_pass"
	Model     string
	System    string
	Parts     []Part
	Schema    map[string]any // JSON schema the answer must follow
	MaxTokens int64
	Timeout   time.Duration // zero means two minutes
}

// Response is the model's answer.
type Response struct {
	Text       string
	StopReason string // "end_turn", "max_tokens", ...
	InTokens   int64
	OutTokens  int64
	Latency    time.Duration
}

// Asker is what callers depend on, so tests and benches can stand in.
type Asker interface {
	Ask(ctx context.Context, req Request) (Response, error)
}

type Options struct {
	APIKey        string
	Logger        *slog.Logger
	MaxConcurrent int // zero means 4
}

type Client struct {
	send      func(context.Context, Request) (Response, error)
	slots     chan struct{}
	log       *slog.Logger
	now       func() time.Time
	mu        sync.Mutex
	fails     int
	openUntil time.Time
}

// New returns a client. With no key every Ask returns ErrOff.
func New(opts Options) *Client {
	var send func(context.Context, Request) (Response, error)
	if opts.APIKey != "" {
		send = sdkSend(anthropic.NewClient(option.WithAPIKey(opts.APIKey), option.WithMaxRetries(1)))
	}
	return newWithSend(opts, send)
}

func newWithSend(opts Options, send func(context.Context, Request) (Response, error)) *Client {
	n := opts.MaxConcurrent
	if n <= 0 {
		n = 4
	}
	log := opts.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Client{send: send, slots: make(chan struct{}, n), log: log, now: time.Now}
}

// On reports whether a key is configured.
func (c *Client) On() bool { return c != nil && c.send != nil }

func (c *Client) Ask(ctx context.Context, req Request) (Response, error) {
	if !c.On() {
		return Response{}, ErrOff
	}
	c.mu.Lock()
	open := c.now().Before(c.openUntil)
	c.mu.Unlock()
	if open {
		return Response{}, ErrBusy
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return Response{}, ctx.Err()
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	began := c.now()
	res, err := c.send(ctx, req)
	res.Latency = c.now().Sub(began)
	c.mu.Lock()
	switch {
	case err == nil || errors.Is(err, ErrRefused):
		c.fails = 0
	default:
		c.fails++
		if c.fails >= 5 {
			c.openUntil, c.fails = c.now().Add(time.Minute), 0
		}
	}
	c.mu.Unlock()
	class := "ok"
	if err != nil {
		class = errorClass(err)
	}
	// Never log the request or the answer: they hold a firm's prices.
	c.log.Info("llm call", "name", req.Name, "model", req.Model, "class", class, "stop", res.StopReason,
		"latency_ms", res.Latency.Milliseconds(), "in_tokens", res.InTokens, "out_tokens", res.OutTokens)
	return res, err
}

func errorClass(err error) string {
	var apiErr *anthropic.Error
	switch {
	case errors.Is(err, ErrRefused):
		return "refused"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.As(err, &apiErr):
		return "http_" + http.StatusText(apiErr.StatusCode)
	}
	return "transport"
}

// Retryable reports whether a later attempt could succeed.
func Retryable(err error) bool {
	if err == nil || errors.Is(err, ErrOff) || errors.Is(err, ErrRefused) {
		return false
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 408 || apiErr.StatusCode == 429 || apiErr.StatusCode >= 500
	}
	return true
}

func sdkSend(client anthropic.Client) func(context.Context, Request) (Response, error) {
	return func(ctx context.Context, req Request) (Response, error) {
		blocks := make([]anthropic.ContentBlockParamUnion, 0, len(req.Parts))
		for _, p := range req.Parts {
			if p.ImageJPEG != "" {
				blocks = append(blocks, anthropic.NewImageBlockBase64("image/jpeg", p.ImageJPEG))
				continue
			}
			blocks = append(blocks, anthropic.NewTextBlock(p.Text))
		}
		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(req.Model),
			MaxTokens: req.MaxTokens,
			System:    []anthropic.TextBlockParam{{Text: req.System}},
			Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(blocks...)},
			// Thinking off and low effort were the fastest set-up that held
			// accuracy in the trial (design section 14).
			Thinking: anthropic.ThinkingConfigParamUnion{OfDisabled: &anthropic.ThinkingConfigDisabledParam{}},
			OutputConfig: anthropic.OutputConfigParam{
				Effort: anthropic.OutputConfigEffortLow,
				Format: anthropic.JSONOutputFormatParam{Schema: req.Schema},
			},
		}
		// Streamed, so a long answer does not hit an HTTP timeout.
		stream := client.Messages.NewStreaming(ctx, params)
		msg := anthropic.Message{}
		for stream.Next() {
			if err := msg.Accumulate(stream.Current()); err != nil {
				return Response{}, err
			}
		}
		if err := stream.Err(); err != nil {
			return Response{}, err
		}
		out := Response{StopReason: string(msg.StopReason), InTokens: msg.Usage.InputTokens, OutTokens: msg.Usage.OutputTokens}
		if msg.StopReason == anthropic.StopReasonRefusal {
			return out, ErrRefused
		}
		for _, block := range msg.Content {
			if t, ok := block.AsAny().(anthropic.TextBlock); ok {
				out.Text += t.Text
			}
		}
		return out, nil
	}
}
