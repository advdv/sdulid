// Package main describes automation tasks.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func init() {
	mustBeInRootIfNotTest()
}

// Dev namespace holds development commands.
type Dev mg.Namespace

// Lint our codebase.
func (Dev) Lint() error {
	if err := sh.Run("golangci-lint", "run"); err != nil {
		return fmt.Errorf("failed to run golang-ci: %w", err)
	}

	return nil
}

// Test the whole codebase.
func (Dev) Test() error {
	return (Dev{}).TestSome("!e2e")
}

// TestSome tests some parts of the codebase.
func (Dev) TestSome(labelFilter string) error {
	if err := (Dev{}).testSome(labelFilter, "./..."); err != nil {
		return fmt.Errorf("failed to run ginkgo: %w", err)
	}

	return nil
}

func (Dev) testSome(labelFilter, dir string) error {
	if err := sh.Run(
		"go", "run", "-mod=readonly", "github.com/onsi/ginkgo/v2/ginkgo",
		"-p", "-randomize-all", "--fail-on-pending", "--race", "--trace",
		"--junit-report=test-report.xml",
		"--label-filter", labelFilter,
		dir,
	); err != nil {
		return fmt.Errorf("failed to run ginkgo: %w", err)
	}

	return nil
}

// error when wrong version format is used.
var errVersionFormat = fmt.Errorf("version must be in format vX,Y,Z")

// Release tags a new version and pushes it.
func (Dev) Release(version string) error {
	if !regexp.MustCompile(`^v([0-9]+).([0-9]+).([0-9]+)$`).Match([]byte(version)) {
		return errVersionFormat
	}

	if err := sh.Run("git", "tag", version); err != nil {
		return fmt.Errorf("failed to tag version: %w", err)
	}

	if err := sh.Run("git", "push", "origin", version); err != nil {
		return fmt.Errorf("failed to push version tag: %w", err)
	}

	return nil
}

func mustBeInRootIfNotTest() {
	if _, err := os.ReadFile("go.mod"); err != nil && !strings.Contains(strings.Join(os.Args, ""), "-test.") {
		panic("must be in project root, couldn't stat go.mod file: " + err.Error())
	}
}
