package steganalysis

import "testing"

func TestHolmMonotoneAndAdjusted(t *testing.T) {
	got := Holm([]float64{0.04, 0.01, 0.03})
	// Sorted p: 0.01, 0.03, 0.04. Multipliers 3, 2, 1 give 0.03, 0.06, 0.04,
	// and the step-down forces the last up to 0.06. Input order is restored.
	want := []float64{0.06, 0.03, 0.06}
	if len(got) != len(want) {
		t.Fatalf("len %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %v want %v", i, got, want)
		}
	}
}

func TestBHStepUp(t *testing.T) {
	got := BH([]float64{0.01, 0.04, 0.03})
	want := []float64{0.03, 0.04, 0.04}
	if len(got) != len(want) {
		t.Fatalf("len %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %v want %v", i, got, want)
		}
	}
}
