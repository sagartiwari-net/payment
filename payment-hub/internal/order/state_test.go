package order

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"pending", "processing", true},
		{"pending", "expired", true},
		{"pending", "success", false},
		{"processing", "success", true},
		{"processing", "failed", true},
		{"success", "refunded", true},
		{"success", "failed", false},
		{"failed", "pending", false},
		{"success", "success", true},
	}

	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestIsFinalStatus(t *testing.T) {
	if !IsFinalStatus("success") || !IsFinalStatus("failed") {
		t.Fatal("expected final statuses")
	}
	if IsFinalStatus("pending") {
		t.Fatal("pending should not be final")
	}
}
