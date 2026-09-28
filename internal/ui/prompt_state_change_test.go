package ui

import (
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mark3labs/kit/internal/app"
)

// --------------------------------------------------------------------------
// Extension modal observability
// --------------------------------------------------------------------------

// promptEdge is one PromptStateChange delivery, in the order the extension
// received it.
type promptEdge struct {
	kind     string
	promptTy string
	state    string
	previous string
}

// turnEdge is one TurnStateChange delivery.
type turnEdge struct {
	state    string
	previous string
}

// modalRecorder captures both event streams from the OnPromptStateChange and
// OnTurnStateChange emitters.
//
// Handlers are wired the way cmd/root.go wires them — straight into
// notifyPromptState, which queues on the model's extension event dispatcher —
// so the test observes the real asynchrony. That is the point: emission order
// is guaranteed only by the dispatcher queue, and an assertion made against a
// synchronous stub would prove nothing about what an extension actually sees.
type modalRecorder struct {
	mu    sync.Mutex
	sig   chan struct{} // capacity 1; nudged on every delivery
	edges []promptEdge
	turns []turnEdge
}

func newModalRecorder() *modalRecorder {
	return &modalRecorder{sig: make(chan struct{}, 1)}
}

// nudge wakes a waiter. Buffered with a capacity of one because the count of
// events is what is being waited on, not the number of nudges: a coalesced
// signal is enough because every waiter re-checks the condition in a loop.
func (r *modalRecorder) nudge() {
	select {
	case r.sig <- struct{}{}:
	default:
	}
}

// emitPromptStateChange and emitTurnStateChange match the AppModelOpts
// signatures.
func (r *modalRecorder) emitPromptStateChange(kind, promptType, state, previous string) {
	r.mu.Lock()
	r.edges = append(r.edges, promptEdge{kind: kind, promptTy: promptType, state: state, previous: previous})
	r.mu.Unlock()
	r.nudge()
}

func (r *modalRecorder) emitTurnStateChange(state, previous string) {
	r.mu.Lock()
	r.turns = append(r.turns, turnEdge{state: state, previous: previous})
	r.mu.Unlock()
	r.nudge()
}

// await blocks until n prompt events have been delivered. Dispatch runs on a
// background goroutine, so a test that read the slices straight out of Update
// would race — and would pass or fail on scheduler timing rather than on the
// behaviour under test.
func (r *modalRecorder) await(n int) []promptEdge {
	deadline := time.After(2 * time.Second)
	for {
		r.mu.Lock()
		edges := append([]promptEdge(nil), r.edges...)
		r.mu.Unlock()
		if len(edges) >= n {
			return edges
		}

		select {
		case <-r.sig:
		case <-deadline:
			return edges
		}
	}
}

// awaitTurns is await for the turn-state stream, which must arrive alongside
// the prompt events it is being compared against.
func (r *modalRecorder) awaitTurns(n int) []turnEdge {
	deadline := time.After(2 * time.Second)
	for {
		r.mu.Lock()
		turns := append([]turnEdge(nil), r.turns...)
		r.mu.Unlock()
		if len(turns) >= n {
			return turns
		}

		select {
		case <-r.sig:
		case <-deadline:
			return turns
		}
	}
}

// quiet asserts nothing more arrives, for the paths that must stay silent.
// A short fixed wait is the honest tool here: a negative assertion cannot be
// proved by a condition, only by time passing without a delivery.
func (r *modalRecorder) quiet(t *testing.T, d time.Duration) []promptEdge {
	t.Helper()
	time.Sleep(d)
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]promptEdge(nil), r.edges...)
}

// modalTestModel builds a model wired to a recorder, with its frame clock and
// event dispatcher running. Frame ticks arriving from a nil frameClock are
// dropped without touching the recorder, so a stray tick cannot desynchronize
// the expectations below.
func modalTestModel(t *testing.T) (*AppModel, *modalRecorder) {
	t.Helper()
	m, _, _ := newTestAppModel(&stubAppController{})
	m.frames = frameClock{}

	rec := newModalRecorder()
	m.emitPromptStateChange = rec.emitPromptStateChange
	m.emitTurnStateChange = rec.emitTurnStateChange
	m.extEvents = newExtEventDispatcher()

	return m, rec
}

// TestPromptStateChange_ParkedTurnIsDistinguishableFromFinishedTurn is the
// regression guard for the whole event.
//
// An extension tool that blocks inside ctx.PromptSelect parks the turn: the
// modal owns m.state, so the deferred working/idle comparison in update
// reports "idle" — and no AgentEnd will arrive until the user answers. A
// status line reading TurnStateChange alone therefore renders "Ready" for as
// long as the modal is up, which is exactly the bug this event exists for.
//
// The two events together are asserted, in one pass, because either alone is
// ambiguous: TurnStateChange "idle" here means "the keyboard moved", not "the
// turn ended".
func TestPromptStateChange_ParkedTurnIsDistinguishableFromFinishedTurn(t *testing.T) {
	m, rec := modalTestModel(t)
	m.state = stateWorking

	// A real tool would be blocked on this read, so it must be buffered for
	// the same reason it is in production: resolvePrompt sends from inside
	// Update, and an unbuffered channel would deadlock the event loop.
	respCh := make(chan app.PromptResponse, 1)

	sendMsg(m, app.PromptRequestEvent{
		PromptType: "select",
		Message:    "Which branch?",
		Options:    []string{"main", "dev"},
		ResponseCh: respCh,
	})

	if m.state != statePrompt {
		t.Fatalf("state = %v, want statePrompt while the modal owns the keyboard", m.state)
	}

	edges := rec.await(1)
	if len(edges) != 1 {
		t.Fatalf("got %d prompt events (%+v), want the open edge", len(edges), edges)
	}
	if edges[0] != (promptEdge{kind: "prompt", promptTy: "select", state: "open", previous: "closed"}) {
		t.Errorf("open edge = %+v", edges[0])
	}

	// Answering: enter selects the highlighted option and resolves the prompt.
	sendMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.state != stateWorking {
		t.Errorf("state after answer = %v, want stateWorking restored", m.state)
	}
	edges = rec.await(2)
	if edges[1] != (promptEdge{kind: "prompt", promptTy: "select", state: "closed", previous: "open"}) {
		t.Errorf("close edge = %+v, want the same surface the open edge named", edges[1])
	}

	// The verdict the bug was about. Opening the modal produced a turn event
	// that reads "idle" while the agent was still blocked in the tool; the
	// prompt edges around it are what disambiguate that from a finished turn.
	// Dismissing the modal puts the working state straight back, so nothing
	// after it can be mistaken for the turn having ended.
	turns := rec.awaitTurns(2)
	want := []turnEdge{
		{state: "idle", previous: "working"},
		{state: "working", previous: "idle"},
	}
	if len(turns) != len(want) {
		t.Fatalf("got %d turn events (%+v), want %d", len(turns), turns, len(want))
	}
	for i := range want {
		if turns[i] != want[i] {
			t.Errorf("turn event %d = %+v, want %+v", i, turns[i], want[i])
		}
	}
	if turns[0].state != "idle" {
		t.Fatalf("TurnStateChange reported %q while a modal was open; this test "+
			"only proves anything while the turn reads as idle", turns[0].state)
	}

	if got := <-respCh; got.Value != "main" {
		t.Errorf("tool received %+v, want the selected option", got)
	}
}

// TestPromptStateChange_CloseEdgeLandsAfterTheStateRestore pins the ordering
// inside a single Update. The close edge must be emitted once m.state is back
// to the pre-modal value, so an extension that reacts to "closed" by consulting
// the live UI sees the same world the event describes — never a phantom modal
// that has already been torn down.
func TestPromptStateChange_CloseEdgeLandsAfterTheStateRestore(t *testing.T) {
	m, rec := modalTestModel(t)

	respCh := make(chan app.PromptResponse, 1)
	sendMsg(m, app.PromptRequestEvent{
		PromptType: "confirm",
		Message:    "Apply?",
		ResponseCh: respCh,
	})

	// The recorder reads m.state at delivery time. Delivery happens on the
	// dispatcher goroutine, after Update has returned, so this observes the
	// post-restore state rather than a mid-Update one.
	var mu sync.Mutex
	var stateAtClose appState
	m.emitPromptStateChange = func(kind, promptType, state, previous string) {
		if state == "closed" {
			mu.Lock()
			stateAtClose = m.state
			mu.Unlock()
		}
		rec.emitPromptStateChange(kind, promptType, state, previous)
	}

	sendMsg(m, tea.KeyPressMsg{Code: tea.KeyEscape})

	rec.await(2)

	mu.Lock()
	defer mu.Unlock()
	if stateAtClose != stateInput {
		t.Errorf("state observed by the close handler = %v, want stateInput (already restored)", stateAtClose)
	}
	if (<-respCh).Cancelled != true {
		t.Error("escape did not cancel the prompt")
	}
}

// TestPromptStateChange_OverlayIsItsOwnKind verifies ShowOverlay rides the
// overlay state rather than the prompt state, and is reported as a distinct
// surface. A handler that cannot tell a question it must wait on from a
// dialog it merely displayed has no way to avoid declaring the turn finished.
func TestPromptStateChange_OverlayIsItsOwnKind(t *testing.T) {
	m, rec := modalTestModel(t)
	m.state = stateWorking

	respCh := make(chan app.OverlayResponse, 1)
	sendMsg(m, app.OverlayRequestEvent{
		Title:      "Results",
		Content:    "42 files",
		ResponseCh: respCh,
	})

	edges := rec.await(1)
	if len(edges) != 1 {
		t.Fatalf("got %d events (%+v), want the open edge", len(edges), edges)
	}
	if edges[0].kind != "overlay" {
		t.Errorf("kind = %q, want %q", edges[0].kind, "overlay")
	}
	if edges[0].state != "open" || edges[0].previous != "closed" {
		t.Errorf("open edge = %+v", edges[0])
	}

	sendMsg(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	edges = rec.await(2)
	if edges[1] != (promptEdge{kind: "overlay", promptTy: "overlay", state: "closed", previous: "open"}) {
		t.Errorf("close edge = %+v", edges[1])
	}
	if m.state != stateWorking {
		t.Errorf("state after dismissal = %v, want stateWorking", m.state)
	}
}

// TestPromptStateChange_NoEventForInvisiblePrompt guards the unknown prompt
// type path.
//
// A prompt type with no overlay behind it is cancelled in the same Update that
// set statePrompt, so the user never sees a modal. Announcing one anyway would
// leave an extension waiting for a "closed" edge that has no meaning — the
// exact stranded state this event is meant to prevent.
func TestPromptStateChange_NoEventForInvisiblePrompt(t *testing.T) {
	m, rec := modalTestModel(t)
	m.state = stateWorking

	respCh := make(chan app.PromptResponse, 1)
	sendMsg(m, app.PromptRequestEvent{
		PromptType: "hologram",
		Message:    "?",
		ResponseCh: respCh,
	})

	if m.state != stateWorking {
		t.Errorf("state = %v, want the pre-prompt state restored", m.state)
	}
	if got := (<-respCh).Cancelled; !got {
		t.Error("an unsupported prompt type must be cancelled, not left hanging")
	}

	// Give a wrongly-emitted event time to arrive before declaring it absent.
	if edges := rec.quiet(t, 50*time.Millisecond); len(edges) != 0 {
		t.Errorf("emitted %+v for a prompt that was never shown", edges)
	}
}

// TestPromptStateChange_RejectedRequestsAreSilent keeps concurrent requests
// from producing edges that describe no modal. A second prompt arriving while
// one is already up is refused, and an extension that sees an "open" for it
// would wait forever on a "closed" that belongs to the first prompt.
func TestPromptStateChange_RejectedRequestsAreSilent(t *testing.T) {
	m, rec := modalTestModel(t)

	first := make(chan app.PromptResponse, 1)
	sendMsg(m, app.PromptRequestEvent{
		PromptType: "input",
		Message:    "first",
		ResponseCh: first,
	})

	second := make(chan app.PromptResponse, 1)
	sendMsg(m, app.PromptRequestEvent{
		PromptType: "input",
		Message:    "second",
		ResponseCh: second,
	})
	overlayCh := make(chan app.OverlayResponse, 1)
	sendMsg(m, app.OverlayRequestEvent{Title: "nope", ResponseCh: overlayCh})

	if got := (<-second).Cancelled; !got {
		t.Error("the second prompt was not rejected")
	}
	if got := (<-overlayCh).Cancelled; !got {
		t.Error("an overlay was allowed on top of a prompt")
	}

	sendMsg(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	edges := rec.await(2)
	if len(edges) != 2 {
		t.Fatalf("got %d events (%+v), want only the first prompt's open and close", len(edges), edges)
	}
	if edges[0].state != "open" || edges[1].state != "closed" {
		t.Errorf("edges = %+v, want a matched open/close pair", edges)
	}
}

// TestPromptStateChange_NotFiredByMessageInspector guards the shared-state
// hazard. The inspector reuses stateOverlay without any extension having
// raised it, so firing an open/close pair for it would report a modal the
// user never asked for — and would fight the real event stream for the same
// state.
func TestPromptStateChange_NotFiredByMessageInspector(t *testing.T) {
	m, rec := modalTestModel(t)
	m.printSystemMessage("first")
	m.printSystemMessage("second")

	// Opens the overlay directly, exactly as inspectSelectedMessage does.
	m.preOverlayState = stateMessageNav
	m.state = stateOverlay
	m.overlayResponseCh = nil
	m.overlay = newOverlayDialog("Tool Result", "body", false, "", "", 60, 16, "center", nil, 80, 24)

	rec.quiet(t, 50*time.Millisecond)
	m.resolveOverlay(app.OverlayResponse{Cancelled: true})

	if m.state != stateMessageNav {
		t.Errorf("state = %v, want stateMessageNav restored", m.state)
	}
	if edges := rec.quiet(t, 50*time.Millisecond); len(edges) != 0 {
		t.Errorf("emitted %+v for a UI-owned overlay", edges)
	}
}

// TestPromptStateChange_NilEmitterIsSafe covers the no-extensions case: every
// emit* option is nil then, and a stray notification must not panic a TUI that
// has no extension runtime at all.
func TestPromptStateChange_NilEmitterIsSafe(t *testing.T) {
	m, _, _ := newTestAppModel(&stubAppController{})
	m.frames = frameClock{}
	m.extEvents = newExtEventDispatcher()

	respCh := make(chan app.PromptResponse, 1)
	sendMsg(m, app.PromptRequestEvent{PromptType: "select", ResponseCh: respCh})
	sendMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if got := <-respCh; got.Cancelled {
		t.Error("select prompt was cancelled instead of answered")
	}
}
