// Package headels manages HTML head elements with automatic deduplication.
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
package headels

import (
	"fmt"
	"html/template"
	"maps"
	"slices"
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

func NewInstance(dataAttribute string) *Instance {
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

type SortedHeadEls struct {
	Title string
	Meta  []*htmlutil.Element
	Rest  []*htmlutil.Element
}

func (inst *Instance) Render(input *SortedHeadEls) (template.HTML, error) {
	var b strings.Builder

	// Add title
	err := htmlutil.RenderElementToBuilder(&htmlutil.Element{Tag: "title", TextContent: input.Title}, &b)
	if err != nil {
		return "", fmt.Errorf("error rendering title: %w", err)
	}

	// Add meta head els
	b.WriteString(inst.metaStart)
	b.WriteString("\n")
	for _, el := range input.Meta {
		if err := htmlutil.RenderElementToBuilder(el, &b); err != nil {
			return "", fmt.Errorf("error rendering meta head el: %w", err)
		}
	}
	b.WriteString(inst.metaEnd)
	b.WriteString("\n")

	// Add rest head els
	b.WriteString(inst.restStart)
	b.WriteString("\n")
	for _, el := range input.Rest {
		if err := htmlutil.RenderElementToBuilder(el, &b); err != nil {
			return "", fmt.Errorf("error rendering rest head el: %w", err)
		}
	}
	b.WriteString(inst.restEnd)
	b.WriteString("\n")

	return template.HTML(b.String()), nil
}

// ToSortedHeadEls deduplicates and organizes a slice of *htmlutil.Elements
// into a *SortedHeadEls struct.
func ToSortedHeadEls(els []*htmlutil.Element, uniqueRules *HeadEls) *SortedHeadEls {
	deduped := dedupeHeadEls(els, uniqueRules)

	headEls := &SortedHeadEls{
		Meta: make([]*htmlutil.Element, 0, len(els)),
		Rest: make([]*htmlutil.Element, 0, len(els)),
	}

	for _, el := range deduped {
		safeEl := htmlutil.EscapeIntoTrusted(el)
		switch {
		case isTitle(&safeEl):
			headEls.Title = safeEl.DangerousInnerHTML
		case isMeta(el):
			headEls.Meta = append(headEls.Meta, &safeEl)
		default:
			headEls.Rest = append(headEls.Rest, &safeEl)
		}
	}

	return headEls
}

func dedupeHeadEls(els []*htmlutil.Element, uniqueRules *HeadEls) []*htmlutil.Element {
	// Pre-process unique rules into an efficient lookup structure
	// Map of tag -> slice of rules for that tag
	rulesByTag := make(map[string][]*ruleAttrs)
	if uniqueRules != nil {
		for _, rule := range uniqueRules.Collect() {
			attrs := extractRuleAttrs(rule)
			rulesByTag[rule.Tag] = append(rulesByTag[rule.Tag], attrs)
		}
	}

	// Track unique elements and their positions
	type elementKey struct {
		ruleIndex int    // which rule matched (-1 for no rule)
		hash      string // for elements not matching rules
	}
	seen := make(map[elementKey]int) // maps to position in dedupedEls
	dedupedEls := make([]*htmlutil.Element, 0, len(els))

	// Special handling for title/description to preserve existing behavior
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
			continue

		case isDescription(el):
			if descriptionIdx == -1 {
				descriptionIdx = len(dedupedEls)
				dedupedEls = append(dedupedEls, el)
			} else {
				dedupedEls[descriptionIdx] = el
			}
			continue
		}

		// Check if element matches any rules for its tag
		if rules, hasRules := rulesByTag[el.Tag]; hasRules {
			matched := false
			for ruleIdx, rule := range rules {
				if matchesRule(el, rule) {
					key := elementKey{ruleIndex: ruleIdx}
					if pos, exists := seen[key]; exists {
						// Replace existing element at position
						dedupedEls[pos] = el
					} else {
						seen[key] = len(dedupedEls)
						dedupedEls = append(dedupedEls, el)
					}
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}

		// Fall back to existing hash-based deduplication
		key := elementKey{ruleIndex: -1, hash: headElStableHash(el)}
		if _, exists := seen[key]; !exists {
			seen[key] = len(dedupedEls)
			dedupedEls = append(dedupedEls, el)
		}
	}

	return dedupedEls
}

type ruleAttrs struct {
	attrs   map[string]string
	trusted map[string]string
	boolean []string
}

func extractRuleAttrs(rule *htmlutil.Element) *ruleAttrs {
	return &ruleAttrs{
		attrs:   maps.Clone(rule.Attributes),
		trusted: maps.Clone(rule.AttributesDangerousVals),
		boolean: slices.Clone(rule.BooleanAttributes),
	}
}

func matchesRule(el *htmlutil.Element, rule *ruleAttrs) bool {
	// Check regular attributes
	for k, v := range rule.attrs {
		if el.Attributes[k] != v {
			return false
		}
	}

	// Check trusted attributes
	for k, v := range rule.trusted {
		if el.AttributesDangerousVals[k] != v {
			return false
		}
	}

	// Check boolean attributes
	for _, attr := range rule.boolean {
		if !slices.Contains(el.BooleanAttributes, attr) {
			return false
		}
	}

	return true
}

func isTitle(el *htmlutil.Element) bool {
	return el.Tag == "title"
}

func isMeta(el *htmlutil.Element) bool {
	return el.Tag == "meta"
}

func isDescription(el *htmlutil.Element) bool {
	return el.Tag == "meta" && (el.Attributes["name"] == "description" || el.AttributesDangerousVals["name"] == "description")
}

func headElStableHash(el *htmlutil.Element) string {
	parts := make([]string, 0, len(el.Attributes)+len(el.AttributesDangerousVals)+len(el.BooleanAttributes))

	for key, value := range el.Attributes {
		parts = append(parts, fmt.Sprintf("attr:%s=%s", key, value))
	}
	for key, value := range el.AttributesDangerousVals {
		parts = append(parts, fmt.Sprintf("trusted:%s=%s", key, value))
	}
	for _, attr := range el.BooleanAttributes {
		parts = append(parts, fmt.Sprintf("bool:%s", attr))
	}

	sort.Strings(parts)

	// Calculate initial capacity for string builder
	// Initial size: tag + separator + innerHTML + separators between attributes
	initialSize := len(el.Tag) + 1 + len(el.TextContent) + (len(parts) * 16)

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
	if len(el.TextContent) > 0 {
		sb.WriteString("|")
		sb.WriteString(string(el.TextContent))
	}

	// Add self-closing flag if true
	if el.SelfClosing {
		sb.WriteString("|self-closing")
	}

	return sb.String()
}

/////////////////////////////////////////////////////////////////////
/////// HIGH LEVEL
/////////////////////////////////////////////////////////////////////

type typeInterface interface{ GetType() htmlutilType }

type htmlutilType string

const (
	typeTag              htmlutilType = "tag"
	typeAttribute        htmlutilType = "attribute"
	typeBooleanAttribute htmlutilType = "boolean-attribute"
	typeInnerHTML        htmlutilType = "inner-html"
	typeTextContent      htmlutilType = "text-content"
	typeSelfClosing      htmlutilType = "self-closing"
)

type Tag string
type Attr struct {
	attr    [2]string
	trusted bool
	unique  bool
}
type BooleanAttribute string
type InnerHTML string
type TextContent string
type SelfClosing bool

func (Tag) GetType() htmlutilType              { return typeTag }
func (Attr) GetType() htmlutilType             { return typeAttribute }
func (BooleanAttribute) GetType() htmlutilType { return typeBooleanAttribute }
func (InnerHTML) GetType() htmlutilType        { return typeInnerHTML }
func (TextContent) GetType() htmlutilType      { return typeTextContent }
func (SelfClosing) GetType() htmlutilType      { return typeSelfClosing }

type HeadEls struct {
	els []*htmlutil.Element
}

func New(size ...int) HeadEls {
	var els []*htmlutil.Element
	if len(size) > 0 {
		els = make([]*htmlutil.Element, 0, size[0])
	} else {
		els = make([]*htmlutil.Element, 0)
	}
	return HeadEls{els: els}
}

func (h *HeadEls) Add(defs ...typeInterface) {
	el := new(htmlutil.Element)

	el.Attributes = make(map[string]string)
	el.AttributesDangerousVals = make(map[string]string)
	el.BooleanAttributes = make([]string, 0)

	for _, def := range defs {
		switch def.GetType() {
		case typeTag:
			el.Tag = string(def.(Tag))
		case typeAttribute:
			attr := def.(*Attr)
			if attr.trusted {
				el.AttributesDangerousVals[attr.attr[0]] = attr.attr[1]
			} else {
				el.Attributes[attr.attr[0]] = attr.attr[1]
			}
		case typeBooleanAttribute:
			attr := def.(BooleanAttribute)
			el.BooleanAttributes = append(el.BooleanAttributes, string(attr))
		case typeInnerHTML:
			el.DangerousInnerHTML = string(def.(InnerHTML))
		case typeTextContent:
			el.TextContent = string(def.(TextContent))
		case typeSelfClosing:
			el.SelfClosing = bool(def.(SelfClosing))
		default:
			panic(fmt.Sprintf("unknown type %T", def))
		}
	}

	h.els = append(h.els, el)
}

func (h *HeadEls) Collect() []*htmlutil.Element {
	return h.els
}

func (a *Attr) DangerousVal() *Attr {
	a.trusted = true
	return a
}
func (a *Attr) Unique() *Attr {
	a.unique = true
	return a
}

func (h *HeadEls) Title(title string) {
	h.Add(Tag("title"), InnerHTML(title))
}
func (h *HeadEls) Description(description string) {
	h.Meta(h.Attr("name", "description"), h.Attr("content", description))
}
func (h *HeadEls) Meta(defs ...typeInterface) {
	h.Add(append(defs, Tag("meta"))...)
}
func (h *HeadEls) Link(defs ...typeInterface) {
	h.Add(append(defs, Tag("link"))...)
}

func (h *HeadEls) Attr(name, value string) *Attr {
	return &Attr{attr: [2]string{name, value}}
}
func (h *HeadEls) Property(property string) *Attr {
	return h.Attr("property", property)
}
func (h *HeadEls) Name(name string) *Attr {
	return h.Attr("name", name)
}
func (h *HeadEls) Content(content string) *Attr {
	return h.Attr("content", content)
}
func (h *HeadEls) Rel(rel string) *Attr {
	return h.Attr("rel", rel)
}
func (h *HeadEls) Href(href string) *Attr {
	return h.Attr("href", href)
}
