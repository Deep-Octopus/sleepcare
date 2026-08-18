package careclient

import (
	"testing"
	"time"
)

func TestCareAssignmentEffectiveStatus(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	ended := now.Add(-time.Minute)
	cancelled := now
	tests := []struct {
		name string
		item CareAssignment
		want string
	}{
		{name: "scheduled", item: CareAssignment{ValidFrom: future}, want: AssignmentStatusScheduled},
		{name: "active", item: CareAssignment{ValidFrom: past}, want: AssignmentStatusActive},
		{name: "ended", item: CareAssignment{ValidFrom: past, ValidUntil: &ended}, want: AssignmentStatusEnded},
		{name: "cancelled", item: CareAssignment{ValidFrom: past, CancelledAt: &cancelled}, want: AssignmentStatusCancelled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.EffectiveStatus(now); got != tt.want {
				t.Fatalf("EffectiveStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
