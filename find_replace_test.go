package main

import "testing"

func TestRandomStringLengthNegativeOne(t *testing.T) {
	got := randomString(-1)
	if len(got) != 0 {
		t.Errorf("len(RandomString(-1)) = %v; want 0", got)
	}
}

func TestRandomStringLengthZero(t *testing.T) {
	got := randomString(0)
	if len(got) != 0 {
		t.Errorf("len(RandomString(0)) = %v; want 0", got)
	}
}

func TestRandomStringLengthOne(t *testing.T) {
	got := randomString(1)
	if len(got) != 1 {
		t.Errorf("len(RandomString(1)) = %v; want 1", got)
	}
}

func TestRandomStringLengthTen(t *testing.T) {
	got := randomString(10)
	if len(got) != 10 {
		t.Errorf("len(RandomString(10)) = %v; want 10", got)
	}
}

func TestRandomStringLengthTwenty(t *testing.T) {
	got := randomString(20)
	if len(got) != 20 {
		t.Errorf("len(RandomString(20)) = %v; want 20", got)
	}
}
