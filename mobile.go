package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// The path to the real go binary is stored under this key in environment
const garbleOgGo = "GARBLE_OG_GO"

// If go mobile calls this binary with a command other than "build" we need to call the real go binary.
func redirectToOgGo(args []string) int {

	command := os.Getenv(garbleOgGo)

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				// exit with same code returned from cmd
				return status.ExitStatus()
			}
		}
		return 1
	}

	return 0
}

// Copy this binary to a temp dir, name it "go", and return the path
func copyGarbleToTempDirAsGo() (dir string, err error) {
	pathToGarble, err := os.Executable()
	if err != nil {
		return
	}
	absPathToGarble, err := filepath.Abs(pathToGarble)
	if err != nil {
		return
	}

	dir, err = copyFileToTmp(absPathToGarble, "go")
	if err != nil {
		return
	}

	// Add execute permissions
	err = os.Chmod(filepath.Join(dir, "go"), 0700)

	return
}

// The copied file will have newName as its name.
func copyFileToTmp(src, newName string) (tmpDir string, err error) {

	sourceFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("unable to open source file: %w", err)
	}
	defer sourceFile.Close()

	// It is important that "garble" is part of the name as the "resetPath" function
	// looks for the string "garble"
	tmpDir, err = os.MkdirTemp("", "garble")
	if err != nil {
		return "", fmt.Errorf("unable to create temp directory: %w", err)
	}

	destPath := filepath.Join(tmpDir, newName)
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("unable to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return "", fmt.Errorf("failed to copy file contents: %w", err)
	}

	return tmpDir, nil
}

// If "garble mobile" was called previously, it modified our path. We want to reset it so that we will call
// the real go binary instead of our redirection binary.
func resetPath() {
	path := os.Getenv("PATH")
	if !strings.Contains(path, "garble") {
		return
	}
	path = trimUntilChar(path, os.PathListSeparator)
	os.Setenv("PATH", path)
}

func trimUntilChar(s string, c rune) string {
	index := strings.IndexRune(s, c)
	if index == -1 {
		return s // Character not found, return the original string
	}
	return s[index+1:] // Slice the string from the character onwards
}

func prependToPath(dir string) error {
	// Normalize based on OS
	dir = filepath.FromSlash(dir)

	path := os.Getenv("PATH")

	// Append the directory to the PATH
	path = dir + string(os.PathListSeparator) + path

	return os.Setenv("PATH", path)
}

// When this binary is called as "go" by gomobile, if the command is not in the slice below,
// we want to exec the real go binary.
func isPassThroughCommand(cmd string) bool {
	switch cmd {
	case "build", "toolexec":
		return false
	}
	return true
}
