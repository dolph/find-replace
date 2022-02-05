package main

import (
	"fmt"
	"log"
	"os"
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

	// Recursively explore the hierarchy depth first, rewrite files as needed,
	// and rename files last (after we don't have to revisit them).
	// path.filepath.WalkDir() won't work here because it walks files
	// alphabetically, breadth-first (and you'd be renaming files that you haven't explored yet).
	fr.WalkDir("", ".")
}

func (fr *findReplace) WalkDir(baseDir string, path string) {
	// List the files in this path.
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.Name() != ".git" {
			if file.IsDir() {
				fr.WalkDir(baseDir+string(os.PathSeparator)+path, file.Name())
			}

			// Rename the file now that we're otherwise done with it
			newName := strings.Replace(file.Name(), fr.find, fr.replace, -1)
			if file.Name() != newName {
				fmt.Printf("%v%v%v -> %v%v%v\n", path, string(os.PathSeparator), file.Name(), path, string(os.PathSeparator), newName)
				os.Rename(path+string(os.PathSeparator)+file.Name(), path+string(os.PathSeparator)+newName)
			} else {
				fmt.Printf("%v%v%v\n", path, string(os.PathSeparator), file.Name())
			}

		}
	}
}
