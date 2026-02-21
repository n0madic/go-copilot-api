package config

const (
	DefaultPort         = 4141
	DefaultAccountType  = "individual"
	FallbackVSCode      = "1.104.3"
	CopilotVersion      = "0.26.7"
	GitHubBaseURL       = "https://github.com"
	GitHubAPIBaseURL    = "https://api.github.com"
	GitHubClientID      = "Iv1.b507a08c87ecfe98"
	GitHubAppScopes     = "read:user"
	CopilotAPIVersion   = "2025-04-01"
	VSCodeVersionSource = "https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=visual-studio-code-bin"
)

type StartOptions struct {
	Port          int
	Verbose       bool
	AccountType   string
	Manual        bool
	RateLimit     *int
	RateLimitWait bool
	GitHubToken   string
	ClaudeCode    bool
	ShowToken     bool
	ProxyEnv      bool
}

type AuthOptions struct {
	Verbose   bool
	ShowToken bool
}

type DebugOptions struct {
	JSON bool
}
