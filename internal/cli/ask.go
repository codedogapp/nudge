package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

// ask runs nudge ask: it asks the question on the command line, and prints its answer.
func ask(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	question, ok := parseQuestion(args, stderr)
	if !ok {
		return exitUsage
	}

	got, err := send(ctx, question)
	if err != nil {
		return failed(stderr, "ask", err)
	}

	if _, err := io.WriteString(stdout, answerLines(got)); err != nil {
		return failed(stderr, "ask", fmt.Errorf("print the answer: %w", err))
	}

	return exitOK
}

// parseQuestion reads the question from args. On a usage error, it prints why, and the usage.
func parseQuestion(args []string, stderr io.Writer) (agentdoor.Question, bool) {
	var question agentdoor.Question

	flags := flag.NewFlagSet("nudge ask", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { misused(stderr) }
	flags.StringVar(&question.Details, "details", "", "what the user needs to know to answer")
	flags.Func("option", "an answer to offer; repeat it for more", func(option string) error {
		question.Options = append(question.Options, option)
		return nil
	})

	if err := flags.Parse(args); err != nil {
		return question, false
	}

	if flags.NArg() != 1 {
		flags.Usage()
		return question, false
	}

	question.Text = flags.Arg(0)

	return question, true
}

// send asks question, from the project in the current folder, through the agent door.
func send(ctx context.Context, question agentdoor.Question) (agentdoor.Answer, error) {
	project, err := currentProject()
	if err != nil {
		return agentdoor.Answer{}, err
	}

	question.Project = project

	path, err := agentdoor.SocketPath()
	if err != nil {
		return agentdoor.Answer{}, err
	}

	return agentdoor.Ask(ctx, agentdoor.NewClient(path), question)
}

// currentProject is the name of the project in the current folder.
func currentProject() (string, error) {
	folder, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("find the current folder: %w", err)
	}

	return filepath.Base(folder), nil
}

// answerLines is the answer as nudge ask prints it: an option line and a text line, each only
// when it has a value.
func answerLines(got agentdoor.Answer) string {
	var lines strings.Builder

	if got.Option != "" {
		lines.WriteString("option: " + got.Option + "\n")
	}

	if got.Text != "" {
		lines.WriteString("text: " + got.Text + "\n")
	}

	return lines.String()
}
