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
package stream

import (
	"log"
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

var supportedMediaTypes = []contenttype.MediaType{
	contenttype.NewMediaType("application/vnd.apple.mpegurl"),
	contenttype.NewMediaType("application/x-mpegurl"),
	contenttype.NewMediaType("audio/x-mpegurl"),
	contenttype.NewMediaType("audio/mpeg"),
	contenttype.NewMediaType("audio/aacp"),
	contenttype.NewMediaType("audio/aac"),
	contenttype.NewMediaType("audio/mp4"),
	contenttype.NewMediaType("audio/mp3"),
	contenttype.NewMediaType("audio/ac3"),
	contenttype.NewMediaType("audio/x-aac"),
	contenttype.NewMediaType("video/mp2t"),
	contenttype.NewMediaType("video/m2ts"),
	contenttype.NewMediaType("video/mp4"),
	contenttype.NewMediaType("binary/octet-stream"),
	contenttype.NewMediaType("application/dash+xml"),
}

func NewStream(
	m3u m3uparser.M3UEntry,
	headers map[string]string,
	httpProxy string,
	forceKodiHeaders bool,
	radio bool,
	transport *http.Transport,
	disableRemap bool,
	timeout int,
) Stream {

	resp, err := executeRequest("GET", m3u.URI, transport, headers, timeout)
	if err != nil {
		return nil
	}

	resp.Body.Close()

	if resp.Request.URL.String() != m3u.URI {
		m3u.URI = resp.Request.URL.String()
	}

	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := contenttype.GetAcceptableMediaTypeFromHeader(ct, supportedMediaTypes)
	if err != nil {
		log.Printf("Error getting media type: %s\n", err)
		return nil
	}

	switch {
	case mediaType.Subtype == "dash+xml":
		return &MPDStream{
			BaseStream: BaseStream{
				m3u:              m3u,
				headers:          headers,
				httpProxy:        httpProxy,
				forceKodiHeaders: forceKodiHeaders,
				radio:            radio,
				transport:        transport,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}
	default:
		return &M3UStream{
			BaseStream: BaseStream{
				m3u:              m3u,
				headers:          headers,
				httpProxy:        httpProxy,
				forceKodiHeaders: forceKodiHeaders,
				radio:            radio,
				transport:        transport,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}
	}
}

func (stream *BaseStream) HealthCheck(timeout int) error {
	resp, err := executeRequest("GET", stream.m3u.URI, stream.transport, stream.headers, timeout)
	if err != nil {
		return err
	}
	resp.Body.Close()

	stream.mux.Lock()
	if resp.Request.URL.String() != stream.m3u.URI {
		stream.m3u.URI = resp.Request.URL.String()
	}
	stream.mux.Unlock()

	_, err = verifyStream(stream.m3u.URI, stream.transport, stream.headers, timeout)

	stream.mux.Lock()
	stream.active = err == nil
	stream.mux.Unlock()

	if err != nil {
		log.Printf("Stream %s is not healthy: %s\n", stream.m3u.Title, err)
	}
	return err
}

func (stream *BaseStream) Active() bool {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.active
}

func (stream *BaseStream) MediaType() contenttype.MediaType {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.mediaType
}

func (stream *BaseStream) MediaName() string {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.m3u.Title
}

func (stream *BaseStream) M3UTags() m3uparser.M3UTags {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.m3u.Tags
}

func (stream *BaseStream) IsRadio() bool {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.radio
}
