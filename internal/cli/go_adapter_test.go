package cli

import "testing"

func TestAdapterStatusExit(t *testing.T) {
	cases := map[string]int{
		"passed":      ExitOK,
		"failed":      ExitAdapterFailure,
		"unsupported": ExitUnsupported,
		"unexpected":  ExitInternal,
	}
	for status, expected := range cases {
		if got := adapterStatusExit(status); got != expected {
			t.Fatalf("%s: expected %d, got %d", status, expected, got)
		}
	}
}
