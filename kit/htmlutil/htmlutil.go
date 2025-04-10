package htmlutil

import (
	"fmt"
	"html/template"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/sjc5/river/kit/bytesutil"
	"github.com/sjc5/river/kit/cryptoutil"
	"github.com/sjc5/river/kit/id"
)

// Element is a structure for defining HTML elements. For both Attributes and
// TrustedAttributes, consumers are in charge of making sure keys are safe.
// This means that if you are using user input as a key, you need to escape
// it before passing it to this package. This package will only escape the
// values of the Attributes map (and not the TrustedAttributes map).
type Element struct {
	Tag        string            `json:"tag,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	// TrustedAttributes are attributes whose values are safe to render as-is.
	// This means you either know the value is safe (because you control it)
	// or you have escaped it already. If you aren't sure, use the Attributes
	// field instead.
	TrustedAttributes map[string]string `json:"safeAttributes,omitempty"`
	BooleanAttributes []string          `json:"booleanAttributes,omitempty"`
	InnerHTML         template.HTML     `json:"innerHTML,omitempty"`
	SelfClosing       bool              `json:"-"`
}

var (
	// see https://html.spec.whatwg.org/multipage/syntax.html#void-elements
	// If you need something to self-close something that isn't on this list, set the SelfClosing field to true
	selfClosingTags = []string{"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr"}
)

func AddSha256HashInline(el *Element, includeConvenienceIntegrityAttribute bool) (string, error) {
	if el.TrustedAttributes == nil {
		el.TrustedAttributes = make(map[string]string)
	}
	sha256Hash := cryptoutil.Sha256Hash([]byte(el.InnerHTML))
	sha256HashBase64 := bytesutil.ToBase64(sha256Hash[:])
	if includeConvenienceIntegrityAttribute {
		el.TrustedAttributes["integrity"] = "sha256-" + sha256HashBase64
	}
	return sha256HashBase64, nil
}

func AddSha256HashExternal(el *Element, externalSha256Hash string) (string, error) {
	if el.TrustedAttributes == nil {
		el.TrustedAttributes = make(map[string]string)
	}
	if externalSha256Hash == "" {
		return "", fmt.Errorf("no sha256 hash provided for external resource")
	}
	el.TrustedAttributes["integrity"] = "sha256-" + externalSha256Hash
	return externalSha256Hash, nil
}

func AddNonce(el *Element, len uint8) (string, error) {
	if el.TrustedAttributes == nil {
		el.TrustedAttributes = make(map[string]string)
	}
	if len == 0 {
		len = 16
	}
	nonce, err := id.New(len)
	if err != nil {
		return "", fmt.Errorf("could not generate nonce: %v", err)
	}
	el.TrustedAttributes["nonce"] = nonce
	return nonce, nil
}

func RenderElement(el *Element) (template.HTML, error) {
	var htmlBuilder strings.Builder

	err := RenderElementToBuilder(el, &htmlBuilder)
	if err != nil {
		return "", fmt.Errorf("could not render element: %v", err)
	}

	return template.HTML(htmlBuilder.String()), nil
}

func RenderElementToBuilder(el *Element, htmlBuilder *strings.Builder) error {
	isSelfClosing := slices.Contains(selfClosingTags, el.Tag) || el.SelfClosing

	attributes := EscapeAllIntoNewMap(el)
	hasAttributes := len(attributes) > 0
	hasBooleanAttributes := len(el.BooleanAttributes) > 0

	htmlBuilder.WriteString("<")
	htmlBuilder.WriteString(el.Tag)

	if hasAttributes {
		keys := slices.Collect(maps.Keys(attributes))
		sort.Strings(keys)
		for _, key := range keys {
			writeAttribute(htmlBuilder, key, attributes[key])
		}
	}

	if hasBooleanAttributes {
		for _, key := range el.BooleanAttributes {
			htmlBuilder.WriteString(" ")
			htmlBuilder.WriteString(key)
		}
	}

	if isSelfClosing {
		htmlBuilder.WriteString(" />")
	} else {
		htmlBuilder.WriteString(">")
		htmlBuilder.WriteString(string(el.InnerHTML))
		htmlBuilder.WriteString("</")
		htmlBuilder.WriteString(el.Tag)
		htmlBuilder.WriteString(">")
	}

	return nil
}

func writeAttribute(htmlBuilder *strings.Builder, key, value string) {
	htmlBuilder.WriteString(" ")
	htmlBuilder.WriteString(key)
	htmlBuilder.WriteString(`="`)
	htmlBuilder.WriteString(value)
	htmlBuilder.WriteString(`"`)
}

// EscapeAllIntoNewMap returns a new TrustedAttributes map containing both the original
// TrustedAttributes items and the Attributes items, with the values escaped. In the case
// of a key collision, the value from the original TrustedAttributes map will be used.
func EscapeAllIntoNewMap(el *Element) map[string]string {
	attributes := make(map[string]string, len(el.Attributes)+len(el.TrustedAttributes))
	for k, v := range el.Attributes {
		attributes[k] = template.HTMLEscapeString(v)
	}
	maps.Copy(attributes, el.TrustedAttributes)
	return attributes
}

// EscapeIntoTrusted returns a new Element identical to the provided Element, but with
// the values of the Attributes map escaped and moved to the TrustedAttributes map. In
// the case of a key collision, the value from the original TrustedAttributes map will
// be used.
func EscapeIntoTrusted(el *Element) Element {
	return Element{
		Tag:               el.Tag,
		TrustedAttributes: EscapeAllIntoNewMap(el),
		BooleanAttributes: el.BooleanAttributes,
		InnerHTML:         el.InnerHTML,
		SelfClosing:       el.SelfClosing,
	}
}

func RenderModuleScriptToBuilder(src string, htmlBuilder *strings.Builder) error {
	return RenderElementToBuilder(&Element{
		Tag:               "script",
		TrustedAttributes: map[string]string{"type": "module", "src": src},
	}, htmlBuilder)
}
