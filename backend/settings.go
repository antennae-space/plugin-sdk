package backend

import "context"

// Settings decodes the plugin's settings (contributes.settings, the defaults where the user
// set nothing) into out.
func Settings(ctx context.Context, out any) error {
	return Core(ctx, "settings.get", map[string]any{}, out)
}

// Target is what an action of the plugin acts on (a connection), for the user's AI access.
type Target struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// DeclareTargets tells the IDE what the plugin's actions act on: Settings > Plugins lists them,
// each with the user's choice of what the AI may change there (read only by default).
func DeclareTargets(ctx context.Context, targets []Target) error {
	if targets == nil {
		targets = []Target{}
	}
	return Core(ctx, "ai.targets", map[string]any{"targets": targets}, nil)
}
