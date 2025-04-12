package headblocks

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/river-now/river/kit/htmlutil"
)

var testInstance = New("bob")

func TestGetHeadElements(t *testing.T) {
	routeData := &HeadBlocks{
		Title: "Test Title",
		Meta: []*htmlutil.Element{
			{Tag: "meta", Attributes: map[string]string{"name": "description", "content": "Test Description"}},
		},
		Rest: []*htmlutil.Element{
			{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
		},
	}

	html, err := testInstance.Render(routeData)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if !strings.Contains(string(html), "<title>Test Title</title>") {
		t.Errorf("Expected title tag, but it's missing")
	}
	if !strings.Contains(string(html), `name="description"`) || !strings.Contains(string(html), `content="Test Description"`) {
		t.Errorf("Expected meta description tag, but it's missing")
	}
	if !strings.Contains(string(html), `rel="stylesheet"`) || !strings.Contains(string(html), `href="/style.css"`) {
		t.Errorf("Expected link tag, but it's missing")
	}
}

const (
	testTitle         = "Test Title"
	testTitle_2       = "Different Test Title"
	testDescription   = "This is a test description."
	testDescription_2 = "This is a different test description."
)

// Test cases for dedupeHeadBlocks
func TestDedupeHeadBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    []*htmlutil.Element
		expected []*htmlutil.Element
	}{
		{
			name: "No duplicates, with title and description",
			input: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": testDescription}},
				{Tag: "meta", Attributes: map[string]string{"name": "og:image", "content": "image.webp"}},
			},
			expected: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": testDescription}},
				{Tag: "meta", Attributes: map[string]string{"name": "og:image", "content": "image.webp"}},
			},
		},
		{
			name: "With duplicates",
			input: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": testDescription}},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": testDescription_2}},
			},
			expected: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": testDescription_2}},
			},
		},
		{
			name: "With duplicates TrustedAttributes",
			input: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", TrustedAttributes: map[string]string{"name": "description", "content": testDescription}},
				{Tag: "meta", TrustedAttributes: map[string]string{"name": "description", "content": testDescription_2}},
			},
			expected: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", TrustedAttributes: map[string]string{"name": "description", "content": testDescription_2}},
			},
		},
		{
			name: "With duplicates mixed",
			input: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": testDescription}},
				{Tag: "meta", TrustedAttributes: map[string]string{"name": "description", "content": testDescription_2}},
			},
			expected: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "meta", TrustedAttributes: map[string]string{"name": "description", "content": testDescription_2}},
			},
		},
		{
			name: "No title or description",
			input: []*htmlutil.Element{
				{Tag: "meta", Attributes: map[string]string{"name": "keywords", "content": "go, test"}},
				{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
			},
			expected: []*htmlutil.Element{
				{Tag: "meta", Attributes: map[string]string{"name": "keywords", "content": "go, test"}},
				{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
			},
		},
		{
			name: "Multiple titles and descriptions",
			input: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle, Attributes: nil},
				{Tag: "title", InnerHTML: testTitle_2, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": "Description 1"}},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": "Description 2"}},
			},
			expected: []*htmlutil.Element{
				{Tag: "title", InnerHTML: testTitle_2, Attributes: nil},
				{Tag: "meta", Attributes: map[string]string{"name": "description", "content": "Description 2"}},
			},
		},
		{
			name: "Different tags with same attributes",
			input: []*htmlutil.Element{
				{Tag: "meta", Attributes: map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1"}},
				{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
				{Tag: "meta", Attributes: map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1"}},
			},
			expected: []*htmlutil.Element{
				{Tag: "meta", Attributes: map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1"}},
				{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupeHeadBlocks(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				fmt.Println("Result:")
				for _, block := range result {
					t.Logf("%+v", block)
				}

				fmt.Println("Expected:")
				for _, block := range tt.expected {
					t.Logf("%+v", block)
				}

				t.Errorf("dedupeHeadBlocks() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Ensure stableHash function produces consistent hashes
func TestStableHash(t *testing.T) {
	tests := []struct {
		name     string
		input    *htmlutil.Element
		expected string
	}{
		{
			name: "Simple meta tag",
			input: &htmlutil.Element{
				Tag:        "meta",
				Attributes: map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1"},
			},
			expected: "meta|attr:content=width=device-width, initial-scale=1&attr:name=viewport",
		},
		{
			name: "Title tag with innerHTML",
			input: &htmlutil.Element{
				Tag:       "title",
				InnerHTML: "Test Title",
			},
			expected: "title||Test Title",
		},
		{
			name: "Empty element",
			input: &htmlutil.Element{
				Tag: "div",
			},
			expected: "div|",
		},
		{
			name: "Mixed attributes types",
			input: &htmlutil.Element{
				Tag: "input",
				Attributes: map[string]string{
					"type": "text",
					"name": "username",
				},
				TrustedAttributes: map[string]string{
					"data-custom": "<safe-value>",
				},
				BooleanAttributes: []string{"required", "autofocus"},
			},
			expected: "input|attr:name=username&attr:type=text&bool:autofocus&bool:required&trusted:data-custom=<safe-value>",
		},
		{
			name: "Self-closing tag with attributes",
			input: &htmlutil.Element{
				Tag:         "img",
				SelfClosing: true,
				Attributes: map[string]string{
					"src": "image.jpg",
					"alt": "Test Image",
				},
			},
			expected: "img|attr:alt=Test Image&attr:src=image.jpg|self-closing",
		},
		{
			name: "Complex script tag",
			input: &htmlutil.Element{
				Tag:       "script",
				InnerHTML: "console.log('test');",
				Attributes: map[string]string{
					"type": "text/javascript",
				},
				TrustedAttributes: map[string]string{
					"nonce": "abc123",
				},
			},
			expected: "script|attr:type=text/javascript&trusted:nonce=abc123|console.log('test');",
		},
		{
			name: "All fields populated",
			input: &htmlutil.Element{
				Tag:       "div",
				InnerHTML: "content",
				Attributes: map[string]string{
					"class": "main",
					"id":    "container",
				},
				TrustedAttributes: map[string]string{
					"data-safe": "<html>",
				},
				BooleanAttributes: []string{"hidden", "draggable"},
				SelfClosing:       true,
			},
			expected: "div|attr:class=main&attr:id=container&bool:draggable&bool:hidden&trusted:data-safe=<html>|content|self-closing",
		},
		{
			name: "Special characters in attributes",
			input: &htmlutil.Element{
				Tag: "div",
				Attributes: map[string]string{
					"data-test": "a|b&c=d",
				},
			},
			expected: "div|attr:data-test=a|b&c=d",
		},
		{
			name: "Only boolean attributes",
			input: &htmlutil.Element{
				Tag:               "input",
				BooleanAttributes: []string{"required", "readonly", "disabled"},
			},
			expected: "input|bool:disabled&bool:readonly&bool:required",
		},
		{
			name: "Only trusted attributes",
			input: &htmlutil.Element{
				Tag: "div",
				TrustedAttributes: map[string]string{
					"data-html":  "<p>safe</p>",
					"data-html2": "<div>also safe</div>",
				},
			},
			expected: "div|trusted:data-html2=<div>also safe</div>&trusted:data-html=<p>safe</p>",
		},
		{
			name: "Empty attributes but with innerHTML",
			input: &htmlutil.Element{
				Tag:               "span",
				InnerHTML:         "Some content",
				Attributes:        map[string]string{},
				TrustedAttributes: map[string]string{},
				BooleanAttributes: []string{},
			},
			expected: "span||Some content",
		},
		{
			name: "Unicode content",
			input: &htmlutil.Element{
				Tag:       "p",
				InnerHTML: "Hello, 世界",
				Attributes: map[string]string{
					"lang": "zh",
				},
			},
			expected: "p|attr:lang=zh|Hello, 世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := headBlockStableHash(tt.input)
			if result != tt.expected {
				t.Errorf("stableHash() =\n%v\nexpected:\n%v", result, tt.expected)
			}
		})
	}
}

func TestDedupeHeadBlocksEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []*htmlutil.Element
		expected []*htmlutil.Element
	}{
		{
			name: "Same tag different attributes",
			input: []*htmlutil.Element{
				{Tag: "meta", Attributes: map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1"}},
				{Tag: "meta", Attributes: map[string]string{"name": "charset", "content": "UTF-8"}},
			},
			expected: []*htmlutil.Element{
				{Tag: "meta", Attributes: map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1"}},
				{Tag: "meta", Attributes: map[string]string{"name": "charset", "content": "UTF-8"}},
			},
		},
		{
			name: "Script and link tags",
			input: []*htmlutil.Element{
				{Tag: "script", Attributes: map[string]string{"src": "/script.js"}},
				{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
			},
			expected: []*htmlutil.Element{
				{Tag: "script", Attributes: map[string]string{"src": "/script.js"}},
				{Tag: "link", Attributes: map[string]string{"rel": "stylesheet", "href": "/style.css"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupeHeadBlocks(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				fmt.Println("Result:")
				for _, block := range result {
					t.Logf("%+v", block)
				}

				fmt.Println("Expected:")
				for _, block := range tt.expected {
					t.Logf("%+v", block)
				}

				t.Errorf("dedupeHeadBlocks() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
