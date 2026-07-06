package leftoff

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type CompatibilityResult struct {
	Output string
}

type agentTarget struct {
	Name string
	Path string
	Note string
}

func CompatibilityReport(storePath string) CompatibilityResult {
	defaultStore, err := DefaultStoreRoot()
	if err != nil {
		defaultStore = "unknown"
	}
	if strings.TrimSpace(storePath) == "" {
		storePath = defaultStore
	}

	var b strings.Builder
	fmt.Fprintf(&b, "COMPATIBILITY\n")
	fmt.Fprintf(&b, "- OS: %s\n", runtime.GOOS)
	fmt.Fprintf(&b, "- Architecture: %s\n", runtime.GOARCH)
	fmt.Fprintf(&b, "- Default store: %s\n", defaultStore)
	fmt.Fprintf(&b, "- Selected store: %s\n", storePath)
	fmt.Fprintf(&b, "- Agent skill targets:\n")
	for _, target := range agentSkillTargets(homeOrTilde()) {
		if target.Note == "" {
			fmt.Fprintf(&b, "  - %s: %s\n", target.Name, target.Path)
		} else {
			fmt.Fprintf(&b, "  - %s: %s (%s)\n", target.Name, target.Path, target.Note)
		}
	}
	fmt.Fprintf(&b, "- Core workflow: no network required.\n")
	fmt.Fprintf(&b, "- Optional GitHub metadata: requires explicit `github --refresh` and the GitHub CLI.\n")
	fmt.Fprintf(&b, "- Output convention: evidence, inference, and uncertainty are labeled in user-facing reports.\n")
	return CompatibilityResult{Output: b.String()}
}

func agentSkillTargets(home string) []agentTarget {
	codexPath := filepath.Join(home, ".codex", "skills", "leftoff")
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		codexPath = filepath.Join(codexHome, "skills", "leftoff")
	}

	return []agentTarget{
		{Name: "Generic AI agent", Path: filepath.Join(home, ".leftoff", "skills", "leftoff")},
		{Name: "Claude Code", Path: filepath.Join(home, ".claude", "skills", "leftoff"), Note: "aliases: claude-code, claude"},
		{Name: "Codex", Path: codexPath, Note: "uses CODEX_HOME when set"},
		{Name: "Cursor", Path: filepath.Join(home, ".cursor", "skills", "leftoff")},
		{Name: "Pi", Path: filepath.Join(home, ".pi", "skills", "leftoff")},
		{Name: "GitHub Copilot CLI", Path: filepath.Join(home, ".github-copilot-cli", "skills", "leftoff"), Note: "aliases: copilot-cli, copilot"},
		{Name: "OpenCode", Path: filepath.Join(home, ".opencode", "skills", "leftoff")},
		{Name: "Gemini CLI / Antigravity", Path: filepath.Join(home, ".gemini", "skills", "leftoff"), Note: "aliases: gemini, google-gemini, antigravity"},
		{Name: "Factory AI Droid", Path: filepath.Join(home, ".factory-ai-droid", "skills", "leftoff"), Note: "aliases: factory-droid, droid"},
		{Name: "OpenClaw", Path: filepath.Join(home, ".openclaw", "skills", "leftoff")},
		{Name: "Hermes Agent", Path: filepath.Join(home, ".hermes", "skills", "leftoff"), Note: "alias: hermes"},
		{Name: "AstrBot", Path: filepath.Join(home, ".astrbot", "skills", "leftoff")},
		{Name: "NanoClaw", Path: filepath.Join(home, ".nanoclaw", "skills", "leftoff")},
		{Name: "Shelley", Path: filepath.Join(home, ".shelley", "skills", "leftoff")},
		{Name: "Auggie / Augment", Path: filepath.Join(home, ".augment", "skills", "leftoff"), Note: "aliases: auggie, augment"},
		{Name: "Cline / Roo Code", Path: filepath.Join(home, ".cline", "skills", "leftoff"), Note: "aliases: cline, roo, roo-code"},
		{Name: "CodeBuddy", Path: filepath.Join(home, ".codebuddy", "skills", "leftoff")},
		{Name: "Continue", Path: filepath.Join(home, ".continue", "skills", "leftoff")},
		{Name: "Crush", Path: filepath.Join(home, ".crush", "skills", "leftoff")},
		{Name: "Deep Agents", Path: filepath.Join(home, ".deep-agents", "skills", "leftoff"), Note: "alias: deepagents"},
		{Name: "Firebender", Path: filepath.Join(home, ".firebender", "skills", "leftoff")},
		{Name: "ForgeCode", Path: filepath.Join(home, ".forgecode", "skills", "leftoff")},
		{Name: "Goose", Path: filepath.Join(home, ".goose", "skills", "leftoff")},
		{Name: "Junie", Path: filepath.Join(home, ".junie", "skills", "leftoff")},
		{Name: "Kilo Code", Path: filepath.Join(home, ".kilo-code", "skills", "leftoff"), Note: "alias: kilocode"},
		{Name: "Kimi Code CLI", Path: filepath.Join(home, ".kimi-code-cli", "skills", "leftoff"), Note: "alias: kimi"},
		{Name: "Kiro CLI", Path: filepath.Join(home, ".kiro", "skills", "leftoff"), Note: "alias: kiro"},
		{Name: "Lingma", Path: filepath.Join(home, ".lingma", "skills", "leftoff")},
		{Name: "Mistral Vibe", Path: filepath.Join(home, ".mistral-vibe", "skills", "leftoff")},
		{Name: "Mux", Path: filepath.Join(home, ".mux", "skills", "leftoff")},
		{Name: "OpenHands", Path: filepath.Join(home, ".openhands", "skills", "leftoff")},
		{Name: "Qoder", Path: filepath.Join(home, ".qoder", "skills", "leftoff")},
		{Name: "Qwen Code", Path: filepath.Join(home, ".qwen-code", "skills", "leftoff"), Note: "alias: qwen"},
		{Name: "Rovo Dev", Path: filepath.Join(home, ".rovo", "skills", "leftoff"), Note: "alias: rovo"},
		{Name: "Tabnine CLI", Path: filepath.Join(home, ".tabnine", "skills", "leftoff"), Note: "alias: tabnine"},
		{Name: "Trae / Trae CN", Path: filepath.Join(home, ".trae", "skills", "leftoff"), Note: "aliases: trae, trae-cn"},
		{Name: "Warp", Path: filepath.Join(home, ".warp", "skills", "leftoff")},
		{Name: "Windsurf", Path: filepath.Join(home, ".windsurf", "skills", "leftoff")},
		{Name: "Zed", Path: filepath.Join(home, ".zed", "skills", "leftoff")},
	}
}

func homeOrTilde() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "~"
	}
	return home
}
