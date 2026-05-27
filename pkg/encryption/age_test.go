package encryption_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/sgaunet/gitlab-backup/pkg/encryption"
	"github.com/stretchr/testify/require"
)

const sampleArchive = "this is a synthetic gitlab archive payload\n"

func newIdentity(t *testing.T) *age.X25519Identity {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	return id
}

func TestParseRecipients_FromInlineStrings(t *testing.T) {
	id := newIdentity(t)
	recipients, err := encryption.ParseRecipients([]string{id.Recipient().String()})
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestParseRecipients_TrimsAndSkipsBlanks(t *testing.T) {
	id := newIdentity(t)
	recipients, err := encryption.ParseRecipients([]string{"", "  " + id.Recipient().String() + "  ", ""})
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestParseRecipients_EmptyReturnsSentinel(t *testing.T) {
	_, err := encryption.ParseRecipients(nil)
	require.ErrorIs(t, err, encryption.ErrNoRecipients)

	_, err = encryption.ParseRecipients([]string{"", "   "})
	require.ErrorIs(t, err, encryption.ErrNoRecipients)
}

func TestParseRecipients_InvalidString(t *testing.T) {
	_, err := encryption.ParseRecipients([]string{"not-a-real-key"})
	require.Error(t, err)
	require.NotErrorIs(t, err, encryption.ErrNoRecipients)
}

func TestParseRecipientsFile_HandlesCommentsAndBlanks(t *testing.T) {
	id := newIdentity(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "recipients.txt")
	content := "# managed by helm\n\n" + id.Recipient().String() + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	recipients, err := encryption.ParseRecipientsFile(path)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestParseRecipientsFile_Missing(t *testing.T) {
	_, err := encryption.ParseRecipientsFile(filepath.Join(t.TempDir(), "missing.txt"))
	require.Error(t, err)
}

func TestParseRecipientsFile_EmptyReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(path, []byte("# only comments\n\n"), 0o600))

	_, err := encryption.ParseRecipientsFile(path)
	require.ErrorIs(t, err, encryption.ErrNoRecipients)
}

func TestEncryptFileInPlace_RoundTrip(t *testing.T) {
	id := newIdentity(t)
	dir := t.TempDir()
	archive := filepath.Join(dir, "project-42.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte(sampleArchive), 0o600))

	require.NoError(t, encryption.EncryptFileInPlace(archive, []age.Recipient{id.Recipient()}, false))

	// File should still exist at the same path with same name.
	encData, err := os.ReadFile(archive) //nolint:gosec // test fixture
	require.NoError(t, err)
	require.NotEqual(t, sampleArchive, string(encData), "file should be encrypted, not plaintext")

	// Decrypt and verify content matches.
	r, err := age.Decrypt(bytes.NewReader(encData), id)
	require.NoError(t, err)
	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, sampleArchive, string(plaintext))
}

func TestEncryptFileInPlace_ArmorRoundTrip(t *testing.T) {
	id := newIdentity(t)
	dir := t.TempDir()
	archive := filepath.Join(dir, "project-43.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte(sampleArchive), 0o600))

	require.NoError(t, encryption.EncryptFileInPlace(archive, []age.Recipient{id.Recipient()}, true))

	encData, err := os.ReadFile(archive) //nolint:gosec // test fixture
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(encData), "-----BEGIN AGE ENCRYPTED FILE-----"),
		"armored output should start with PEM header, got %q", string(encData[:min(60, len(encData))]))

	armorR := armor.NewReader(bytes.NewReader(encData))
	r, err := age.Decrypt(armorR, id)
	require.NoError(t, err)
	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, sampleArchive, string(plaintext))
}

func TestEncryptFileInPlace_MultipleRecipients(t *testing.T) {
	id1 := newIdentity(t)
	id2 := newIdentity(t)
	dir := t.TempDir()
	archive := filepath.Join(dir, "project-44.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte(sampleArchive), 0o600))

	require.NoError(t, encryption.EncryptFileInPlace(archive,
		[]age.Recipient{id1.Recipient(), id2.Recipient()}, false))

	encData, err := os.ReadFile(archive) //nolint:gosec // test fixture
	require.NoError(t, err)

	// Each recipient must be able to decrypt independently.
	for _, id := range []*age.X25519Identity{id1, id2} {
		r, err := age.Decrypt(bytes.NewReader(encData), id)
		require.NoError(t, err)
		plaintext, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, sampleArchive, string(plaintext))
	}
}

func TestEncryptFileInPlace_NoRecipients(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "project-45.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte(sampleArchive), 0o600))

	err := encryption.EncryptFileInPlace(archive, nil, false)
	require.ErrorIs(t, err, encryption.ErrNoRecipients)

	// File must be untouched.
	data, err := os.ReadFile(archive) //nolint:gosec // test fixture
	require.NoError(t, err)
	require.Equal(t, sampleArchive, string(data))
}

func TestEncryptFileInPlace_MissingInput(t *testing.T) {
	id := newIdentity(t)
	err := encryption.EncryptFileInPlace(filepath.Join(t.TempDir(), "nope.tar.gz"),
		[]age.Recipient{id.Recipient()}, false)
	require.Error(t, err)
	require.True(t, errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "open archive"),
		"want open-archive error, got %v", err)
}

func TestEncryptFileInPlace_LeavesNoTempOnSuccess(t *testing.T) {
	id := newIdentity(t)
	dir := t.TempDir()
	archive := filepath.Join(dir, "project-46.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte(sampleArchive), 0o600))

	require.NoError(t, encryption.EncryptFileInPlace(archive, []age.Recipient{id.Recipient()}, false))

	_, err := os.Stat(archive + ".age.tmp")
	require.True(t, os.IsNotExist(err), "temp file should not remain after success")
}
