package servicos

import "testing"

func TestFloor2(t *testing.T) {
	if got := floor2(0.199); got != 0.19 {
		t.Fatalf("floor2(0.199) = %v; want 0.19", got)
	}
	if got := floor2(1.0); got != 1.0 {
		t.Fatalf("floor2(1.0) = %v; want 1.0", got)
	}
}

func TestNormalizarPercentual(t *testing.T) {
	if got := normalizarPercentual(0.5); got != 50 {
		t.Fatalf("normalizarPercentual(0.5) = %v; want 50", got)
	}
	if got := normalizarPercentual(1.5); got != 1.5 {
		t.Fatalf("normalizarPercentual(1.5) = %v; want 1.5", got)
	}
}
