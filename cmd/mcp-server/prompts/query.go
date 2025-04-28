package prompts

import (
	"encoding/json"
	"fmt"
)

type QueryPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
}

func QueryPrompt(promptName string, arguments map[string]string) string {
	query := QueryPromptParams{
		Name:      promptName,
		Arguments: arguments,
	}

	s, err := json.Marshal(query)
	if err != nil {
		s = []byte(fmt.Sprintf("Failed to marshal query prompt: %v", err))
	}

	return string(s)
}
