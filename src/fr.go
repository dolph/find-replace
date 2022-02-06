package main

import (
	"errors"
	"io"
	"io/fs"
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
		if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
			log.Print("Renaming " + oldPath + " to " + newBaseName)
			os.Rename(oldPath, newPath)
		} else {
			log.Print("Refusing to rename " + oldPath + " to " + newBaseName + " because " + newPath + " already exists.")
		}
	}
}

func readFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("Unable to open "+path, err)
	}
	defer f.Close()
	builder := new(strings.Builder)
	io.Copy(builder, f)
	return builder.String()
}

// Atomically write file.
func writeFile(dirName string, file fs.DirEntry, content string) {
	path := dirName + string(os.PathSeparator) + file.Name()

	info, err := os.Stat(path)
	if err != nil {
		log.Print("Error getting stats on " + path)
		log.Fatal(err)
	}

	tempName := dirName + string(os.PathSeparator) + randomString(20)
	if err := os.WriteFile(tempName, []byte(content), info.Mode()); err != nil {
		log.Print("Error creating tempfile in " + dirName)
		log.Fatal(err)
	}

	log.Print("Rewriting " + path)
	os.Rename(tempName, path)
}

func (fr *findReplace) ReplaceContents(dirName string, file fs.DirEntry) {
	path := dirName + string(os.PathSeparator) + file.Name()

	// Find & replace the contents of file.
	content := readFile(path)
	if strings.Contains(content, fr.find) {
		newContent := strings.Replace(content, fr.find, fr.replace, -1)
		writeFile(dirName, file, newContent)
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
