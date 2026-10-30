package auth

import (
	"context"
	"strconv"
	"testing"
	"time"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

func authWithWeeklyReset(id string, reset time.Time) *Auth {
	return &Auth{ID: id, Quota: QuotaState{Signals: map[string]string{
		claudeWeeklyResetSignal: strconv.FormatInt(reset.Unix(), 10),
	}}}
}

func pickSoonestReset(t *testing.T, auths []*Auth) string {
	t.Helper()
	got, err := (&SoonestResetSelector{}).Pick(context.Background(), "claude", "", cliproxyexecutor.Options{}, auths)
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	return got.ID
}

func TestSoonestResetSelectorPick_PrefersEarliestWeeklyReset(t *testing.T) {
	t.Parallel()

	now := time.Now()
	got := pickSoonestReset(t, []*Auth{
		authWithWeeklyReset("a", now.Add(5*24*time.Hour)),
		authWithWeeklyReset("b", now.Add(10*time.Hour)),
		authWithWeeklyReset("c", now.Add(3*24*time.Hour)),
	})
	if got != "b" {
		t.Fatalf("Pick() auth.ID = %q, want %q", got, "b")
	}
}

func TestSoonestResetSelectorPick_UnknownResetGoesFirst(t *testing.T) {
	t.Parallel()

	got := pickSoonestReset(t, []*Auth{
		authWithWeeklyReset("a", time.Now().Add(time.Hour)),
		{ID: "b"},
	})
	if got != "b" {
		t.Fatalf("Pick() auth.ID = %q, want %q", got, "b")
	}
}

func TestSoonestResetSelectorPick_PastResetCountsAsUnknown(t *testing.T) {
	t.Parallel()

	now := time.Now()
	got := pickSoonestReset(t, []*Auth{
		authWithWeeklyReset("a", now.Add(time.Hour)),
		authWithWeeklyReset("b", now.Add(-time.Minute)),
	})
	if got != "b" {
		t.Fatalf("Pick() auth.ID = %q, want %q", got, "b")
	}
}

func TestSoonestResetSelectorPick_SkipsCredentialInQuotaCooldown(t *testing.T) {
	t.Parallel()

	now := time.Now()
	exhausted := authWithWeeklyReset("a", now.Add(time.Hour))
	exhausted.Quota.Exceeded = true
	exhausted.Quota.Reason = "credential_quota"
	exhausted.Quota.NextRecoverAt = now.Add(time.Hour)

	got := pickSoonestReset(t, []*Auth{
		exhausted,
		authWithWeeklyReset("b", now.Add(4*24*time.Hour)),
	})
	if got != "b" {
		t.Fatalf("Pick() auth.ID = %q, want %q", got, "b")
	}
}
