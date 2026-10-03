// Package jev is the bounded System One client.
//
// One call posts a question map to the pinned model. Code admits the call,
// validates the response, and decides what to do with it.
//
//	https://docs.typesafe.ai/api
//	https://docs.typesafe.ai/models
//	https://docs.typesafe.ai/patterns/fan-out
//	https://docs.typesafe.ai/confidence
//	https://docs.typesafe.ai/concepts/how-to-build-with-system-one
//	https://docs.typesafe.ai/model-jaggedness/jev-1.13
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"sitewise/internal/config"
)

// Options configures a Client. Zero values take the production defaults.
// A non-zero Slots value must be paired with InteractiveReserve; the default
// reserve belongs to the default pool of TotalSlots.
type Options struct {
	BaseURL string
	APIKey  string
	Model   string
	// Transport defaults to NewTransport. Evaluation passes a recording or
	// replaying round tripper; production keeps the default.
	Transport          http.RoundTripper
	Logger             *slog.Logger
	Slots              int
	InteractiveReserve int
	Deadline           time.Duration
	MaxBody            int64
	BreakerThreshold   int
	BreakerCooldown    time.Duration
	BackgroundAttempts int
	Backoff            time.Duration
	Now                func() time.Time
	Sleep              func(context.Context, time.Duration) error
}

// Client talks to System One with a reused HTTP transport, a slot limit and
// a circuit breaker. Interactive calls are not retried.
type Client struct {
	baseURL            string
	apiKey             string
	model              string
	http               *http.Client
	admission          *Admission
	breaker            *breaker
	log                *slog.Logger
	deadline           time.Duration
	maxBody            int64
	backgroundAttempts int
	backoff            time.Duration
	now                func() time.Time
	sleep              func(context.Context, time.Duration) error
	// reached is the unix-nano time of the last HTTP response from the
	// endpoint, whatever its status; zero means none yet.
	reached atomic.Int64
}

type outbound struct {
	State     any                 `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]Question `json:"questions"`
}

type responseWire struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type answerWire struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul"`
	Choice        string             `json:"choice"`
	Score         *float64           `json:"score"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    *float64           `json:"confidence"`
	Legend        map[string]string  `json:"legend"`
}

// NewTransport is the process transport: idle HTTP/2 connections are reused
// up to the slot cap. The call context is the deadline; the transport does
// not add a second header timer.
func NewTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          TotalSlots,
		MaxIdleConnsPerHost:   TotalSlots,
		MaxConnsPerHost:       TotalSlots,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

// New builds a client. The model must be the pinned version; a moving alias
// is rejected. The API key is never copied into errors.
func New(opts Options) (*Client, error) {
	key := strings.TrimSpace(opts.APIKey)
	if key == "" {
		return nil, fmt.Errorf("%w: api key", ErrRequest)
	}
	model := opts.Model
	if model == "" {
		model = config.PinnedJevModel
	}
	if model != config.PinnedJevModel {
		return nil, fmt.Errorf("jev model %q is not the pinned version %s", model, config.PinnedJevModel)
	}
	base := opts.BaseURL
	if base == "" {
		base = Endpoint
	}
	if !strings.HasPrefix(base, "https://") && !strings.HasPrefix(base, "http://") {
		return nil, fmt.Errorf("%w: endpoint", ErrRequest)
	}

	slots := opts.Slots
	reserve := opts.InteractiveReserve
	if slots == 0 {
		slots = TotalSlots
	}
	if reserve == 0 {
		if slots != TotalSlots {
			return nil, fmt.Errorf("%w: interactive reserve", ErrRequest)
		}
		reserve = InteractiveReserve
	}
	admission, err := newAdmission(slots, reserve)
	if err != nil {
		return nil, err
	}

	deadline := opts.Deadline
	if deadline == 0 {
		deadline = InteractiveDeadline
	}
	if deadline < 0 {
		return nil, fmt.Errorf("%w: deadline", ErrRequest)
	}
	maxBody := opts.MaxBody
	if maxBody == 0 {
		maxBody = MaxResponseBytes
	}
	if maxBody < 1 {
		return nil, fmt.Errorf("%w: body limit", ErrRequest)
	}
	threshold := opts.BreakerThreshold
	if threshold == 0 {
		threshold = breakerThreshold
	}
	if threshold < 1 {
		return nil, fmt.Errorf("%w: breaker", ErrRequest)
	}
	cooldown := opts.BreakerCooldown
	if cooldown == 0 {
		cooldown = breakerCooldown
	}
	if cooldown < 0 {
		return nil, fmt.Errorf("%w: breaker", ErrRequest)
	}
	attempts := opts.BackgroundAttempts
	if attempts == 0 {
		attempts = backgroundAttempts
	}
	if attempts < 1 || attempts > 8 {
		return nil, fmt.Errorf("%w: background attempts", ErrRequest)
	}
	backoff := opts.Backoff
	if backoff == 0 {
		backoff = backgroundBackoff
	}
	if backoff < 0 {
		return nil, fmt.Errorf("%w: backoff", ErrRequest)
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	transport := opts.Transport
	if transport == nil {
		transport = NewTransport()
	}

	return &Client{
		baseURL:            base,
		apiKey:             key,
		model:              model,
		admission:          admission,
		breaker:            newBreaker(threshold, cooldown),
		log:                logger,
		deadline:           deadline,
		maxBody:            maxBody,
		backgroundAttempts: attempts,
		backoff:            backoff,
		now:                now,
		sleep:              sleep,
		http: &http.Client{
			Transport: transport,
			// Redirects would resend the bearer token on this host.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Warm opens a connection on the reused transport so the first filing call
// does not pay the handshake. Any HTTP response counts as warm; the API key
// is not sent.
func (c *Client) Warm(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.markReached()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return nil
}

// Probe warms the connection every interval until ctx ends, so health can
// tell whether the provider is reachable without asking Jev anything. The
// probe is a HEAD without the API key: it spends no evaluation and no slot.
func (c *Client) Probe(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		probe, cancel := context.WithTimeout(ctx, every)
		_ = c.Warm(probe)
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Status is the provider state health reports.
type Status struct {
	// Circuit is CircuitClosed, CircuitOpen or CircuitHalfOpen.
	Circuit string
	// Reached is the last HTTP response from the endpoint, from a probe or a
	// call; zero means none since start.
	Reached time.Time
}

// Status reads the breaker and last reach without blocking a call.
func (c *Client) Status() Status {
	s := Status{Circuit: c.breaker.State()}
	if n := c.reached.Load(); n != 0 {
		s.Reached = time.Unix(0, n).In(c.now().Location())
	}
	return s
}

func (c *Client) markReached() {
	c.reached.Store(c.now().UnixNano())
}

// Ask posts one fan-out. Valid answers are returned even when other questions
// are missing or rejected. Nothing is invented for a question that failed
// validation. Interactive calls make one HTTP attempt.
func (c *Client) Ask(ctx context.Context, call Call) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateCall(&call); err != nil {
		return Result{}, err
	}
	payload, err := json.Marshal(outbound{
		State:     call.State,
		Model:     c.model,
		Questions: call.Questions,
	})
	if err != nil {
		return Result{}, fmt.Errorf("%w: encode", ErrRequest)
	}
	attempts := 1
	if call.Priority == PriorityBackground {
		attempts = c.backgroundAttempts
	}
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			wait := c.backoff << (attempt - 1)
			if err := c.sleep(ctx, wait); err != nil {
				return Result{}, err
			}
		}
		result, err, retry := c.attempt(ctx, payload, call)
		if err == nil {
			return result, nil
		}
		last = err
		if !retry {
			return Result{}, err
		}
	}
	return Result{}, last
}

func (c *Client) attempt(parent context.Context, payload []byte, call Call) (Result, error, bool) {
	start := c.now()
	if err := c.breaker.Allow(start); err != nil {
		c.logCall(slog.LevelWarn, call, 0, c.now().Sub(start), Usage{}, nil, nil, errorClass(err))
		return Result{}, err, false
	}
	settled := false
	defer func() {
		if !settled {
			c.breaker.Abandon()
		}
	}()

	ctx, cancel := c.withDeadline(parent, call)
	defer cancel()
	if err := c.admission.Acquire(ctx, call.Priority); err != nil {
		c.logCall(slog.LevelWarn, call, 0, c.now().Sub(start), Usage{}, nil, nil, errorClass(err))
		return Result{}, err, call.Priority == PriorityBackground && parent.Err() == nil
	}
	defer c.admission.Release(call.Priority)

	status, body, err := c.post(ctx, payload)
	latency := c.now().Sub(start)
	if err != nil {
		if errors.Is(err, ErrBodyLimit) {
			settled = true
			c.breaker.Success()
			c.logCall(slog.LevelWarn, call, status, latency, Usage{}, nil, nil, errorClass(err))
			return Result{}, err, false
		}
		settled = true
		c.breaker.Fail(c.now())
		c.logCall(slog.LevelWarn, call, status, latency, Usage{}, nil, nil, errorClass(err))
		if parent.Err() != nil {
			return Result{}, err, false
		}
		return Result{}, err, call.Priority == PriorityBackground
	}
	settled = true
	return c.interpret(call, status, body, latency)
}

func (c *Client) withDeadline(parent context.Context, call Call) (context.Context, context.CancelFunc) {
	d := c.deadline
	if call.Deadline > 0 {
		d = call.Deadline
	}
	return context.WithTimeout(parent, d)
}

func (c *Client) post(ctx context.Context, payload []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sitewise")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	c.markReached()
	body, tooBig, err := readBounded(resp.Body, c.maxBody)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil, nil
	}
	if tooBig {
		return resp.StatusCode, nil, ErrBodyLimit
	}
	return resp.StatusCode, body, nil
}

func readBounded(r io.Reader, max int64) ([]byte, bool, error) {
	body, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > max {
		return nil, true, nil
	}
	return body, false, nil
}

func (c *Client) interpret(call Call, status int, body []byte, latency time.Duration) (Result, error, bool) {
	if status != http.StatusOK {
		err, retry := statusError(status)
		if retry {
			c.breaker.Fail(c.now())
		} else {
			c.breaker.Success()
		}
		c.logCall(slog.LevelWarn, call, status, latency, Usage{}, nil, nil, errorClass(err))
		if retry && call.Priority != PriorityBackground {
			retry = false
		}
		return Result{}, err, retry
	}
	var wire responseWire
	if err := json.Unmarshal(body, &wire); err != nil {
		c.breaker.Fail(c.now())
		c.logCall(slog.LevelWarn, call, status, latency, Usage{}, nil, nil, "bad_response")
		return Result{}, ErrBadResponse, false
	}
	usage := Usage{InputTokens: wire.Usage.InputTokens, OutputTokens: wire.Usage.OutputTokens}
	if wire.Model != c.model {
		c.breaker.Success()
		c.logCall(slog.LevelWarn, call, status, latency, usage, nil, nil, "wrong_model")
		return Result{}, ErrBadResponse, false
	}
	answers := make(map[string]Answer, len(call.Questions))
	unresolved := make([]string, 0)
	for id, q := range call.Questions {
		raw, ok := wire.Answers[id]
		if !ok || string(bytes.TrimSpace(raw)) == "" || string(raw) == "null" {
			unresolved = append(unresolved, id)
			continue
		}
		ans, ok := validateOne(q, raw)
		if !ok {
			unresolved = append(unresolved, id)
			continue
		}
		answers[id] = ans
	}
	slices.Sort(unresolved)
	c.breaker.Success()
	result := Result{
		Model:           wire.Model,
		Answers:         answers,
		Unresolved:      unresolved,
		Usage:           usage,
		Latency:         latency,
		QuestionVersion: call.QuestionVersion,
	}
	c.logCall(slog.LevelInfo, call, status, latency, usage, uncertaintyOf(answers), unresolved, "")
	return result, nil, false
}

func statusError(code int) (error, bool) {
	switch code {
	case http.StatusTooManyRequests:
		return ErrRateLimited, true
	case 529:
		return ErrOverloaded, true
	case http.StatusUnauthorized:
		return ErrUnauthorized, false
	case http.StatusUnprocessableEntity:
		return ErrRequest, false
	default:
		if code >= 500 && code <= 599 {
			return fmt.Errorf("%w: status %d", ErrBadResponse, code), true
		}
		if code != http.StatusOK {
			return fmt.Errorf("%w: status %d", ErrBadResponse, code), false
		}
		return nil, false
	}
}

func validateCall(call *Call) error {
	if call.State == nil {
		return fmt.Errorf("%w: state", ErrRequest)
	}
	switch call.Priority {
	case "":
		call.Priority = PriorityInteractive
	case PriorityInteractive, PriorityBackground:
	default:
		return fmt.Errorf("%w: priority", ErrRequest)
	}
	if len(call.Questions) == 0 {
		return fmt.Errorf("%w: questions", ErrRequest)
	}
	for id, q := range call.Questions {
		if id == "" {
			return fmt.Errorf("%w: question id", ErrRequest)
		}
		if err := validateQuestion(q); err != nil {
			return fmt.Errorf("%w: question %q", ErrRequest, id)
		}
	}
	return nil
}

func validateQuestion(q Question) error {
	if !instructionsOK(q.Instructions) {
		return errors.New("instructions")
	}
	switch q.Type {
	case TypeNoul:
		if q.Criteria == nil {
			return nil
		}
		raw, err := json.Marshal(q.Criteria)
		if err != nil || len(raw) == 0 || raw[0] != '{' {
			return errors.New("noul criteria")
		}
		return nil
	case TypeChoice:
		_, err := choiceOptions(q.Criteria)
		return err
	case TypeScore:
		_, err := scoreLevels(q.Criteria)
		return err
	default:
		return errors.New("type")
	}
}

func instructionsOK(v any) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) != ""
	case json.RawMessage:
		text := bytes.TrimSpace(x)
		return len(text) > 0 && string(text) != "null" && string(text) != `""`
	default:
		return true
	}
}

func choiceOptions(criteria any) (map[string]struct{}, error) {
	raw, err := json.Marshal(criteria)
	if err != nil {
		return nil, errors.New("choice criteria")
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded) == 0 || len(decoded) > MaxChoiceOptions {
		return nil, errors.New("choice criteria")
	}
	out := make(map[string]struct{}, len(decoded))
	for key := range decoded {
		if key == "" {
			return nil, errors.New("choice criteria")
		}
		out[key] = struct{}{}
	}
	return out, nil
}

func scoreLevels(criteria any) ([]string, error) {
	raw, err := json.Marshal(criteria)
	if err != nil {
		return nil, errors.New("score criteria")
	}
	var levels []string
	if err := json.Unmarshal(raw, &levels); err != nil {
		return nil, errors.New("score criteria")
	}
	if len(levels) < 2 || len(levels) > 10 {
		return nil, errors.New("score criteria")
	}
	for _, level := range levels {
		if strings.TrimSpace(level) == "" {
			return nil, errors.New("score criteria")
		}
	}
	return levels, nil
}

func validateOne(q Question, raw json.RawMessage) (Answer, bool) {
	var parsed answerWire
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Answer{}, false
	}
	if parsed.Type != q.Type {
		return Answer{}, false
	}
	switch q.Type {
	case TypeNoul:
		return validateNoul(parsed)
	case TypeChoice:
		return validateChoice(q, parsed)
	case TypeScore:
		return validateScore(q, parsed)
	default:
		return Answer{}, false
	}
}

func validateNoul(parsed answerWire) (Answer, bool) {
	if parsed.Noul == nil || !finiteUnit(*parsed.Noul) {
		return Answer{}, false
	}
	return Answer{Type: TypeNoul, Noul: *parsed.Noul}, true
}

func validateChoice(q Question, parsed answerWire) (Answer, bool) {
	options, err := choiceOptions(q.Criteria)
	if err != nil {
		return Answer{}, false
	}
	if _, ok := options[parsed.Choice]; !ok {
		return Answer{}, false
	}
	if !distributionOK(parsed.Probabilities, options) {
		return Answer{}, false
	}
	conf, ok := optionalConfidence(parsed.Confidence)
	if !ok {
		return Answer{}, false
	}
	return Answer{
		Type:          TypeChoice,
		Choice:        parsed.Choice,
		Probabilities: parsed.Probabilities,
		Confidence:    conf,
	}, true
}

func validateScore(q Question, parsed answerWire) (Answer, bool) {
	levels, err := scoreLevels(q.Criteria)
	if err != nil {
		return Answer{}, false
	}
	if parsed.Score == nil || math.IsNaN(*parsed.Score) || math.IsInf(*parsed.Score, 0) {
		return Answer{}, false
	}
	if *parsed.Score < 0 || *parsed.Score > float64(len(levels)-1) {
		return Answer{}, false
	}
	if len(parsed.Legend) != len(levels) {
		return Answer{}, false
	}
	keys := make(map[string]struct{}, len(levels))
	for i, level := range levels {
		key := strconv.Itoa(i)
		if parsed.Legend[key] != level {
			return Answer{}, false
		}
		keys[key] = struct{}{}
	}
	if !distributionOK(parsed.Probabilities, keys) {
		return Answer{}, false
	}
	conf, ok := optionalConfidence(parsed.Confidence)
	if !ok {
		return Answer{}, false
	}
	return Answer{
		Type:          TypeScore,
		Score:         *parsed.Score,
		Probabilities: parsed.Probabilities,
		Confidence:    conf,
	}, true
}

func optionalConfidence(v *float64) (*float64, bool) {
	if v == nil {
		return nil, true
	}
	if !finiteUnit(*v) {
		return nil, false
	}
	copied := *v
	return &copied, true
}

func distributionOK(probs map[string]float64, keys map[string]struct{}) bool {
	if len(probs) != len(keys) {
		return false
	}
	var sum float64
	centPrecision := true
	for key, value := range probs {
		if _, ok := keys[key]; !ok || !finiteUnit(value) {
			return false
		}
		sum += value
		centPrecision = centPrecision && math.Abs(value*100-math.Round(value*100)) < 1e-8
	}
	if math.Abs(sum-1) <= 1e-3 {
		return true
	}
	// The API specifies a unit-sum distribution (https://docs.typesafe.ai/api).
	// Recorded jev-1.13.0 responses sometimes round every probability to cents
	// and total 0.99. Tolerate only one cent at that precision; retain the raw
	// values and provider confidence rather than inventing a new confidence.
	return centPrecision && math.Abs(sum-1) <= .0100000001
}

func finiteUnit(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1
}

func uncertaintyOf(answers map[string]Answer) map[string]float64 {
	if len(answers) == 0 {
		return nil
	}
	out := make(map[string]float64, len(answers))
	for id, ans := range answers {
		switch ans.Type {
		case TypeNoul:
			out[id] = ans.Noul
		case TypeChoice, TypeScore:
			if ans.Confidence != nil {
				out[id] = *ans.Confidence
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *Client) logCall(level slog.Level, call Call, status int, latency time.Duration, usage Usage, uncertainty map[string]float64, unresolved []string, class string) {
	attrs := []any{
		slog.String("model", c.model),
		slog.String("question_version", call.QuestionVersion),
		slog.Int64("latency_us", latency.Microseconds()),
		slog.String("priority", string(call.Priority)),
		slog.Int("status", status),
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 {
		attrs = append(attrs,
			slog.Int("input_tokens", usage.InputTokens),
			slog.Int("output_tokens", usage.OutputTokens),
		)
	}
	if len(uncertainty) > 0 {
		attrs = append(attrs, slog.Any("uncertainty", uncertainty))
	}
	if len(unresolved) > 0 {
		attrs = append(attrs, slog.Any("unresolved", unresolved))
	}
	if class != "" {
		attrs = append(attrs, slog.String("error_class", class))
	}
	c.log.Log(context.Background(), level, "jev", attrs...)
}

func errorClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrCircuitOpen):
		return "circuit_open"
	case errors.Is(err, ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrOverloaded):
		return "overloaded"
	case errors.Is(err, ErrBodyLimit):
		return "body_limit"
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrRequest):
		return "request"
	case errors.Is(err, ErrBadResponse):
		return "bad_response"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "transport"
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
