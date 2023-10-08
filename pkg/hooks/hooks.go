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

func (h *Hooks) GeneratePreBackupCmd() string {
	return h.PreBackup
}

func (h *Hooks) GeneratePostBackupCmd(file string) string {
	cmd := strings.ReplaceAll(h.PostBackup, "%INPUTFILE%", file)
	return cmd
}

func (h *Hooks) HasPreBackup() bool {
	return h.PreBackup != ""
}

func (h *Hooks) HasPostBackup() bool {
	return h.PostBackup != ""
}

func (h *Hooks) ExecutePreBackup() error {
	return execute(h.GeneratePreBackupCmd())
}

func (h *Hooks) ExecutePostBackup(file string) error {
	return execute(h.GeneratePostBackupCmd(file))
}

func execute(command string) error {
	if command == "" {
		return nil
	}
	// cmd := exec.Command("sh", "-c", command)
	// // cmd.Stdout = os.Stdout
	// // cmd.Stderr = os.Stderr
	// err := cmd.Run()
	// if err != nil {
	// 	return err
	// }
	commandSplitter, _ := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	trimmer := splitter.Trim("'\"")

	splitCmd, _ := commandSplitter.Split(command, trimmer)

	// splitCmd := strings.Split(cmd, " ")
	if len(splitCmd) == 0 {
		return nil
	}
	_, err := exec.Command(splitCmd[0], splitCmd[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute : %s - %s", command, err.Error())
	}

	return nil
}
