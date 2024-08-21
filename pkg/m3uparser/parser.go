package m3uparser

import (
	"bufio"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type M3UTag struct {
	Tag   string
	Value string
}

// M3UEntry represents a single entry in the M3U file.
type M3UEntry struct {
	URI       string            // The URI of the media file.
	Duration  int               // The duration of the media in seconds (if available).
	Title     string            // The title of the media (if available).
	keyValues map[string]string // Additional key-value pairs associated with the entry.
	Tags      []M3UTag          // Additional tags associated with the entry.
}

// M3UPlaylist represents the parsed M3U playlist.
type M3UPlaylist struct {
	Version int        // The version of the M3U (EXTM3U).
	Entries []M3UEntry // The list of media entries in the playlist.
	Tags    []M3UTag   // Additional tags associated with the entry.
}

const (
	// M3U8Version3 represents version 3 of the M3U8 format.
	M3U8Version3 = 3
)

var (
	M3U8Directives = []string{
		// M3U Extensions
		"EXTM3U",
		"EXTINF",
		"PLAYLIST",
		"EXTGRP",
		"EXTALB",
		"EXTART",
		"EXTGENRE",
		"EXTM3A",
		"EXTBYT",
		"EXTBIN",
		"EXTENC",
		"EXTIMG",
		// HLS M3U extensions
		"EXT-X-START",
		"EXT-X-INDEPENDENT-SEGMENTS",
		"EXT-X-PLAYLIST-TYPE",
		"EXT-X-TARGETDURATION",
		"EXT-X-VERSION",
		"EXT-X-MEDIA-SEQUENCE",
		"EXT-X-MEDIA",
		"EXT-X-STREAM-INF",
		"EXT-X-BYTERANGE",
		"EXT-X-DISCONTINUITY",
		"EXT-X-DISCONTINUITY-SEQUENCE",
		"EXT-X-GAP",
		"EXT-X-KEY",
		"EXT-X-MAP",
		"EXT-X-PROGRAM-DATE-TIME",
		"EXT-X-DATERANGE",
		"EXT-X-I-FRAMES-ONLY",
		"EXT-X-SESSION-DATA",
		"EXT-X-SESSION-KEY",
		"EXT-X-ENDLIST",
		// VLC M3U extensions
		"EXTVLCOPT",
		// Kodi M3U extensions
		"KODIPROP",
	}
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ParseM3UFile reads an M3U file and returns a parsed M3UPlaylist.
func ParseM3UFile(filePath string) (*M3UPlaylist, error) {

	var scanner *bufio.Scanner

	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		// Load content from URL
		resp, err := http.Get(filePath)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		scanner = bufio.NewScanner(resp.Body)

	} else {

		// Load content from local file
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		scanner = bufio.NewScanner(file)
	}

	playlist := &M3UPlaylist{
		Version: M3U8Version3, // Default M3U8 version
		Entries: make([]M3UEntry, 0),
		Tags:    make([]M3UTag, 0),
	}

	var currentEntry *M3UEntry
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			// Ignore empty lines and comments that aren't tags
			continue
		}

		if strings.HasPrefix(line, "#") {
			// Handle tags
			tag, err := parseTag(line)
			if err != nil {
				// Ignore invalid tags or comments
				continue
			}

			if !contains(M3U8Directives, tag.Tag) {
				// Ignore unknown tags
				continue
			}

			if tag.Tag == "EXTM3U" {

				playlist.Version = M3U8Version3
				continue
			}

			if tag.Tag == "EXTINF" {
				// Handle EXTINF tag
				currentEntry = &M3UEntry{
					keyValues: make(map[string]string),
					Tags:      []M3UTag{tag},
				}
				parts := strings.SplitN(line[8:], ",", 2)
				if len(parts) > 0 {
					currentEntry.Duration = parseDuration(parts[0])
					if currentEntry.Duration == -1 {
						// Extract the channel metadata, such as group-title, tvg-id, tvg-logo, etc

						properties := strings.TrimSpace(parts[0][len("-1"):])

						var key string
						var currValue string
						var inQuotes bool
						for i := 0; i < len(properties); i++ {

							if properties[i] == ' ' && key == "" {
								continue
							}

							if properties[i] == '"' {

								if inQuotes {
									currentEntry.keyValues[key] = currValue
									currValue = ""
									key = ""
									inQuotes = false
									continue
								}

								inQuotes = true
								continue

							}

							if properties[i] == '=' {
								key = currValue
								currValue = ""
								continue
							}

							currValue += string(properties[i])
						}

					}

				}
				if len(parts) > 1 {
					currentEntry.Title = parts[1]
				}
				continue
			}

			if tag.Tag == "EXT-X-STREAM-INF" {
				currentEntry = &M3UEntry{
					keyValues: make(map[string]string),
					Tags:      []M3UTag{tag}, // Add the EXT-X-STREAM-INF tag
				}
				continue
			}

			if currentEntry != nil {
				currentEntry.Tags = append(currentEntry.Tags, tag)
			} else {
				playlist.Tags = append(playlist.Tags, tag)
			}

			continue
		}

		// Handle URI (must be after EXTINF)
		if currentEntry != nil {
			currentEntry.URI = line
			playlist.Entries = append(playlist.Entries, *currentEntry)
			currentEntry = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return playlist, nil
}

// parseDuration parses the duration from the EXTINF tag.
func parseDuration(durationStr string) int {
	duration, err := strconv.Atoi(strings.TrimSpace(durationStr))
	if err != nil {
		return -1 // or handle error as required
	}
	return duration
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

func (entry *M3UEntry) Get(key string) string {
	return entry.keyValues[key]
}

func (entry *M3UEntry) Set(key, value string) {
	entry.keyValues[key] = value
}

func (entry *M3UEntry) GetTags() []M3UTag {
	return entry.Tags
}

func (entry *M3UEntry) GetURI() string {
	return entry.URI
}

func (entry *M3UEntry) GetDuration() int {
	return entry.Duration
}

func (entry *M3UEntry) GetTitle() string {
	return entry.Title
}

func (entry *M3UEntry) String() string {
	var result string
	for _, tag := range entry.Tags {
		result += "#" + tag.Tag + ":" + tag.Value + "\n"
	}
	result += entry.URI + "\n"
	return strings.Trim(result, "\n")
}

func (playlist *M3UPlaylist) GetVersion() int {
	return playlist.Version
}

func (playlist *M3UPlaylist) GetEntries() []M3UEntry {
	return playlist.Entries
}

func (playlist *M3UPlaylist) String() string {
	var result string
	result += "#EXTM3U\n"
	for _, tag := range playlist.Tags {
		result += "#" + tag.Tag + ":" + tag.Value + "\n"
	}
	for _, entry := range playlist.Entries {
		result += entry.String() + "\n"
	}
	return strings.Trim(result, "\n")
}
