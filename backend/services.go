package backend

import "context"

// The services between external plugins (IDE 1.24): a backend calls the backend providing a
// capability its manifest requires, and hears the events it emits.

type fromKey struct{}

// FromOf is the plugin whose backend made the call through ServiceCall ("" for a window).
func FromOf(ctx context.Context) string {
	f, _ := ctx.Value(fromKey{}).(string)
	return f
}

// ServiceCall calls method (of the provider's namespace, `<provider>.*`) of the backend
// providing capability, one the manifest requires; its result is decoded into out (nil:
// dropped). The provider's method sees the caller with FromOf.
func ServiceCall(ctx context.Context, capability, method string, params, out any) error {
	return Core(ctx, "service.call", map[string]any{"capability": capability, "method": method, "params": params, "lang": LangOf(ctx)}, out)
}

// ServiceOn makes the events the backend providing capability emits reach Plugin.Events, while
// both run (again after this backend starts again); it answers the provider's id.
func ServiceOn(ctx context.Context, capability string) (string, error) {
	var r struct {
		Provider string `json:"provider"`
	}
	err := Core(ctx, "service.on", map[string]any{"capability": capability}, &r)
	return r.Provider, err
}
