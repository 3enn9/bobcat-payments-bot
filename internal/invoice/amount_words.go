package invoice

import (
	"fmt"
	"math"
	"strings"
)

var (
	onesMale = []string{
		"", "один", "два", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять",
		"десять", "одиннадцать", "двенадцать", "тринадцать", "четырнадцать", "пятнадцать",
		"шестнадцать", "семнадцать", "восемнадцать", "девятнадцать",
	}
	onesFemale = []string{
		"", "одна", "две", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять",
		"десять", "одиннадцать", "двенадцать", "тринадцать", "четырнадцать", "пятнадцать",
		"шестнадцать", "семнадцать", "восемнадцать", "девятнадцать",
	}
	tens = []string{
		"", "", "двадцать", "тридцать", "сорок", "пятьдесят",
		"шестьдесят", "семьдесят", "восемьдесят", "девяносто",
	}
	hundreds = []string{
		"", "сто", "двести", "триста", "четыреста", "пятьсот",
		"шестьсот", "семьсот", "восемьсот", "девятьсот",
	}
)

func AmountInWords(amount float64) string {
	if amount < 0 {
		amount = -amount
	}
	rubles := int64(math.Floor(amount + 1e-9))
	kopecks := int(math.Round((amount - float64(rubles)) * 100))
	if kopecks == 100 {
		rubles++
		kopecks = 0
	}

	words := numberToWords(rubles, true)
	result := fmt.Sprintf("%s %s %02d %s",
		capitalize(words),
		rubleForm(rubles),
		kopecks,
		kopeckForm(int64(kopecks)),
	)
	return result
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func numberToWords(n int64, male bool) string {
	if n == 0 {
		return "ноль"
	}

	parts := make([]string, 0)
	billions := n / 1_000_000_000
	millions := (n / 1_000_000) % 1000
	thousands := (n / 1000) % 1000
	rest := n % 1000

	if billions > 0 {
		parts = append(parts, triadToWords(billions, true), plural(billions, "миллиард", "миллиарда", "миллиардов"))
	}
	if millions > 0 {
		parts = append(parts, triadToWords(millions, true), plural(millions, "миллион", "миллиона", "миллионов"))
	}
	if thousands > 0 {
		parts = append(parts, triadToWords(thousands, false), plural(thousands, "тысяча", "тысячи", "тысяч"))
	}
	if rest > 0 || len(parts) == 0 {
		parts = append(parts, triadToWords(rest, male))
	}

	return strings.Join(filterEmpty(parts), " ")
}

func triadToWords(n int64, male bool) string {
	if n == 0 {
		return ""
	}
	ones := onesMale
	if !male {
		ones = onesFemale
	}

	h := n / 100
	t := n % 100
	parts := make([]string, 0, 3)
	if h > 0 {
		parts = append(parts, hundreds[h])
	}
	if t < 20 {
		if t > 0 {
			parts = append(parts, ones[t])
		}
	} else {
		parts = append(parts, tens[t/10])
		if t%10 > 0 {
			parts = append(parts, ones[t%10])
		}
	}
	return strings.Join(parts, " ")
}

func plural(n int64, one, few, many string) string {
	n = n % 100
	if n >= 11 && n <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}

func rubleForm(n int64) string {
	return plural(n, "рубль", "рубля", "рублей")
}

func kopeckForm(n int64) string {
	return plural(n, "копейка", "копейки", "копеек")
}

func filterEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}
