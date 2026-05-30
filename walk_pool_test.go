package main

import "testing"

func TestWorkerLimit(t *testing.T) {
	n := workerLimit()
	if n < 4 || n > 32 {
		t.Fatalf("workerLimit() = %d; want in [4, 32]", n)
	}
}
