package invoice

import (
	"regexp"
	"strings"
)

var orgFormPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*открытое\s+акционерное\s+общество\s*`),
	regexp.MustCompile(`(?i)^\s*акционерное\s+общество\s*`),
	regexp.MustCompile(`(?i)^\s*общество\s+с\s+ограниченной\s+ответственностью\s*`),
	regexp.MustCompile(`(?i)^\s*индивидуальный\s+предприниматель\s*`),
	regexp.MustCompile(`(?i)^\s*оао\s*`),
	regexp.MustCompile(`(?i)^\s*ооо\s*`),
	regexp.MustCompile(`(?i)^\s*ао\s*`),
	regexp.MustCompile(`(?i)^\s*ип\s*`),
}

var municipalNameRE = regexp.MustCompile(`(?i)^\s*муниципальн`)
var quotedChunkRE = regexp.MustCompile(`["«„]([^"»“]+)["»“]`)

// ShortenBuyerName убирает типовые формы собственности для компактных таблиц.
func ShortenBuyerName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return s
	}

	// Муниципальное … "Короткое имя" → берём содержимое кавычек.
	if municipalNameRE.MatchString(s) {
		if q := extractQuotedName(s); q != "" {
			return q
		}
	}

	for {
		changed := false
		for _, re := range orgFormPatterns {
			next := strings.TrimSpace(re.ReplaceAllString(s, ""))
			if next != s {
				s = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	s = strings.Trim(s, " \t\"'«»„“")
	if s == "" {
		return strings.TrimSpace(name)
	}
	return s
}

func extractQuotedName(s string) string {
	m := quotedChunkRE.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
