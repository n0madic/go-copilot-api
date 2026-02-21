package api

import (
	"crypto/rand"
	"fmt"

	"github.com/n0madic/go-copilot-api/internal/config"
	"github.com/n0madic/go-copilot-api/internal/state"
)

func StandardHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
}

func CopilotBaseURL(st *state.State) string {
	if st.AccountType() == "individual" {
		return "https://api.githubcopilot.com"
	}
	return fmt.Sprintf("https://api.%s.githubcopilot.com", st.AccountType())
}

func CopilotHeaders(st *state.State, vision bool) map[string]string {
	h := map[string]string{
		"Authorization":                       "Bearer " + st.CopilotToken(),
		"Content-Type":                        "application/json",
		"Copilot-Integration-Id":              "vscode-chat",
		"Editor-Version":                      "vscode/" + st.VSCodeVersion(),
		"Editor-Plugin-Version":               "copilot-chat/" + config.CopilotVersion,
		"User-Agent":                          "GitHubCopilotChat/" + config.CopilotVersion,
		"Openai-Intent":                       "conversation-panel",
		"X-Github-Api-Version":                config.CopilotAPIVersion,
		"X-Request-Id":                        randomUUID(),
		"X-Vscode-User-Agent-Library-Version": "electron-fetch",
	}
	if vision {
		h["Copilot-Vision-Request"] = "true"
	}
	return h
}

func GitHubHeaders(st *state.State) map[string]string {
	h := StandardHeaders()
	h["Authorization"] = "token " + st.GitHubToken()
	h["Editor-Version"] = "vscode/" + st.VSCodeVersion()
	h["Editor-Plugin-Version"] = "copilot-chat/" + config.CopilotVersion
	h["User-Agent"] = "GitHubCopilotChat/" + config.CopilotVersion
	h["X-Github-Api-Version"] = config.CopilotAPIVersion
	h["X-Vscode-User-Agent-Library-Version"] = "electron-fetch"
	return h
}

func randomUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	)
}
