package questionnaire

import (
	"encoding/json"
	"testing"
)

func TestContentLifecycleTransitionsAreForwardOnly(t *testing.T) {
	allowed := [][2]string{
		{LifecycleDraft, LifecycleInReview},
		{LifecycleInReview, LifecycleApproved},
		{LifecycleApproved, LifecyclePublished},
		{LifecyclePublished, LifecycleDisabled},
	}
	for _, transition := range allowed {
		if !CanTransitionLifecycle(transition[0], transition[1]) {
			t.Fatalf("expected transition %s -> %s", transition[0], transition[1])
		}
	}
	denied := [][2]string{
		{LifecycleDraft, LifecyclePublished},
		{LifecyclePublished, LifecycleDraft},
		{LifecycleDisabled, LifecyclePublished},
		{LifecycleApproved, LifecycleInReview},
	}
	for _, transition := range denied {
		if CanTransitionLifecycle(transition[0], transition[1]) {
			t.Fatalf("unexpected transition %s -> %s", transition[0], transition[1])
		}
	}
}

func TestCanonicalJSONMakesObjectKeyOrderStable(t *testing.T) {
	left := CanonicalJSON(json.RawMessage(`{"value":"A","operator":"EQUALS","questionCode":"q"}`))
	right := CanonicalJSON(json.RawMessage(`{"operator":"EQUALS","questionCode":"q","value":"A"}`))
	if string(left) != string(right) {
		t.Fatalf("canonical JSON differs: %s vs %s", left, right)
	}
}
