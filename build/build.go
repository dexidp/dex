package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

var version = "dev"

var targetDir = "./build/target"

type Target struct {
	OS     string
	Arch   string
	Target string
	CC     string
	ARM    string
}

func main() {
	wd, _ := os.Getwd()
	log.Printf("Executing script from %q", wd)

	version = os.Getenv("VERSION")
	log.Printf("Dex version %q", version)

	targets := []Target{
		{
			OS:     "linux",
			Arch:   "amd64",
			Target: "linux/amd64",
		},
		{
			OS:     "linux",
			Arch:   "arm64",
			Target: "linux/arm64",
			CC:     "aarch64-linux-gnu-gcc",
		},
		{
			OS:     "linux",
			Arch:   "arm",
			Target: "linux/arm/v7",
			ARM:    "7",
			CC:     "arm-linux-gnueabihf-gcc",
		},
		{
			OS:     "linux",
			Arch:   "ppc64le",
			Target: "linux/ppc64le",
			CC:     "powerpc64le-linux-gnu-gcc",
		},
	}

	apps := []string{
		"dex",
		"docker-entrypoint",
	}

	_ = os.Mkdir(targetDir, 0777)

	wg := sync.WaitGroup{}
	for _, target := range targets {
		for _, app := range apps {
			wg.Add(1)
			go executeBuild(&wg, app, target)
		}
	}

	wg.Wait()
}

func executeBuild(wg *sync.WaitGroup, name string, target Target) {
	defer wg.Done()

	GOARM := ""
	if target.ARM != "" {
		GOARM = "v" + target.ARM
	}

	ldFlags := fmt.Sprintf(`-w -X main.version=%s -extldflags "-static"`, version)

	src := fmt.Sprintf("./cmd/%s", name)
	dest := fmt.Sprintf("%s/%s-%s-%s%s", targetDir, name, target.OS, target.Arch, GOARM)

	cmd := exec.Command("go", "build",
		"-v",
		"-ldflags", ldFlags,
		"-tags", "netgo",
		"-o", dest,
		src,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	envCopy := append([]string{}, os.Environ()...)
	cmd.Env = append(envCopy,
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
		"GOARM="+target.ARM,
		"CC="+target.CC,
		"CGO_ENABLED=1",
	)
	log.Printf("Start building %q for %q", name, target.Target)

	if err := cmd.Run(); err != nil {
		log.Println(err.Error())
	}
}
