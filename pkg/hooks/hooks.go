package hooks

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-andiamo/splitter"
)

type Hooks struct {
	PreBackup  string `yaml:"prebackup" env:"PREBACKUP" env-default:""`
	PostBackup string `yaml:"postbackup" env:"POSTBACKUP" env-default:""`
}

// GeneratePreBackupCmd generates the pre backup command
func (h *Hooks) GeneratePreBackupCmd() string {
	return h.PreBackup
}

// GeneratePostBackupCmd generates the post backup command
func (h *Hooks) GeneratePostBackupCmd(file string) string {
	cmd := strings.ReplaceAll(h.PostBackup, "%INPUTFILE%", file)
	return cmd
}

// HasPreBackup returns true if a pre backup command is defined
func (h *Hooks) HasPreBackup() bool {
	return h.PreBackup != ""
}

// HasPostBackup returns true if a post backup command is defined
func (h *Hooks) HasPostBackup() bool {
	return h.PostBackup != ""
}

// ExecutePreBackup executes the pre backup command
func (h *Hooks) ExecutePreBackup() error {
	return execute(h.GeneratePreBackupCmd())
}

// ExecutePostBackup executes the post backup command
func (h *Hooks) ExecutePostBackup(file string) error {
	return execute(h.GeneratePostBackupCmd(file))
}

// execute executes the given command
func execute(command string) error {
	if command == "" {
		return nil
	}
	commandSplitter, _ := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	trimmer := splitter.Trim("'\"")
	splitCmd, _ := commandSplitter.Split(command, trimmer)
	if len(splitCmd) == 0 {
		return nil
	}
	_, err := exec.Command(splitCmd[0], splitCmd[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute : %s - %s", command, err.Error())
	}
	return nil
}
