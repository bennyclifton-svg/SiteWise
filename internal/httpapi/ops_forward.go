package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// A forwarded operation runs the screen's own handler for one write route,
// with the caller's own session, inside the operation's marked change. The
// handler validates exactly as it does for the screen and calls the same
// store write, so the record still has one writer and one set of rules.
//
// Whether such an operation applies at once depends on Undo:
//
//   - Undo true: every table the route writes carries the capture trigger,
//     so the change log can put it back. It applies at once.
//   - Undo false: the change could not be undone from chat, so it is
//     prepared and waits for the PM's click (owner's rule: apply at once
//     only with Undo).
//   - Click true: the PM vouches for something by doing it. Always prepared.
type forward struct {
	Name, About, Fields string
	Route               string   // "POST /projects/{id}/procurement/rows"
	Also                []string // other routes the same operation stands for
	Undo, Click         bool
	Figures             map[string]string
	// Send turns the fields into what the screen would send: values for the
	// route's path variables, a query string, and the JSON body. Nil sends
	// the fields as the body, taking path variables from fields of the same
	// name.
	Send func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error)
	// Done says in plain words what was changed.
	Done func(f map[string]any) string
	// Card, when set, checks the fields when chat prepares the card and
	// gives the card's own words (operation.Card).
	Card func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error)
}

type sent struct {
	Route string // overrides forward.Route when the fields choose between routes
	Path  map[string]string
	Query url.Values
	Body  any
}

type opRequestKey struct{}

// withOpRequest keeps the caller's request, so a forwarded operation can
// act with the same session and origin. Nothing else is taken from it.
func withOpRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, opRequestKey{}, r)
}

// routeRecorder captures a handler's answer.
type routeRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *routeRecorder) Header() http.Header { return r.header }
func (r *routeRecorder) WriteHeader(s int) {
	if r.status == 0 {
		r.status = s
	}
}
func (r *routeRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(b)
}

// callRoute runs one route's handler in this process.
func callRoute(ctx context.Context, d Deps, project, route string, s sent) (int, []byte, error) {
	method, pattern, _ := strings.Cut(route, " ")
	handler := apiRoutes()[route]
	origin, _ := ctx.Value(opRequestKey{}).(*http.Request)
	if handler == nil || origin == nil {
		return 0, nil, refuse("", "This cannot be done from here.")
	}
	path := pattern
	values := map[string]string{"id": project}
	for k, v := range s.Path {
		values[k] = v
	}
	for name, v := range values {
		path = strings.ReplaceAll(path, "{"+name+"}", url.PathEscape(v))
	}
	if strings.Contains(path, "{") {
		return 0, nil, refuse("", "A field this operation needs is missing.")
	}
	var body bytes.Buffer
	if s.Body != nil {
		if err := json.NewEncoder(&body).Encode(s.Body); err != nil {
			return 0, nil, err
		}
	}
	target := "/api" + path
	if len(s.Query) > 0 {
		target += "?" + s.Query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, &body)
	if err != nil {
		return 0, nil, err
	}
	for name, v := range values {
		req.SetPathValue(name, v)
	}
	// The mux sets the matched pattern; writeSpecification chooses its write
	// by it, so a forwarded call must carry it too.
	req.Pattern = route
	for _, h := range []string{"Cookie", "Origin"} {
		if v := origin.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Host = origin.Host
	rec := &routeRecorder{header: http.Header{}}
	handler(rec, req, d)
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	return rec.status, rec.body.Bytes(), nil
}

// routeRefusal is the handler's own words for a request it refused.
func routeRefusal(status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	var j struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(body, &j) == nil && (j.Message != "" || j.Error != "") {
		msg = j.Message
		if msg == "" {
			msg = j.Error
		}
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	if status == http.StatusConflict && !strings.Contains(msg, " ") {
		msg = "This changed while the edit was being made. Read it again."
	}
	return &opError{Message: strings.ToUpper(msg[:1]) + msg[1:], status: status}
}

func (fw forward) operation() operation {
	op := operation{Name: fw.Name, About: fw.About, Fields: fw.Fields, Figures: fw.Figures, Click: fw.Click, NoUndo: !fw.Undo,
		Routes: append([]string{fw.Route}, fw.Also...)}
	if !fw.Undo && !fw.Click {
		op.Hold = func(context.Context, Deps, opActor, string, json.RawMessage) (string, error) {
			return "This cannot be undone from chat yet, so it waits for your click.", nil
		}
	}
	if fw.Card != nil {
		op.Card = func(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
			f, err := forwardFields(raw)
			if err != nil {
				return "", err
			}
			return fw.Card(ctx, d, who, project, f)
		}
	}
	op.Run = func(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
		f, err := forwardFields(raw)
		if err != nil {
			return "", err
		}
		s := sent{Body: f}
		if fw.Send != nil {
			var err error
			if s, err = fw.Send(ctx, d, who, project, f); err != nil {
				return "", err
			}
		} else {
			// Path variables come out of the fields; the rest is the body.
			s.Path = map[string]string{}
			_, pattern, _ := strings.Cut(fw.Route, " ")
			for _, part := range strings.Split(pattern, "/") {
				if name := strings.Trim(part, "{}"); name != part && name != "id" {
					v, _ := f[name].(string)
					s.Path[name] = v
					delete(f, name)
				}
			}
		}
		route := fw.Route
		if s.Route != "" {
			route = s.Route
		}
		status, body, err := callRoute(ctx, d, project, route, s)
		if err != nil {
			return "", err
		}
		if status < 200 || status > 299 {
			return "", routeRefusal(status, body)
		}
		if fw.Done != nil {
			return fw.Done(f), nil
		}
		return strings.ToUpper(fw.About[:1]) + strings.TrimSuffix(fw.About[1:], ".") + ": done.", nil
	}
	return op
}

// forwardFields reads a forwarded operation's fields as the screen's JSON.
func forwardFields(raw json.RawMessage) (map[string]any, error) {
	f := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, refuse("", "The fields are not in the shape this operation takes.")
		}
	}
	if f == nil { // "fields": null
		f = map[string]any{}
	}
	return f, nil
}

// str reads a text field.
func fieldText(f map[string]any, name string) string {
	s, _ := f[name].(string)
	return strings.TrimSpace(s)
}

// need refuses when a field is missing.
func need(f map[string]any, names ...string) error {
	for _, n := range names {
		if v, ok := f[n]; !ok || v == nil || v == "" {
			return refuse(n, "%s is needed.", n)
		}
	}
	return nil
}

func quoted(s string) string { return fmt.Sprintf("%q", s) }
