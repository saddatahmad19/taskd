package tasklist

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

func taskFilter(term string, targets []string) []list.Rank {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		// No filter → return everything in order.
		ranks := make([]list.Rank, len(targets))
		for i := range targets {
			ranks[i] = list.Rank{Index: i}
		}
		return ranks
	}

	terms := strings.Fields(term)
	var ranks []list.Rank
	for i, target := range targets {
		if allTermsMatch(terms, target) {
			// MatchedIndexes is intentionally nil — we don't need character
			// highlighting since our delegate renders its own styled output.
			ranks = append(ranks, list.Rank{Index: i})
		}
	}
	return ranks
}

func allTermsMatch(terms []string, target string) bool {
	desc := extractDesc(target)
	for _, t := range terms {
		if strings.Contains(t, ":") {
			// field:value — match the whole "field:value" token anywhere.
			if !strings.Contains(target, t) {
				return false
			}
		} else {
			// Plain text — only match within the description segment so that
			// searching "go" doesn't accidentally match "tag:django".
			if !strings.Contains(desc, t) {
				return false
			}
		}
	}
	return true
}

func extractDesc(fv string) string {
	for _, marker := range []string{" tag:", " project:", " priority:", " due:"} {
		if idx := strings.Index(fv, marker); idx != -1 {
			return fv[:idx]
		}
	}
	return fv
}
