package mcp

import "github.com/axonops/axonops-schema-registry/internal/analysis"

// FuzzyMatch is an alias for analysis.FuzzyMatch.
type FuzzyMatch = analysis.FuzzyMatch

// LevenshteinDistance delegates to analysis.LevenshteinDistance.
func LevenshteinDistance(a, b string) int {
	return analysis.LevenshteinDistance(a, b)
}

// FuzzyScore delegates to analysis.FuzzyScore.
func FuzzyScore(query, target string) float64 {
	return analysis.FuzzyScore(query, target)
}

// NamingVariants delegates to analysis.NamingVariants.
func NamingVariants(name string) []string {
	return analysis.NamingVariants(name)
}

// MatchFuzzy delegates to analysis.MatchFuzzy.
func MatchFuzzy(query string, candidates []string, threshold float64) []FuzzyMatch {
	return analysis.MatchFuzzy(query, candidates, threshold)
}
