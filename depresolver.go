package depresolver

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"
)

const (
	queryTmpl         = "kind(\"source file\", deps(set(%s)))"
	buildQueryTmpl    = "buildfiles(deps(set(%s)))"
	externalPrefix    = "@"
	externalName      = "external"
	labelPrefix       = "//"
	workspaceFileName = "WORKSPACE"
)

type CommandDepResolver struct {
	BazelBin  string
	BazelArgs []string
}

func (c *CommandDepResolver) bazelCmd(ctx context.Context, extraArgs ...string) *exec.Cmd {
	return exec.CommandContext(ctx, c.BazelBin, append(c.BazelArgs, extraArgs...)...)
}

func (c *CommandDepResolver) info(ctx context.Context, noun string) (string, error) {
	cmd := c.bazelCmd(ctx, "info", noun)
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(stdout)), nil
}

func (c *CommandDepResolver) mapLabelsToFiles(labels []string, outputBase, workspace string) []string {
	files := make([]string, 0, len(labels))
	for _, label := range labels {
		if label == "" {
			continue
		}

		dir := workspace
		if strings.HasPrefix(label, externalPrefix) {
			repoName := label[len(externalPrefix):strings.Index(label, labelPrefix)]
			dir = path.Join(outputBase, externalName, repoName)
			label = label[len(repoName)+len(externalPrefix):]
		}

		label = strings.Replace(label[len(labelPrefix):], ":", "/", 1)
		files = append(files, path.Join(dir, label))
	}
	return files
}

func (c *CommandDepResolver) resolveQuery(ctx context.Context, query, outputBase, workspace string) ([]string, error) {
	cmd := c.bazelCmd(ctx, "query", query)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return c.mapLabelsToFiles(strings.Split(string(stdout), "\n"), outputBase, workspace), nil
}

func (c *CommandDepResolver) Resolve(ctx context.Context, target string) (sourceFiles, buildFiles []string, err error) {
	outputBase, err := c.info(ctx, "output_base")
	if err != nil {
		return nil, nil, err
	}
	workspace, err := c.info(ctx, "workspace")
	if err != nil {
		return nil, nil, err
	}

	sourceFiles, err = c.resolveQuery(ctx, fmt.Sprintf(queryTmpl, target), outputBase, workspace)
	if err != nil {
		return nil, nil, err
	}

	buildFiles, err = c.resolveQuery(ctx, fmt.Sprintf(buildQueryTmpl, target), outputBase, workspace)
	if err != nil {
		return nil, nil, err
	}
	// explicitly add the WORKSPACE file to the list of buildFiles we care about
	buildFiles = append(buildFiles, path.Join(workspace, workspaceFileName))

	return sourceFiles, buildFiles, err
}
