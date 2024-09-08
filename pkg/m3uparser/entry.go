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
	"errors"
	"strings"
)

type M3UTag struct {
	Tag   string
	Value string
}

type M3UTags []M3UTag

// M3UEntry represents a single entry in the M3U file.
type M3UEntry struct {
	URI      string     `json:"uri"`      // The URI of the media.
	Duration int        `json:"duration"` // The duration of the media in seconds (if available).
	Title    string     `json:"title"`    // The title of the media (if available).
	Tags     M3UTags    `json:"tags"`     // Additional tags associated with the entry.
	TVGTags  M3UTvgTags `json:"tvg_tags"` // Additional tags associated with the entry.
}

type M3UEntries []M3UEntry

func (entry *M3UEntry) String() string {
	var result string
	for _, tag := range entry.Tags {
		result += "#" + tag.Tag + ":" + tag.Value + "\n"
	}
	result += entry.URI + "\n"
	return strings.Trim(result, "\n")
}

// parseTag parses a line that starts with '#' and extracts the tag name and value.
func parseTag(line string) (M3UTag, error) {
	line = strings.TrimPrefix(line, "#")
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 0 {
		return M3UTag{}, errors.New("invalid tag")
	}
	if len(parts[0]) == 0 {
		return M3UTag{}, errors.New("invalid tag")
	}
	if len(parts) == 1 {
		return M3UTag{parts[0], ""}, nil
	}
	return M3UTag{parts[0], parts[1]}, nil
}

func (tags M3UTags) GetValue(tag string) string {
	for _, tags := range tags {
		if tags.Tag == tag {
			return tags.Value
		}
	}
	return ""
}

func (entry *M3UEntry) SearchTags(tag string) []M3UTag {
	var result []M3UTag
	for _, t := range entry.Tags {
		if t.Tag == tag {
			result = append(result, t)
		}
	}
	return result
}

func (entry *M3UEntry) RemoveTags(tag string) {

	// first pass to count the number of tags to remove
	count := 0
	for _, t := range entry.Tags {
		if t.Tag == tag {
			count++
		}
	}

	// second pass to remove the tags
	for i := 0; i < count; i++ {
		for i, t := range entry.Tags {
			if t.Tag == tag {
				if i == len(entry.Tags)-1 {
					entry.Tags = entry.Tags[:i]
				} else if i == 0 {
					entry.Tags = entry.Tags[1:]
				} else {
					entry.Tags = append(entry.Tags[:i], entry.Tags[i+1:]...)
					break
				}
			}
		}
	}
}

func (entry *M3UEntry) AddTag(tag string, value string) {
	entry.Tags = append(entry.Tags, M3UTag{tag, value})
}

func (entry *M3UEntry) ClearTags() {
	tags := M3UTags{}
	for _, tag := range entry.Tags {
		if tag.Tag == "EXTINF" {
			tags = append(tags, tag)
		}
	}
	entry.Tags = tags
}

func (entries M3UEntries) SearchByTvgTag(tag string, value string) *M3UEntry {
	for _, entry := range entries {
		if entry.TVGTags.GetValue(tag) == value {
			return &entry
		}
	}
	return nil
}

func (entries M3UEntries) SearchIndexByTvgTag(tag string, value string) int {
	for i, entry := range entries {
		if entry.TVGTags.GetValue(tag) == value {
			return i
		}
	}
	return -1
}

func (entries M3UEntries) RemoveByTvgTag(tag string, value string) {
	for i, entry := range entries {
		if entry.TVGTags.GetValue(tag) == value {
			if i == len(entries)-1 {
				entries = entries[:i]
			} else if i == 0 {
				entries = entries[1:]
			} else {
				entries = append(entries[:i], entries[i+1:]...)
				break
			}
		}
	}
}
