package testcover

import "testing"

func TestAbs(t *testing.T) {
	if Abs(-2) != 2 {
		t.Fatal("Abs(-2) != 2")
	}
}

func TestAbs_Positive(t *testing.T) {
	if Abs(3) != 3 {
		t.Fatal("Abs(3) != 3")
	}
}

func TestAbs_Negative(t *testing.T) {
	if Abs(-5) != 5 {
		t.Fatal("Abs(-5) != 5")
	}
}
