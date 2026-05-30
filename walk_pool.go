package main

import "runtime"

func workerLimit() int {
	n := runtime.GOMAXPROCS(0) * 2
	if n < 4 {
		return 4
	}
	if n > 32 {
		return 32
	}
	return n
}
