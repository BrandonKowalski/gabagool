package gabagool

// Deprecated: The FSM (finite state machine) is deprecated and will be removed in a future version.
// Use the router package instead, which provides explicit data flow and type-safe screen transitions.
//
// Migration example:
//
// Before (FSM):
//
//	fsm := gabagool.NewFSM()
//	gabagool.AddState(fsm, "list", func(ctx *gabagool.Context) (ListResult, gabagool.ExitCode) {
//	    // ...
//	})
//	fsm.Start("list").Run()
//
// After (Router):
//
//	r := router.New()
//	r.Register(ScreenList, func(input any) (any, error) {
//	    return listScreen(input.(ListInput)), nil
//	})
//	r.OnTransition(func(from router.Screen, result any, stack *router.Stack) (router.Screen, any) {
//	    // centralized transition logic
//	})
//	r.Run(ScreenList, ListInput{...})
//
// See the router package documentation for complete examples.

import (
	"fmt"
	"reflect"
)

// ExitCode represents the result of a screen draw operation
type ExitCode int

const (
	ExitCodeSuccess ExitCode = 0
	ExitCodeBack    ExitCode = 1
	ExitCodeCancel  ExitCode = 2
	ExitCodeQuit    ExitCode = 3
	ExitCodeAction  ExitCode = 4
	ExitCodeError   ExitCode = -1
	// Custom exit codes can start from 100
)

type StateName string

// Context holds shared state between screens during FSM execution
type Context struct {
	data map[reflect.Type]any
}

// NewContext creates a new context
func NewContext() *Context {
	return &Context{data: make(map[reflect.Type]any)}
}

// Set stores a value in the context by its type
func Set[T any](c *Context, value T) {
	c.data[reflect.TypeOf((*T)(nil)).Elem()] = value
}

// Get retrieves a typed value from the context
func Get[T any](c *Context) (T, bool) {
	val, ok := c.data[reflect.TypeOf((*T)(nil)).Elem()]
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}

// MustGet retrieves a typed value or panics
func MustGet[T any](c *Context) T {
	val, ok := Get[T](c)
	if !ok {
		panic(fmt.Sprintf("type %T not found in context", *new(T)))
	}
	return val
}

// node represents an executable state in the FSM
type node interface {
	execute(ctx *Context) (ExitCode, error)
	name() StateName
}

// stateNode wraps a function that returns a typed value
type stateNode[T any] struct {
	nodeName StateName
	fn       func(*Context) (T, ExitCode)
}

func (n *stateNode[T]) name() StateName {
	return n.nodeName
}

func (n *stateNode[T]) execute(ctx *Context) (ExitCode, error) {
	value, exitCode := n.fn(ctx)
	Set(ctx, value) // Auto-store by type
	return exitCode, nil
}

// actionNode wraps a simple function that only returns an exit code
type actionNode struct {
	nodeName StateName
	fn       func(*Context) ExitCode
}

func (n *actionNode) name() StateName {
	return n.nodeName
}

func (n *actionNode) execute(ctx *Context) (ExitCode, error) {
	exitCode := n.fn(ctx)
	return exitCode, nil
}

// transition defines a state transition
type transition struct {
	from StateName
	code ExitCode
	to   StateName
	hook func(*Context) error
}

// FSM represents the finite state machine
type FSM struct {
	nodes       map[StateName]node
	transitions []transition
	initialNode StateName
	ctx         *Context
}

// NewFSM creates a new FSM
func NewFSM() *FSM {
	return &FSM{
		nodes:       make(map[StateName]node),
		transitions: []transition{},
		ctx:         NewContext(),
	}
}

// AddState adds a state that returns a typed value (auto-stored in context by type)
func AddState[T any](fsm *FSM, name StateName, fn func(*Context) (T, ExitCode)) *StateBuilder {
	fsm.nodes[name] = &stateNode[T]{
		nodeName: name,
		fn:       fn,
	}
	return &StateBuilder{fsm: fsm, state: name}
}

// AddAction adds a simple action that only returns an exit code
func AddAction(fsm *FSM, name StateName, fn func(*Context) ExitCode) *StateBuilder {
	fsm.nodes[name] = &actionNode{
		nodeName: name,
		fn:       fn,
	}
	return &StateBuilder{fsm: fsm, state: name}
}

// Start sets the initial state
func (f *FSM) Start(name StateName) *FSM {
	f.initialNode = name
	return f
}

// Context returns the FSM context
func (f *FSM) Context() *Context {
	return f.ctx
}

// Run executes the FSM
func (f *FSM) Run() error {
	if f.initialNode == "" {
		return fmt.Errorf("no initial state set")
	}

	currentNode := f.initialNode

	for {
		node, exists := f.nodes[currentNode]
		if !exists {
			return fmt.Errorf("state not found: %s", currentNode)
		}

		exitCode, err := node.execute(f.ctx)
		if err != nil {
			return fmt.Errorf("error in state %s: %w", currentNode, err)
		}

		var matched *transition
		for i := range f.transitions {
			t := &f.transitions[i]
			if t.from == currentNode && t.code == exitCode {
				matched = t
				break
			}
		}

		if matched == nil {
			return nil
		}

		if matched.hook != nil {
			if err := matched.hook(f.ctx); err != nil {
				return fmt.Errorf("transition hook error: %w", err)
			}
		}

		if matched.to == "" {
			return nil
		}

		currentNode = matched.to
	}
}

// StateBuilder provides a fluent API for defining transitions
type StateBuilder struct {
	fsm   *FSM
	state StateName
}

// On defines a transition for an exit code
func (b *StateBuilder) On(code ExitCode, to StateName) *StateBuilder {
	b.fsm.transitions = append(b.fsm.transitions, transition{
		from: b.state,
		code: code,
		to:   to,
	})
	return b
}

// OnWithHook defines a transition with a pre-transition hook
func (b *StateBuilder) OnWithHook(code ExitCode, to StateName, hook func(*Context) error) *StateBuilder {
	b.fsm.transitions = append(b.fsm.transitions, transition{
		from: b.state,
		code: code,
		to:   to,
		hook: hook,
	})
	return b
}

// Exit defines a terminal transition
func (b *StateBuilder) Exit(code ExitCode) *StateBuilder {
	return b.On(code, "")
}
