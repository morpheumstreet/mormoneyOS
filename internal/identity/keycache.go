// Package identity: short-lived cache for derived keys.
// Reduces exposure window; cleared on shutdown and periodically.
package identity

import (
	"crypto/ecdsa"
	"sync"
	"time"
)

const defaultCacheTTL = 10 * time.Minute

type evmCacheKey struct {
	index uint32
}

type evmCacheEntry struct {
	priv *ecdsa.PrivateKey
	addr string
	exp  time.Time
}

type addrCacheKey struct {
	chain string
	index uint32
}

type addrCacheEntry struct {
	addr string
	exp  time.Time
}

var (
	evmCache   = make(map[evmCacheKey]*evmCacheEntry)
	addrCache  = make(map[addrCacheKey]*addrCacheEntry)
	cacheMu    sync.RWMutex
	cacheTTL   = defaultCacheTTL
	cacheEnabled = true
)

// SetKeyCacheTTL sets the TTL for cached derived keys. Default 10 minutes.
func SetKeyCacheTTL(d time.Duration) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheTTL = d
}

// ClearDerivedKeys clears all cached derived keys from memory.
// Call on: graceful shutdown (SIGTERM), after sensitive operations, periodically.
func ClearDerivedKeys() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	evmCache = make(map[evmCacheKey]*evmCacheEntry)
	addrCache = make(map[addrCacheKey]*addrCacheEntry)
}

// getOrDeriveEVM returns cached EVM key or derives and caches.
func getOrDeriveEVM(mnemonic string, index uint32) (*ecdsa.PrivateKey, string, error) {
	if !cacheEnabled {
		return DeriveEVMPrivateKeyFromMnemonic(mnemonic, index)
	}
	key := evmCacheKey{index: index}
	cacheMu.RLock()
	if e, ok := evmCache[key]; ok && time.Now().Before(e.exp) {
		cacheMu.RUnlock()
		return e.priv, e.addr, nil
	}
	cacheMu.RUnlock()

	priv, addr, err := DeriveEVMPrivateKeyFromMnemonic(mnemonic, index)
	if err != nil {
		return nil, "", err
	}
	cacheMu.Lock()
	evmCache[key] = &evmCacheEntry{priv: priv, addr: addr, exp: time.Now().Add(cacheTTL)}
	cacheMu.Unlock()
	return priv, addr, nil
}

// getOrDeriveAddress returns cached address or derives and caches.
func getOrDeriveAddress(mnemonic, chainCAIP2 string, index uint32) (string, error) {
	if !cacheEnabled {
		return DeriveAddressFromMnemonic(mnemonic, chainCAIP2, index)
	}
	key := addrCacheKey{chain: chainCAIP2, index: index}
	cacheMu.RLock()
	if e, ok := addrCache[key]; ok && time.Now().Before(e.exp) {
		cacheMu.RUnlock()
		return e.addr, nil
	}
	cacheMu.RUnlock()

	addr, err := DeriveAddressFromMnemonic(mnemonic, chainCAIP2, index)
	if err != nil {
		return "", err
	}
	cacheMu.Lock()
	addrCache[key] = &addrCacheEntry{addr: addr, exp: time.Now().Add(cacheTTL)}
	cacheMu.Unlock()
	return addr, nil
}
