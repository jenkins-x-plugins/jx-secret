package survey

import (
	"io"
	"os"
	"strings"

	"github.com/jenkins-x/jx-extsecret/pkg/input"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type client struct {
	in  terminal.FileReader
	out terminal.FileWriter
	err io.Writer
}

// NewInput creates a new input using std in/out/err
func NewInput() input.Interface {
	return NewInputFrom(os.Stdin, os.Stdout, os.Stderr)
}

// NewInputFrom creates a new input from the given in/out/err
func NewInputFrom(in terminal.FileReader, out terminal.FileWriter, err io.Writer) input.Interface {
	return &client{
		in:  in,
		out: out,
		err: err,
	}
}

// PickPassword gets a password (via hidden input) from a user's free-form input
func (c *client) PickPassword(message string, help string) (string, error) {
	answer := ""
	prompt := &survey.Password{
		Message: message,
		Help:    help,
	}
	validator := survey.Required
	surveyOpts := survey.WithStdio(c.in, c.out, c.err)
	err := survey.AskOne(prompt, &answer, validator, surveyOpts)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}
