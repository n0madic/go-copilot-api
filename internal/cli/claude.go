package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/types"
)

func generateClaudeCodeCommand(st *state.State, serverURL string) error {
	models := st.Models()
	if models == nil || len(models.Data) == 0 {
		return errors.New("no models available")
	}
	selectedModel, err := promptSelect("Select a model to use with Claude Code", models.Data)
	if err != nil {
		return err
	}
	selectedSmallModel, err := promptSelect("Select a small model to use with Claude Code", models.Data)
	if err != nil {
		return err
	}
	command := generateEnvScript(envVars{
		"ANTHROPIC_BASE_URL":                       serverURL,
		"ANTHROPIC_AUTH_TOKEN":                     "dummy",
		"ANTHROPIC_MODEL":                          selectedModel,
		"ANTHROPIC_DEFAULT_SONNET_MODEL":           selectedModel,
		"ANTHROPIC_SMALL_FAST_MODEL":               selectedSmallModel,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":            selectedSmallModel,
		"DISABLE_NON_ESSENTIAL_MODEL_CALLS":        "1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
	}, "claude")

	fmt.Println(command)
	return nil
}

func promptSelect(label string, models []types.Model) (string, error) {
	fmt.Println(label)
	for idx, model := range models {
		fmt.Printf("  %d) %s\n", idx+1, model.ID)
	}
	fmt.Print("Select number: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || value < 1 || value > len(models) {
		return "", errors.New("invalid model selection")
	}
	return models[value-1].ID, nil
}
