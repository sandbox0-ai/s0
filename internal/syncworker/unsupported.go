package syncworker

import (
	"archive/tar"
	"fmt"
	"io/fs"
	"strings"
)

const unsupportedTypePolicy = "s0 sync currently supports only regular files and directories"

type UnsupportedWorkspaceEntryError struct {
	Path      string
	EntryType string
}

func (e *UnsupportedWorkspaceEntryError) Error() string {
	return fmt.Sprintf("%s; workspace path %q is %s", unsupportedTypePolicy, strings.TrimSpace(e.Path), strings.TrimSpace(e.EntryType))
}

type UnsupportedRemoteChangeError struct {
	Path      string
	EventType string
	EntryKind string
}

func (e *UnsupportedRemoteChangeError) Error() string {
	path := strings.TrimSpace(e.Path)
	eventType := strings.TrimSpace(e.EventType)
	entryKind := strings.TrimSpace(e.EntryKind)
	switch {
	case entryKind != "" && eventType != "":
		return fmt.Sprintf("%s; remote change %q at %q uses unsupported entry kind %q", unsupportedTypePolicy, eventType, path, entryKind)
	case eventType != "":
		return fmt.Sprintf("%s; remote change %q at %q is unsupported", unsupportedTypePolicy, eventType, path)
	default:
		return fmt.Sprintf("%s; remote change at %q is unsupported", unsupportedTypePolicy, path)
	}
}

type UnsupportedBootstrapEntryError struct {
	Path      string
	EntryType string
}

func (e *UnsupportedBootstrapEntryError) Error() string {
	return fmt.Sprintf("%s; bootstrap archive entry %q is %s", unsupportedTypePolicy, strings.TrimSpace(e.Path), strings.TrimSpace(e.EntryType))
}

func unsupportedWorkspaceEntryError(path string, mode fs.FileMode) error {
	return &UnsupportedWorkspaceEntryError{
		Path:      path,
		EntryType: workspaceEntryType(mode),
	}
}

func unsupportedRemoteChangeError(path, eventType, entryKind string) error {
	return &UnsupportedRemoteChangeError{
		Path:      path,
		EventType: eventType,
		EntryKind: entryKind,
	}
}

func unsupportedBootstrapEntryError(path string, typeflag byte) error {
	return &UnsupportedBootstrapEntryError{
		Path:      path,
		EntryType: archiveEntryType(typeflag),
	}
}

func workspaceEntryType(mode fs.FileMode) string {
	switch {
	case mode&fs.ModeSymlink != 0:
		return "symlink"
	case mode&fs.ModeNamedPipe != 0:
		return "named pipe"
	case mode&fs.ModeSocket != 0:
		return "socket"
	case mode&fs.ModeDevice != 0 && mode&fs.ModeCharDevice != 0:
		return "character device"
	case mode&fs.ModeDevice != 0:
		return "block device"
	case mode&fs.ModeIrregular != 0:
		return "irregular file"
	default:
		return "unsupported entry"
	}
}

func archiveEntryType(typeflag byte) string {
	switch typeflag {
	case tar.TypeSymlink:
		return "symlink"
	case tar.TypeLink:
		return "hard link"
	case tar.TypeChar:
		return "character device"
	case tar.TypeBlock:
		return "block device"
	case tar.TypeFifo:
		return "named pipe"
	case tar.TypeXGlobalHeader, tar.TypeXHeader:
		return "extended header"
	default:
		return fmt.Sprintf("unsupported tar entry type %q", typeflag)
	}
}
