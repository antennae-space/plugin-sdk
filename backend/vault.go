package backend

import "context"

// Secret reads an entry of the plugin's vault (the grant vault): its value, and whether it
// exists. The vault is synced with the user's other pods; a plugin reaches its own entries only.
func Secret(ctx context.Context, key string) (string, bool, error) {
	var r struct {
		Value string `json:"value"`
		Found bool   `json:"found"`
	}
	err := Core(ctx, "vault.get", map[string]string{"key": key}, &r)
	return r.Value, r.Found, err
}

// SetSecret writes an entry of the plugin's vault.
func SetSecret(ctx context.Context, key, value string) error {
	return Core(ctx, "vault.set", map[string]string{"key": key, "value": value}, nil)
}

// DeleteSecret removes an entry of the plugin's vault.
func DeleteSecret(ctx context.Context, key string) error {
	return Core(ctx, "vault.delete", map[string]string{"key": key}, nil)
}
