package web

import (
	"embed"
	"io/fs"
)

//go:embed static/* templates/* themes/* robots.txt
var files embed.FS

// FS returns a filesystem with the default web assets.
func FS() fs.FS {
	return files
}
