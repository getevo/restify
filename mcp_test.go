package restify

import (
	"net/url"
	"strings"
	"testing"
)

// The filter grammar accepts a restricted character set, so a value that
// carries anything else has to survive the round trip through percent
// encoding: escapeFilterValue writes it, filterRegEx matches it and
// filterMapper unescapes it.
func TestEscapeFilterValueSurvivesTheGrammar(t *testing.T) {
	var values = []string{
		"Acme Spa",
		"ap@globex.example",
		"50% off",
		"a,b,c",
		"2026-08-10 12:30:00",
		"quote'and\"double",
		"plus+and~tilde",
		"unicode Ünïcode",
	}

	for _, value := range values {
		var escaped = escapeFilterValue(value)
		var matches = filterRegEx("name[eq]=" + escaped)
		if len(matches) != 1 {
			t.Fatalf("value %q escaped to %q, which the grammar does not match", value, escaped)
			continue
		}
		if matches[0]["column"] != "name" || matches[0]["condition"] != "eq" {
			t.Fatalf("value %q escaped to %q, parsed as %v", value, escaped, matches[0])
		}
		decoded, err := url.QueryUnescape(matches[0]["value"])
		if err != nil {
			t.Fatalf("value %q escaped to %q, which does not unescape: %v", value, escaped, err)
		}
		if decoded != value {
			t.Fatalf("value %q escaped to %q, unescaped to %q", value, escaped, decoded)
		}
	}
}

func TestInputAddFilter(t *testing.T) {
	var input = &Input{}
	input.AddFilter("status", "eq", "paid")
	input.AddFilter("total", "gt", "100")
	input.AddFilter("deleted_at", "isnull", "")
	input.AddFilter("name", "contains", "Acme Spa")

	var context = &Context{Input: input}
	var want = "status[eq]=paid&total[gt]=100&deleted_at[isnull]&name[contains]=Acme%20Spa"
	if got := context.QueryString(); got != want {
		t.Fatalf("query string is %q, want %q", got, want)
	}
}

// A null check carries no value, and writing "=" with nothing after it would
// make the grammar match an empty value instead.
func TestInputAddFilterOmitsValueForNullChecks(t *testing.T) {
	for _, condition := range []string{IsNullOperator, NotNullOperator} {
		var input = &Input{}
		input.AddFilter("deleted_at", condition, "ignored")
		if len(input.Filters) != 1 || strings.Contains(input.Filters[0], "=") {
			t.Fatalf("%s filter is %v, want no value", condition, input.Filters)
		}
	}
}

func TestInputSetSkipsEmptyValues(t *testing.T) {
	var input = &Input{}
	input.Set("order", "id.asc")
	input.Set("fields", "")

	var context = &Context{Input: input}
	if got := context.Query("order").String(); got != "id.asc" {
		t.Fatalf("order is %q, want id.asc", got)
	}
	if got := context.Query("fields").String(); got != "" {
		t.Fatalf("fields is %q, want empty", got)
	}
	if _, exists := input.Values["fields"]; exists {
		t.Fatal("an empty value was recorded, which makes an absent parameter look present")
	}
}

func TestContextReadsFromInput(t *testing.T) {
	var context = &Context{Input: &Input{
		Values: map[string]string{"size": "25"},
		Params: map[string]string{"id": "7"},
		Body:   []byte(`{"name":"Acme"}`),
		Verb:   "PATCH",
	}}

	if got := context.Query("size").Int(); got != 25 {
		t.Fatalf("size is %d, want 25", got)
	}
	if got := context.Param("id").Int(); got != 7 {
		t.Fatalf("id is %d, want 7", got)
	}
	if got := context.Method(); got != "PATCH" {
		t.Fatalf("method is %q, want PATCH", got)
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := context.ParseBody(&body); err != nil {
		t.Fatalf("parsing the body failed: %v", err)
	}
	if body.Name != "Acme" {
		t.Fatalf("name is %q, want Acme", body.Name)
	}
}

func TestParseBodyRejectsAnEmptyBody(t *testing.T) {
	var context = &Context{Input: &Input{}}
	var body struct{}
	if err := context.ParseBody(&body); err == nil {
		t.Fatal("an empty body was accepted")
	}
}

// SetResponse writes to the HTTP response, which a tool call does not have.
// Reaching for the request would panic, so the guard is what keeps All and
// Paginate usable outside HTTP.
func TestSetResponseWritesNothingForANonHTTPCall(t *testing.T) {
	var context = &Context{Input: &Input{}}
	context.SetResponse([]string{"a", "b"})
}

func TestMCPSplitRules(t *testing.T) {
	var cases = []struct {
		tag  string
		want []string
	}{
		{"", nil},
		{"required", []string{"required"}},
		{"required,email", []string{"required", "email"}},
		{"required,in(draft,sent,paid)", []string{"required", "in(draft,sent,paid)"}},
		{" required , >=10 , <=100 ", []string{"required", ">=10", "<=100"}},
		{"regex(^INV-[0-9]{4}$)", []string{"regex(^INV-[0-9]{4}$)"}},
	}

	for _, item := range cases {
		var got = mcpSplitRules(item.tag)
		if strings.Join(got, "|") != strings.Join(item.want, "|") {
			t.Fatalf("tag %q split to %v, want %v", item.tag, got, item.want)
		}
	}
}

func TestMCPDropRequired(t *testing.T) {
	var got = mcpDropRequired([]string{"required", "in(a,b)", "REQUIRED"})
	if strings.Join(got, "|") != "in(a,b)" {
		t.Fatalf("rules are %v, want in(a,b) alone", got)
	}
}

// A model may answer with a number, a boolean or a list where the schema says
// string. All of them have to reach the query grammar as text.
func TestMCPFilterValue(t *testing.T) {
	var cases = []struct {
		value any
		want  string
	}{
		{nil, ""},
		{"paid", "paid"},
		{float64(100), "100"},
		{float64(240.5), "240.5"},
		{true, "true"},
		{[]any{"draft", "sent"}, "draft,sent"},
		{[]any{float64(1), float64(2)}, "1,2"},
	}

	for _, item := range cases {
		if got := mcpFilterValue(item.value); got != item.want {
			t.Fatalf("value %#v rendered as %q, want %q", item.value, got, item.want)
		}
	}
}

func TestMCPContains(t *testing.T) {
	if !mcpContains(mcpFilterOperators, "between") {
		t.Fatal("between is not reported as an operator")
	}
	if mcpContains(mcpFilterOperators, "nope") {
		t.Fatal("nope is reported as an operator")
	}
}
