package main

import "log"

func (fr *findReplace) noteError(format string, args ...interface{}) {
	log.Printf(format, args...)
	fr.hadErrors.Store(true)
}
