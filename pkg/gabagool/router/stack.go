package router

// StackEntry represents a single entry in the navigation stack.
// It stores the screen identifier, the input that was used to call the screen,
// and any resume state returned by the screen.
type StackEntry struct {
	Screen Screen
	Input  any
	Resume any
}

// Stack manages navigation history for back navigation.
// It stores entries that allow returning to previous screens
// with their original input and resume state.
type Stack struct {
	entries []StackEntry
}

// NewStack creates a new empty navigation stack.
func NewStack() *Stack {
	return &Stack{
		entries: make([]StackEntry, 0),
	}
}

// Push adds a new entry to the stack.
// Called when navigating forward to a new screen.
func (s *Stack) Push(screen Screen, input any, resume any) {
	s.entries = append(s.entries, StackEntry{
		Screen: screen,
		Input:  input,
		Resume: resume,
	})
}

// Pop removes and returns the top entry from the stack.
// Returns nil if the stack is empty.
func (s *Stack) Pop() *StackEntry {
	if len(s.entries) == 0 {
		return nil
	}
	entry := s.entries[len(s.entries)-1]
	s.entries = s.entries[:len(s.entries)-1]
	return &entry
}

// Peek returns the top entry without removing it.
// Returns nil if the stack is empty.
func (s *Stack) Peek() *StackEntry {
	if len(s.entries) == 0 {
		return nil
	}
	return &s.entries[len(s.entries)-1]
}

// IsEmpty returns true if the stack has no entries.
func (s *Stack) IsEmpty() bool {
	return len(s.entries) == 0
}

// Len returns the number of entries in the stack.
func (s *Stack) Len() int {
	return len(s.entries)
}

// Clear removes all entries from the stack.
func (s *Stack) Clear() {
	s.entries = s.entries[:0]
}
