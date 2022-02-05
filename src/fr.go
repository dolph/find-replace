package main

import (
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

func main() {
	if len(os.Args) != 3 {
		log.Print("usage: fr FIND REPLACE")
	}

	find := os.Args[1]
	replace := os.Args[2]

	fr := findReplace{find: find, replace: replace}

	// Recursively explore the hierarchy depth first, rewrite files as needed,
	// and rename files last (after we don't have to revisit them).
	// path.filepath.WalkDir() won't work here because it walks files
	// alphabetically, breadth-first (and you'd be renaming files that you haven't explored yet).
	fr.WalkDir(".", ".")
}

func (fr *findReplace) WalkDir(baseDir string, path string) {
	// List the files in this path.
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal("Unable to read directory: ", err)
	}

	for _, file := range files {
		if file.Name() != ".git" {
			// If this a directory, recurse immediately (depth-first).
			if file.IsDir() {
				fr.WalkDir(baseDir+string(os.PathSeparator)+path, file.Name())
			}

			fr.ReplaceContents(baseDir+string(os.PathSeparator)+path, file)

			// Rename the file now that we're otherwise done with it
			newName := strings.Replace(file.Name(), fr.find, fr.replace, -1)
			if file.Name() != newName {
				// TODO: abort if destination file already exists
				fmt.Printf("%v%v%v -> %v%v%v\n", path, string(os.PathSeparator), file.Name(), path, string(os.PathSeparator), newName)
				os.Rename(path+string(os.PathSeparator)+file.Name(), path+string(os.PathSeparator)+newName)
			} else {
				fmt.Printf("%v%v%v\n", path, string(os.PathSeparator), file.Name())
			}

		}
	}
}

func (fr *findReplace) ReplaceContents(dirName string, file fs.DirEntry) {
	// Find & replace contents of file
	if !file.IsDir() {
		f, err := os.Open(dirName + string(os.PathSeparator) + file.Name())
		if err != nil {
			log.Fatal("Unable to open "+dirName+string(os.PathSeparator)+file.Name(), err)
		}
		defer f.Close()
		builder := new(strings.Builder)
		io.Copy(builder, f)
		str := builder.String()
		if strings.Contains(str, fr.find) {
			content := strings.Replace(builder.String(), fr.find, fr.replace, -1)
			tmpfile, err := ioutil.TempFile(dirName, randomString(8))
			if err != nil {
				log.Print("Error creating tempfile")
				log.Fatal(err)
			}

			defer os.Rename(tmpfile.Name(), file.Name())

			if _, err := tmpfile.WriteString(content); err != nil {
				log.Print("Error writing to tempfile")
				log.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				log.Print("Error closing tempfile")
				log.Fatal(err)
			}
		}
	}
}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}
