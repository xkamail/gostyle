package main

import (
	"slices"
	"testing"
)

func TestTestCover(t *testing.T) {
	tests, err := testCover("testdata/testcover", []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, tc := range tests {
		if tc.Failed {
			t.Fatalf("%s failed", tc.Test)
		}
		got[tc.Test] = tc.Unique
	}
	for name, want := range map[string][]string{
		"TestAbs":          {},
		"TestAbs_Positive": {"testdata/testcover/abs.go:7-7"},
		"TestAbs_Negative": {},
	} {
		if !slices.Equal(got[name], want) {
			t.Errorf("%s unique = %v, want %v", name, got[name], want)
		}
	}
}
