// Package conway: x402 payment protocol for USDC credit topup.
//
// GET /pay/{amountUsd}/{address} returns 402 with X-Payment-Required.
// Client signs USDC TransferWithAuthorization (EIP-3009) and retries with X-Payment header.
// TS-aligned: src/conway/x402.ts
package conway

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// x402Result holds the result of an x402 fetch (with or without payment).
type x402Result struct {
	Success           bool
	Error             string
	CreditsCentsAdded int64
}

// x402Fetch performs GET with automatic x402 payment when server returns 402.
func x402Fetch(ctx context.Context, url string, key *ecdsa.PrivateKey, fromAddr string, method string, body []byte, client *http.Client) (*x402Result, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		// 200 or other: parse body and return
		return parseX402Response(resp)
	}

	// 402: parse payment requirements and sign
	reqData, err := parsePaymentRequired(resp)
	if err != nil {
		return &x402Result{Success: false, Error: err.Error()}, nil
	}
	if reqData == nil {
		return &x402Result{Success: false, Error: "could not parse payment requirements"}, nil
	}

	payment, err := signTransferWithAuthorization(key, fromAddr, reqData)
	if err != nil {
		return &x402Result{Success: false, Error: fmt.Sprintf("sign payment: %v", err)}, nil
	}

	paymentJSON, err := json.Marshal(payment)
	if err != nil {
		return &x402Result{Success: false, Error: err.Error()}, nil
	}
	paymentHeader := base64.StdEncoding.EncodeToString(paymentJSON)

	// Retry with X-Payment header
	req2, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Payment", paymentHeader)

	resp2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	return parseX402Response(resp2)
}

func parseX402Response(resp *http.Response) (*x402Result, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return &x402Result{
			Success: false,
			Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(data)),
		}, nil
	}
	var result x402Result
	result.Success = true
	// Try to extract credits_cents or amount_cents from JSON body
	var m map[string]interface{}
	if json.Unmarshal(data, &m) == nil {
		if v, ok := m["credits_cents"].(float64); ok {
			result.CreditsCentsAdded = int64(v)
		} else if v, ok := m["amount_cents"].(float64); ok {
			result.CreditsCentsAdded = int64(v)
		}
	}
	return &result, nil
}

type paymentRequirement struct {
	Scheme                 string `json:"scheme"`
	Network                string `json:"network"`
	MaxAmountRequired      string `json:"maxAmountRequired"`
	PayToAddress          string `json:"payToAddress"`
	PayTo                 string `json:"payTo"`
	USDCAddress           string `json:"usdcAddress"`
	Asset                 string `json:"asset"`
	RequiredDeadlineSeconds int   `json:"requiredDeadlineSeconds"`
	MaxTimeoutSeconds     int    `json:"maxTimeoutSeconds"`
}

type paymentRequiredResponse struct {
	X402Version int                  `json:"x402Version"`
	Accepts     []paymentRequirement `json:"accepts"`
}

func parsePaymentRequired(resp *http.Response) (*paymentRequirement, error) {
	// Try X-Payment-Required header first
	if h := resp.Header.Get("X-Payment-Required"); h != "" {
		var raw interface{}
		if json.Unmarshal([]byte(h), &raw) == nil {
			if pr := parseRequirementFromRaw(raw); pr != nil {
				return pr, nil
			}
		}
		decoded, err := base64.StdEncoding.DecodeString(h)
		if err == nil {
			var raw interface{}
			if json.Unmarshal(decoded, &raw) == nil {
				if pr := parseRequirementFromRaw(raw); pr != nil {
					return pr, nil
				}
			}
		}
	}
	// Try body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed paymentRequiredResponse
	if json.Unmarshal(data, &parsed) != nil {
		return nil, fmt.Errorf("parse payment required body")
	}
	if len(parsed.Accepts) == 0 {
		return nil, fmt.Errorf("no accepts in payment required")
	}
	// Prefer "exact" scheme for Base
	for i := range parsed.Accepts {
		if parsed.Accepts[i].Scheme == "exact" && (parsed.Accepts[i].Network == "eip155:8453" || parsed.Accepts[i].Network == "eip155:84532") {
			return normalizeRequirement(&parsed.Accepts[i]), nil
		}
	}
	return normalizeRequirement(&parsed.Accepts[0]), nil
}

func parseRequirementFromRaw(raw interface{}) *paymentRequirement {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	accepts, ok := m["accepts"].([]interface{})
	if !ok || len(accepts) == 0 {
		return nil
	}
	first, ok := accepts[0].(map[string]interface{})
	if !ok {
		return nil
	}
	pr := &paymentRequirement{}
	if v, ok := first["scheme"].(string); ok {
		pr.Scheme = v
	}
	if v, ok := first["network"].(string); ok {
		pr.Network = v
	}
	if v, ok := first["maxAmountRequired"].(string); ok {
		pr.MaxAmountRequired = v
	} else if v, ok := first["maxAmountRequired"].(float64); ok {
		pr.MaxAmountRequired = strconv.FormatFloat(v, 'f', 0, 64)
	}
	if v, ok := first["payToAddress"].(string); ok {
		pr.PayToAddress = v
	} else if v, ok := first["payTo"].(string); ok {
		pr.PayToAddress = v
	}
	if v, ok := first["usdcAddress"].(string); ok {
		pr.USDCAddress = v
	} else if v, ok := first["asset"].(string); ok {
		pr.USDCAddress = v
	}
	if v, ok := first["requiredDeadlineSeconds"].(float64); ok {
		pr.RequiredDeadlineSeconds = int(v)
	} else if v, ok := first["maxTimeoutSeconds"].(float64); ok {
		pr.RequiredDeadlineSeconds = int(v)
	}
	if pr.RequiredDeadlineSeconds <= 0 {
		pr.RequiredDeadlineSeconds = 300
	}
	return normalizeRequirement(pr)
}

func normalizeRequirement(pr *paymentRequirement) *paymentRequirement {
	if pr.PayToAddress == "" {
		pr.PayToAddress = pr.PayTo
	}
	if pr.USDCAddress == "" {
		switch pr.Network {
		case "eip155:8453":
			pr.USDCAddress = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
		case "eip155:84532":
			pr.USDCAddress = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
		}
	}
	return pr
}

// signTransferWithAuthorization signs EIP-3009 TransferWithAuthorization for USDC.
func signTransferWithAuthorization(key *ecdsa.PrivateKey, from string, req *paymentRequirement) (map[string]interface{}, error) {
	chainID := int64(8453)
	if req.Network == "eip155:84532" {
		chainID = 84532
	}

	// Parse amount: maxAmountRequired can be "5" (USD) or "5000000" (6 decimals)
	amount := new(big.Int)
	amountStr := strings.TrimSpace(req.MaxAmountRequired)
	if strings.Contains(amountStr, ".") {
		// e.g. "5.0" -> 5 * 1e6
		f, _, err := big.ParseFloat(amountStr, 10, 0, big.ToNearestEven)
		if err != nil {
			return nil, err
		}
		f.Mul(f, big.NewFloat(1e6))
		f.Int(amount)
	} else {
		if _, ok := amount.SetString(amountStr, 10); !ok {
			return nil, fmt.Errorf("invalid maxAmountRequired: %s", amountStr)
		}
		if amount.Cmp(big.NewInt(1e9)) < 0 {
			// Assume USD if small number
			amount.Mul(amount, big.NewInt(1e6))
		}
	}

	now := time.Now().Unix()
	validAfter := now - 60
	validBefore := now + int64(req.RequiredDeadlineSeconds)
	nonce := common.BytesToHash(crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", now))).Bytes())

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"TransferWithAuthorization": {
				{Name: "from", Type: "address"},
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "validAfter", Type: "uint256"},
				{Name: "validBefore", Type: "uint256"},
				{Name: "nonce", Type: "bytes32"},
			},
		},
		PrimaryType: "TransferWithAuthorization",
		Domain: apitypes.TypedDataDomain{
			Name:              "USD Coin",
			Version:           "2",
			ChainId:           math.NewHexOrDecimal256(chainID),
			VerifyingContract: req.USDCAddress,
		},
		Message: apitypes.TypedDataMessage{
			"from":        common.HexToAddress(from),
			"to":          common.HexToAddress(req.PayToAddress),
			"value":       amount.String(),
			"validAfter":  strconv.FormatInt(validAfter, 10),
			"validBefore": strconv.FormatInt(validBefore, 10),
			"nonce":       nonce,
		},
	}

	sighash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(sighash, key)
	if err != nil {
		return nil, err
	}
	// Adjust V for EIP-155
	if sig[64] < 27 {
		sig[64] += 27
	}

	return map[string]interface{}{
		"x402Version": 1,
		"scheme":      req.Scheme,
		"network":     req.Network,
		"payload": map[string]interface{}{
			"signature": hexutil.Bytes(sig),
			"authorization": map[string]interface{}{
				"from":        from,
				"to":          req.PayToAddress,
				"value":       amount.String(),
				"validAfter":  strconv.FormatInt(validAfter, 10),
				"validBefore": strconv.FormatInt(validBefore, 10),
				"nonce":       "0x" + common.Bytes2Hex(nonce.Bytes()),
			},
		},
	}, nil
}
