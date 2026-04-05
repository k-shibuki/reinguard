package prquery

import (
	"testing"
	"time"
)

func TestAdjustRateLimitRemainingForStatusCommentAge_table(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	//nolint:govet // test table: readable field order
	tests := []struct {
		name     string
		statusAt string
		now      time.Time
		parsed   int
		want     int
	}{
		{
			name:     "same_instant_no_elapsed",
			statusAt: base.Format(time.RFC3339),
			now:      base,
			parsed:   125,
			want:     125,
		},
		{
			name:     "one_minute_elapsed",
			statusAt: base.Format(time.RFC3339),
			now:      base.Add(60 * time.Second),
			parsed:   125,
			want:     65,
		},
		{
			name:     "elapsed_exceeds_parsed_clamps_to_zero",
			statusAt: base.Format(time.RFC3339),
			now:      base.Add(200 * time.Second),
			parsed:   125,
			want:     0,
		},
		{
			name:     "negative_elapsed_clock_skew_treats_elapsed_as_zero",
			statusAt: base.Format(time.RFC3339),
			now:      base.Add(-90 * time.Second),
			parsed:   125,
			want:     125,
		},
		{
			name:     "parsed_zero_long_elapsed_stays_zero",
			statusAt: base.Format(time.RFC3339),
			now:      base.Add(500 * time.Second),
			parsed:   0,
			want:     0,
		},
		{
			name:     "edited_comment_new_anchor_simulated_later_updatedAt",
			statusAt: base.Add(5 * time.Minute).Format(time.RFC3339),
			now:      base.Add(5*time.Minute + 30*time.Second),
			parsed:   120,
			want:     90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status := map[string]any{
				"rate_limit_remaining_seconds": tt.parsed,
				"status_comment_at":            tt.statusAt,
			}
			adjustRateLimitRemainingForStatusCommentAge(status, tt.now)
			got, ok := status["rate_limit_remaining_seconds"].(int)
			if !ok {
				t.Fatalf("missing or wrong type: %+v", status)
			}
			if got != tt.want {
				t.Fatalf("remaining = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAdjustRateLimitRemainingForStatusCommentAge_skipsWithoutAnchorOrParsed(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	t.Run("empty_status_comment_at", func(t *testing.T) {
		t.Parallel()
		status := map[string]any{"rate_limit_remaining_seconds": 100}
		adjustRateLimitRemainingForStatusCommentAge(status, now)
		if got := status["rate_limit_remaining_seconds"].(int); got != 100 {
			t.Fatalf("want unchanged 100, got %d", got)
		}
	})

	t.Run("invalid_rfc3339_timestamp", func(t *testing.T) {
		t.Parallel()
		status := map[string]any{
			"rate_limit_remaining_seconds": 100,
			"status_comment_at":            "not-a-time",
		}
		adjustRateLimitRemainingForStatusCommentAge(status, now)
		if got := status["rate_limit_remaining_seconds"].(int); got != 100 {
			t.Fatalf("want unchanged 100, got %d", got)
		}
	})

	t.Run("negative_parsed_skipped", func(t *testing.T) {
		t.Parallel()
		status := map[string]any{
			"rate_limit_remaining_seconds": -5,
			"status_comment_at":            now.Format(time.RFC3339),
		}
		adjustRateLimitRemainingForStatusCommentAge(status, now)
		if got := status["rate_limit_remaining_seconds"].(int); got != -5 {
			t.Fatalf("want unchanged -5, got %d", got)
		}
	})
}
