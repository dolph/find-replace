package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type findReplace struct {
	find    string
	replace string
}

// ** Variable terminology **
// dirName: the name of a directory, without a trailing separator
// baseName: the relative name of a file, without a directory
// path: the relative path to a specific file or directory, including both dirName and baseName

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 3 {
		log.Print("usage: fr FIND REPLACE")
	}

	find := os.Args[1]
	replace := os.Args[2]

	fr := findReplace{find: find, replace: replace}

	// Recursively explore the hierarchy depth first, rewrite files as needed,
	// and rename files last (after we don't have to revisit them).
	// path.filepath.WalkDir() won't work here because it walks files
	// alphabetically, breadth-first (and you'd be renaming files that you
	// haven't explored yet).
	fr.WalkDir(".")
}

func (fr *findReplace) WalkDir(path string) {
	// List the files in this path.
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal("Unable to read directory: ", err)
	}

	for _, file := range files {
		if file.Name() != ".git" {
			// If file is a directory, recurse immediately (depth-first).
			if file.IsDir() {
				fr.WalkDir(path + string(os.PathSeparator) + file.Name())
			} else {
				// Replace the contents of regular files
				fr.ReplaceContents(path, file)
			}

			// Rename the file now that we're otherwise done with it
			fr.RenameFile(path, file)
		}
	}
}

// Renames a file if the destination does not already exist.
func (fr *findReplace) RenameFile(dirName string, file fs.DirEntry) {
	oldPath := dirName + string(os.PathSeparator) + file.Name()
	newBaseName := strings.Replace(file.Name(), fr.find, fr.replace, -1)
	newPath := dirName + string(os.PathSeparator) + newBaseName

	if file.Name() != newBaseName {
		fmt.Printf("%v -> %v\n", oldPath, newPath)
		if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
			os.Rename(oldPath, newPath)
		} else {
			log.Print("Refusing to rename " + oldPath + " to " + newBaseName + " because " + newPath + " already exists.")
		}
	} else {
		fmt.Printf("%v\n", oldPath)
	}
}

func (fr *findReplace) ReplaceContents(dirName string, file fs.DirEntry) {
	path := dirName + string(os.PathSeparator) + file.Name()

	// Find & replace contents of file
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("Unable to open "+path, err)
	}
	defer f.Close()
	builder := new(strings.Builder)
	io.Copy(builder, f)
	str := builder.String()
	if strings.Contains(str, fr.find) {
		content := strings.Replace(builder.String(), fr.find, fr.replace, -1)
		tmpfile, err := ioutil.TempFile(dirName, randomString(20))
		if err != nil {
			log.Print("Error creating tempfile")
			log.Fatal(err)
		}
		log.Print(tmpfile.Name())

		if _, err := tmpfile.WriteString(content); err != nil {
			log.Print("Error writing to tempfile")
			log.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			log.Print("Error closing tempfile")
			log.Fatal(err)
		}

		os.Rename(tmpfile.Name(), path)
	}
}

var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
