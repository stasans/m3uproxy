package m3uparser

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

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
		// M3UPROXYHEADER
		"M3UPROXYHEADER",
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

func ParseM3UFile(filePath string) (*M3UPlaylist, error) {
	var reader io.ReadCloser

	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		// Load content from URL
		resp, err := http.Get(filePath)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		reader = resp.Body

	} else {

		// Load content from local file
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		reader = file
	}

	return DecodeFromReader(reader)
}

func readString(buf io.Reader) (string, error) {
	var content string

	b := make([]byte, 1)
	for {
		// Read line
		_, err := buf.Read(b)
		if err != nil {
			if err == io.EOF {
				return content, err
			}
			return "", err
		}
		if b[0] == '\n' {
			break
		}
		content += string(b)
	}

	return content, nil
}

func assertM3UHeader(buf io.Reader) error {
	// Read first line
	line, err := readString(buf)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(line, "#EXTM3U") {
		return errors.New("invalid M3U file")
	}

	return nil
}

func processLine(buf io.Reader) (M3UTag, string, error) {

	eof := false
	for {
		// Read line
		line, err := readString(buf)
		eof = err == io.EOF
		if err != nil && !eof {
			return M3UTag{}, "", err
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			// Ignore empty lines and comments that aren't tags
			if eof {
				return M3UTag{}, "", io.EOF
			}
			continue
		}

		if !strings.HasPrefix(line, "#") {
			return M3UTag{}, line, nil
		}

		tag, err := parseTag(line)
		if err != nil {
			// Ignore invalid tags or comments
			if eof {
				return M3UTag{}, "", io.EOF
			}
			continue
		}

		if !contains(M3U8Directives, tag.Tag) {
			// Ignore unknown tags
			if eof {
				return M3UTag{}, "", io.EOF
			}
			continue
		}

		return tag, "", err
	}
}

func DecodeFromReader(buf io.Reader) (*M3UPlaylist, error) {

	// Consume header
	err := assertM3UHeader(buf)
	if err != nil {
		return nil, err
	}

	playlist := &M3UPlaylist{
		Version: M3U8Version3, // Default M3U8 version
		Entries: make([]M3UEntry, 0),
		Tags:    make([]M3UTag, 0),
		Type:    "master",
	}

	// Read all content from buf
	var currentEntry *M3UEntry

	for {

		tag, line, err := processLine(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if line != "" {
			if currentEntry != nil {
				currentEntry.URI = line
				playlist.Entries = append(playlist.Entries, *currentEntry)
				currentEntry = nil
			} else {
				return nil, errors.New("invalid M3U file")
			}
			continue
		}

		if tag.Tag == "EXTINF" {
			// Handle EXTINF tag
			currentEntry = &M3UEntry{
				Tags: []M3UTag{tag},
			}
			parts := strings.SplitN(tag.Value, ",", 2)
			if len(parts) > 0 {
				currentEntry.Duration = parseDuration(parts[0])
				if currentEntry.Duration == -1 {
					currentEntry.TVGTags = ParseTVGTags(parts[0][2:])
				}
			} else {
				currentEntry.Duration = -1
			}
			if len(parts) > 1 {
				currentEntry.Title = parts[1]
			}
			continue
		}

		if tag.Tag == "EXT-X-STREAM-INF" {
			currentEntry = &M3UEntry{
				Tags: []M3UTag{tag}, // Add the EXT-X-STREAM-INF tag
			}
			continue
		}

		if currentEntry != nil {
			currentEntry.Tags = append(currentEntry.Tags, tag)
		} else {

			if tag.Tag == "EXT-X-INDEPENDENT-SEGMENTS" {
				playlist.Type = "master"
			} else if tag.Tag == "EXT-X-MEDIA-SEQUENCE" {
				playlist.Type = "media"
			}

			playlist.Tags = append(playlist.Tags, tag)
		}

	}

	if playlist.Version == 0 {
		return nil, errors.New("invalid M3U file")
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
