package version

import (
	"strings"
	"testing"
)

func TestVersionDefaults(t *testing.T) {
	if Version == "" {
		t.Error("expected non-empty Version")
	}
	if Commit == "" {
		t.Error("expected non-empty Commit")
	}
	if Date == "" {
		t.Error("expected non-empty Date")
	}
}

func TestVersionInfo(t *testing.T) {
	info := Info()
	if !strings.Contains(info, "devgit version") {
		t.Errorf("expected 'devgit version' in Info, got: %s", info)
	}
	if !strings.Contains(info, "commit:") {
		t.Errorf("expected 'commit:' in Info, got: %s", info)
	}
	if !strings.Contains(info, "built:") {
		t.Errorf("expected 'built:' in Info, got: %s", info)
	}
	if !strings.Contains(info, "os/arch:") {
		t.Errorf("expected 'os/arch:' in Info, got: %s", info)
	}
}

func TestVersionShort(t *testing.T) {
	short := Short()
	if !strings.Contains(short, "devgit version "+Version) {
		t.Errorf("expected version %s in Short, got: %s", Version, short)
	}
}
