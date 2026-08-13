package sample

import "testing"

func TestPass(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatal("unexpected sum")
	}
}

func TestFail(t *testing.T) {
	t.Fatal("deterministic fixture failure")
}

func FuzzSafe(f *testing.F) {
	f.Add([]byte("seed"))
	f.Fuzz(func(t *testing.T, input []byte) {
		_ = append([]byte(nil), input...)
	})
}

func FuzzCrash(f *testing.F) {
	f.Fuzz(func(t *testing.T, input []byte) {
		t.Fatal("deterministic fuzz fixture failure")
	})
}
