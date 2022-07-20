package genericcli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/metal-stack/metal-lib/pkg/pointer"
)

type PromptConfig struct {
	Message         string
	No              string
	AcceptedAnswers []string
	ShowAnswers     bool
	In              io.Reader
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
	if c.Message == "" || len(c.AcceptedAnswers) == 0 {
		panic("internal error: prompt not properly configured")
	}
	if c.In == nil {
		c.In = os.Stdin
	}

	if c.ShowAnswers {
		if c.No == "" {
			fmt.Printf("%s [%s] ", c.Message, pointer.FirstOrZero(c.AcceptedAnswers))
		} else {
			fmt.Printf("%s [%s/%s] ", c.Message, pointer.FirstOrZero(c.AcceptedAnswers), c.No)
		}
	} else {
		fmt.Printf("%s ", c.Message)
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return err
	}

	text := scanner.Text()
	for _, accepted := range c.AcceptedAnswers {
		if strings.EqualFold(text, accepted) {
			return nil
		}
	}

	return fmt.Errorf("aborting due to given answer (%q)", text)
}
