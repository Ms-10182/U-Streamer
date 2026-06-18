package main

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ignoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".idea":        true,
	".vscode":      true,
}

func shouldIgnore(path string) bool {
	parts := strings.Split(path, string(os.PathSeparator))
	for _, p := range parts {
		if ignoreDirs[p] {
			return true
		}
	}
	return false
}

func main() {
	outputFile := "encoded_repo.txt"
	out, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	root := "."

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if shouldIgnore(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		encoded := base64.StdEncoding.EncodeToString(data)

		_, err = fmt.Fprintf(
			out,
			"\n===== FILE: %s =====\n%s\n===== END FILE =====\n",
			path,
			encoded,
		)
		return err
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Repo encoded successfully into", outputFile)
	fmt.Println("📦 You can now share this single file.")
}
