package backend

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// A service between plugins: the caller's id in a call, ServiceCall and ServiceOn through the
// core, an event of the provider to Events.
func TestServices(t *testing.T) {
	heard := make(chan string, 1)
	p := Plugin{
		Methods: map[string]Method{
			"beta.who": func(ctx context.Context, _ json.RawMessage) (any, error) { return FromOf(ctx), nil },
			"beta.ask": func(ctx context.Context, _ json.RawMessage) (any, error) {
				var s string
				if err := ServiceCall(ctx, "greeting", "alpha.greet", map[string]string{"name": "beta"}, &s); err != nil {
					return nil, err
				}
				provider, err := ServiceOn(ctx, "greeting")
				return s + " / " + provider, err
			},
		},
		Events: func(_ context.Context, event string, data json.RawMessage) { heard <- event + " " + string(data) },
	}
	f := start(t, p, func(method string, params json.RawMessage) (any, *Error) {
		switch method {
		case "service.call":
			var a struct{ Capability, Method string }
			_ = json.Unmarshal(params, &a)
			return a.Capability + ":" + a.Method, nil
		case "service.on":
			return map[string]string{"provider": "alpha"}, nil
		}
		return nil, &Error{Code: -32000, Message: "unexpected " + method}
	})
	f.ask("initialize", Info{APIVersion: 1, Plugin: "beta"})
	if r := f.ask("call", map[string]any{"method": "beta.who", "from": "gamma"}); string(r.Result) != `"gamma"` {
		t.Fatalf("from: %s", r.Result)
	}
	if r := f.ask("call", map[string]any{"method": "beta.ask"}); string(r.Result) != `"greeting:alpha.greet / alpha"` {
		t.Fatalf("service calls: %s %v", r.Result, r.Error)
	}
	f.write(message{JSONRPC: "2.0", Method: "event", Params: mustJSON(map[string]any{"event": "alpha.greeted", "data": "beta"})})
	select {
	case h := <-heard:
		if h != `alpha.greeted "beta"` {
			t.Fatal(h)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
}
