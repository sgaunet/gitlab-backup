// Package encryption provides backup-archive encryption using the age file
// encryption format (https://age-encryption.org).
//
// age uses public-key cryptography (X25519) and is the recommended encryption
// option for automated backups: recipient public keys can be safely stored in
// the cluster, while the matching private identity stays offline and is only
// used for restore. A compromised backup runner can therefore encrypt new
// archives but cannot decrypt past ones.
//
// SSH public keys (ssh-ed25519, ssh-rsa) are also accepted as recipients.
package encryption

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// ErrNoRecipients is returned when no age recipients are configured or parsed.
var ErrNoRecipients = errors.New("no age recipients provided")

// tempFilePerm is the mode used for the intermediate encrypted file.
const tempFilePerm os.FileMode = 0o600

// ParseRecipients parses age recipients from a list of strings.
// Each entry is one recipient (age1..., or an SSH public key line).
// Empty strings and lines starting with '#' (comments) are ignored.
// Returns ErrNoRecipients if nothing parses.
func ParseRecipients(lines []string) ([]age.Recipient, error) {
	nonEmpty := make([]string, 0, len(lines))
	for _, l := range lines {
		s := strings.TrimSpace(l)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		nonEmpty = append(nonEmpty, s)
	}
	if len(nonEmpty) == 0 {
		return nil, ErrNoRecipients
	}
	r := strings.NewReader(strings.Join(nonEmpty, "\n"))
	recipients, err := age.ParseRecipients(r)
	if err != nil {
		return nil, fmt.Errorf("parse age recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil, ErrNoRecipients
	}
	return recipients, nil
}

// ParseRecipientsFile reads age recipients from a file. Blank lines and
// comment lines (starting with '#') are ignored. Returns ErrNoRecipients if
// the file contains no usable recipient lines.
func ParseRecipientsFile(path string) ([]age.Recipient, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from operator config
	if err != nil {
		return nil, fmt.Errorf("read recipients file %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	recipients, err := ParseRecipients(lines)
	if err != nil {
		if errors.Is(err, ErrNoRecipients) {
			return nil, fmt.Errorf("recipients file %s: %w", path, ErrNoRecipients)
		}
		return nil, fmt.Errorf("recipients file %s: %w", path, err)
	}
	return recipients, nil
}

// EncryptFileInPlace encrypts the file at path with the given recipients and
// replaces the original on success. If armorEnabled is true, the output is
// ASCII-armored (PEM-like). The file path is preserved so downstream upload
// logic does not need to be aware of encryption.
func EncryptFileInPlace(path string, recipients []age.Recipient, armorEnabled bool) error {
	if len(recipients) == 0 {
		return ErrNoRecipients
	}

	in, err := os.Open(path) //nolint:gosec // path is produced by this app
	if err != nil {
		return fmt.Errorf("open archive %s: %w", path, err)
	}
	defer func() { _ = in.Close() }()

	tmpPath := path + ".age.tmp"
	//nolint:gosec // tmpPath derives from app-controlled archive path
	out, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, tempFilePerm)
	if err != nil {
		return fmt.Errorf("create temp encrypted file: %w", err)
	}
	removeTmp := func() { _ = os.Remove(tmpPath) }

	if encErr := encryptStream(in, out, recipients, armorEnabled); encErr != nil {
		_ = out.Close()
		removeTmp()
		return encErr
	}
	if closeErr := out.Close(); closeErr != nil {
		removeTmp()
		return fmt.Errorf("close temp encrypted file: %w", closeErr)
	}
	if closeErr := in.Close(); closeErr != nil {
		removeTmp()
		return fmt.Errorf("close archive: %w", closeErr)
	}
	if renameErr := os.Rename(tmpPath, path); renameErr != nil {
		removeTmp()
		return fmt.Errorf("replace archive with encrypted version: %w", renameErr)
	}
	return nil
}

// encryptStream encrypts r to w using the provided recipients.
// When armorEnabled is true, the output is wrapped in age's ASCII armor.
func encryptStream(r io.Reader, w io.Writer, recipients []age.Recipient, armorEnabled bool) error {
	sink := w
	var armorW io.WriteCloser
	if armorEnabled {
		armorW = armor.NewWriter(w)
		sink = armorW
	}

	encW, err := age.Encrypt(sink, recipients...)
	if err != nil {
		return fmt.Errorf("init age writer: %w", err)
	}

	if _, copyErr := io.Copy(encW, r); copyErr != nil {
		_ = encW.Close()
		if armorW != nil {
			_ = armorW.Close()
		}
		return fmt.Errorf("encrypt archive: %w", copyErr)
	}

	if closeErr := encW.Close(); closeErr != nil {
		if armorW != nil {
			_ = armorW.Close()
		}
		return fmt.Errorf("close age writer: %w", closeErr)
	}
	if armorW != nil {
		if closeErr := armorW.Close(); closeErr != nil {
			return fmt.Errorf("close armor writer: %w", closeErr)
		}
	}
	return nil
}
