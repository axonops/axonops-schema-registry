package analysis

import (
	"strings"
	"unicode"
)

// LevenshteinDistance computes the edit distance between two strings.
func LevenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// FuzzyScore returns a similarity score between 0.0 and 1.0.
// 1.0 means exact match, 0.0 means completely different.
func FuzzyScore(query, target string) float64 {
	q := strings.ToLower(query)
	t := strings.ToLower(target)
	if q == t {
		return 1.0
	}
	maxLen := len(q)
	if len(t) > maxLen {
		maxLen = len(t)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := LevenshteinDistance(q, t)
	return 1.0 - float64(dist)/float64(maxLen)
}

// NamingVariants generates common casing variants of a field name.
// Given "user_name", returns ["user_name", "userName", "UserName", "user-name"].
func NamingVariants(name string) []string {
	snake := toSnakeCase(name)
	parts := strings.Split(snake, "_")
	if len(parts) == 0 {
		return []string{name}
	}

	snakeCase := strings.Join(parts, "_")
	camelParts := make([]string, len(parts))
	camelParts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			camelParts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	camelCase := strings.Join(camelParts, "")
	pascalParts := make([]string, len(parts))
	for i := 0; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			pascalParts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	pascalCase := strings.Join(pascalParts, "")
	kebabCase := strings.Join(parts, "-")

	seen := make(map[string]bool)
	var variants []string
	for _, v := range []string{snakeCase, camelCase, pascalCase, kebabCase} {
		if !seen[v] {
			seen[v] = true
			variants = append(variants, v)
		}
	}
	return variants
}

func toSnakeCase(name string) string {
	var result []rune
	for i, r := range name {
		if r == '-' || r == '.' || r == ' ' {
			result = append(result, '_')
			continue
		}
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(name[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// FuzzyMatch represents a fuzzy search match result.
type FuzzyMatch struct {
	Value string  `json:"value"`
	Score float64 `json:"score"`
}

// MatchFuzzy finds candidates whose fuzzy score against query exceeds threshold.
func MatchFuzzy(query string, candidates []string, threshold float64) []FuzzyMatch {
	var matches []FuzzyMatch
	for _, c := range candidates {
		score := FuzzyScore(query, c)
		if score >= threshold {
			matches = append(matches, FuzzyMatch{Value: c, Score: score})
		}
	}
	return matches
}
