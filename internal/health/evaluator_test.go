package health

import (
	"testing"
	"time"
)

func TestEvaluate(t *testing.T) {
	store := NewStore()
	store.Update(Result{
		Available:     true,
		ReadOnly:      false,
		ReadOnlyKnown: true,
		Healthy:       true,
		Reason:        "mysql is healthy",
		CheckedAt:     time.Now(),
	})

	t.Setenv("MYSQL_MASTER", "1")
	got := Evaluate(store)
	if !got.Healthy {
		t.Fatalf("Evaluate() = %+v, want healthy", got)
	}
	if got.FailureReason != FailureReasonNone {
		t.Fatalf("FailureReason = %q, want none", got.FailureReason)
	}

	t.Setenv("MYSQL_MASTER", "0")
	got = Evaluate(store)
	if got.Healthy {
		t.Fatalf("Evaluate() = %+v, want unhealthy when MYSQL_MASTER=0", got)
	}
	if got.FailureReason != FailureReasonMasterEnv {
		t.Fatalf("FailureReason = %q", got.FailureReason)
	}

	store.Update(Result{Available: false, Reason: "mysql is unavailable"})
	t.Setenv("MYSQL_MASTER", "true")
	got = Evaluate(store)
	if got.Healthy {
		t.Fatalf("Evaluate() = %+v, want unhealthy mysql", got)
	}
	if got.FailureReason != FailureReasonMySQLUnavailable {
		t.Fatalf("FailureReason = %q", got.FailureReason)
	}

	store.Update(Result{
		Available:     true,
		ReadOnly:      true,
		ReadOnlyKnown: true,
	})
	got = Evaluate(store)
	if got.FailureReason != FailureReasonReadOnly {
		t.Fatalf("FailureReason = %q, want mysql_read_only", got.FailureReason)
	}
}
