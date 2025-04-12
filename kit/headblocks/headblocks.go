// Package headblocks manages HTML head elements with automatic deduplication.
//
// Deduplication Behavior:
//
// Title tags: Only the last title tag is kept. Earlier ones are discarded.
//
// Meta description: Only the last description meta tag is kept. Earlier ones
// are discarded.
//
// All other head elements: Exact duplicates are automatically removed, keeping
// only one instance.
//
// For example, if your page tries to insert multiple identical stylesheet links
// or meta tags, only one will appear in the final HTML. This makes the package
// ideal for component-based systems where multiple components might independently
// request the same resources or set the same metadata.
package headblocks

import (
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/river-now/river/kit/htmlutil"
)

type Instance struct {
	metaStart string
	metaEnd   string
	restStart string
	restEnd   string
}

const prefix = `<!-- data-`

func New(dataAttribute string) *Instance {
	return &Instance{
		metaStart: prefix + dataAttribute + suffix("meta-start"),
		metaEnd:   prefix + dataAttribute + suffix("meta-end"),
		restStart: prefix + dataAttribute + suffix("rest-start"),
		restEnd:   prefix + dataAttribute + suffix("rest-end"),
	}
}

func suffix(val string) string {
	return fmt.Sprintf(`="%s" -->`, val)
}

type HeadBlocks struct {
	Title string
	Meta  []*htmlutil.Element
	Rest  []*htmlutil.Element
}

func (inst *Instance) Render(input *HeadBlocks) (template.HTML, error) {
	var b strings.Builder

	// Add title
	err := htmlutil.RenderElementToBuilder(&htmlutil.Element{Tag: "title", InnerHTML: template.HTML(input.Title)}, &b)
	if err != nil {
		return "", fmt.Errorf("error rendering title: %v", err)
	}

	// Add meta head els
	b.WriteString(inst.metaStart)
	b.WriteString("\n")
	for _, el := range input.Meta {
		if err := htmlutil.RenderElementToBuilder(el, &b); err != nil {
			return "", fmt.Errorf("error rendering meta head el: %v", err)
		}
	}
	b.WriteString(inst.metaEnd)
	b.WriteString("\n")

	// Add rest head els
	b.WriteString(inst.restStart)
	b.WriteString("\n")
	for _, el := range input.Rest {
		if err := htmlutil.RenderElementToBuilder(el, &b); err != nil {
			return "", fmt.Errorf("error rendering rest head el: %v", err)
		}
	}
	b.WriteString(inst.restEnd)
	b.WriteString("\n")

	return template.HTML(b.String()), nil
}

// ToHeadBlocks deduplicates and organizes a slice of *htmlutil.Elements
// into a *HeadBlocks struct.
func ToHeadBlocks(els []*htmlutil.Element) *HeadBlocks {
	deduped := dedupeHeadBlocks(els)

	headblocks := &HeadBlocks{
		Meta: make([]*htmlutil.Element, 0, len(els)),
		Rest: make([]*htmlutil.Element, 0, len(els)),
	}

	for _, el := range deduped {
		switch {
		case isTitle(el):
			headblocks.Title = template.HTMLEscapeString(string(el.InnerHTML))
		case isMeta(el):
			safeEl := htmlutil.EscapeIntoTrusted(el)
			headblocks.Meta = append(headblocks.Meta, &safeEl)
		default:
			safeEl := htmlutil.EscapeIntoTrusted(el)
			headblocks.Rest = append(headblocks.Rest, &safeEl)
		}
	}

	return headblocks
}

func dedupeHeadBlocks(els []*htmlutil.Element) []*htmlutil.Element {
	uniqueEls := make(map[string]*htmlutil.Element, len(els))
	dedupedEls := make([]*htmlutil.Element, 0, len(els))

	titleIdx := -1
	descriptionIdx := -1

	for _, el := range els {
		switch {
		case isTitle(el):
			if titleIdx == -1 {
				titleIdx = len(dedupedEls)
				dedupedEls = append(dedupedEls, el)
			} else {
				dedupedEls[titleIdx] = el
			}

		case isDescription(el):
			if descriptionIdx == -1 {
				descriptionIdx = len(dedupedEls)
				dedupedEls = append(dedupedEls, el)
			} else {
				dedupedEls[descriptionIdx] = el
			}

		default:
			key := headBlockStableHash(el)
			if _, exists := uniqueEls[key]; !exists {
				uniqueEls[key] = el
				dedupedEls = append(dedupedEls, el)
			}
		}
	}

	return dedupedEls
}

func isTitle(el *htmlutil.Element) bool {
	return el.Tag == "title"
}

func isMeta(el *htmlutil.Element) bool {
	return el.Tag == "meta"
}

func isDescription(el *htmlutil.Element) bool {
	return el.Tag == "meta" && (el.Attributes["name"] == "description" || el.TrustedAttributes["name"] == "description")
}

func headBlockStableHash(el *htmlutil.Element) string {
	parts := make([]string, 0, len(el.Attributes)+len(el.TrustedAttributes)+len(el.BooleanAttributes))

	for key, value := range el.Attributes {
		parts = append(parts, fmt.Sprintf("attr:%s=%s", key, value))
	}
	for key, value := range el.TrustedAttributes {
		parts = append(parts, fmt.Sprintf("trusted:%s=%s", key, value))
	}
	for _, attr := range el.BooleanAttributes {
		parts = append(parts, fmt.Sprintf("bool:%s", attr))
	}

	sort.Strings(parts)

	// Calculate initial capacity for string builder
	// Initial size: tag + separator + innerHTML + separators between attributes
	initialSize := len(el.Tag) + 1 + len(el.InnerHTML) + (len(parts) * 16)

	var sb strings.Builder
	sb.Grow(initialSize)

	// Add tag
	sb.WriteString(el.Tag)
	sb.WriteString("|")

	// Add all attributes
	for i, part := range parts {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(part)
	}

	// Add innerHTML if present
	if len(el.InnerHTML) > 0 {
		sb.WriteString("|")
		sb.WriteString(string(el.InnerHTML))
	}

	// Add self-closing flag if true
	if el.SelfClosing {
		sb.WriteString("|self-closing")
	}

	return sb.String()
}
