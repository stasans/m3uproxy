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
package streamstore_test

import (
	"testing"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/streamstore"
)

func TestLoadPlaylist(t *testing.T) {
	// Create a sample M3U playlist
	playlist := &m3uparser.M3UPlaylist{
		Entries: []m3uparser.M3UEntry{
			{Title: "Channel 1", URI: "http://example.com/channel1"},
			{Title: "Channel 2", URI: "http://example.com/channel2"},
			{Title: "Channel 3", URI: "http://example.com/channel3"},
		},
	}

	// Load the playlist into the channel store
	err := streamstore.LoadPlaylist(playlist)
	if err != nil {
		t.Errorf("Failed to load playlist: %v", err)
	}

	// Verify that the number of channels is correct
	expectedChannelCount := 3
	if streamstore.GetStreamCount() != expectedChannelCount {
		t.Errorf("Expected %d channels, but got %d", expectedChannelCount, streamstore.GetStreamCount())
	}
}

func TestSetDefaultTimeout(t *testing.T) {
	// Set a default timeout of 5 seconds
	streamstore.SetDefaultTimeout(5)

	// Get the default timeout from the channel store
	defaultTimeout := streamstore.GetDefaultTimeout()

	// Verify that the default timeout is correct
	expectedTimeout := 5
	if defaultTimeout != expectedTimeout {
		t.Errorf("Expected default timeout of %d seconds, but got %d seconds", expectedTimeout, defaultTimeout)
	}
}
