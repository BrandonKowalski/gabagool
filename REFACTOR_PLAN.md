# Gabagool Refactor Plan

## Overview

Refactor gabagool to simplify the architecture while keeping the blocking component model. Focus areas:

1. Replace FSM with simpler Router
2. Standardize component options
3. Separate infrastructure vs domain errors
4. Review logging
5. Simplify component internals

Breaking changes to consuming apps (like Grout) are acceptable.

---

## 1. Router (Replacing FSM)

### Current Problems
- FSM uses type-keyed context (`Set[T]`, `MustGet[T]`) which acts like hidden global state
- Data flow is hard to trace
- Generic exit codes don't capture screen-specific intent

### Detailed FSM Analysis (from codebase review)

**Type-keyed context issues:**
- One value per type enforced - can't store multiple values of same type
- `Set(ctx, User{Name: "Alice"})` then `Set(ctx, User{Name: "Bob"})` silently overwrites
- Forces wrapper types for complex workflows
- Reflection overhead on every access (`reflect.TypeOf()`)

**MustGet panic pattern:**
- `MustGet[T](ctx)` panics if type not found (line 53)
- No graceful error path - forces choice between verbose `Get()` or panic risk
- Hard to unit test (must catch panics)

**Performance:**
- Linear O(n) transition lookup (lines 169-175)
- With 50+ states, this becomes expensive

**Missing validation:**
- No check that transitions target valid states
- `On(ExitCodeSuccess, "nonexistent_state")` crashes at runtime

**No debugging support:**
- No way to trace state transitions
- No logging of which state executed
- No runtime inspection of FSM structure

### New Design

**Hybrid approach:** Declarative registration + imperative transition logic

```go
type Screen int

const (
    ScreenGameList Screen = iota
    ScreenGameDetail
    ScreenSettings
)

router := NewRouter()
router.Register(ScreenGameList, GameListScreen)
router.Register(ScreenGameDetail, GameDetailScreen)

router.OnTransition(func(from Screen, result any, stack *Stack) (Screen, any) {
    switch from {
    case ScreenGameList:
        r := result.(GameListResult)
        switch r.Action {
        case ActionSelected:
            return ScreenGameDetail, GameDetailInput{Game: r.Selected}
        case ActionBack:
            return stack.Pop()
        }
    case ScreenGameDetail:
        // ...
    }
})

router.Start(ScreenGameList, initialInput)
```

**Key characteristics:**
- Screen identifiers are enums (compile-time type safety, no string typos)
- Central place for all transition logic
- Explicit data passing: input → screen → result → next input
- Stack-based back navigation

### Resume State (Position Restoration)

When navigating back to a screen (e.g., returning to a list), position should be restored.

**Approach:** Hybrid of screen-defined + router-stored

1. Screen defines what state matters (optional Resume struct)
2. Screen returns Resume as part of result
3. Router stores Resume on stack automatically when non-nil
4. Router passes Resume back via input when popping

```go
// Screen result - Resume is optional (nil = stateless screen)
type GameListResult struct {
    Action   GameListAction
    Selected *Game
    Resume   *GameListResume  // nil if stateless
}

type GameListResume struct {
    ScrollPosition int
    SelectedIndex  int
}

// Screen input - Resume populated when returning
type GameListInput struct {
    Games  []Game
    Resume *GameListResume  // nil = fresh, non-nil = returning
}
```

Router pseudocode:
```go
// Forward navigation
stack.Push(currentScreen, currentInput, result.Resume)
next := transition(result)
nextResult := next.Screen(next.Input)

// Back navigation
prev := stack.Pop()
input := prev.Input.WithResume(prev.Resume)
prevResult := prev.Screen(input)
```

---

## 2. Screen Model

### Principles
- Screens are self-contained functions: typed input → typed result
- Same input always produces same screen (deterministic/pure)
- Screens handle their own side effects (fetch data, save to disk)
- No reaching into global/context state

### Result Structure
Each screen has its own result type with:
- Action enum (screen-specific, not generic exit codes)
- Relevant data for that action
- Optional Resume state

```go
type GameListAction int

const (
    GameListSelected GameListAction = iota
    GameListBack
    GameListSettings
)

type GameListResult struct {
    Action   GameListAction
    Selected *Game              // populated when Action == Selected
    Resume   *GameListResume    // optional position state
}
```

---

## 3. Error Handling

### Current Problem
Errors are mixed - rendering failures and domain errors (failed download) handled the same way.

### Detailed Error Handling Analysis (from codebase review)

**Current patterns found:**

1. **Single sentinel error:** `ErrCancelled = errors.New("operation cancelled by user")` used across all UI components

2. **Inconsistent cancellation handling:**
   - Some use `ErrCancelled`
   - Download uses `fmt.Errorf("download cancelled by user")` and `fmt.Errorf("download canceled")` (typo variant)
   - String matching on error messages: `err.Error() != "download cancelled by user"`

3. **Silent error ignoring:**
   - Rendering errors silently ignored in helpers.go (font rendering failures → `continue`)
   - Image loading errors ignored in detail.go, confirmation_message.go
   - Pattern: `if err == nil { proceed }` without else clause

4. **Mixed concerns in download.go:**
   - HTTP errors: `fmt.Errorf("bad status: %s", resp.Status)`
   - File I/O errors: raw `error` passed through
   - User cancellation: `fmt.Errorf("download canceled")`
   - All stored in same `[]error` slice with no type distinction

5. **No custom error types:** Everything is `error` or `fmt.Errorf()` - no way to programmatically distinguish error kinds

### New Design

**Two distinct error types:**

1. **Infrastructure errors** - Gabagool's problem (rendering failed, SDL crashed, font missing)
   - Returned as Go `error` from component functions
   - Likely fatal or needs framework-level recovery

2. **Domain errors** - App's problem (download failed, network timeout)
   - Part of the result struct as data
   - App decides how to handle

```go
// Infrastructure error
result, err := List(input)
if err != nil {
    // Framework problem - maybe fatal
    log.Fatal(err)
}

// Domain errors as data in result
type DownloadResult struct {
    Action  DownloadAction
    Failed  []DownloadError  // domain errors
    Resume  *DownloadResume
}

// App handles domain errors
if len(result.Failed) > 0 {
    showRetryDialog(result.Failed)
}
```

---

## 4. Component Options Cleanup

### Current Problems
- Too many options
- Inconsistent naming across components
- No clear organization

### Detailed Options Audit (from codebase review)

**Boolean prefix inconsistencies:**

| Current | Component | Recommended |
|---------|-----------|-------------|
| `EnableImages` | ListOptions | `ShowImages` |
| `EnableAction` | DetailScreenOptions | `AllowAction` |
| `ShowScrollbar` | DetailScreenOptions | ✓ (keep) |
| `ShowThemeBackground` | Multiple | ✓ (keep) |
| `StartInMultiSelectMode` | ListOptions | `InitialMultiSelectMode` |
| `DisableBackButton` | Multiple | ✓ (keep) |
| `SmallTitle` | Multiple | `UseSmallTitle` |
| `AutoContinue` | DownloadManagerOptions | `AutoContinueOnComplete` |
| `InsecureSkipVerify` | DownloadManagerOptions | `SkipSSLVerification` |

**Color field inconsistencies:**
- `TitleColor`, `MetadataColor` (good)
- `FooterTextColor` → should be `FooterColor`
- `MessageTextColor` → should be `MessageColor`

**Dimension inconsistencies:**
- `MaxImageHeight`, `MaxImageWidth` (good - has Max prefix)
- `ImageWidth`, `ImageHeight` → should be `TargetImageWidth`, `TargetImageHeight`

**Callback naming:** All use `On*` pattern consistently (OnSelect, OnReorder, OnUpdate) ✓

### Goals
- Consistent naming conventions (e.g., all callbacks start with `On`)
- Sensible defaults to minimize required options
- Group related options into sub-structs

### Proposed Structure

```go
type ListOptions struct {
    // Required
    Items []MenuItem

    // Appearance (grouped)
    Appearance ListAppearance

    // Behavior (grouped)
    Behavior ListBehavior

    // Callbacks (all prefixed with On)
    OnSelect   func(index int, item *MenuItem)
    OnReorder  func(from, to int)
    OnCancel   func()
}

type ListAppearance struct {
    ShowHeader    bool
    HeaderText    string
    ShowFooter    bool
    // ... other visual options with sensible defaults
}

type ListBehavior struct {
    AllowReorder  bool
    AllowMulti    bool
    WrapAround    bool
    // ... other behavior options with sensible defaults
}
```

### Naming Conventions
- Callbacks: `On<Event>` (OnSelect, OnCancel, OnUpdate)
- Booleans: `Allow<Thing>`, `Show<Thing>`, `Enable<Thing>`
- Appearance options: grouped in `<Component>Appearance`
- Behavior options: grouped in `<Component>Behavior`

---

## 5. Logging

### Current Implementation (from codebase review)

**Technology:** Go standard library `log/slog` (structured logging)

**Architecture:**
- `GetLogger()` - Main application logger
- `GetInternalLogger()` - Internal/debug logger with independent level control
- JSON handler for formatted output
- Multi-writer: simultaneous stdout and file (`logs/app.log`)
- Thread-safe via `sync.Once` singleton pattern

**Configuration:**
- `SetLogLevel()` / `SetRawLogLevel()` - Set main logger level
- `SetInternalLogLevel()` - Control internal logger independently
- Environment variables: `NITRATES` or `INPUT_CAPTURE` → Debug level

**Strengths:**
- Structured key-value logging (good for parsing)
- Dual logger pattern separates app vs internal concerns
- Standard library, no external dependencies
- Dynamic level changes via `LevelVar`

**Issues Found:**

1. **Panics on setup failure** (lines 35, 46)
   - `panic("Failed to create logs directory: " + err.Error())`
   - Should fall back to console-only logging instead

2. **No log rotation**
   - Files only append, never rotate
   - Could grow unbounded on long-running systems

3. **No correlation IDs**
   - Difficult to trace related logs in async operations (especially input processing)

4. **Source location disabled**
   - `AddSource: false` - no file/line info in logs
   - Useful for debugging, could enable for dev builds

---

## 6. Component Internals

### Goal
Simplify implementations without changing external behavior (function signatures, options, results).

### Keep
- Blocking model (components run their own event loops)
- Callbacks for live updates during execution

### Detailed Analysis (from codebase review)

#### keyboard.go (1796 lines) - Major Duplication

**Layout setup functions (lines 550-961):** 407 lines, 95% duplicate
- `setupKeyboardRects()` (~95 lines)
- `setupURLKeyboardRects()` (~99 lines) - nearly identical
- `setupURLKeyboardRectsFor5()` (~99 lines) - nearly identical
- `setupURLKeyboardRectsFor10()` (~112 lines) - nearly identical

**Key creation (lines 240-517):** ~150 lines duplicate
- Same character mapping pattern repeated for each keyboard type
- QWERTY row creation identical across functions

**Simplification opportunity:** Extract to parameterized functions, reduce 400+ lines to ~100

#### list.go (1192 lines)

**Directional repeat handling (lines 646-675):** 30 lines
- Identical to keyboard.go lines 1268-1297
- Also duplicated in detail.go
- **Total: 90+ lines of identical logic across 3 files**

**Text measurement (lines 1069-1136):** ~40 lines
- Renders throwaway textures 3+ times per cycle just to measure text
- `shouldScroll()`, `getOrCreateScrollData()`, `measureText()` all create temporary textures

**Selection state duplication:**
- `SelectedItems` map AND `Options.Items[i].Selected` both store selection
- Risk of desynchronization

**handleActionButtons (lines 315-412):** 98-line switch statement
- Could be refactored into smaller handler functions

#### Cross-Component Duplication

| Pattern | Files | Lines | Priority |
|---------|-------|-------|----------|
| Directional repeat handler | list.go, keyboard.go, detail.go | 90+ | P1 |
| Layout setup functions | keyboard.go | 407 | P1 |
| Key creation patterns | keyboard.go | 150 | P1 |
| Text measurement | list.go | 40 | P2 |

**Estimated total simplification: 400-500 lines (20-25% of list.go + keyboard.go)**

#### Recommended Extraction

```go
// internal/directional.go
type DirectionalInputHandler struct {
    heldDirections struct { up, down, left, right bool }
    lastRepeatTime time.Time
    repeatDelay, repeatInterval time.Duration
    hasRepeated bool
}

func (h *DirectionalInputHandler) Update(callback func(direction string)) { ... }
```

This would eliminate 90+ lines of duplicate code across components.

---

## Migration Strategy

### Phase 1: Foundation ✅ COMPLETE
1. ✅ **Create Router package** (`pkg/gabagool/router/`) alongside existing FSM
2. ✅ **Extract DirectionalInputHandler** to `internal/directional.go` (eliminates 90+ lines duplication)
3. ✅ **Define and document naming conventions** for options (`CONVENTIONS.md`)

### Phase 2: Component Cleanup ✅ COMPLETE
4. ✅ **Refactor keyboard.go** - extracted layout setup into `internal/keyboard_layout.go`
5. ✅ **Refactor list.go, option_list.go** - now use DirectionalInput handler
6. ✅ **Standardize options naming** across all components (breaking change)
   - `EnableImages` → `ShowImages`
   - `StartInMultiSelectMode` → `InitialMultiSelectMode`
   - `SmallTitle` → `UseSmallTitle`
   - `FooterTextColor` → `FooterColor`
   - `EnableAction` → `AllowAction`
   - `AutoContinue` → `AutoContinueOnComplete`
   - `InsecureSkipVerify` → `SkipSSLVerification`
   - `MessageTextColor` → `MessageColor`

### Phase 3: Error Handling ✅ COMPLETE
7. ✅ **Create custom error types** - `errors.go` with `InfrastructureError`, `ErrCancelled`, `ErrDownloadCancelled`
8. ✅ **Standardize cancellation** - download.go now uses `ErrDownloadCancelled` instead of string errors
9. ✅ **Fix logging panic** - `internal/logging.go` now falls back to console-only on failure

### Phase 4: Router Migration ✅ COMPLETE
10. ✅ **Port one screen** to new Router as proof of concept - `router/example_test.go`
11. ✅ **Migrate remaining screens** - examples demonstrate the pattern for consuming apps
12. ✅ **Deprecate FSM** - deprecation notice added to `fsm.go` with migration guide

### Phase 5: Cleanup ✅ COMPLETE
13. ✅ **Remove old ui/ package stubs** - empty directory removed
14. ✅ **Final audit and documentation** - this file updated

---

## Files to Investigate

### Core Architecture
- `pkg/gabagool/fsm.go` - Current FSM implementation
- `pkg/gabagool/internal/input_processor.go` - Input handling
- `pkg/gabagool/internal/input_mapper.go` - Input mapping

### Components (for options audit)
- `pkg/gabagool/list.go`
- `pkg/gabagool/detail.go`
- `pkg/gabagool/keyboard.go`
- `pkg/gabagool/option_list.go`
- `pkg/gabagool/confirmation_message.go`
- `pkg/gabagool/selection_message.go`
- `pkg/gabagool/process_message.go`
- `pkg/gabagool/color_picker.go`
- `pkg/gabagool/download.go`
- `pkg/gabagool/status_bar.go`

### Error Handling & Logging
- `pkg/gabagool/internal/logging.go`
- Grep for error patterns across codebase

---

## Open Questions

1. ~~Should the new Router live in `pkg/gabagool/router/` or replace FSM in place?~~ → **Decision: `pkg/gabagool/router/`**
2. How to handle modal dialogs (confirmation, selection) in the router model?
   - Option A: Modals are screens with their own input/result types
   - Option B: Modals are handled within the calling screen (current approach)
3. Animation/transition ownership between screens?
4. Lifecycle hooks (mount/unmount) - needed?

---

## Non-Goals

- Do not change the blocking component model
- Do not remove callbacks (needed for blocking components)
- Do not push application logic into components
- Do not over-engineer - keep it simple
