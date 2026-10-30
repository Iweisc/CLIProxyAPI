package auth

import (
	"context"
	"strconv"
	"strings"
	"time"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

const claudeWeeklyResetSignal = "Anthropic-Ratelimit-Unified-7d-Reset"

// SoonestResetSelector picks the credential whose weekly quota window resets
// first. Subscription quota left over at a reset is lost, so spending the
// account that resets soonest wastes less than spreading load evenly.
//
// The reset time comes from the passive quota snapshot of the credential's last
// response. Credentials without a known future reset sort first, so a new or
// freshly reset account gets one request and records its next reset time.
type SoonestResetSelector struct{}

// Pick selects the available auth with the earliest weekly quota reset.
func (s *SoonestResetSelector) Pick(ctx context.Context, provider, model string, opts cliproxyexecutor.Options, auths []*Auth) (*Auth, error) {
	_ = opts
	now := time.Now()
	available, err := getSelectorAvailableAuths(ctx, auths, provider, model, now)
	if err != nil {
		return nil, err
	}
	available = preferCodexWebsocketAuths(ctx, provider, available)
	picked := available[0]
	pickedReset, pickedKnown := weeklyQuotaResetAt(picked, now)
	for _, auth := range available[1:] {
		if !pickedKnown {
			break
		}
		reset, known := weeklyQuotaResetAt(auth, now)
		if !known || reset.Before(pickedReset) {
			picked, pickedReset, pickedKnown = auth, reset, known
		}
	}
	return picked, nil
}

// weeklyQuotaResetAt returns the observed weekly reset when it is still ahead.
// A reset in the past means the window already rolled over and the next reset
// is unknown until the credential answers again.
func weeklyQuotaResetAt(auth *Auth, now time.Time) (time.Time, bool) {
	if auth == nil {
		return time.Time{}, false
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(auth.Quota.Signals[claudeWeeklyResetSignal]), 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}, false
	}
	reset := time.Unix(seconds, 0)
	if !reset.After(now) {
		return time.Time{}, false
	}
	return reset, true
}
