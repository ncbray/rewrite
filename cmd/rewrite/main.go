// Tool for rewriting files.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var eol = regexp.MustCompile("(?m)^.*$")

type Query struct {
	Directory      string
	FileSuffixes   []string
	MatchContent   string
	ReplaceContent string
	Commit         bool
}

type Matchers struct {
	FileSuffixes []string
	Content      *regexp.Regexp
	Replacement  []byte
	Commit       bool
}

type LineResult struct {
	Line      int
	Text      string
	Rewritten string
}

type FileResult struct {
	Path  string
	Lines []LineResult
}

type Result struct {
	Error string
	Files []FileResult
}

func matchContent(matchers *Matchers, file_path string, result *Result) {
	lineResults := []LineResult{}

	if matchers.Content != nil {
		// TODO error handling.
		data, _ := ioutil.ReadFile(file_path)
		if matchers.Content.Find(data) == nil {
			return
		}

		if matchers.Commit {
			data = matchers.Content.ReplaceAll(data, matchers.Replacement)
			ioutil.WriteFile(file_path, data, 0644)
		}

		matches := matchers.Content.FindAllIndex(data, -1)
		lines := eol.FindAllIndex(data, -1)

		lineMatched := make([]bool, len(lines))

		line := 0
		match := 0
		for {
			if line >= len(lines) || match >= len(matches) {
				break
			}

			lr := lines[line]
			mr := matches[match]

			if mr[0] < lr[0] {
				// Match starts before line start
				if mr[1] <= lr[0] {
					// Match entirely before line.
					match += 1
				} else if mr[1] <= lr[1] {
					// Match overlaps start of line.
					lineMatched[line] = true
					// We know the line matches.
					line += 1
					// Move on to the next match.
					match += 1
				} else {
					// Line is contained in match.
					lineMatched[line] = true
					line += 1
				}
			} else if mr[0] < lr[1] {
				// Match starts inside line.
				lineMatched[line] = true
				if mr[1] <= lr[1] {
					// Match is contained in line
					match += 1
				}
				// We know the line matches.
				line += 1
			} else {
				// Match is after line.
				line += 1
			}
		}

		for line := 0; line < len(lines); line++ {
			if lineMatched[line] {
				lr := lines[line]
				lineBytes := data[lr[0]:lr[1]]
				result := LineResult{Line: line, Text: string(lineBytes)}
				// HACK does not support empty replacements.
				if len(matchers.Replacement) > 0 {
					// HACK assumes match will not cross line boundary.
					result.Rewritten = string(matchers.Content.ReplaceAll(lineBytes, matchers.Replacement))
				}
				lineResults = append(lineResults, result)
			}
		}
	}

	result.Files = append(result.Files, FileResult{Path: file_path, Lines: lineResults})
}

func fileSuffixMatches(name string, suffixes []string) bool {
	if len(suffixes) == 0 {
		return true
	}
	ext := filepath.Ext(name)
	if ext == "" {
		return false
	}
	ext = ext[1:]
	for _, suffix := range suffixes {
		if suffix == ext {
			return true
		}
	}
	return false
}

func findFiles(matchers *Matchers, dir_path string, result *Result) {
	p := dir_path
	if p == "" {
		p = "."
	}
	files, err := ioutil.ReadDir(p)
	if err != nil {
		result.Error = err.Error()
	}
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		rel := filepath.Join(dir_path, name)
		if f.IsDir() {
			findFiles(matchers, rel, result)
		} else {
			if fileSuffixMatches(name, matchers.FileSuffixes) {
				matchContent(matchers, rel, result)
			}
		}
	}
}

func performQuery(query *Query) *Result {
	result := &Result{Files: []FileResult{}}

	matchers := &Matchers{
		FileSuffixes: query.FileSuffixes,
	}

	if query.MatchContent != "" {
		r, err := regexp.Compile(query.MatchContent)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		matchers.Content = r
		matchers.Replacement = []byte(query.ReplaceContent)
		matchers.Commit = query.Commit
	}

	cwd, _ := os.Getwd()
	os.Chdir(data_dir)
	defer os.Chdir(cwd)

	findFiles(matchers, query.Directory, result)
	return result
}

func query(w http.ResponseWriter, r *http.Request) {
	data, _ := ioutil.ReadAll(r.Body)
	query := &Query{}
	json.Unmarshal(data, query)

	result := performQuery(query)
	if result.Error != "" {
		result.Files = result.Files[0:0]
	}

	serialized, _ := json.Marshal(result)

	w.Header().Set("Content-Type", "application/json")
	w.Write(serialized)
}

func checkDir(path string) error {
	if path == "" {
		return errors.New("not specified")
	}
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return errors.New("not a directory")
	}
	return nil
}

var data_dir string
var static_dir string
var port int

func main() {
	flag.StringVar(&data_dir, "data_dir", "", "Directory of files to rewrite.")
	flag.StringVar(&static_dir, "static_dir", "", "Directory of static web content.")
	flag.IntVar(&port, "port", 5432, "Web server port.")

	flag.Parse()

	err := checkDir(data_dir)
	if err != nil {
		fmt.Printf("data_dir - %s\n", err.Error())
		os.Exit(1)
	}
	err = checkDir(static_dir)
	if err != nil {
		fmt.Printf("static_dir - %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Launching on port %d\n", port)
	http.HandleFunc("/query", query)
	http.Handle("/view/", http.StripPrefix("/view/", http.FileServer(http.Dir(data_dir))))
	http.Handle("/", http.FileServer(http.Dir(static_dir)))
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
