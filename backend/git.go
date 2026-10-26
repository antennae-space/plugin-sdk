package backend

import "context"

// Git and the worktree projects, run by the IDE for the backend (IDE 1.24): git needs the
// user's keys and configuration, the worktrees are projects of the IDE.

// Git runs git with args in dir (the project's main folder for "", or one of its
// .ide/worktrees/*), with the user's keys (grant git); an option that would run another program
// is refused. It answers git's output.
func Git(ctx context.Context, dir string, args ...string) (string, error) {
	var r struct {
		Output string `json:"output"`
	}
	err := Core(ctx, "git", map[string]any{"args": args, "dir": dir}, &r)
	return r.Output, err
}

// GitAny is Git with its output whatever its exit code (git diff --no-index exits 1 on a
// difference), and that code; an error only when git could not run (IDE 1.24).
func GitAny(ctx context.Context, dir string, args ...string) (string, int, error) {
	var r struct {
		Output string `json:"output"`
		Exit   int    `json:"exit"`
	}
	err := Core(ctx, "git", map[string]any{"args": args, "dir": dir, "anyExit": true}, &r)
	return r.Output, r.Exit, err
}

// GH runs gh pr … (grant gh) in the project's main folder.
func GH(ctx context.Context, args ...string) (string, error) {
	var r struct {
		Output string `json:"output"`
	}
	err := Core(ctx, "gh", map[string]any{"args": args}, &r)
	return r.Output, err
}

// RegisterWorktree makes a folder of .ide/worktrees the project of the ticket owner, titled
// title (grant worktrees); it answers the id of that project.
func RegisterWorktree(ctx context.Context, owner int64, title, path string) (string, error) {
	var r struct {
		Project string `json:"project"`
	}
	err := Core(ctx, "worktree.register", map[string]any{"owner": owner, "title": title, "path": path}, &r)
	return r.Project, err
}

// UnregisterWorktree removes the project of the worktree of the ticket owner (its folder stays).
func UnregisterWorktree(ctx context.Context, owner int64) error {
	return Core(ctx, "worktree.unregister", map[string]any{"owner": owner}, nil)
}

// SetupWorktree runs the setup commands of the project in a worktree: their output, and whether
// they all succeeded.
func SetupWorktree(ctx context.Context, path string) (output string, ok bool, err error) {
	var r struct {
		Output string `json:"output"`
		OK     bool   `json:"ok"`
	}
	err = Core(ctx, "worktree.setup", map[string]any{"path": path}, &r)
	return r.Output, r.OK, err
}

// KeepIdeOut keeps the .ide folder of the project out of its git repository.
func KeepIdeOut(ctx context.Context) error { return Core(ctx, "ide.keepOut", map[string]any{}, nil) }
