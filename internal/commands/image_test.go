package commands

import (
	"bytes"
	"testing"

	"github.com/sandbox0-ai/s0/internal/client"
)

func TestImagePushReferencesPreferCompleteProviderReferences(t *testing.T) {
	t.Parallel()

	target, template := imagePushReferences(&client.RegistryCredentials{
		PushRegistry: "public.example.com/sandbox0/t-team",
		PullRegistry: "vpc.example.com/sandbox0/t-team",
		PushImage:    "public.example.com/sandbox0/t-team:probe-test-hash",
		PullImage:    "vpc.example.com/sandbox0/t-team:probe-test-hash",
	}, "probe:test")

	if got, want := target, "public.example.com/sandbox0/t-team:probe-test-hash"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
	if got, want := template, "vpc.example.com/sandbox0/t-team:probe-test-hash"; got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
}

func TestImagePushReferencesPreserveRegistryCompositionFallback(t *testing.T) {
	t.Parallel()

	target, template := imagePushReferences(&client.RegistryCredentials{
		PushRegistry: "public.example.com/t-team",
		PullRegistry: "vpc.example.com/t-team",
	}, "probe:test")

	if got, want := target, "public.example.com/t-team/probe:test"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
	if got, want := template, "vpc.example.com/t-team/probe:test"; got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
}

func TestWriteImagePushResultIncludesTemplateReferenceWhenRegistriesMatch(t *testing.T) {
	var output bytes.Buffer
	image := "registry.example.com/t-team/my-image:v1"

	writeImagePushResult(&output, image, image)

	want := "\nImage pushed successfully: " + image + "\n" +
		"Template image reference: " + image + "\n"
	if output.String() != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestWriteImagePushResultUsesPullReferenceWhenRegistriesDiffer(t *testing.T) {
	var output bytes.Buffer

	writeImagePushResult(
		&output,
		"push.example.com/t-team/my-image:v1",
		"pull.example.com/t-team/my-image:v1",
	)

	want := "\nImage pushed successfully: push.example.com/t-team/my-image:v1\n" +
		"Template image reference: pull.example.com/t-team/my-image:v1\n"
	if output.String() != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", output.String(), want)
	}
}
