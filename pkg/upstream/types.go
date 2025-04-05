package upstream

import (
	"github.com/valyala/fasthttp"
)

type UpstreamConnection struct {
	client  *fasthttp.Client
	headers map[string]string
}
