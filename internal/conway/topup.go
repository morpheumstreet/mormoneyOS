// Package conway provides Conway API client and credit topup via x402.
//
// Bootstrap topup: buy minimum $5 credits on startup when balance is low.
// TopupCredits: execute x402 payment via GET /pay/{amountUsd}/{address}.
package conway

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TopupTiers are valid topup amounts in USD (TS-aligned).
var TopupTiers = []int{5, 25, 100, 500, 1000, 2500}

// TopupResult is the result of a credit topup attempt.
type TopupResult struct {
	Success          bool
	AmountUSD        float64
	CreditsCentsAdded int64
	Error            string
}

// BootstrapTopupParams configures the bootstrap topup.
type BootstrapTopupParams struct {
	APIURL               string
	Account              *ecdsa.PrivateKey
	Address              string
	CreditsCents         int64
	CreditThresholdCents int64
	DefaultChain         string
	HTTPClient           *http.Client
}

// BootstrapTopup buys the minimum tier ($5) when credits are below threshold.
// Returns nil if skipped (credits sufficient, no USDC, or no wallet).
// TS-aligned: src/conway/topup.ts bootstrapTopup().
func BootstrapTopup(ctx context.Context, params BootstrapTopupParams) (*TopupResult, error) {
	threshold := params.CreditThresholdCents
	if threshold == 0 {
		threshold = 500 // $5 default
	}
	if params.CreditsCents >= threshold {
		return nil, nil
	}
	if params.Account == nil || params.Address == "" {
		slog.Debug("bootstrap topup skipped: no wallet for x402 signing")
		return nil, nil
	}
	if params.APIURL == "" {
		slog.Debug("bootstrap topup skipped: no Conway API URL")
		return nil, nil
	}

	// USDC balance check: skip if below minimum tier.
	usdcBalance, err := GetUSDCBalance(ctx, params.Address, params.DefaultChain, nil)
	if err != nil {
		slog.Warn("bootstrap topup: USDC balance check failed, attempting anyway", "err", err)
		// Continue - topup will fail if insufficient USDC
	} else {
		minTier := float64(TopupTiers[0])
		if usdcBalance < minTier {
			slog.Info("bootstrap topup skipped: USDC balance below minimum tier",
				"usdc", usdcBalance, "min_tier", minTier)
			return nil, nil
		}
	}

	minTier := TopupTiers[0]
	slog.Info("bootstrap topup: credits below threshold, buying minimum tier",
		"credits_cents", params.CreditsCents, "min_tier_usd", minTier)

	return TopupCredits(ctx, TopupCreditsParams{
		APIURL:     params.APIURL,
		Account:    params.Account,
		Address:    params.Address,
		AmountUSD:  minTier,
		HTTPClient: params.HTTPClient,
	})
}

// TopupCreditsParams configures a credit topup.
type TopupCreditsParams struct {
	APIURL     string
	Account    *ecdsa.PrivateKey
	Address    string
	AmountUSD  int
	HTTPClient *http.Client
}

// TopupCredits executes a credit topup via x402 (GET /pay/{amountUsd}/{address}).
// TS-aligned: src/conway/topup.ts topupCredits().
func TopupCredits(ctx context.Context, params TopupCreditsParams) (*TopupResult, error) {
	url := strings.TrimSuffix(params.APIURL, "/") + "/pay/" + strconv.Itoa(params.AmountUSD) + "/" + params.Address
	client := params.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	result, err := x402Fetch(ctx, url, params.Account, params.Address, "GET", nil, client)
	if err != nil {
		return &TopupResult{
			Success:   false,
			AmountUSD: float64(params.AmountUSD),
			Error:     err.Error(),
		}, nil
	}
	if !result.Success {
		return &TopupResult{
			Success:   false,
			AmountUSD: float64(params.AmountUSD),
			Error:     result.Error,
		}, nil
	}
	creditsCents := result.CreditsCentsAdded
	if creditsCents == 0 {
		creditsCents = int64(params.AmountUSD * 100)
	}
	return &TopupResult{
		Success:           true,
		AmountUSD:         float64(params.AmountUSD),
		CreditsCentsAdded: creditsCents,
	}, nil
}

func ethCall(ctx context.Context, rpcURL string, payload interface{}, out *struct {
	Result string `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rpc %s: %d", rpcURL, resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return err
	}
	if out.Error != nil {
		return fmt.Errorf("rpc error: %s", out.Error.Message)
	}
	return nil
}
