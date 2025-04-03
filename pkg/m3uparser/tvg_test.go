package m3uparser

import (
	"testing"
)

func TestParseTvgTag(t *testing.T) {
	data := "-1 tvg-id=\"Channel 1\" tvg-logo=\"logo1.png\",Channel 1"

	expectedTvgID := "Channel 1"
	expectedTvgLogo := "logo1.png"
	tags := ExtractExtinfTags(data[2:])
	if len(tags) != 2 {
		t.Errorf("Unexpected number of tags. Expected: 2, Got: %d", len(tags))
	}

	if tags[0].Tag != "tvg-id" || tags[0].Value != expectedTvgID {
		t.Errorf("Unexpected tag. Expected: tvg-id=%s, Got: %s=%s", expectedTvgID, tags[0].Tag, tags[0].Value)
	}

	if tags[1].Tag != "tvg-logo" || tags[1].Value != expectedTvgLogo {
		t.Errorf("Unexpected tag. Expected: tvg-logo=%s, Got: %s=%s", expectedTvgLogo, tags[1].Tag, tags[1].Value)
	}

	if tags.GetValue("tvg-id") != expectedTvgID {
		t.Errorf("Unexpected tag value. Expected: %s, Got: %s", expectedTvgID, tags.GetValue("tvg-id"))
	}
}
