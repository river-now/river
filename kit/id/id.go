package id

import (
	"encoding/base64"
	"strings"

	"github.com/river-now/river/kit/bytesutil"
)

// New generates a cryptographically random string ID of length idLen,
// consisting of mixed-case alphanumeric characters (A-Z, a-z, 0-9),
// with no special characters or padding. The idLen parameter must be
// between 0 and 255 inclusive.
func New(idLen uint8) (string, error) {

	// if zero length, return empty string
	if idLen == 0 {
		return "", nil
	}

	l := int(idLen)

	// get random bytes
	bytes, err := bytesutil.Random(l)
	if err != nil {
		return "", err
	}

	// encode as unpadded, url-safe base64
	encoded := base64.RawURLEncoding.EncodeToString(bytes)

	// use a strings.Builder and set capacity to idLen
	result := strings.Builder{}
	result.Grow(l)

	// iterate over encoded bytes, filtering out non-alphanumeric characters
	for _, ch := range encoded {
		if ch != '-' && ch != '_' {
			result.WriteRune(ch)
			if result.Len() == l {
				break
			}
		}
	}

	// if len is less than idLen, get another ID and append to result
	// extremely unlikely this would ever happen, if even possible
	if result.Len() < l {
		supplIdLen := uint8(l - result.Len())
		suppl, err := New(supplIdLen)
		if err != nil {
			return "", err
		}
		result.WriteString(suppl)
	}

	// return the final id
	return result.String(), nil
}

// NewMulti generates count number of cryptographically random string IDs of length idLen,
// consisting of mixed-case alphanumeric characters (A-Z, a-z, 0-9), with no special
// characters or padding. The idLen parameter must be between 0 and 255 inclusive.
func NewMulti(idLen uint8, count uint8) ([]string, error) {
	ids := make([]string, count)
	for i := uint8(0); i < count; i++ {
		id, err := New(idLen)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}
