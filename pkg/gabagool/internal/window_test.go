package internal

import (
	"testing"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
)

// On-device (non-dev mode) non-NextUI apps never start the power-button
// handler, so initPowerButtonHandling — and therefore PowerButtonWG.Add(1) —
// is never called. closeWindow must not call Done() in that case, otherwise
// the WaitGroup counter goes negative and the process panics on exit with
// "sync: negative WaitGroup counter".
func TestStopPowerButtonHandling_NoHandlerStarted_DoesNotPanic(t *testing.T) {
	t.Setenv(constants.EnvironmentEnvVar, "PROD") // force non-dev mode
	if constants.IsDevMode() {
		t.Fatal("test precondition failed: expected non-dev mode")
	}

	w := &Window{} // zero value: no power-button handler started

	w.stopPowerButtonHandling() // must not panic
}

// When the handler was started (Add(1) called), teardown must release the
// WaitGroup exactly once and be safe to call again (e.g. a double cleanup)
// without driving the counter negative.
func TestStopPowerButtonHandling_HandlerStarted_BalancesAndIsIdempotent(t *testing.T) {
	t.Setenv(constants.EnvironmentEnvVar, "PROD") // force non-dev mode

	w := &Window{}
	w.PowerButtonWG.Add(1)
	w.powerButtonStarted = true

	w.stopPowerButtonHandling() // releases the Add(1): counter 1 -> 0
	w.stopPowerButtonHandling() // no-op: must not go negative
}
