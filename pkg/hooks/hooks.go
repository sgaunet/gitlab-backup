// Package hooks provides pre and post backup hook functionality.
package hooks

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-andiamo/splitter"
)

// Hooks holds the configuration for pre and post backup hooks.
type Hooks struct {
	PreBackup  string `env:"PREBACKUP"  env-default:"" yaml:"prebackup"`
	PostBackup string `env:"POSTBACKUP" env-default:"" yaml:"postbackup"`
}

// GeneratePreBackupCmd generates the pre backup command.
func (h *Hooks) GeneratePreBackupCmd() string {
	return h.PreBackup
}

// GeneratePostBackupCmd generates the post backup command.
func (h *Hooks) GeneratePostBackupCmd(file string) string {
	cmd := strings.ReplaceAll(h.PostBackup, "%INPUTFILE%", file)
	return cmd
}

// HasPreBackup returns true if a pre backup command is defined.
func (h *Hooks) HasPreBackup() bool {
	return h.PreBackup != ""
}

// HasPostBackup returns true if a post backup command is defined.
func (h *Hooks) HasPostBackup() bool {
	return h.PostBackup != ""
}

// ExecutePreBackup executes the pre backup command.
func (h *Hooks) ExecutePreBackup() error {
	return execute(h.GeneratePreBackupCmd())
}

// ExecutePostBackup executes the post backup command.
func (h *Hooks) ExecutePostBackup(file string) error {
	return execute(h.GeneratePostBackupCmd(file))
}

// execute executes the given command.
func execute(command string) error {
	if command == "" {
		return nil
	}
	commandSplitter, err := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	if err != nil {
		return fmt.Errorf("failed to create command splitter: %w", err)
	}
	trimmer := splitter.Trim("'\"")
	splitCmd, err := commandSplitter.Split(command, trimmer)
	if err != nil {
		return fmt.Errorf("failed to parse command '%s': %w", command, err)
	}
	if len(splitCmd) == 0 {
		return nil
	}
	//nolint:gosec,noctx // G204: Command execution with user input is intentional for hook functionality
	_, err = exec.Command(splitCmd[0], splitCmd[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute %s: %w", command, err)
	}
	return nil
}
