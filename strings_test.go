package main

import (
	"testing"
)

// assertRandomStringLength ensures that the generated string matches the
// desired length.
func assertRandomStringLength(t *testing.T, ask int, want int) {
	got := len(RandomString(ask))
	if got != want {
		t.Errorf("len(RandomString(%v)) = %v; want %v", ask, got, want)
	}
}

func TestRandomStringLengthNegativeOne(t *testing.T) {
	assertRandomStringLength(t, -1, 0)
}

func TestRandomStringLengthZero(t *testing.T) {
	assertRandomStringLength(t, 0, 0)
}

func TestRandomStringLengthOne(t *testing.T) {
	assertRandomStringLength(t, 1, 1)
}

func TestRandomStringLengthTen(t *testing.T) {
	assertRandomStringLength(t, 10, 10)
}

func TestRandomStringLengthTwenty(t *testing.T) {
	assertRandomStringLength(t, 20, 20)
}
