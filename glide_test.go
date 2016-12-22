package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

type glideLock struct {
	Imports []struct {
		Name        string   `yaml:"name"`
		Subpackages []string `yaml:"subpackages,omitempty"`
	} `yaml:"imports"`
	TestImports []struct {
		Name        string   `yaml:"name"`
		Subpackages []string `yaml:"subpackages,omitempty"`
	} `yaml:"testImports"`
}

type glideYAML struct {
	Imports []struct {
		Name string `yaml:"package"`
	} `yaml:"import"`
}

func loadYAML(t *testing.T, file string, v interface{}) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("read file %s: %v", file, err)
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal file %s: %v", file, err)
	}
	return
}

// TestGlideYAMLPinsAllDependencies ensures that all packages listed in glide.lock also
// appear in glide.yaml which can get out of sync if glide.yaml fails to list transitive
// dependencies.
//
// Testing this ensures developers can update individual packages without grabbing the HEAD
// of an unspecified dependency.
func TestGlideYAMLPinsAllDependencies(t *testing.T) {
	var (
		lockPackages glideLock
		yamlPackages glideYAML
	)
	loadYAML(t, "glide.lock", &lockPackages)
	loadYAML(t, "glide.yaml", &yamlPackages)

	if len(yamlPackages.Imports) == 0 {
		t.Fatalf("no packages found in glide.yaml")
	}

	pkgs := make(map[string]bool)
	for _, pkg := range yamlPackages.Imports {
		pkgs[pkg.Name] = true
	}

	for _, pkg := range lockPackages.Imports {
		if pkgs[pkg.Name] {
			continue
		}
		if len(pkg.Subpackages) == 0 {
			t.Errorf("package in glide lock but not pinned in glide yaml: %s", pkg.Name)
			continue
		}

		for _, subpkg := range pkg.Subpackages {
			pkgName := path.Join(pkg.Name, subpkg)
			if !pkgs[pkgName] {
				t.Errorf("package in glide lock but not pinned in glide yaml: %s", pkgName)
			}
		}
	}

	for _, pkg := range lockPackages.TestImports {
		if pkgs[pkg.Name] {
			continue
		}
		if len(pkg.Subpackages) == 0 {
			t.Errorf("package in glide lock but not pinned in glide yaml: %s", pkg.Name)
			continue
		}

		for _, subpkg := range pkg.Subpackages {
			pkgName := path.Join(pkg.Name, subpkg)
			if !pkgs[pkgName] {
				t.Errorf("package in glide lock but not pinned in glide yaml: %s", pkgName)
			}
		}
	}
}

func TestGlideVCUseLockFile(t *testing.T) {
	_, err := os.Stat("vendor/github.com/golang/protobuf/protoc-gen-go")
	if err != nil {
		t.Fatalf("vendor did not use glide-vc --use-lock-file. Revendor packages using 'make revendor' to use the correct glide and glide-vc flags")
	}
}

func TestGlideFlagsAndGlideVC(t *testing.T) {
	err := filepath.Walk("vendor", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Fatalf("walk: stat path %s failed: %v", path, err)
		}
		if info.IsDir() && filepath.Base(path) == ".git" {
			t.Fatalf(".git directory detected in vendor: %s. Revendor packages using 'make revendor' to use the correct glide and glide-vc flags", path)
		}
		if !info.IsDir() && strings.HasSuffix(path, "_test.go") {
			t.Fatalf("'_test.go' file detected in vendor: %s. Revendor packages using 'make revendor' to use the correct glide and glide-vc flags", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
