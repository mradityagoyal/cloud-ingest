package helpers

import (
	"testing"
)

func TestUniformGetNext(t *testing.T) {
	d := NewUniformDistribution(0, 10, 1)
	// rand.Intn with seed of 1 will generate {1 7 7 9 1 8 5 0 6 0 ...} sequence.
	expected_rand := []int{1, 7, 7, 9, 1, 8, 5, 0, 6, 0}
	for _, r := range expected_rand {
		next := d.GetNext()
		if r != next {
			t.Errorf("expected %d, but found %v", r, next)
		}
	}
}

func TestUniformGetNextNonZeroStart(t *testing.T) {
	d := NewUniformDistribution(3, 13, 1)
	// rand.Intn with seed of 1 will generate {1 7 7 9 1 8 5 0 6 0 ...} sequence.
	expected_rand := []int{1, 7, 7, 9, 1, 8, 5, 0, 6, 0}
	for _, r := range expected_rand {
		next := d.GetNext()
		if r+3 != next {
			t.Errorf("expected %d, but found %v", r, next)
		}
	}
}

func TestUniformGetNextSingleValue(t *testing.T) {
	d := NewUniformDistribution(10, 11, 1)
	for i := 0; i < 5; i++ {
		r := d.GetNext()
		if r != 10 {
			t.Errorf("expected 10, but found %v", r)
		}
	}
}

func TestUniformMax(t *testing.T) {
	d := NewUniformDistribution(10, 11, 1)
	m := d.Max()
	if m != 10 {
		t.Errorf("expected 10, but found %v", m)
	}
}
