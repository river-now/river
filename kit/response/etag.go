package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/river-now/river/kit/cryptoutil"
)

func (res *Response) SetETag(etag string) *Response {
	res.SetHeader("ETag", etag)
	return res
}

// Only responds true for GET and HEAD requests with exact matches of
// the provided ETag to an If-None-Match header value.
func ShouldReturn304Conservative(r *http.Request, etag string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	if etag == "" {
		return false
	}

	match := r.Header.Get("If-None-Match")

	if match == "" {
		return false
	}

	if etag[0] != '"' {
		etag = fmt.Sprintf(`"%s"`, etag)
	}

	for tag := range strings.SplitSeq(match, ",") {
		if strings.TrimPrefix(strings.TrimSpace(tag), "W/") == etag {
			return true
		}
	}

	return false
}

func ToQuotedSha256Etag(data []byte) string {
	return fmt.Sprintf(`"%x"`, cryptoutil.Sha256Hash(data))
}
