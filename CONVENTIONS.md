# Gabagool Naming Conventions

This document defines the naming conventions for component options, callbacks, and other public API elements.

## Options Struct Naming

Options structs should be named `<Component>Options`:
- `ListOptions`
- `DetailScreenOptions`
- `KeyboardOptions`

## Boolean Fields

### Visibility toggles: `Show<Element>`

Use for options that control whether something is displayed:
- `ShowHeader`
- `ShowFooter`
- `ShowScrollbar`
- `ShowProgressBar`
- `ShowThemeBackground`

### Feature enablement: `Allow<Feature>`

Use for options that enable or permit a behavior:
- `AllowReorder`
- `AllowMultiSelect`
- `AllowWrapAround`
- `AllowAction`

### Disabling defaults: `Disable<Feature>`

Use sparingly, only when the default is "enabled" and you need to turn it off:
- `DisableBackButton`

Prefer positive naming (`Allow*`) over negative (`Disable*`) when possible.

### Initial state: `Initial<State>`

Use for options that set the starting state:
- `InitialSelectedIndex`
- `InitialMultiSelectMode`
- `InitialScrollPosition`

### Modifiers: `Use<Variant>`

Use for options that select a variant of default behavior:
- `UseSmallTitle`
- `UseCompactLayout`

## Callbacks

### Event callbacks: `On<Event>`

All callbacks should be prefixed with `On`:
- `OnSelect`
- `OnReorder`
- `OnCancel`
- `OnUpdate`
- `OnComplete`

The event name should be a past-tense verb or noun describing what happened:
- `OnSelect` (not `OnSelecting`)
- `OnComplete` (not `OnCompleting`)
- `OnTextChanged` (for text input changes)

## Color Fields

### Pattern: `<Element>Color`

Color fields should use the element name followed by `Color`:
- `TitleColor`
- `BackgroundColor`
- `TextColor`
- `FooterColor`
- `MessageColor`

Avoid redundant words:
- `FooterColor` (not `FooterTextColor`)
- `MessageColor` (not `MessageTextColor`)

## Dimension Fields

### Constraints: `Max<Dimension>` / `Min<Dimension>`

Use for maximum/minimum constraints:
- `MaxVisibleItems`
- `MaxImageHeight`
- `MaxImageWidth`
- `MinWidth`

### Target values: `<Element><Dimension>`

Use for target/desired dimensions without constraint semantics:
- `ImageWidth`
- `ImageHeight`
- `ItemSpacing`
- `Margins`

## Duration Fields

### Pattern: `<Action>Delay` or `<Action>Duration`

- `InputDelay`
- `ScrollPauseDuration`
- `AnimationDuration`

Use `Delay` for waiting before an action, `Duration` for how long something takes.

## Button Configuration

### Pattern: `<Purpose>Button`

Button assignments should describe their purpose:
- `ActionButton` (primary action)
- `SecondaryActionButton` (secondary action)
- `ConfirmButton`
- `CancelButton`
- `BackButton`

## Grouped Options

For components with many options, group related options into sub-structs:

```go
type ListOptions struct {
    // Required
    Items []MenuItem

    // Grouped options
    Appearance ListAppearance
    Behavior   ListBehavior

    // Callbacks
    OnSelect  func(index int, item *MenuItem)
    OnReorder func(from, to int)
}

type ListAppearance struct {
    ShowHeader    bool
    ShowFooter    bool
    UseSmallTitle bool
    TitleColor    sdl.Color
    // ...
}

type ListBehavior struct {
    AllowReorder     bool
    AllowMultiSelect bool
    AllowWrapAround  bool
    // ...
}
```

## Result Struct Naming

Result structs should be named `<Component>Result`:
- `ListResult`
- `DetailScreenResult`
- `KeyboardResult`

## Action Enums

Screen-specific action enums should be named `<Screen>Action`:

```go
type ListAction int

const (
    ListActionSelected ListAction = iota
    ListActionBack
    ListActionSettings
)
```

## Resume State

Resume structs for position restoration should be named `<Screen>Resume`:

```go
type ListResume struct {
    ScrollPosition int
    SelectedIndex  int
}
```

Resume fields in results and inputs should be named `Resume`:

```go
type ListResult struct {
    Action   ListAction
    Selected *Item
    Resume   *ListResume  // nil for stateless
}

type ListInput struct {
    Items  []Item
    Resume *ListResume  // nil if fresh, populated if returning
}
```
