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
	regexp.MustCompile(`(?i)^\s*индувидуальный\s+предприниматель\s*`), // опечатка бухгалтера
	regexp.MustCompile(`(?i)^\s*оао\s*`),
	regexp.MustCompile(`(?i)^\s*ооо\s*`),
	regexp.MustCompile(`(?i)^\s*ао\s*`),
	regexp.MustCompile(`(?i)^\s*ип\s*`),
}

// ShortenBuyerName уплотняет имя покупателя для таблицы:
// если есть открывающая кавычка — берём текст от неё до конца (закрывающая необязательна);
// иначе срезает форму собственности.
func ShortenBuyerName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return s
	}

	if q := extractQuotedName(s); q != "" {
		return q
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
	runes := []rune(s)
	openIdx := -1
	for i, r := range runes {
		if r == '"' || r == '«' || r == '„' {
			openIdx = i
			break
		}
	}
	if openIdx < 0 || openIdx+1 >= len(runes) {
		return ""
	}

	// От кавычки до конца строки; хвостовые кавычки/пробелы срезаем.
	body := strings.TrimSpace(string(runes[openIdx+1:]))
	body = strings.TrimRight(body, " \t\"»“")
	body = strings.TrimSpace(body)
	return body
}
