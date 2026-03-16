package simulation

import (
	"math/rand"
	"time"
)

// ChaosInjector applies configurable failures for stability testing.
type ChaosInjector struct {
	level ChaosLevel
	rng   *rand.Rand
}

// NewChaosInjector creates an injector with the given level and seed.
func NewChaosInjector(level ChaosLevel, seed int64) *ChaosInjector {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &ChaosInjector{
		level: level,
		rng:   rand.New(rand.NewSource(seed)),
	}
}

// ShouldInjectAPITimeout returns true when an API timeout should be simulated.
func (c *ChaosInjector) ShouldInjectAPITimeout() bool {
	switch c.level {
	case ChaosNone:
		return false
	case ChaosLow:
		return c.rng.Float64() < 0.02 // 2%
	case ChaosMedium:
		return c.rng.Float64() < 0.10 // 10%
	case ChaosHigh:
		return c.rng.Float64() < 0.25 // 25%
	default:
		return false
	}
}

// ShouldInjectEmptyLLMResponse returns true when LLM should return empty (tests trimmer).
func (c *ChaosInjector) ShouldInjectEmptyLLMResponse() bool {
	switch c.level {
	case ChaosNone, ChaosLow:
		return false
	case ChaosMedium:
		return c.rng.Float64() < 0.01 // 1%
	case ChaosHigh:
		return c.rng.Float64() < 0.05 // 5%
	default:
		return false
	}
}

// ShouldInjectPriceFlashCrash returns true when a price shock should occur.
func (c *ChaosInjector) ShouldInjectPriceFlashCrash() bool {
	switch c.level {
	case ChaosNone, ChaosLow:
		return false
	case ChaosMedium:
		return c.rng.Float64() < 0.05 // 5%
	case ChaosHigh:
		return c.rng.Float64() < 0.15 // 15%
	default:
		return false
	}
}
