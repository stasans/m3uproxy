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
