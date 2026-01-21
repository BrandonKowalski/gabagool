package gabagool

type ListAction int

const (
	ListActionSelected ListAction = iota
	ListActionTriggered
	ListActionSecondaryTriggered
	ListActionConfirmed
)

type DetailAction int

const (
	DetailActionNone DetailAction = iota
	DetailActionTriggered
	DetailActionConfirmed
	DetailActionCancelled
)
