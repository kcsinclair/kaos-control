// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFile_CreatesWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	fresh := []byte(genStart + "\nHELLO\n" + genEnd + "\n")

	fw, err := writeFile(path, fresh, false)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	if !fw.Created || !fw.Changed {
		t.Errorf("expected Created and Changed, got %+v", fw)
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(fresh) {
		t.Errorf("file content: got %q want %q", got, fresh)
	}
}

func TestWriteFile_SurgicalRefresh_NoForceNeeded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	initial := []byte("prose above\n\n" + genStart + "\nOLD\n" + genEnd + "\n\nprose below\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}

	fresh := []byte(genStart + "\nNEW\n" + genEnd + "\n")
	fw, err := writeFile(path, fresh, false)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	if fw.Diff != "" {
		t.Errorf("expected no diff gate when markers intact, got: %s", fw.Diff)
	}
	if !fw.Changed {
		t.Error("expected Changed=true")
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "prose above") || !strings.Contains(string(got), "prose below") || !strings.Contains(string(got), "NEW") {
		t.Errorf("unexpected merged content: %s", got)
	}
}

func TestWriteFile_NoMarkers_RequiresForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	legacy := []byte("# Legacy hand-written file, no markers.\n")
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	fresh := []byte(genStart + "\nNEW\n" + genEnd + "\n")

	fw, err := writeFile(path, fresh, false)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	if fw.Diff == "" {
		t.Fatal("expected Diff to be set without force")
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(legacy) {
		t.Error("file should not have been written without force")
	}

	fw2, err := writeFile(path, fresh, true)
	if err != nil {
		t.Fatalf("writeFile with force: %v", err)
	}
	if fw2.Diff != "" {
		t.Errorf("expected no diff when force=true, got: %s", fw2.Diff)
	}
	got2, _ := os.ReadFile(path)
	if string(got2) != string(fresh) {
		t.Errorf("expected file replaced with force, got: %s", got2)
	}
}

func TestWriteFile_NoOpWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	fresh := []byte(genStart + "\nSAME\n" + genEnd + "\n")
	if err := os.WriteFile(path, fresh, 0o644); err != nil {
		t.Fatal(err)
	}

	fw, err := writeFile(path, fresh, false)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	if !fw.Skipped {
		t.Errorf("expected Skipped=true, got %+v", fw)
	}
}
