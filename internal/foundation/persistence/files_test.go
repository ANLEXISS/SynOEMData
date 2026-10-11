package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceSyncedReplacesStateFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "state.json")
	if err := os.WriteFile(destination, []byte("old\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceSynced(dir, "state.json", ".state-*.tmp", []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new\n" {
		t.Fatalf("replaced state=%q, want %q", body, "new\n")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Fatalf("temporary file remained after replacement: %v", entries)
	}
}

func TestReplaceSyncedReportsMissingDirectory(t *testing.T) {
	err := ReplaceSynced(filepath.Join(t.TempDir(), "missing"), "state.json", ".state-*.tmp", []byte("{}\n"))
	if err == nil {
		t.Fatal("expected missing directory error")
	}
}

func TestAppendSyncedAndWriteSynced(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "journal.jsonl")
	if err := AppendSynced(journal, []byte("one\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := AppendSynced(journal, []byte("two\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "one\ntwo\n" {
		t.Fatalf("journal=%q", body)
	}

	segment := filepath.Join(dir, "segment.jsonl")
	if err := WriteSynced(segment, []byte("segment\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(segment)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "segment\n" {
		t.Fatalf("segment=%q", body)
	}
	if err := SyncDir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestDurableFileHelpersReportFilesystemErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing", "file")
	if err := AppendSynced(missing, []byte("x"), 0o600); err == nil {
		t.Fatal("append should fail for a missing parent directory")
	}
	if err := WriteSynced(missing, []byte("x"), 0o600); err == nil {
		t.Fatal("write should fail for a missing parent directory")
	}
	if err := SyncDir(missing); err == nil {
		t.Fatal("directory sync should fail for a missing path")
	}
}
