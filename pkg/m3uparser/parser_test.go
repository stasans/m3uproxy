/*
Copyright © 2024 Alexandre Pires

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package m3uparser

import (
	"io"
	"os"
	"testing"
)

func TestParseM3UFile0(t *testing.T) {
	filePath := "../../tests/test0.m3u8"
	playlist, err := ParseM3UFile(filePath)
	if err != nil {
		t.Errorf("Failed to parse M3U file: %v", err)
	}

	// Assert that the playlist is not nil
	if playlist == nil {
		t.Error("Playlist is nil")
	}

	// Assert that the version is correct
	expectedVersion := 3
	if playlist != nil && playlist.Version != expectedVersion {
		t.Errorf("Unexpected version. Expected: %d, Got: %d", expectedVersion, playlist.Version)
	}

	// Assert that the number of entries is correct
	expectedNumEntries := 3
	if len(playlist.Entries) != expectedNumEntries {
		t.Errorf("Unexpected number of entries. Expected: %d, Got: %d", expectedNumEntries, len(playlist.Entries))
	}

	// Assert that the first entry is correct
	expectedURI := "http://example.com/channel1.m3u8"
	expectedDuration := -1
	expectedTitle := "Channel 1"
	if playlist.Entries[0].URI != expectedURI || playlist.Entries[0].Duration != expectedDuration || playlist.Entries[0].Title != expectedTitle {
		t.Errorf("Unexpected entry. Expected: %s, %d, %s, Got: %s, %d, %s", expectedURI, expectedDuration, expectedTitle, playlist.Entries[0].URI, playlist.Entries[0].Duration, playlist.Entries[0].Title)
	}

	// Assert that first entry has the correct tags
	expectedTags := []M3UTag{
		{"EXTINF", "-1 group-title=\"TV\" tvg-id=\"Channel 1\" tvg-logo=\"logo1.png\",Channel 1"},
		{"EXTVLCOPT", "http-user-agent=Firefox"},
		{"EXTVLCOPT", "http-referrer=test"},
		{"KODIPROP", "inputstream=inputstream.adaptive"},
		{"KODIPROP", "inputstream.adaptive.manifest_ty"},
	}

	if len(playlist.Entries[0].Tags) != len(expectedTags) {
		t.Errorf("Unexpected number of tags. Expected: %d, Got: %d", len(expectedTags), len(playlist.Entries[0].Tags))
	}

	for i, tag := range playlist.Entries[0].Tags {
		if playlist.Entries[0].Tags[i].Tag != tag.Tag || playlist.Entries[0].Tags[i].Value != tag.Value {
			t.Errorf("Unexpected tag. Expected: %s=%s, Got: %s=%s", tag.Tag, tag.Value, playlist.Entries[0].Tags[i].Tag, playlist.Entries[0].Tags[i].Value)
		}
	}
}

func TestParseM3UFile1(t *testing.T) {
	filePath := "../../tests/test1.m3u8"
	playlist, err := ParseM3UFile(filePath)
	if err != nil {
		t.Errorf("Failed to parse M3U file: %v", err)
	}

	// Assert that the playlist is not nil
	if playlist == nil {
		t.Error("Playlist is nil")
	}

	// Assert that the version is correct
	expectedVersion := 3
	if playlist.Version != expectedVersion {
		t.Errorf("Unexpected version. Expected: %d, Got: %d", expectedVersion, playlist.Version)
	}

	// Assert that the number of entries is correct
	expectedNumEntries := 4
	if len(playlist.Entries) != expectedNumEntries {
		t.Errorf("Unexpected number of entries. Expected: %d, Got: %d", expectedNumEntries, len(playlist.Entries))
	}

	// Assert that all entries have the correct URI
	expectedURIs := []string{
		"edge_servers/720_passthrough/chunks.m3u8",
		"edge_servers/480p/chunks.m3u8",
		"edge_servers/360p/chunks.m3u8",
		"edge_servers/240p/chunks.m3u8",
	}

	if len(playlist.Entries) != len(expectedURIs) {
		t.Errorf("Unexpected number of URIs. Expected: %d, Got: %d", len(expectedURIs), len(playlist.Entries))
	}

	for i, uri := range expectedURIs {
		if playlist.Entries[i].URI != uri {
			t.Errorf("Unexpected URI. Expected: %s, Got: %s", uri, playlist.Entries[i].URI)
		}
	}

	// Read the test file and compare the results
	// Load content from local file
	file, err := os.Open(filePath)
	if err != nil {
		t.Errorf("Failed to open file: %v", err)
	}

	defer file.Close()

	// Read all content from the file
	content, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	// Assert that the content is the same
	expectedContent := string(content)
	if playlist.String() != expectedContent {
		t.Errorf("Unexpected content. Expected: %s, Got: %s", expectedContent, playlist.String())
	}
}

func TestParseM3UFile2(t *testing.T) {
	filePath := "../../tests/test2.m3u8"
	playlist, err := ParseM3UFile(filePath)
	if err != nil {
		t.Errorf("Failed to parse M3U file: %v", err)
	}

	// Assert that the playlist is not nil
	if playlist == nil {
		t.Error("Playlist is nil")
	}

	// Assert that the version is correct
	expectedVersion := 3
	if playlist.Version != expectedVersion {
		t.Errorf("Unexpected version. Expected: %d, Got: %d", expectedVersion, playlist.Version)
	}

	// Assert that the number of entries is correct
	expectedNumEntries := 6
	if len(playlist.Entries) != expectedNumEntries {
		t.Errorf("Unexpected number of entries. Expected: %d, Got: %d", expectedNumEntries, len(playlist.Entries))
	}

	// Assert that all entries have the correct URI
	expectedURIs := []string{
		"l_2309_1470371560_79529.ts",
		"l_2309_1470381560_79530.ts",
		"l_2309_1470391560_79531.ts",
		"l_2309_1470401560_79532.ts",
		"l_2309_1470411560_79533.ts",
		"l_2309_1470421560_79534.ts",
	}

	if len(playlist.Entries) != len(expectedURIs) {
		t.Errorf("Unexpected number of URIs. Expected: %d, Got: %d", len(expectedURIs), len(playlist.Entries))
	}

	for i, uri := range expectedURIs {
		if playlist.Entries[i].URI != uri {
			t.Errorf("Unexpected URI. Expected: %s, Got: %s", uri, playlist.Entries[i].URI)
		}
	}

	// Read the test file and compare the results
	// Load content from local file
	file, err := os.Open(filePath)
	if err != nil {
		t.Errorf("Failed to open file: %v", err)
	}

	defer file.Close()

	// Read all content from the file
	content, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	// Assert that the content is the same
	expectedContent := string(content)
	if playlist.String() != expectedContent {
		t.Errorf("Unexpected content. Expected: %s, Got: %s", expectedContent, playlist.String())
	}
}

func TestParseDuration(t *testing.T) {
	durationStr := "123"
	expectedDuration := 123
	duration := parseDuration(durationStr)
	if duration != expectedDuration {
		t.Errorf("Unexpected duration. Expected: %d, Got: %d", expectedDuration, duration)
	}
}

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
