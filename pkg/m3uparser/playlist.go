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
	"strings"
)

// M3UPlaylist represents the parsed M3U playlist.
type M3UPlaylist struct {
	Version int        // The version of the M3U (EXTM3U).
	Entries M3UEntries // The list of media entries in the playlist.
	Tags    M3UTags    // Additional tags associated with the entry.
}

func (playlist *M3UPlaylist) GetVersion() int {
	return playlist.Version
}

func (playlist *M3UPlaylist) GetEntries() M3UEntries {
	return playlist.Entries
}

func (playlist *M3UPlaylist) EntriesString() string {
	var result string
	for _, tag := range playlist.Tags {
		result += "#" + tag.Tag + ":" + tag.Value + "\n"
	}
	for _, entry := range playlist.Entries {
		result += entry.String() + "\n"
	}
	return strings.Trim(result, "\n")
}

func (playlist *M3UPlaylist) String() string {
	var result string
	result += "#EXTM3U\n"
	result += playlist.EntriesString()
	return result
}

func (playlist *M3UPlaylist) GetEntryByTitle(title string) *M3UEntry {
	for _, entry := range playlist.Entries {
		if entry.Title == title {
			return &entry
		}
	}
	return nil
}

func (playlist *M3UPlaylist) GetEntryByURI(uri string) *M3UEntry {
	for _, entry := range playlist.Entries {
		if entry.URI == uri {
			return &entry
		}
	}
	return nil
}

func (playlist *M3UPlaylist) GetStreamCount() int {
	return len(playlist.Entries)
}

func (playlist *M3UPlaylist) GetEntryByTvgTag(tag, value string) *M3UEntry {
	return playlist.Entries.GetByTvgTag(tag, value)
}

func (playlist *M3UPlaylist) GetEntryIndexByTvgTag(tag, value string) int {
	return playlist.Entries.GetIndexByTvgTag(tag, value)
}

func (playlist *M3UPlaylist) RemoveEntryByTvgTag(tag, value string) {
	playlist.Entries.RemoveByTvgTag(tag, value)
}
