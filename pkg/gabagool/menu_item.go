package gabagool

// MenuItem represents a single item in a List component.
type MenuItem struct {
	Text               string      // Display text for the item
	Selected           bool        // Whether this item is selected (for multi-select mode)
	Focused            bool        // Whether this item has focus (managed by List)
	NotMultiSelectable bool        // Prevent this item from being multi-selected
	NotReorderable     bool        // Prevent this item from being moved in reorder mode
	Metadata           interface{} // Application-specific data attached to the item
	ImageFilename      string      // Path to image displayed when this item is focused
	BackgroundFilename string      // Path to background image when this item is focused
}

// ListResult is the standardized return type for the List component
type ListResult struct {
	Items           []MenuItem
	Selected        []int      // Indices of selected items (always a slice, even for single selection)
	Action          ListAction // The action taken when exiting (Selected or Triggered)
	VisiblePosition int        // Position of first selected item relative to VisibleStartIndex (for scroll restoration)
}
