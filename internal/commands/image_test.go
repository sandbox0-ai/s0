package commands

import (
	"bytes"
	"testing"
)

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
