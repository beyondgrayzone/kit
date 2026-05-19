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
