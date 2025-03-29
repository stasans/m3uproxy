package m3uparser

import (
	"testing"
)

func TestParseTag(t *testing.T) {
	line := "#EXTINF:123,Sample Title"
	expectedTagName := "EXTINF"
	expectedTagValue := "123,Sample Title"
	tag, err := parseTag(line)
	if err != nil {
		t.Errorf("Error parsing tag: %v", err)
	}

	if tag.Tag != expectedTagName || tag.Value != expectedTagValue {
		t.Errorf("Unexpected tag. Expected: %s=%s, Got: %s=%s", expectedTagName, expectedTagValue, tag.Tag, tag.Value)
	}
}

func TestParseTag_EmptyLine(t *testing.T) {
	line := ""
	_, err := parseTag(line)
	if err == nil {
		t.Error("Error should not be nil")
	}
}
