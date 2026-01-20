// Package router provides screen navigation with explicit data flow.
//
// Unlike the FSM package, router uses explicit input/output types for each screen
// and a centralized transition function for all routing logic. This makes data
// flow traceable and avoids hidden global state.
//
// # Basic Usage
//
//	// Define screen identifiers as typed constants
//	const (
//	    ScreenList Screen = iota
//	    ScreenDetail
//	)
//
//	// Define input/output types for each screen
//	type ListInput struct {
//	    Items  []Item
//	    Resume *ListResume // nil if fresh, populated if returning
//	}
//
//	type ListResult struct {
//	    Action   ListAction
//	    Selected *Item
//	    Resume   *ListResume // position state for back navigation
//	}
//
//	// Create and configure router
//	r := router.New()
//
//	r.Register(ScreenList, func(input any) (any, error) {
//	    in := input.(ListInput)
//	    return listScreen(in), nil
//	})
//
//	r.Register(ScreenDetail, func(input any) (any, error) {
//	    in := input.(DetailInput)
//	    return detailScreen(in), nil
//	})
//
//	r.OnTransition(func(from router.Screen, result any, stack *router.Stack) (router.Screen, any) {
//	    switch from {
//	    case ScreenList:
//	        r := result.(ListResult)
//	        switch r.Action {
//	        case ActionSelected:
//	            // Push current state for back navigation
//	            stack.Push(from, input, r.Resume)
//	            return ScreenDetail, DetailInput{Item: r.Selected}
//	        case ActionBack:
//	            if stack.IsEmpty() {
//	                return router.ScreenExit, nil
//	            }
//	            entry := stack.Pop()
//	            // Restore with resume state
//	            in := entry.Input.(ListInput)
//	            in.Resume = entry.Resume.(*ListResume)
//	            return entry.Screen, in
//	        }
//	    case ScreenDetail:
//	        // ...
//	    }
//	    return router.ScreenExit, nil
//	})
//
//	r.Run(ScreenList, ListInput{Items: items})
//
// # Resume State
//
// Screens can return resume state (like scroll position) that gets stored
// on the stack when navigating forward. When navigating back, this state
// is passed back to the screen via its input, allowing it to restore position.
//
// The Resume field should be nil for stateless screens (dialogs, confirmations).
package router
