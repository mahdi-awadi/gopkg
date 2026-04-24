package pipeline

import (
	"errors"
	"testing"
)

func TestSentinelErrorsUnique(t *testing.T) {
	errs := []error{ErrFormatBridge, ErrToolExecutorPanicked}
	for i, a := range errs {
		for j, b := range errs {
			if i != j && errors.Is(a, b) {
				t.Errorf("errs[%d] and errs[%d] compare equal", i, j)
			}
		}
	}
}

func TestSentinelErrorsHaveMessages(t *testing.T) {
	if ErrFormatBridge.Error() == "" || ErrToolExecutorPanicked.Error() == "" {
		t.Error("sentinel errors must have non-empty messages")
	}
}
