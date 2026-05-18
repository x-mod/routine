package routine

import (
	"context"
	"testing"
)

func TestWithArgments(t *testing.T) {
	ctx := WithArgments(context.Background(), "a", 1, true)
	args, ok := ArgumentsFrom(ctx)
	if !ok {
		t.Fatal("expected args")
	}
	if len(args) != 3 || args[0] != "a" || args[1] != 1 || args[2] != true {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestArgumentsFromMissing(t *testing.T) {
	args, ok := ArgumentsFrom(context.Background())
	if ok || args != nil {
		t.Fatal("expected no args")
	}
}

func TestArgumentsFromNilCtx(t *testing.T) {
	args, ok := ArgumentsFrom(nil)
	if ok || args != nil {
		t.Fatal("expected no args for nil context")
	}
}

func TestWithArgmentsNilCtx(t *testing.T) {
	ctx := WithArgments(nil, "x")
	if ctx != nil {
		t.Fatal("expected nil context back")
	}
}

func TestWithParentAndParentFrom(t *testing.T) {
	r := New(ExecutorFunc(func(ctx context.Context) error { return nil }))
	ctx := WithParent(context.Background(), r)
	got, ok := ParentFrom(ctx)
	if !ok || got != r {
		t.Fatal("expected parent routine")
	}
}

func TestParentFromMissing(t *testing.T) {
	_, ok := ParentFrom(context.Background())
	if ok {
		t.Fatal("expected no parent")
	}
}
