package m3uparser

import (
	"errors"
	"io"
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

func (entry *M3UEntry) WriteTo(w io.Writer) (int64, error) {

	n := 0
	for _, tag := range entry.Tags {
		nBytes, _ := w.Write([]byte("#" + tag.Tag + ":" + tag.Value + "\n"))
		n += nBytes
	}
	nBytes, _ := w.Write([]byte(entry.URI + "\n"))
	n += nBytes
	return int64(n), nil
}

// parseTag parses a line that starts with '#' and extracts the tag name and value.
func parseTag(line string) (M3UTag, error) {
	if strings.HasPrefix(line, "#EXTM3U") {
		return M3UTag{"EXTM3U", ""}, nil
	}

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

func (tags M3UTags) Exist(tag string) bool {
	for _, tags := range tags {
		if tags.Tag == tag {
			return true
		}
	}
	return false
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
