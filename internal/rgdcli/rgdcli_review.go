package rgdcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/k-shibuki/reinguard/internal/configdir"
	"github.com/k-shibuki/reinguard/internal/githubapi"
	"github.com/urfave/cli/v2"
)

func runReviewReplyThread(c *cli.Context) error {
	wd, err := reviewWorkDir(c)
	if err != nil {
		return err
	}
	body, err := reviewBodyFromFlags(wd, c)
	if err != nil {
		return err
	}
	return githubapi.ReplyToPullRequestThread(c.Context, wd, githubapi.PullRequestThreadReplyInput{
		PRNumber:  c.Int("pr"),
		Body:      body,
		InReplyTo: c.Int("in-reply-to"),
		CommitSHA: c.String("commit-sha"),
		Path:      c.String("path"),
		Line:      c.Int("line"),
	})
}

func runReviewResolveThread(c *cli.Context) error {
	wd, err := reviewWorkDir(c)
	if err != nil {
		return err
	}
	return githubapi.ResolveReviewThread(c.Context, wd, c.String("thread-id"))
}

func reviewWorkDir(c *cli.Context) (string, error) {
	if wd := strings.TrimSpace(c.String("cwd")); wd != "" {
		return wd, nil
	}
	return configdir.WorkingDir()
}

func reviewBodyFromFlags(wd string, c *cli.Context) (string, error) {
	inline := c.String("body")
	path := c.String("body-file")
	hasInline := inline != ""
	hasPath := path != ""
	switch {
	case hasInline && hasPath:
		return "", fmt.Errorf("--body and --body-file are mutually exclusive")
	case hasInline:
		if strings.TrimSpace(inline) == "" {
			return "", fmt.Errorf("--body must be non-empty")
		}
		return inline, nil
	case hasPath:
		data, err := os.ReadFile(resolveInputPath(wd, path))
		if err != nil {
			return "", err
		}
		body := string(data)
		if strings.TrimSpace(body) == "" {
			return "", fmt.Errorf("--body-file must be non-empty")
		}
		return body, nil
	default:
		return "", fmt.Errorf("either --body or --body-file is required")
	}
}

func reviewReplyThreadFlags() []cli.Flag {
	return []cli.Flag{
		newCwdFlag(),
		&cli.IntFlag{Name: "pr", Required: true, Usage: "pull request number"},
		&cli.IntFlag{Name: "in-reply-to", Required: true, Usage: "root review comment database id"},
		&cli.StringFlag{Name: "commit-sha", Required: true, Usage: "full 40-character commit SHA"},
		&cli.StringFlag{Name: "path", Required: true, Usage: "review comment file path"},
		&cli.IntFlag{Name: "line", Required: true, Usage: "review comment line number"},
		&cli.StringFlag{Name: "body", Usage: "reply body text"},
		&cli.StringFlag{Name: "body-file", Usage: "path to a file containing the reply body"},
	}
}

func reviewResolveThreadFlags() []cli.Flag {
	return []cli.Flag{
		newCwdFlag(),
		&cli.StringFlag{Name: "thread-id", Required: true, Usage: "GraphQL review thread node id"},
	}
}
