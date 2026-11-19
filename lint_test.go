package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestMoneyName(t *testing.T) {
	for name, want := range map[string]bool{
		"price":       true,
		"totalAmount": true,
		"FeeRate":     true,
		"Fees":        true,
		"price_list":  true,
		"management":  false,
		"Feedback":    false,
		"pricing":     false,
	} {
		if got := moneyRegexp(moneyWords).MatchString(name); got != want {
			t.Errorf("moneyRegexp(moneyWords).MatchString(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsCommentedCode(t *testing.T) {
	for name, tc := range map[string]struct {
		lines []string
		want  bool
	}{
		"if block":       {[]string{" if err != nil {", " \treturn err", " }"}, true},
		"func call":      {[]string{" x := load()", " use(x)"}, true},
		"prose":          {[]string{" Retry once: the gateway drops the first", " request after a cold start."}, false},
		"path notes":     {[]string{" v2/otp/request", "v2/otp/verify"}, false},
		"doc example":    {[]string{"\tgostyle lint ./...", "\tgostyle helpers ./..."}, false},
		"example output": {[]string{" Output:", " f(1)"}, false},
		"single line":    {[]string{" x := load()"}, false},
	} {
		if got := isCommentedCode(tc.lines); got != tc.want {
			t.Errorf("%s: isCommentedCode = %v, want %v", name, got, tc.want)
		}
	}
}

func TestBanner(t *testing.T) {
	for text, want := range map[string]bool{
		" ---- Response types ----": true,
		" ===========":              true,
		" -- not a banner":          false,
		" fee = amount * 7%":        false,
	} {
		if got := banner.MatchString(text); got != want {
			t.Errorf("banner.MatchString(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestExportFirst(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), exportFirst, "exportfirst")
}
