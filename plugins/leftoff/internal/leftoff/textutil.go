package leftoff

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var wordPattern = regexp.MustCompile(`[a-z0-9]+`)
var isoDatePattern = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
var deadlineDatePattern = regexp.MustCompile(`(?i)\b(?:due|deadline|by)\b\s*:?\s*(\d{4}-\d{2}-\d{2})\b`)
var effortPattern = regexp.MustCompile(`(?i)\b(\d+)\s*(?:-|to)?\s*(\d+)?\s*(min|mins|minute|minutes|h|hr|hour|hours)\b`)

func tokenize(text string) []string {
	text = strings.ToLower(text)
	matches := wordPattern.FindAllString(text, -1)
	var tokens []string
	stop := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "in": true, "is": true,
		"it": true, "of": true, "on": true, "or": true, "the": true, "this": true,
		"to": true, "with": true,
	}
	for _, match := range matches {
		if len(match) < 2 || stop[match] {
			continue
		}
		tokens = append(tokens, match)
	}
	return tokens
}

func tokenSet(text string) map[string]bool {
	set := map[string]bool{}
	for _, token := range tokenize(text) {
		set[token] = true
	}
	return set
}

func tokenOverlapScore(query string, text string) int {
	queryTokens := tokenSet(query)
	if len(queryTokens) == 0 {
		return 1
	}
	textTokens := tokenSet(text)
	score := 0
	for token := range queryTokens {
		if textTokens[token] {
			score++
		}
	}
	return score
}

func jaccardSimilarity(left string, right string) float64 {
	leftSet := tokenSet(left)
	rightSet := tokenSet(right)
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0
	}
	intersection := 0
	for token := range leftSet {
		if rightSet[token] {
			intersection++
		}
	}
	union := len(leftSet)
	for token := range rightSet {
		if !leftSet[token] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func sortedTokens(text string) []string {
	tokens := tokenize(text)
	sort.Strings(tokens)
	return tokens
}

func parseEffortMinutes(value string) (int, int, bool) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\u2013", "-")
	value = strings.ReplaceAll(value, "\u2014", "-")
	if value == "" || strings.EqualFold(value, "unknown") {
		return 0, 0, false
	}
	match := effortPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, 0, false
	}
	first, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, false
	}
	second := first
	if match[2] != "" {
		if parsed, err := strconv.Atoi(match[2]); err == nil {
			second = parsed
		}
	}
	unit := strings.ToLower(match[3])
	if strings.HasPrefix(unit, "h") {
		first *= 60
		second *= 60
	}
	if second < first {
		first, second = second, first
	}
	return first, second, true
}

func extractISODate(text string) time.Time {
	match := isoDatePattern.FindString(text)
	if match == "" {
		return time.Time{}
	}
	parsed, err := time.Parse("2006-01-02", match)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func extractDeadlineDate(text string) time.Time {
	match := deadlineDatePattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return time.Time{}
	}
	parsed, err := time.Parse("2006-01-02", match[1])
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func lowerContainsAny(text string, values ...string) bool {
	text = strings.ToLower(text)
	for _, value := range values {
		if strings.Contains(text, strings.ToLower(value)) {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
