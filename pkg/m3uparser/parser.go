package m3uparser

import (
	"bufio"
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

var M3U8DirectivesMap = func() map[string]struct{} {
	m := make(map[string]struct{}, len(M3U8Directives))
	for _, directive := range M3U8Directives {
		m[directive] = struct{}{}
	}
	return m
}()

func contains(item string) bool {
	_, exists := M3U8DirectivesMap[item]
	return exists
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

func readString(buf *bufio.Reader) (string, error) {
	line, err := buf.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), err
}

func assertM3UHeader(buf *bufio.Reader) error {
	line, err := readString(buf)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "#EXTM3U") {
		return errors.New("invalid M3U file")
	}
	return nil
}

func processLine(buf *bufio.Reader) (M3UTag, string, error) {
	for {
		line, err := readString(buf)
		if err != nil && line == "" {
			if err == io.EOF {
				return M3UTag{}, "", io.EOF
			}
			return M3UTag{}, "", err
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 || !strings.HasPrefix(line, "#") {
			// Return non-tag lines as URIs
			return M3UTag{}, line, nil
		}

		tag, err := parseTag(line)
		if err == nil && contains(tag.Tag) {
			return tag, "", nil
		}
	}
}

func DecodeFromReader(r io.Reader) (*M3UPlaylist, error) {
	buf := bufio.NewReader(r)

	// Validate header
	if err := assertM3UHeader(buf); err != nil {
		return nil, err
	}

	playlist := &M3UPlaylist{
		Version: M3U8Version3,
		Entries: make([]M3UEntry, 0),
		Tags:    make([]M3UTag, 0),
		Type:    "master",
	}

	var currentEntry *M3UEntry

	for {
		tag, line, err := processLine(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
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

		switch tag.Tag {
		case "EXTINF":
			currentEntry = &M3UEntry{
				Tags: []M3UTag{tag},
			}
			parts := strings.SplitN(tag.Value, ",", 2)
			if len(parts) > 0 {
				currentEntry.Duration = parseDuration(parts[0])
				if currentEntry.Duration == -1 {
					currentEntry.ExtInfTags = ExtractExtinfTags(parts[0][2:])
				}
			} else {
				currentEntry.Duration = -1
			}
			if len(parts) > 1 {
				currentEntry.Title = parts[1]
			}
		case "EXT-X-STREAM-INF":
			currentEntry = &M3UEntry{
				Tags: []M3UTag{tag},
			}
		default:
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
