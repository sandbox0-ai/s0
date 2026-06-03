package commands

import "testing"

func TestFilesystemSnapshotCommandRegistration(t *testing.T) {
	subcommands := map[string]bool{}
	for _, cmd := range filesystemSnapshotCmd.Commands() {
		subcommands[cmd.Name()] = true
	}

	for _, name := range []string{"list", "get", "create", "delete", "restore"} {
		if !subcommands[name] {
			t.Fatalf("expected filesystem snapshot subcommand %q to be registered", name)
		}
	}
}

func TestFilesystemSnapshotCommandFlags(t *testing.T) {
	if flag := filesystemSnapshotCreateCmd.Flags().Lookup("name"); flag == nil {
		t.Fatal("expected filesystem snapshot create --name flag")
	}
	if flag := filesystemSnapshotCreateCmd.Flags().Lookup("description"); flag == nil {
		t.Fatal("expected filesystem snapshot create --description flag")
	}
}
