package issueworkflow

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kxn/codex-remote-feishu/internal/execlaunch"
)

type GitClient interface {
	TrackedDirtyFiles(context.Context) ([]string, error)
	PullFFOnly(context.Context) error
	CurrentBranch(context.Context) (string, error)
	HeadCommit(context.Context) (string, error)
	OriginRemoteURL(context.Context) (string, error)
}

type gitCLI struct {
	rootDir string
}

func NewGitCLI(rootDir string) GitClient {
	return &gitCLI{rootDir: rootDir}
}

func (g *gitCLI) TrackedDirtyFiles(ctx context.Context) ([]string, error) {
	output, err := g.run(ctx, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return nil, err
	}
	lines := splitNonEmptyLines(output)
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		files = append(files, filepath.Clean(strings.TrimSpace(line[3:])))
	}
	return files, nil
}

func (g *gitCLI) PullFFOnly(ctx context.Context) error {
	_, err := g.run(ctx, "pull", "--ff-only")
	return err
}

func (g *gitCLI) CurrentBranch(ctx context.Context) (string, error) {
	output, err := g.run(ctx, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (g *gitCLI) HeadCommit(ctx context.Context) (string, error) {
	output, err := g.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (g *gitCLI) OriginRemoteURL(ctx context.Context) (string, error) {
	output, err := g.run(ctx, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (g *gitCLI) run(ctx context.Context, args ...string) (string, error) {
	cmd := execlaunch.CommandContext(ctx, "git", args...)
	cmd.Dir = g.rootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return strings.TrimSpace(string(output)), fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return strings.TrimSpace(string(output)), nil
}

func splitNonEmptyLines(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
