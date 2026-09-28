package test

import (
	"testing"

	"github.com/mark3labs/kit/extensions"
)

// TestPromptStateChangeCrossesYaegiBoundary proves the modal-visibility event
// survives the interpreter with all of its fields intact.
//
// Yaegi hands an interpreted callback a zero value for a type it cannot
// resolve, and a missing symbols.go entry fails just that field rather than
// the load — so a test that only checked the extension compiled would pass
// against a half-registered event. Asserting on every field is what makes this
// a real check.
//
// The extension also reproduces the state machine the event exists for: a
// modal open arrives bracketed by a turn event that reads "idle", and only the
// prompt edge reveals that the turn is still parked on the user. Tracking
// that here, in interpreted code, is the shape a status-line extension needs.
func TestPromptStateChangeCrossesYaegiBoundary(t *testing.T) {
	src := `package main

import (
	"fmt"

	"kit/ext"
)

// modalOpen mirrors the flag a status-line extension keeps: TurnStateChange
// alone would clear it the instant a modal took the keyboard.
var modalOpen bool

func Init(api ext.API) {
	api.OnPromptStateChange(func(e ext.PromptStateChangeEvent, ctx ext.Context) {
		modalOpen = e.State == "open"
		ctx.Print(fmt.Sprintf("prompt %s %s %s->%s", e.Kind, e.PromptType, e.Previous, e.State))
	})

	api.OnTurnStateChange(func(e ext.TurnStateChangeEvent, ctx ext.Context) {
		if e.State == "idle" && !modalOpen {
			ctx.Print("title: ✅ KIT: Ready")
			return
		}
		if modalOpen {
			ctx.Print("title: ⏸ KIT: Waiting for you")
			return
		}
		ctx.Print("title: 🔨 KIT: Working")
	})
}
`

	harness := New(t)
	harness.LoadString(src, "prompt_state.go")

	emit := func(e extensions.Event) {
		t.Helper()
		if _, err := harness.Emit(e); err != nil {
			t.Fatalf("Emit(%T) error = %v", e, err)
		}
	}

	// A tool blocks on PromptSelect: the modal takes the keyboard, and the
	// turn state reports idle underneath it.
	emit(extensions.PromptStateChangeEvent{
		Kind: "prompt", PromptType: "select", State: "open", Previous: "closed",
	})
	emit(extensions.TurnStateChangeEvent{State: "idle", Previous: "working"})

	// The user answers; the modal closes and the turn is genuinely done.
	emit(extensions.PromptStateChangeEvent{
		Kind: "prompt", PromptType: "select", State: "closed", Previous: "open",
	})
	emit(extensions.TurnStateChangeEvent{State: "working", Previous: "idle"})

	want := []string{
		"prompt prompt select closed->open",
		"title: ⏸ KIT: Waiting for you",
		"prompt prompt select open->closed",
		"title: 🔨 KIT: Working",
	}

	got := harness.Context().GetPrints()
	if len(got) != len(want) {
		t.Fatalf("got %d prints (%v); want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("print[%d]:\n got %q\nwant %q", i, got[i], want[i])
		}
	}
}

// TestPromptStateChangeOverlayTypeIsNamed keeps the overlay half honest. An
// overlay carries no prompt type of its own, and a handler that formats the
// field directly must still get something meaningful rather than a blank.
func TestPromptStateChangeOverlayTypeIsNamed(t *testing.T) {
	src := `package main

import (
	"fmt"

	"kit/ext"
)

func Init(api ext.API) {
	api.OnPromptStateChange(func(e ext.PromptStateChangeEvent, ctx ext.Context) {
		ctx.Print(fmt.Sprintf("%s|%s", e.Kind, e.PromptType))
	})
}
`

	harness := New(t)
	harness.LoadString(src, "overlay_type.go")

	if _, err := harness.Emit(extensions.PromptStateChangeEvent{
		Kind: "overlay", PromptType: "overlay", State: "open", Previous: "closed",
	}); err != nil {
		t.Fatalf("Emit error = %v", err)
	}

	got := harness.Context().GetPrints()
	if len(got) != 1 || got[0] != "overlay|overlay" {
		t.Errorf("got %v; want [overlay|overlay]", got)
	}
}
