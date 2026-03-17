package auth

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func generateBcryptHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}
	return string(hash)
}

func writeHTPasswdFile(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "htpasswd")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write htpasswd file: %v", err)
	}
	return path
}

func TestLoadHTPasswdFile_Valid(t *testing.T) {
	dir := t.TempDir()
	hash1 := generateBcryptHash(t, "password1")
	hash2 := generateBcryptHash(t, "password2")

	content := "# Comment line\n" +
		"user1:" + hash1 + "\n" +
		"\n" + // blank line
		"user2:" + hash2 + "\n"

	path := writeHTPasswdFile(t, dir, content)

	store, err := LoadHTPasswdFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("expected 2 entries, got %d", store.Count())
	}

	if !store.Verify("user1", "password1") {
		t.Error("user1/password1 should verify")
	}
	if !store.Verify("user2", "password2") {
		t.Error("user2/password2 should verify")
	}
}

func TestLoadHTPasswdFile_WrongPassword(t *testing.T) {
	dir := t.TempDir()
	hash := generateBcryptHash(t, "correct-password")
	path := writeHTPasswdFile(t, dir, "user1:"+hash+"\n")

	store, err := LoadHTPasswdFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.Verify("user1", "wrong-password") {
		t.Error("wrong password should not verify")
	}
}

func TestLoadHTPasswdFile_UnknownUser(t *testing.T) {
	dir := t.TempDir()
	hash := generateBcryptHash(t, "password")
	path := writeHTPasswdFile(t, dir, "user1:"+hash+"\n")

	store, err := LoadHTPasswdFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.Verify("nonexistent", "password") {
		t.Error("nonexistent user should not verify")
	}
}

func TestLoadHTPasswdFile_FileNotFound(t *testing.T) {
	_, err := LoadHTPasswdFile("/nonexistent/htpasswd")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadHTPasswdFile_Empty(t *testing.T) {
	dir := t.TempDir()
	path := writeHTPasswdFile(t, dir, "# only comments\n\n")

	_, err := LoadHTPasswdFile(path)
	if err == nil {
		t.Error("expected error for empty htpasswd file")
	}
}

func TestLoadHTPasswdFile_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeHTPasswdFile(t, dir, "no-colon-here\n")

	_, err := LoadHTPasswdFile(path)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestLoadHTPasswdFile_UnsupportedHash(t *testing.T) {
	dir := t.TempDir()
	// SHA1 format is not supported
	path := writeHTPasswdFile(t, dir, "user1:{SHA}W6ph5Mm5Pz8GgiULbPgzG37mj9g=\n")

	_, err := LoadHTPasswdFile(path)
	if err == nil {
		t.Error("expected error for unsupported hash format")
	}
}

func TestLoadHTPasswdFile_EmptyUsername(t *testing.T) {
	dir := t.TempDir()
	hash := generateBcryptHash(t, "password")
	path := writeHTPasswdFile(t, dir, ":"+hash+"\n")

	_, err := LoadHTPasswdFile(path)
	if err == nil {
		t.Error("expected error for empty username")
	}
}

func TestLoadHTPasswdFile_AllBcryptVariants(t *testing.T) {
	dir := t.TempDir()
	hash := generateBcryptHash(t, "password")

	// Test $2a$ variant (golang bcrypt uses $2a$)
	path := writeHTPasswdFile(t, dir, "user1:"+hash+"\n")

	store, err := LoadHTPasswdFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !store.Verify("user1", "password") {
		t.Error("bcrypt hash should verify")
	}
}
