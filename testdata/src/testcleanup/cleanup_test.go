package testcleanup

import (
	"context"
	"testing"
)

func TestDeferInParent(t *testing.T) {
	t.Parallel()
	tc := Setup()
	defer tc.Teardown() // want `defer runs when this func returns, but a parallel subtest still uses tc after that; use t.Cleanup`
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel() // want `still uses ctx after that`
	for _, name := range []string{"a", "b"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_ = tc
			_ = ctx
		})
	}
}

func TestCleanupInParent(t *testing.T) {
	t.Parallel()
	tc := Setup()
	t.Cleanup(tc.Teardown)
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
	})
}

func TestDeferInParallelSubtest(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		tc := Setup()
		defer tc.Teardown()
	})
}

func TestDeferSequentialSubtests(t *testing.T) {
	t.Parallel()
	tc := Setup()
	defer tc.Teardown()
	t.Run("ok", func(t *testing.T) {})
}

func TestNoSubtests(t *testing.T) {
	t.Parallel()
	tc := Setup()
	defer tc.Teardown()
}

func TestNestedParent(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		tc := Setup()
		defer tc.Teardown() // want `still uses tc after that`
		t.Run("leaf", func(t *testing.T) {
			t.Parallel()
			_ = tc
		})
	})
}

func TestParallelSubtestOwnSetup(t *testing.T) {
	t.Parallel()
	tc := Setup()
	defer tc.Teardown()
	t.Run("sequential", func(t *testing.T) {
		_ = tc
	})
	t.Run("parallel", func(t *testing.T) {
		t.Parallel()
		tc := Setup()
		defer tc.Teardown()
	})
}
