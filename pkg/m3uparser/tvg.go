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
