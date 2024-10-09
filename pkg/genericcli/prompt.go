package genericcli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/metal-stack/metal-lib/pkg/pointer"
)

type PromptConfig struct {
	// Message is a message shown by the prompt before the input prompt
	Message string
	// Shows accepted answers when set to true
	ShowAnswers     bool
	AcceptedAnswers []string
	// DefaultAnswer is an optional prompt configuration that uses this answer in case the input closes without any content
	DefaultAnswer string
	No            string
	In            io.Reader
	Out           io.Writer
}

func PromptDefaultQuestion() string {
	return "Do you want to continue?"
}

func PromptDefaultAnswers() []string {
	return []string{"y", "yes"}
}

// Prompt the user to given compare text
func Prompt() error {
	return PromptCustom(&PromptConfig{
		Message:         PromptDefaultQuestion(),
		No:              "n",
		AcceptedAnswers: PromptDefaultAnswers(),
		ShowAnswers:     true,
	})
}

// PromptCustomAnswers the user to given compare text
// "no" can be an empty string, "yes" is the list of accepted yes answers.
func PromptCustom(c *PromptConfig) error {
	if c.Message == "" {
		c.Message = PromptDefaultQuestion()
	}
	if len(c.AcceptedAnswers) == 0 {
		c.AcceptedAnswers = PromptDefaultAnswers()
		c.DefaultAnswer = pointer.FirstOrZero(c.AcceptedAnswers)
	}
	if c.In == nil {
		c.In = os.Stdin
	}
	if c.Out == nil {
		c.Out = os.Stdout
	}

	// validate, we need to panic here because this is really a configuration error and code execution needs to stop
	for _, answer := range c.AcceptedAnswers {
		if len(answer) == 0 {
			panic("configured prompt answer must not be an empty string")
		}
	}
	if c.DefaultAnswer != "" && !slices.Contains(append(c.AcceptedAnswers, c.No), c.DefaultAnswer) {
		panic("configured prompt default answer must be contained in accepted answer or no answer")
	}

	if c.ShowAnswers {
		runes := []rune(pointer.FirstOrZero(c.AcceptedAnswers))
		runes[0] = unicode.ToUpper(runes[0])
		answer := string(runes)

		if c.No == "" {
			fmt.Fprintf(c.Out, "%s [%s] ", c.Message, answer)
		} else {
			fmt.Fprintf(c.Out, "%s [%s/%s] ", c.Message, answer, c.No)
		}
	} else {
		fmt.Fprintf(c.Out, "%s ", c.Message)
	}

	scanner := bufio.NewScanner(c.In)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return err
	}

	text := scanner.Text()

	if text == "" {
		text = c.DefaultAnswer
	}

	for _, accepted := range c.AcceptedAnswers {
		if strings.EqualFold(text, accepted) {
			return nil
		}
	}

	return fmt.Errorf("aborting due to given answer (%q)", text)
}
