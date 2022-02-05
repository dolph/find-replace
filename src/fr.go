package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type findReplace struct {
	find    string
	replace string
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: fr FIND REPLACE")
	}

	find := os.Args[1]
	replace := os.Args[2]

	fr := findReplace{find: find, replace: replace}
	filepath.WalkDir(".", fr.Walk)
}

func (fr *findReplace) Walk(path string, info fs.DirEntry, err error) error {
	if err != nil {
		log.Fatal(err)
		return err
	}
	if info.IsDir() && info.Name() == ".git" {
		return fs.SkipDir
	}

	// Rename file
	newPath := strings.Replace(path, fr.find, fr.replace, -1)
	os.Rename(path, newPath)

	if !info.IsDir() {
		fmt.Println(newPath)
	}
	return nil
}
