package router

import "fmt"

// Screen is a type-safe identifier for screens.
// Applications should define their own Screen constants using iota.
//
// Example:
//
//	const (
//	    ScreenMain Screen = iota
//	    ScreenSettings
//	    ScreenDetail
//	)
type Screen int

// ScreenFunc is a function that runs a screen.
// It takes an input and returns a result.
// The input and result types are screen-specific.
type ScreenFunc func(input any) (result any, err error)

// TransitionFunc is called after each screen completes to determine the next screen.
// It receives the screen that just completed, its result, and the navigation stack.
// It returns the next screen to navigate to and its input.
//
// Return (screen, input) to navigate to a new screen.
// Return stack.Pop() values to go back.
// Return (-1, nil) to exit the router.
type TransitionFunc func(from Screen, result any, stack *Stack) (next Screen, input any)

// ScreenExit is a special Screen value that signals the router to exit.
const ScreenExit Screen = -1

// Router manages screen navigation with explicit data flow.
// Screens are registered with their functions, and a single transition
// function handles all routing logic in one place.
type Router struct {
	screens    map[Screen]ScreenFunc
	transition TransitionFunc
	stack      *Stack
}

// New creates a new Router.
func New() *Router {
	return &Router{
		screens: make(map[Screen]ScreenFunc),
		stack:   NewStack(),
	}
}

// Register adds a screen to the router.
// The screen function will be called when navigating to this screen.
func (r *Router) Register(screen Screen, fn ScreenFunc) *Router {
	r.screens[screen] = fn
	return r
}

// OnTransition sets the transition function that determines navigation flow.
// This function is called after each screen completes.
func (r *Router) OnTransition(fn TransitionFunc) *Router {
	r.transition = fn
	return r
}

// Run starts the router at the given screen with the given input.
// It continues running until the transition function returns ScreenExit
// or an error occurs.
func (r *Router) Run(start Screen, input any) error {
	if r.transition == nil {
		return fmt.Errorf("router: no transition function set")
	}

	current := start
	currentInput := input

	for {
		// Get the screen function
		fn, ok := r.screens[current]
		if !ok {
			return fmt.Errorf("router: screen %d not registered", current)
		}

		// Run the screen
		result, err := fn(currentInput)
		if err != nil {
			return fmt.Errorf("router: screen %d error: %w", current, err)
		}

		// Determine next screen
		next, nextInput := r.transition(current, result, r.stack)

		// Check for exit
		if next == ScreenExit {
			return nil
		}

		// Move to next screen
		current = next
		currentInput = nextInput
	}
}

// Stack returns the navigation stack for use in transition functions.
// This allows the transition function to push/pop for back navigation.
func (r *Router) Stack() *Stack {
	return r.stack
}
