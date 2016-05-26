// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

func init() {
	log.SetPrefix("fmtjson: ")
	log.SetFlags(0)
}

func main() {
	if len(os.Args) < 1 {
		log.Fatal(`usage: go run jsonfmt.go [files...]

This is a small utility that standardizes the formatting of JSON files.
`)
	}

	for _, path := range os.Args[1:] {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("reading file %s: %v", path, err)
		}

		buff := new(bytes.Buffer)
		if err := json.Indent(buff, data, "", "  "); err != nil {
			if err, ok := err.(*json.SyntaxError); ok {
				// Calculate line number and column of error.
				data := data[:err.Offset]
				lineNum := 1 + bytes.Count(data, []byte{'\n'})
				lastIndex := bytes.LastIndex(data, []byte{'\n'})

				colNum := err.Offset
				if lastIndex > -1 {
					colNum = int64(len(data) - lastIndex)
				}
				log.Fatalf("file %s: invalid json at line %d, column %d: %v", path, lineNum, colNum, err)
			}

			log.Fatalf("file %s: invalid json: %v", path, err)
		}

		if err := ioutil.WriteFile(path, buff.Bytes(), 0644); err != nil {
			log.Fatalf("write file %s: %v", path, err)
		}
	}
}
