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
	"testing"
)

func TestParseTvgTag(t *testing.T) {
	data := "-1 tvg-id=\"Channel 1\" tvg-logo=\"logo1.png\",Channel 1"

	expectedTvgID := "Channel 1"
	expectedTvgLogo := "logo1.png"
	tags := ParseTVGTags(data[2:])
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
