package commands

import "testing"

func TestVolumeFilesCommandRegistration(t *testing.T) {
	subcommands := map[string]bool{}
	for _, cmd := range volumeFilesCmd.Commands() {
		subcommands[cmd.Name()] = true
	}

	expected := []string{"ls", "cat", "stat", "mkdir", "rm", "mv", "clone", "upload", "download", "write", "watch"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

func TestVolumeFilesCommandFlags(t *testing.T) {
	if volumeFilesMkdirCmd.Flags().Lookup("parents") == nil {
		t.Fatalf("expected mkdir --parents flag")
	}
	if volumeFilesWriteCmd.Flags().Lookup("stdin") == nil {
		t.Fatalf("expected write --stdin flag")
	}
	if volumeFilesWriteCmd.Flags().Lookup("data") == nil {
		t.Fatalf("expected write --data flag")
	}
	if volumeFilesWatchCmd.Flags().Lookup("recursive") == nil {
		t.Fatalf("expected watch --recursive flag")
	}
	if volumeFilesCloneCmd.Flags().Lookup("mode") == nil {
		t.Fatalf("expected clone --mode flag")
	}
	if volumeFilesCloneCmd.Flags().Lookup("overwrite") == nil {
		t.Fatalf("expected clone --overwrite flag")
	}
	if volumeFilesCloneCmd.Flags().Lookup("parents") == nil {
		t.Fatalf("expected clone --parents flag")
	}
}

func TestParseVolumeFilesCloneMode(t *testing.T) {
	tests := []string{"", "cow_or_copy", "cow_required", "copy"}
	for _, mode := range tests {
		if _, err := parseVolumeFilesCloneMode(mode); err != nil {
			t.Fatalf("parseVolumeFilesCloneMode(%q) returned error: %v", mode, err)
		}
	}
	if _, err := parseVolumeFilesCloneMode("invalid"); err == nil {
		t.Fatalf("expected invalid mode to fail")
	}
}
