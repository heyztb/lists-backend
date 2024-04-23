package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/magefile/mage/sh"
)

func Unit() error {
	args := []string{
		"test",
		"github.com/heyztb/lists/internal/crypto",
		"github.com/heyztb/lists/internal/paseto",
		"-cover",
		"-v",
	}
	err := sh.RunV("go", args...)
	if err != nil {
		fmt.Printf("error running tests %s", err)
		return err
	}
	return nil
}

func Integration() error {
	args := []string{
		"test",
		"github.com/heyztb/lists/internal/server",
		"-v",
	}

	err := sh.RunV("go", args...)
	if err != nil {
		fmt.Printf("error running integration test %s", err)
		return err
	}
	return nil
}

func Templ() error {
	err := sh.RunV("templ", "generate")
	if err != nil {
		fmt.Printf("error generating templates %s", err)
		return err
	}
	return nil
}

func TemplWatch() error {
	err := sh.RunV("templ", "generate", "-watch")
	if err != nil {
		fmt.Printf("error generating templates %s", err)
		return err
	}
	return nil
}

func CSS() error {
	err := sh.RunV("npx", "tailwindcss", "-i", "./internal/html/static/dev.css", "-o", "./internal/html/static/assets/css/app.css")
	if err != nil {
		fmt.Printf("error building app.css %s", err)
		return err
	}

	return nil
}

func JS() error {
	err := sh.RunV("npm", "run", "build")
	if err != nil {
		fmt.Printf("error building javascript %s", err)
		return err
	}
	return nil
}

func CopyWasmExec() error {
	goRoot, err := sh.Output("go", "env", "GOROOT")
	if err != nil {
		fmt.Printf("error fetching $GOROOT %s", err)
		return err
	}

	err = sh.RunV("cp", fmt.Sprintf("%s/misc/wasm/wasm_exec.js", goRoot), "./internal/html/static/assets/js/")
	if err != nil {
		fmt.Printf("error coyping wasm_exec.js %s", err)
		return err
	}

	return nil
}

func Wasm() error {
	err := sh.RunWithV(map[string]string{
		"GOOS":   "js",
		"GOARCH": "wasm",
	}, "go", "build", "-o", "./internal/html/static/assets/wasm/srp.wasm", "./cmd/srp/")
	if err != nil {
		fmt.Printf("error building server %s", err)
		return err
	}

	return nil
}

func Docker() error {
	err := sh.RunV("docker", "build", ".", "-t", "lists:latest")
	return err
}

func Run() error {
	godotenv.Load()
	environ := os.Environ()
	env := make(map[string]string, len(environ))
	for _, v := range environ {
		kv := strings.Split(v, "=")
		if len(kv) == 2 {
			env[kv[0]] = kv[1]
		}
	}

	fmt.Printf("Starting server on %s\n", env["LISTEN_ADDRESS"])
	err := sh.RunWithV(env, "go", "run", "./cmd/lists")
	if err != nil {
		fmt.Printf("error running server %s", err)
		return err
	}

	return nil
}

func Build() error {
	err := sh.RunWithV(map[string]string{
		"GOOS":        "linux",
		"GOARCH":      "amd64",
		"CGO_ENABLED": "0",
	}, "go", "build", "-ldflags", `-w -s`, "./cmd/lists")
	if err != nil {
		fmt.Printf("error building server %s", err)
		return err
	}

	return nil
}
