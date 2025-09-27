package main

import (
	"dagger/dex/internal/dagger"
)

type Dex struct {
	// +private
	Source *dagger.Directory
}

func New(
	// +defaultPath="/"
	// +ignore=[".devenv", ".direnv", ".github", "tmp"]
	source *dagger.Directory,
) *Dex {
	return &Dex{
		Source: source,
	}
}
