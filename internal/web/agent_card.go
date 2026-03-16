// Package web serves the agent card at /.well-known/agent-card.json.
// JSON-LD document for ERC-8004 agent discovery (name, address, capabilities, etc.).
package web

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
)

const agentCardPath = "/.well-known/agent-card.json"

// agentCardSectionRe extracts ## Section Name ... content from soul markdown.
var agentCardSectionRe = regexp.MustCompile(`(?m)^##\s+(.+)$`)

func parseSoulSections(body string) map[string]string {
	out := make(map[string]string)
	matches := agentCardSectionRe.FindAllStringSubmatchIndex(body, -1)
	for i := 0; i < len(matches); i++ {
		name := strings.ToLower(strings.TrimSpace(body[matches[i][2]:matches[i][3]]))
		contentStart := matches[i][1]
		contentEnd := len(body)
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		}
		content := strings.TrimSpace(body[contentStart:contentEnd])
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		out[name] = content
	}
	return out
}

// buildAgentCard constructs the JSON-LD agent card from config, DB, and request.
func (s *Server) buildAgentCard(r *http.Request) map[string]any {
	card := map[string]any{
		"@context": []string{"https://schema.org", "https://w3id.org/agent"},
		"@type":   "SoftwareApplication",
		"applicationCategory": "AI Agent",
	}

	// Base URL from request (works behind tunnels/proxies)
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	baseURL := scheme + "://" + host
	cardURL := baseURL + agentCardPath
	card["url"] = cardURL

	// Name from config
	name := "MoneyClaw"
	if s.Cfg != nil && s.Cfg.Name != "" {
		name = s.Cfg.Name
	}
	card["name"] = name

	// Description from soul corePurpose or config
	description := ""
	if s.DB != nil {
		if soulContent, ok, _ := s.DB.GetKV("soul_content"); ok && soulContent != "" {
			sections := parseSoulSections(soulContent)
			if s := sections["core purpose"]; s != "" {
				description = s
			} else if s := sections["mission"]; s != "" {
				description = s
			} else if len(soulContent) > 300 {
				description = strings.TrimSpace(soulContent[:300]) + "..."
			} else {
				description = soulContent
			}
		}
	}
	if description == "" && s.Cfg != nil && s.Cfg.Name != "" {
		description = "Autonomous AI agent (" + s.Cfg.Name + ")"
	}
	if description == "" {
		description = "Autonomous AI agent powered by mormoneyOS"
	}
	card["description"] = description

	// Ethereum address (Base chain for ERC-8004)
	address := ""
	chain := identity.DefaultChainBase
	if s.Cfg != nil && s.Cfg.DefaultChain != "" {
		chain = s.Cfg.DefaultChain
	}
	if s.DB != nil {
		if a, ok, _ := s.DB.GetIdentity(identity.AddressKeyForChain(chain)); ok && a != "" {
			address = a
		}
		if address == "" {
			if a, ok, _ := s.DB.GetIdentity("address"); ok && a != "" {
				address = a
			}
		}
	}
	if address == "" && s.Cfg != nil && s.Cfg.WalletAddress != "" {
		address = s.Cfg.WalletAddress
	}
	if address == "" {
		if addr, err := identity.DeriveAddress(chain); err == nil && addr != "" {
			address = addr
		}
	}
	if address != "" {
		card["identifier"] = []map[string]any{
			{
				"@type":       "PropertyValue",
				"name":        "ethereum:address",
				"value":       address,
				"description": chain,
			},
		}
	}

	// Capabilities from soul sections
	if s.DB != nil {
		if soulContent, ok, _ := s.DB.GetKV("soul_content"); ok && soulContent != "" {
			sections := parseSoulSections(soulContent)
			var caps []string
			if s := sections["capabilities"]; s != "" {
				for _, line := range strings.Split(s, "\n") {
					line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
					if line != "" {
						caps = append(caps, line)
					}
				}
			}
			if len(caps) > 0 {
				card["capabilities"] = caps
			}
		}
	}

	// Creator (optional)
	if s.Cfg != nil && s.Cfg.CreatorAddress != "" {
		card["creator"] = map[string]any{
			"@type": "Person",
			"identifier": []map[string]any{
				{"@type": "PropertyValue", "name": "ethereum:address", "value": s.Cfg.CreatorAddress},
			},
		}
	}

	return card
}

// handleWellKnownAgentCard serves GET /.well-known/agent-card.json
func (s *Server) handleWellKnownAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	card := s.buildAgentCard(r)
	w.Header().Set("Content-Type", "application/ld+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 min cache
	json.NewEncoder(w).Encode(card)
}
