// Package metrics — white-box tests for pure-core helpers in types.go.
// Uses `package metrics` (not `package metrics_test`) so it can reach
// package-private helpers such as validateRTTFloat.
package metrics

import (
	"math"
	"testing"
)

// TestValidateRTTFloat_RejectsNaN verifies that validateRTTFloat returns an
// error for math.NaN(). This exercises the numeric-validation guard directly,
// independent of the JSON string-token branch in UnmarshalJSON (F-P5L2-02).
func TestValidateRTTFloat_RejectsNaN(t *testing.T) {
	t.Parallel()

	if err := validateRTTFloat(math.NaN()); err == nil {
		t.Error("validateRTTFloat(NaN): expected error; got nil")
	}
}

// TestValidateRTTFloat_RejectsPosInf verifies that validateRTTFloat returns an
// error for positive infinity (F-P5L2-02).
func TestValidateRTTFloat_RejectsPosInf(t *testing.T) {
	t.Parallel()

	if err := validateRTTFloat(math.Inf(1)); err == nil {
		t.Error("validateRTTFloat(+Inf): expected error; got nil")
	}
}

// TestValidateRTTFloat_RejectsNegInf verifies that validateRTTFloat returns an
// error for negative infinity (F-P5L2-02).
func TestValidateRTTFloat_RejectsNegInf(t *testing.T) {
	t.Parallel()

	if err := validateRTTFloat(math.Inf(-1)); err == nil {
		t.Error("validateRTTFloat(-Inf): expected error; got nil")
	}
}

// TestValidateRTTFloat_RejectsNegative verifies that validateRTTFloat returns
// an error for any negative value — negative RTT has no physical meaning
// (F-P5L2-02).
func TestValidateRTTFloat_RejectsNegative(t *testing.T) {
	t.Parallel()

	if err := validateRTTFloat(-1.0); err == nil {
		t.Error("validateRTTFloat(-1.0): expected error; got nil")
	}
}

// TestValidateRTTFloat_AcceptsZeroAndPositive verifies that validateRTTFloat
// accepts zero and positive finite values (valid measured RTTs).
func TestValidateRTTFloat_AcceptsZeroAndPositive(t *testing.T) {
	t.Parallel()

	cases := []float64{0, 0.001, 15.0, 100.0, 500.0, 1000.0}
	for _, v := range cases {
		v := v
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if err := validateRTTFloat(v); err != nil {
				t.Errorf("validateRTTFloat(%v): expected nil; got %v", v, err)
			}
		})
	}
}
