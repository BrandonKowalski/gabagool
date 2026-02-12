package gabagool

// ListAction represents user actions that can occur within a List component.
type ListAction int

const (
	ListActionSelected           ListAction = iota // User selected an item (A button)
	ListActionTriggered                            // User triggered action button (X button)
	ListActionSecondaryTriggered                   // User triggered secondary action (Y button)
	ListActionConfirmed                            // User confirmed selection (Start button)
	ListActionTertiaryTriggered                    // User triggered tertiary action (Menu button)
)

// DetailAction represents user actions that can occur within a Detail component.
type DetailAction int

const (
	DetailActionNone      DetailAction = iota // No action taken
	DetailActionTriggered                     // User triggered primary action (A button)
	DetailActionConfirmed                     // User confirmed via action button
	DetailActionCancelled                     // User cancelled/went back (B button)
)
