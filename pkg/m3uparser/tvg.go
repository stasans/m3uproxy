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

type M3UTvgTag struct {
	Tag   string
	Value string
}

type M3UTvgTags []M3UTvgTag

func ParseTVGTags(data string) M3UTvgTags {

	inKey := true
	var key string
	var value string
	var tags []M3UTvgTag
	for token := range data {
		if inKey && (data[token] == ' ' || data[token] == '=') {
			continue
		}
		if inKey && data[token] == ',' {
			// Break, we reached the end of the tags
			break
		}
		if data[token] == '"' {
			if !inKey {
				tags = append(tags, M3UTvgTag{Tag: key, Value: value})
				key = ""
				value = ""
			}
			inKey = !inKey
			continue
		}
		if inKey {
			key += string(data[token])
		} else {
			value += string(data[token])
		}
	}

	return tags
}

func (tag *M3UTvgTag) String() string {
	return tag.Tag + "=\"" + tag.Value + "\""
}

func (tags M3UTvgTags) GetValue(tag string) string {
	for _, t := range tags {
		if t.Tag == tag {
			return t.Value
		}
	}
	return ""
}

func (tags M3UTvgTags) String() string {
	var result string
	for _, tag := range tags {
		result += tag.String() + " "
	}
	return result
}
