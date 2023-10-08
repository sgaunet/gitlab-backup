package hooks_test

import (
	"os"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/hooks"
)

func TestGeneratePreBackupCmd(t *testing.T) {
	type args struct {
		h hooks.Hooks
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "prebackup",
			args: args{
				h: hooks.Hooks{
					PreBackup: "echo 'prebackup'",
					// PostBackup: "echo 'postbackup %INPUTFILE% %INPUTFILE%.out'",
				},
			},
			want: "echo 'prebackup'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.h.GeneratePreBackupCmd(); got != tt.want {
				t.Errorf("GeneratePreBackupCmd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneratePostBackupCmd(t *testing.T) {
	type args struct {
		file string
		h    hooks.Hooks
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "postbackup",
			args: args{
				file: "archive-12345689.tar.gz",
				h: hooks.Hooks{
					// PreBackup: "echo 'prebackup %INPUTFILE% %INPUTFILE%.out'",
					PostBackup: "echo 'postbackup %INPUTFILE% %INPUTFILE%.out'",
				},
			},
			want: "echo 'postbackup archive-12345689.tar.gz archive-12345689.tar.gz.out'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.h.GeneratePostBackupCmd(tt.args.file); got != tt.want {
				t.Errorf("GeneratePostBackupCmd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteHooks(t *testing.T) {
	// Cleanup before test
	filePathPre := "/tmp/tmp.pre"
	if _, error := os.Stat(filePathPre); error == nil {
		err := os.Remove(filePathPre)
		if err != nil {
			t.Errorf("Error removing file %s", filePathPre)
		}
	}
	filePathPost := "/tmp/archive-12345689.tar.gz.post"
	if _, error := os.Stat(filePathPost); error == nil {
		err := os.Remove(filePathPost)
		if err != nil {
			t.Errorf("Error removing file %s", filePathPost)
		}
	}

	file := "archive-12345689.tar.gz"
	h := hooks.Hooks{
		PreBackup:  "touch /tmp/tmp.pre",
		PostBackup: "touch /tmp/%INPUTFILE%.post",
	}

	// Execute prebackup hook
	err := h.ExecutePreBackup()
	if err != nil {
		t.Errorf("Error executing prebackup hook %s", err)
	}
	// Test if /tmp/archive-12345689.tar.gz.pre exists
	if _, error := os.Stat(filePathPre); error != nil {
		t.Errorf("file %s should exists", filePathPre)
	}

	// Execute postbackup hook
	err = h.ExecutePostBackup(file)
	if err != nil {
		t.Errorf("Error executing postbackup hook %s", err)
	}
	// Cleanup
	os.Remove(filePathPre)
	// Test if /tmp/archive-12345689.tar.gz.post exists
	if _, error := os.Stat(filePathPost); error != nil {
		t.Errorf("file %s should exists", filePathPost)
	}
	// Cleanup
	os.Remove(filePathPost)
}
