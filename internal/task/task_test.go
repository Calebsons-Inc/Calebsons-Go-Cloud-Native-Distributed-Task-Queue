package task

import (
	"strings"
	"testing"
)

func TestStatusValid(t *testing.T) {
	for _, s := range ValidStatuses {
		if !s.Valid() {
			t.Fatalf("%q should be valid", s)
		}
	}
	if Status("pending").Valid() {
		t.Fatal("pending should not be valid")
	}
}

func TestParseStatus(t *testing.T) {
	got, ok := ParseStatus("  RUNNING ")
	if !ok || got != StatusRunning {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := ParseStatus("nope"); ok {
		t.Fatal("expected invalid status")
	}
}

func TestCreateRequestValidate(t *testing.T) {
	t.Run("ok trims name", func(t *testing.T) {
		req := &CreateRequest{Name: "  sync-inventory  ", MaxAttempts: 3}
		if err := req.Validate(); err != nil {
			t.Fatal(err)
		}
		if req.Name != "sync-inventory" {
			t.Fatalf("name not trimmed: %q", req.Name)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		req := &CreateRequest{Name: "   "}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("name too long", func(t *testing.T) {
		req := &CreateRequest{Name: strings.Repeat("a", 121)}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("payload too large", func(t *testing.T) {
		req := &CreateRequest{Name: "job", Payload: strings.Repeat("x", 8<<10+1)}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("max attempts bounds", func(t *testing.T) {
		neg := &CreateRequest{Name: "job", MaxAttempts: -1}
		if err := neg.Validate(); err == nil {
			t.Fatal("expected negative error")
		}
		high := &CreateRequest{Name: "job", MaxAttempts: 21}
		if err := high.Validate(); err == nil {
			t.Fatal("expected max error")
		}
		zero := &CreateRequest{Name: "job", MaxAttempts: 0}
		if err := zero.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		var req *CreateRequest
		if err := req.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestTaskCanCancel(t *testing.T) {
	cases := map[Status]bool{
		StatusQueued:    true,
		StatusRunning:   true,
		StatusFailed:    true,
		StatusCompleted: false,
		StatusCanceled:  false,
		StatusDead:      false,
	}
	for status, want := range cases {
		got := Task{Status: status}.CanCancel()
		if got != want {
			t.Fatalf("%s: got %v want %v", status, got, want)
		}
	}
}
