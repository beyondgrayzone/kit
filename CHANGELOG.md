# Changelog (Till 0.88.0)
- Merge v0.86.0 custom changes into v0.88.0
  - Auth refactor with credential manager and device flow for Copilot
  - Extension system: package relocation internal/extensions → extensions/
  - New UI features: Ctrl+Arrow word navigation, crush-style text selection, End key support
  - Compaction improvements: auto-compact, adaptive budgets, retry on overflow
  - Subagent session linking with parent-child relationships and resume support
  - SDK parity: WithProviderWire, sealed SDK types, deduplication of auth/UI/extension paths
  - Various bug fixes: memory leaks, deadlocks, hot-path rendering, subagent nil handle
  - New subagent type display in StreamComponent spinner
  - Agent name sanitization and validation with compiled regexp
  - Bot-review polling loop (/resolve-reviews)

# Changelog (Till 0.86.0)
- Merge all upstream 0.86.0 changes

# Changelog (Till 0.82.2)
- Fix resume token filling 
- Fix Reasoning Delta check
- RawInput attempt
- Expose onStepStart and onStepFinish
- Expose Agent Error

# Changelog (Till 0.82.1)
- Git merge only 

# Changelog (Till 0.80.0)
This changelog summarizes the changes found in the provided diff, categorized by
functional area.

⚠️ Breaking Changes & Refactors

Extension System

  - Expose Kit's thinking API to track token usage more accurately

  - Extension Package Relocation: Moved internal/extensions to the top-level
    extensions/ package. This allows external packages and unit tests to import
    the extension API directly.
      - Action Required: Update import paths from
        github.com/mark3labs/kit/internal/extensions to
        github.com/mark3labs/kit/extensions.
  - Header/Footer API Update: SetHeader and SetFooter now require a
    HeaderFooterConfig with a unique ID. RemoveHeader and RemoveFooter now
    require an id string parameter.

Extension System Improvements

  - Multiple Headers/Footers: Extensions can now render multiple headers and
    footers simultaneously.
      - Added ID field to HeaderFooterConfig for tracking.
      - Added Priority field (lower values render further left).
      - The TUI now joins multiple widgets horizontally using
        lipgloss.JoinHorizontal with gap spacing.
  - Explicit Extension Loading: Added LoadExplicitExtensions to allow loading
    specific extension paths even when automatic discovery is disabled (via
    --no-extensions).
  - Improved Reloading: Fixed a bug in Runner.Reload() to properly re-initialize
    per-extension mutexes.
  - Custom Events: Added a "reload" custom event emitted to extensions when they
    are reloaded via the UI.

UI & Navigation

  - Input Word Navigation: Added Ctrl+Left and Ctrl+Right keybindings to
    navigate by word boundaries in the input area.
  - Input Mouse Support: Implemented "Crush-style" text selection within the
    input component:
      - Single-click to position cursor/start drag.
      - Double-click to select word.
      - Triple-click to select entire line.
      - Automatic copying of selected text to clipboard on mouse release.
  - End Key Support: Added End key binding to force-scroll to the bottom of the
    scrollback buffer in all UI states.
  - Clean Startup: Commented out the ASCII logo/banner to provide more vertical
    real estate upon startup.
  - Layout Fixes: Fixed a bug in setCursorOffset that caused incorrect cursor
    positioning during multi-line navigation.

Agent & Tool Improvements

  - Enhanced Stop Reason Warnings: Added specific warning messages for
    "suspicious" agent stops:
      - Warns if the agent stops with an empty response (often indicating
        MaxSteps reached).
      - Warns if the agent stops while tool calls are still pending.
  - Tool Documentation: Added CODE_WRITING_TOOL.md providing a detailed
    implementation summary of the write and edit tools, including their fuzzy
    matching logic and UI rendering.

Testing & Internal

  - Test Isolation: Improved TestLoadSkills_ProjectLocal by isolating it from
    the user's real global skills using a temporary XDG_CONFIG_HOME.
  - Mock Context Updates: Updated MockContext in the test package to support the
    new ID-based multi-header/footer API.
  - New Loader Tests: Added unit tests for explicit extension loading and
    discovery skipping.

Example Extensions

  - Updated header-footer-demo.go, kit-kit.go, and minimal.go to implement the
    new ID-based header/footer API.

# Changelog (Till 0.70.2)

# Merged upstream 0.70.2 and fix conflicts

## Extension System Improvements

### Multiple Headers/Footers Support
- Extensions can now set multiple headers and footers that display side-by-side
- `HeaderFooterConfig` now requires a unique `ID` field (e.g., `"my-ext:status-header"`)
- Added `Priority` field to control ordering (lower values render further left)
- `SetHeader`/`SetFooter` now store entries in maps instead of single pointers
- `RemoveHeader`/`RemoveFooter` now take an `id string` parameter to remove specific entries
- TUI renders multiple headers/footers horizontally using `lipgloss.JoinHorizontal` with gap spacing
- Added `WithAutoWidth()` rendering option for block_renderer to support side-by-side layout
- Updated `renderHeaderFooterSlot()` to handle multiple widgets with proper height tracking

### Extension Package Relocation
- Moved `internal/extensions` package to top-level `extensions` package
- Enables external packages and unit tests to import and use the extensions API directly
- Updated all import paths across the codebase (`github.com/mark3labs/kit/extensions`)
- Updated example extensions, tests, and documentation to use new import path

### Extension Runner Fixes
- Fixed extension mutex re-initialization in `Runner.Reload()` to properly reinitialize per-extension mutexes
- Added `RemoveHeaderIDs` and `RemoveFooterIDs` tracking to mock context for better test assertions

## Input Navigation

### Word Navigation with Ctrl+Arrow
- Added Ctrl+Left and Ctrl+Right keybindings to navigate word boundaries in the input component
- Implemented `moveCursorWordLeft()` and `moveCursorWordRight()` methods
- Added `getCursorOffset()` and `setCursorOffset()` helper methods for cursor position management
- Word characters defined as alphanumeric and underscore

### Multi-line Cursor Navigation Fix
- Fixed `setCursorOffset` in input component for proper multi-line cursor navigation
- Replaced buggy implementation with correct `CursorUp()`/`CursorDown()` approach
- Cursor now correctly moves to target line and column across multiple lines

### Improved Warning Messages
- Added warning messages for suspicious agent stop reasons:
  - `FinishReasonStop` with empty response (possible MaxSteps reached)
  - `FinishReasonToolCalls` (agent stopped while tool calls pending)
- Max tokens warning now only triggers on `FinishReasonLength`
- Updated tests to cover both normal and suspicious stop scenarios

## UI Changes

### Banner Removal
- Commented out ASCII logo/banner code
- Startup is now more cleaner with bigger real estate

### End Key Support
- Added End key binding to force-scroll to bottom in all states
- Improves navigation in scrollback buffer

### Test Isolation
- `TestLoadSkills_ProjectLocal` now isolates from real global skills by pointing `XDG_CONFIG_HOME` to a temp directory
- Prevents tests from picking up real user skill configurations

### Test Package Updates
- Updated `pkg/extensions/test` assertions to work with new multi-header/footer API
- `AssertHeaderSet` and `AssertFooterSet` now check `len(GetHeaders()) == 0` instead of `GetHeader() == nil`
- Mock context updated to track headers/footers as maps with proper ID-based removal

## Example Extensions

Updated all example extensions to use new API:
- `header-footer-demo.go` - Added IDs (`"hf-demo:header"`, `"hf-demo:footer"`) and priorities
- `kit-kit.go` - Added ID (`"kit-kit:footer"`) to footer
- `minimal.go` - Added ID (`"minimal:footer"`) to footer
- All `RemoveHeader`/`RemoveFooter` calls now pass the appropriate ID
