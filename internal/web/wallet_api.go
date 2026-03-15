package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
)

// handleAPIWalletGet returns wallet info (no mnemonic).
// GET /api/wallet
func (s *Server) handleAPIWalletGet(w http.ResponseWriter, r *http.Request) {
	if !identity.WalletExists() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"exists": false,
			"error":  "no wallet: run 'moneyclaw init' first",
		})
		return
	}

	idx := identity.CurrentIndex()
	addr := identity.GetWalletAddress()

	// Load wallet metadata (wordCount) without exposing mnemonic
	meta, err := identity.GetWalletMetadata()
	if err != nil {
		s.Log.Warn("wallet get: load failed", "err", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"exists": true,
			"error":  err.Error(),
		})
		return
	}

	wordCount := 0
	if meta != nil {
		wordCount = meta.WordCount
	}

	defaultChain := identity.DefaultChainBase
	if s.Cfg != nil && s.Cfg.DefaultChain != "" {
		defaultChain = s.Cfg.DefaultChain
	}
	cfg, _ := config.Load()
	if cfg != nil && cfg.DefaultChain != "" {
		defaultChain = cfg.DefaultChain
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"exists":        true,
		"currentIndex":  idx,
		"address":       addr,
		"defaultChain":  defaultChain,
		"wordCount":     wordCount,
	})
}

// handleAPIWalletAddressGet derives address for chain (optional index).
// GET /api/wallet/address?chain=<CAIP-2>&index=<N>
// chain required; index optional (0 or omit = use wallet's current index).
func (s *Server) handleAPIWalletAddressGet(w http.ResponseWriter, r *http.Request) {
	chain := strings.TrimSpace(r.URL.Query().Get("chain"))
	if chain == "" {
		http.Error(w, "chain query param required (CAIP-2, e.g. eip155:8453)", http.StatusBadRequest)
		return
	}

	var index uint32
	if idxStr := r.URL.Query().Get("index"); idxStr != "" {
		n, err := strconv.ParseUint(idxStr, 10, 32)
		if err != nil {
			http.Error(w, "index must be a non-negative integer", http.StatusBadRequest)
			return
		}
		index = uint32(n)
	}

	var addr string
	var err error
	if index == 0 {
		addr, err = identity.DeriveAddress(chain)
		index = identity.CurrentIndex() // actual index used
	} else {
		addr, err = identity.DeriveAddressAtExplicitIndex(chain, index)
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"chain":   chain,
		"index":   index,
		"address": addr,
	})
}

// handleAPIWalletRotate rotates HD account index (preview or confirm).
// POST /api/wallet/rotate
// Body: { "toIndex": N, "preview": bool?, "confirm": bool? }
// preview: show addresses without writing. confirm: write new index.
func (s *Server) handleAPIWalletRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.validateJWT(r); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		ToIndex uint32 `json:"toIndex"`
		Preview *bool  `json:"preview"`
		Confirm *bool  `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	preview := body.Preview != nil && *body.Preview
	confirm := body.Confirm != nil && *body.Confirm

	if !identity.WalletExists() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "no wallet: run 'moneyclaw init' first"})
		return
	}

	currentIdx := identity.CurrentIndex()
	if body.ToIndex == currentIdx {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "index already " + strconv.FormatUint(uint64(currentIdx), 10)})
		return
	}

	// Collect chains: defaultChain + chainProviders
	cfg, _ := config.Load()
	seen := make(map[string]bool)
	var chains []string
	defaultChain := identity.DefaultChainBase
	if s.Cfg != nil && s.Cfg.DefaultChain != "" {
		defaultChain = s.Cfg.DefaultChain
	}
	if cfg != nil && cfg.DefaultChain != "" {
		defaultChain = cfg.DefaultChain
	}
	chains = append(chains, defaultChain)
	seen[defaultChain] = true
	if cfg != nil {
		for c := range cfg.ChainProviders {
			if c != "" && !seen[c] {
				chains = append(chains, c)
				seen[c] = true
			}
		}
	}

	currentAddrs := make(map[string]string)
	newAddrs := make(map[string]string)
	for _, chain := range chains {
		a, err := identity.DeriveAddressAtExplicitIndex(chain, currentIdx)
		if err != nil {
			currentAddrs[chain] = "error: " + err.Error()
		} else {
			currentAddrs[chain] = a
		}
		b, err := identity.DeriveAddressAtExplicitIndex(chain, body.ToIndex)
		if err != nil {
			newAddrs[chain] = "error: " + err.Error()
		} else {
			newAddrs[chain] = b
		}
	}

	resp := map[string]any{
		"currentIndex":   currentIdx,
		"targetIndex":    body.ToIndex,
		"currentAddresses": currentAddrs,
		"newAddresses":    newAddrs,
		"preview":         preview,
		"confirmed":       false,
	}

	if preview {
		resp["message"] = "Preview only — no changes written. Send confirm: true to apply."
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if !confirm {
		resp["message"] = "Send confirm: true to write new index to wallet.json"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if err := identity.RotateIndex(body.ToIndex, false); err != nil {
		s.Log.Error("wallet rotate failed", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	resp["confirmed"] = true
	resp["message"] = "Rotated to index " + strconv.FormatUint(uint64(body.ToIndex), 10) + ". Migrate balances manually."
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleAPIWalletClearCache clears derived keys cache.
// POST /api/wallet/clear-cache
func (s *Server) handleAPIWalletClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.validateJWT(r); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	identity.ClearDerivedKeys()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": "derived keys cache cleared"})
}
