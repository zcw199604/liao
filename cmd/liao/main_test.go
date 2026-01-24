package main

import (
	"errors"
	"testing"
)

func TestMain_ExitsWhenRunFails(t *testing.T) {
	oldExit := osExit
	oldRun := runFn
	t.Cleanup(func() {
		osExit = oldExit
		runFn = oldRun
	})

	exitCode := 0
	osExit = func(code int) { //nolint:revive // test stub
		exitCode = code
		panic("exit")
	}
	runFn = func() error { return errors.New("boom") }

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected exit")
		}
	}()

	main()
	if exitCode != 1 {
		t.Fatalf("exitCode=%d, want 1", exitCode)
	}
}

func TestMain_DoesNotExitWhenRunSucceeds(t *testing.T) {
	oldExit := osExit
	oldRun := runFn
	t.Cleanup(func() {
		osExit = oldExit
		runFn = oldRun
	})

	osExit = func(code int) { //nolint:revive // test stub
		t.Fatalf("unexpected exit: %d", code)
	}
	runFn = func() error { return nil }

	main()
}
