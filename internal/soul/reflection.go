// Package soul provides soul reflection pipeline (TS-aligned).
// Gathers evidence from tool_calls, inbox_messages, transactions;
// computes genesis alignment; returns suggestions for mutable sections.
package soul

import (
	"regexp"
	"strings"
	"unicode"
)

// ReflectionStore is the minimal interface for soul reflection.
// Implemented by *state.Database when it has GetRecentToolNames, GetRecentInboxAddresses, GetRecentTransactionDescriptions.
type ReflectionStore interface {
	GetKV(key string) (string, bool, error)
	SetKV(key, value string) error
	GetRecentToolNames() ([]string, error)
	GetRecentInboxAddresses() ([]string, error)
	GetRecentTransactionDescriptions() ([]string, error)
}

// SuggestedUpdate is a suggested change to a mutable soul section.
type SuggestedUpdate struct {
	Section         string
	Reason         string
	SuggestedContent string
}

// Reflection holds the result of a soul reflection run.
type Reflection struct {
	CurrentAlignment float64
	SuggestedUpdates []SuggestedUpdate
	AutoUpdated      []string
}

// ReflectOnSoul runs the reflection pipeline (TS reflectOnSoul-aligned).
// Uses soul_content and genesis_prompt from KV; gathers evidence from DB;
// computes alignment; returns suggestions. Does not auto-update soul (Go uses KV, not structured SoulModel).
func ReflectOnSoul(store ReflectionStore) (Reflection, error) {
	out := Reflection{CurrentAlignment: 0, SuggestedUpdates: nil, AutoUpdated: nil}

	soulContent, ok, _ := store.GetKV("soul_content")
	if !ok || soulContent == "" {
		return out, nil
	}

	genesisPrompt, _, _ := store.GetKV("genesis_prompt")
	corePurpose := extractCorePurpose(soulContent)
	if corePurpose == "" {
		corePurpose = soulContent
	}
	if len(corePurpose) > 2000 {
		corePurpose = corePurpose[:2000]
	}

	alignment := computeGenesisAlignment(corePurpose, genesisPrompt)
	out.CurrentAlignment = alignment

	evidence := gatherEvidence(store)
	capSummary := summarizeCapabilities(evidence.ToolsUsed)
	relSummary := summarizeRelationships(evidence.Interactions)
	finSummary := summarizeFinancial(evidence.FinancialActivity)

	if capSummary != "" || relSummary != "" || finSummary != "" {
		out.AutoUpdated = []string{}
		if capSummary != "" {
			out.AutoUpdated = append(out.AutoUpdated, "capabilities")
		}
		if relSummary != "" {
			out.AutoUpdated = append(out.AutoUpdated, "relationships")
		}
		if finSummary != "" {
			out.AutoUpdated = append(out.AutoUpdated, "financialCharacter")
		}
	}

	if alignment < 0.5 && genesisPrompt != "" {
		out.SuggestedUpdates = []SuggestedUpdate{{
			Section:          "corePurpose",
			Reason:           "Genesis alignment is low. Purpose may have drifted from original genesis.",
			SuggestedContent: genesisPrompt,
		}}
	}

	return out, nil
}

func extractCorePurpose(content string) string {
	sections := parseSections(content)
	if s := sections["core purpose"]; s != "" {
		return strings.TrimSpace(s)
	}
	if s := sections["mission"]; s != "" {
		return strings.TrimSpace(s)
	}
	if len(content) > 500 {
		return strings.TrimSpace(content[:500])
	}
	return strings.TrimSpace(content)
}

var sectionRe = regexp.MustCompile(`(?m)^##\s+(.+)$`)

func parseSections(body string) map[string]string {
	out := make(map[string]string)
	matches := sectionRe.FindAllStringSubmatchIndex(body, -1)
	for i := 0; i < len(matches); i++ {
		name := strings.ToLower(strings.TrimSpace(body[matches[i][2]:matches[i][3]]))
		contentStart := matches[i][1]
		contentEnd := len(body)
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		}
		content := strings.TrimSpace(body[contentStart:contentEnd])
		out[name] = content
	}
	return out
}

func computeGenesisAlignment(currentPurpose, genesisPrompt string) float64 {
	currentPurpose = strings.TrimSpace(currentPurpose)
	genesisPrompt = strings.TrimSpace(genesisPrompt)
	if currentPurpose == "" || genesisPrompt == "" {
		return 0
	}
	curr := tokenize(currentPurpose)
	gen := tokenize(genesisPrompt)
	if len(curr) == 0 || len(gen) == 0 {
		return 0
	}
	intersect := 0
	for t := range curr {
		if gen[t] {
			intersect++
		}
	}
	union := len(curr) + len(gen) - intersect
	if union == 0 {
		return 0
	}
	jaccard := float64(intersect) / float64(union)
	recall := float64(intersect) / float64(len(gen))
	score := (jaccard + recall) / 2
	if score > 1 {
		score = 1
	}
	if score < 0 {
		score = 0
	}
	return score
}

func tokenize(s string) map[string]bool {
	f := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }
	parts := strings.FieldsFunc(strings.ToLower(s), f)
	out := make(map[string]bool)
	for _, p := range parts {
		if p != "" {
			out[p] = true
		}
	}
	return out
}

type evidence struct {
	ToolsUsed       []string
	Interactions    []string
	FinancialActivity []string
}

func gatherEvidence(store ReflectionStore) evidence {
	var e evidence
	e.ToolsUsed, _ = store.GetRecentToolNames()
	e.Interactions, _ = store.GetRecentInboxAddresses()
	e.FinancialActivity, _ = store.GetRecentTransactionDescriptions()
	return e
}

func summarizeCapabilities(tools []string) string {
	if len(tools) == 0 {
		return ""
	}
	seen := make(map[string]bool)
	var uniq []string
	for _, t := range tools {
		if !seen[t] {
			seen[t] = true
			uniq = append(uniq, t)
		}
	}
	return "Tools used: " + strings.Join(uniq, ", ")
}

func summarizeRelationships(interactions []string) string {
	if len(interactions) == 0 {
		return ""
	}
	n := 10
	if len(interactions) < n {
		n = len(interactions)
	}
	return "Known contacts: " + strings.Join(interactions[:n], ", ")
}

func summarizeFinancial(activity []string) string {
	if len(activity) == 0 {
		return ""
	}
	n := 5
	if len(activity) < n {
		n = len(activity)
	}
	return "Recent activity: " + strings.Join(activity[:n], "; ")
}
