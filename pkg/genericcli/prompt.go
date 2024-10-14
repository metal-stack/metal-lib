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
	// ShowAnswers shows the accepted answers when set to true
	ShowAnswers bool
	// AcceptedAnswers contains the accepted answers to make the prompt succeed
	AcceptedAnswers []string
	// DefaultAnswer is an optional prompt configuration that uses this answer in case the input closes without any content, it needs to be contained in the list of accepted answers or needs to be the "no" answer
	DefaultAnswer string
	// No is shown in addition to the accepted answers, can be empty
	No  string
	In  io.Reader
	Out io.Writer
}

func PromptDefaultQuestion() string {
	return "Do you want to continue?"
}

func PromptDefaultAnswers() []string {
	return []string{"y", "yes"}
}

func promptDefaultConfig() *PromptConfig {
	return &PromptConfig{
		Message:         PromptDefaultQuestion(),
		No:              "n",
		AcceptedAnswers: PromptDefaultAnswers(),
		ShowAnswers:     true,
	}
}

// Prompt the user to given compare text
func Prompt() error {
	return PromptCustom(promptDefaultConfig())
}

// PromptCustomAnswers the user to given compare text
func PromptCustom(c *PromptConfig) error {
	if c == nil {
		c = promptDefaultConfig()
	}
	if c.Message == "" {
		c.Message = PromptDefaultQuestion()
	}
	if len(c.AcceptedAnswers) == 0 {
		c.AcceptedAnswers = PromptDefaultAnswers()
		c.DefaultAnswer = pointer.FirstOrZero(c.AcceptedAnswers)
		c.No = "n"
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

	defaultAnswerIndex := slices.IndexFunc(c.AcceptedAnswers, func(answer string) bool {
		return answer == c.DefaultAnswer
	})

	if c.DefaultAnswer != "" {
		if defaultAnswerIndex < 0 && c.DefaultAnswer != c.No {
			panic("configured prompt default answer must be contained in accepted answer or no answer")
		}
	}

	if c.ShowAnswers {
		sentenceCase := func(s string) string {
			runes := []rune(s)
			runes[0] = unicode.ToUpper(runes[0])
			return string(runes)
		}

		no := c.No
		yes := pointer.FirstOrZero(c.AcceptedAnswers)

		if c.DefaultAnswer != "" {
			if c.DefaultAnswer == c.No {
				no = sentenceCase(c.No)
			} else {
				yes = sentenceCase(c.AcceptedAnswers[defaultAnswerIndex])
			}
		}

		if c.No == "" {
			fmt.Fprintf(c.Out, "%s [%s] ", c.Message, yes)
		} else {
			fmt.Fprintf(c.Out, "%s [%s/%s] ", c.Message, yes, no)
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
