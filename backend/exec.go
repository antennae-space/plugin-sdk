package backend

import "context"

// Exec runs a program on the host of an SSH project (the grant exec:<name>, exec:sqlite3) in
// dir, a folder of the project ("" its root), and returns its output: the backend runs on the
// pod and cannot reach that host. Nothing confines the program there. On a local project the
// IDE refuses it: run the program yourself, confined with the backend.
func Exec(ctx context.Context, argv []string, dir string) (string, error) {
	var r struct {
		Output string `json:"output"`
	}
	err := Core(ctx, "exec", map[string]any{"argv": argv, "dir": dir}, &r)
	return r.Output, err
}
