package leftoff

import (
	"net/url"
	"regexp"
	"strings"
)

type SecretFinding struct {
	Name     string
	Severity string
}

type secretPattern struct {
	name     string
	severity string
	pattern  *regexp.Regexp
}

var secretPatterns = []secretPattern{
	{name: "private key", severity: "high", pattern: regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)},
	{name: "certificate", severity: "high", pattern: regexp.MustCompile(`-----BEGIN CERTIFICATE-----`)},
	{name: "AWS access key", severity: "high", pattern: regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{name: "GitHub token", severity: "high", pattern: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{30,}`)},
	{name: "sk-prefixed API key", severity: "high", pattern: regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`)},
	{name: "secret assignment", severity: "high", pattern: regexp.MustCompile(`(?i)\b(api[_-]?key|secret|token|password|passwd)\s*[:=]\s*["']?[^"'\s]+`)},
	{name: "authorization header", severity: "high", pattern: regexp.MustCompile(`(?i)\bauthorization\s*:\s*bearer\s+[A-Za-z0-9._~+/=-]+`)},
}

var promptLikeMetadataPattern = regexp.MustCompile(`(?i)\b(ignore (?:all )?(?:previous|prior|above) instructions|system prompt|developer message|act as|you are now)\b`)

const (
	maxMetadataBranchName = 160
	maxMetadataPath       = 512
	maxMetadataTitle      = 160
	maxMetadataCommand    = 512
)

func FindSecrets(text string) []SecretFinding {
	var findings []SecretFinding
	for _, candidate := range secretPatterns {
		if candidate.pattern.MatchString(text) {
			findings = append(findings, SecretFinding{Name: candidate.name, Severity: candidate.severity})
		}
	}
	return findings
}

func HasLikelySecret(text string) bool {
	return len(FindSecrets(text)) > 0
}

func sanitizeExternalMetadata(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	sanitized := text
	for _, candidate := range secretPatterns {
		sanitized = candidate.pattern.ReplaceAllString(sanitized, "[redacted "+candidate.name+"]")
	}
	sanitized = promptLikeMetadataPattern.ReplaceAllString(sanitized, "[redacted prompt-like metadata]")
	if strings.TrimSpace(sanitized) == "" {
		return "[redacted external metadata]"
	}
	return sanitized
}

func sanitizeMetadataBranch(text string) string {
	return sanitizeMetadataField(text, maxMetadataBranchName)
}

func sanitizeMetadataPath(text string) string {
	return sanitizeMetadataField(text, maxMetadataPath)
}

func sanitizeMetadataTitle(text string) string {
	return sanitizeMetadataField(text, maxMetadataTitle)
}

func sanitizeMetadataCommand(text string) string {
	return sanitizeMetadataField(text, maxMetadataCommand)
}

func sanitizeMetadataField(text string, limit int) string {
	sanitized := sanitizeExternalMetadata(text)
	if sanitized == "" {
		return ""
	}
	return cleanSummary(sanitized, limit)
}

func RedactRemoteURL(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}

	if parsed, err := url.Parse(remote); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.User = nil
		return sanitizeExternalMetadata(parsed.String())
	}

	if HasLikelySecret(remote) {
		return "[redacted remote URL]"
	}

	if at := strings.LastIndex(remote, "@"); at > 0 && strings.Contains(remote[:at], ":") {
		return remote[at+1:]
	}

	return sanitizeExternalMetadata(remote)
}
