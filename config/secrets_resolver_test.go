package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
)

// fakeSecretsManager serves canned values and counts lookups so tests can assert on memoization.
type fakeSecretsManager struct {
	values map[string]string
	calls  map[string]int
}

func newFakeSecretsManager(values map[string]string) *fakeSecretsManager {
	return &fakeSecretsManager{values: values, calls: map[string]int{}}
}

func (f *fakeSecretsManager) GetSecret(_ context.Context, key string) (string, error) {
	return f.get("key:" + key)
}

func (f *fakeSecretsManager) GetSecretVerbatim(_ context.Context, id string) (string, error) {
	return f.get("id:" + id)
}

func (f *fakeSecretsManager) get(memo string) (string, error) {
	f.calls[memo]++
	if v, ok := f.values[memo]; ok {
		return v, nil
	}
	return "", fmt.Errorf("secret %s not found", memo)
}

func (f *fakeSecretsManager) totalCalls() int {
	n := 0
	for _, c := range f.calls {
		n += c
	}
	return n
}

func TestParseSecretRef(t *testing.T) {
	tests := []struct {
		value, parentPath, key string
		want                   secretRef
		notSecret              bool
		errContains            string
	}{
		{value: "plain", notSecret: true},
		{value: "<secret", notSecret: true},
		{value: "secret>", notSecret: true},

		{value: "<secret>", parentPath: "Database", key: "Pass", want: secretRef{id: "Database.Pass"}},
		{value: "<secret>", key: "Top", want: secretRef{id: "Top"}},
		{value: "<secret:DB>", parentPath: "Database", key: "Pass", want: secretRef{verbatim: true, id: "DB"}},
		{value: "<secret:prod/db/creds>", key: "x", want: secretRef{verbatim: true, id: "prod/db/creds"}},

		{value: "<secret#>", parentPath: "Database", key: "Pass", want: secretRef{id: "Database", pointer: "/Pass"}},
		{value: "<secret#>", parentPath: "A.B", key: "weird~key/x", want: secretRef{id: "A.B", pointer: "/weird~0key~1x"}},
		{value: "<secret#/password>", parentPath: "Database", key: "Pass", want: secretRef{id: "Database", pointer: "/password"}},
		{value: "<secret:DB#>", parentPath: "Database", key: "Pass", want: secretRef{verbatim: true, id: "DB", pointer: "/Pass"}},
		{value: "<secret:DB#>", key: "Top", want: secretRef{verbatim: true, id: "DB", pointer: "/Top"}},
		{value: "<secret:DB#/a/b>", key: "x", want: secretRef{verbatim: true, id: "DB", pointer: "/a/b"}},
		{value: "<secret:prod/db#/pass#1>", key: "x", want: secretRef{verbatim: true, id: "prod/db", pointer: "/pass#1"}},

		{value: "<secret:>", key: "x", errContains: "provide a custom key or remove ':'"},
		{value: "<secret:#/a>", key: "x", errContains: "provide a custom key or remove ':'"},
		{value: "<secretX>", key: "x", errContains: "invalid secret value: <secretX>, for key: x"},
		{value: "<secret#>", key: "Top", errContains: "a grouped secret needs a parent key"},
		{value: "<secret#/a>", key: "Top", errContains: "a grouped secret needs a parent key"},
		{value: "<secret:DB#noSlash>", key: "x", errContains: "must start with '/'"},
		{value: "<secret#noSlash>", parentPath: "P", key: "x", errContains: "must start with '/'"},
	}

	for _, tt := range tests {
		t.Run(tt.value+"@"+joinConfigPath(tt.parentPath, tt.key), func(t *testing.T) {
			got, ok, err := parseSecretRef(tt.value, tt.parentPath, tt.key)
			if tt.notSecret {
				require.False(t, ok)
				require.NoError(t, err)
				return
			}
			require.True(t, ok)
			if tt.errContains != "" {
				require.Error(t, err)
				require.True(t, errors.IsCode(err, errors.ErrCodeBadState), "code: %v", err)
				require.Contains(t, errors.ErrorMessageWithoutStacktrace(err), tt.errContains)
				return
			}
			require.NoError(t, err)
			tt.want.raw = tt.value
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLoadSecrets_ExistingFormsUnchanged(t *testing.T) {
	sm := newFakeSecretsManager(map[string]string{
		"key:Database.Pass": "plain-by-path",
		"id:OPENAI_API_KEY": "plain-verbatim",
		// A whole-secret reference must never be parsed as JSON, even when it looks like it.
		"id:GCP_SA": `{"type":"service_account","private_key":"-----BEGIN"}`,
	})
	conf := map[string]any{
		"Database": map[string]any{"User": "postgres", "Pass": "<secret>"},
		"Llm":      map[string]any{"ApiKey": "<secret:OPENAI_API_KEY>"},
		"Gcp":      map[string]any{"Sa": "<secret:GCP_SA>"},
	}

	loadSecrets(sm)(conf)

	require.Equal(t, "plain-by-path", conf["Database"].(map[string]any)["Pass"])
	require.Equal(t, "postgres", conf["Database"].(map[string]any)["User"])
	require.Equal(t, "plain-verbatim", conf["Llm"].(map[string]any)["ApiKey"])
	require.Equal(t, `{"type":"service_account","private_key":"-----BEGIN"}`, conf["Gcp"].(map[string]any)["Sa"])
	require.Equal(t, 1, sm.calls["key:Database.Pass"])
	require.Equal(t, 1, sm.calls["id:OPENAI_API_KEY"])
}

func TestLoadSecrets_GroupedByParentFetchesOnce(t *testing.T) {
	sm := newFakeSecretsManager(map[string]string{
		"key:Database.Primary": `{"User":"app","Pass":"s3cr3t","Host":"db.internal"}`,
		"key:Database.Replica": `{"User":"ro","Pass":"r3pl1ca","Host":"replica.internal"}`,
	})
	conf := map[string]any{
		"Database": map[string]any{
			"Primary": map[string]any{"User": "<secret#>", "Pass": "<secret#>", "Host": "<secret#>"},
			"Replica": map[string]any{"User": "<secret#>", "Pass": "<secret#>", "Host": "<secret#>"},
		},
	}

	loadSecrets(sm)(conf)

	primary := conf["Database"].(map[string]any)["Primary"].(map[string]any)
	replica := conf["Database"].(map[string]any)["Replica"].(map[string]any)
	require.Equal(t, map[string]any{"User": "app", "Pass": "s3cr3t", "Host": "db.internal"}, primary)
	require.Equal(t, map[string]any{"User": "ro", "Pass": "r3pl1ca", "Host": "replica.internal"}, replica)
	require.Equal(t, 1, sm.calls["key:Database.Primary"], "one fetch for the whole primary group")
	require.Equal(t, 1, sm.calls["key:Database.Replica"], "one fetch for the whole replica group")
	require.Equal(t, 2, sm.totalCalls())
}

func TestLoadSecrets_ExplicitPointers(t *testing.T) {
	sm := newFakeSecretsManager(map[string]string{
		"key:Email.Smtp":    `{"username":"mailer","pass":"m41l"}`,
		"id:llm-keys":       `{"openai":{"apiKey":"sk-openai"},"anthropic":{"apiKey":"sk-ant"}}`,
		"id:prod/billing":   `{"api_key":"bill-key"}`,
		"id:misc":           `{"api/keys":{"openai":"slash-ok"},"a~b":"tilde-ok","":"empty-key"}`,
		"id:ROT":            `{"passwords":["p1","p2"]}`,
		"id:DB":             `{"port":5432,"ssl":true,"ratio":1.50,"big":12345678901234567890}`,
		"id:WHOLE_AND_PART": `{"token":"tok"}`,
	})
	conf := map[string]any{
		"Email": map[string]any{"Smtp": map[string]any{
			"Username": "<secret#/username>",
			"Password": "<secret#/pass>",
		}},
		"Llm": map[string]any{
			"Openai":    map[string]any{"ApiKey": "<secret:llm-keys#/openai/apiKey>"},
			"Anthropic": map[string]any{"ApiKey": "<secret:llm-keys#/anthropic/apiKey>"},
		},
		"Billing": map[string]any{"Key": "<secret:prod/billing#/api_key>"},
		"Misc": map[string]any{
			"Openai":   "<secret:misc#/api~1keys/openai>",
			"Tilde":    "<secret:misc#/a~0b>",
			"EmptyKey": "<secret:misc#/>",
		},
		"Rotate": map[string]any{"Current": "<secret:ROT#/passwords/0>", "Next": "<secret:ROT#/passwords/1>"},
		"Db": map[string]any{
			"Port":  "<secret:DB#/port>",
			"Ssl":   "<secret:DB#/ssl>",
			"Ratio": "<secret:DB#/ratio>",
			"Big":   "<secret:DB#/big>",
		},
		"Mixed": map[string]any{
			"Raw":   "<secret:WHOLE_AND_PART>",
			"Field": "<secret:WHOLE_AND_PART#/token>",
		},
	}

	loadSecrets(sm)(conf)

	get := func(path ...string) any {
		var cur any = conf
		for _, p := range path {
			cur = cur.(map[string]any)[p]
		}
		return cur
	}
	require.Equal(t, "mailer", get("Email", "Smtp", "Username"))
	require.Equal(t, "m41l", get("Email", "Smtp", "Password"))
	require.Equal(t, "sk-openai", get("Llm", "Openai", "ApiKey"))
	require.Equal(t, "sk-ant", get("Llm", "Anthropic", "ApiKey"))
	require.Equal(t, "bill-key", get("Billing", "Key"))
	require.Equal(t, "slash-ok", get("Misc", "Openai"))
	require.Equal(t, "tilde-ok", get("Misc", "Tilde"))
	require.Equal(t, "empty-key", get("Misc", "EmptyKey"))
	require.Equal(t, "p1", get("Rotate", "Current"))
	require.Equal(t, "p2", get("Rotate", "Next"))
	require.Equal(t, "5432", get("Db", "Port"), "numbers keep their literal text")
	require.Equal(t, "true", get("Db", "Ssl"))
	require.Equal(t, "1.50", get("Db", "Ratio"), "no float round-trip")
	require.Equal(t, "12345678901234567890", get("Db", "Big"), "no precision loss")
	require.Equal(t, `{"token":"tok"}`, get("Mixed", "Raw"))
	require.Equal(t, "tok", get("Mixed", "Field"))

	require.Equal(t, 1, sm.calls["key:Email.Smtp"])
	require.Equal(t, 1, sm.calls["id:llm-keys"], "two fields, one fetch")
	require.Equal(t, 1, sm.calls["id:ROT"])
	require.Equal(t, 1, sm.calls["id:DB"], "four fields, one fetch")
	require.Equal(t, 1, sm.calls["id:WHOLE_AND_PART"], "whole-secret and field reference share one fetch")
	require.Equal(t, 7, sm.totalCalls())
}

func TestLoadSecrets_Errors(t *testing.T) {
	const leak = "SUPER-SECRET-VALUE"
	values := map[string]string{
		"id:DB":       `{"pass":"` + leak + `","obj":{"k":"v"},"arr":[1],"nul":null,"list":["a"]}`,
		"id:NOT_JSON": leak + " is not json",
		"id:ARRAY":    `["` + leak + `"]`,
		"id:SCALAR":   `"` + leak + `"`,
		"id:TRAILING": `{"a":"` + leak + `"} trailing`,
	}
	tests := []struct {
		name        string
		conf        map[string]any
		code        string
		errContains string
	}{
		{"missing member", map[string]any{"X": map[string]any{"V": "<secret:DB#/missing>"}}, errors.ErrCodeBadState, `member "missing" not found`},
		{"nested missing", map[string]any{"X": map[string]any{"V": "<secret:DB#/obj/missing>"}}, errors.ErrCodeBadState, `member "missing" not found`},
		{"object value", map[string]any{"X": map[string]any{"V": "<secret:DB#/obj>"}}, errors.ErrCodeBadState, "resolves to an object"},
		{"array value", map[string]any{"X": map[string]any{"V": "<secret:DB#/arr>"}}, errors.ErrCodeBadState, "resolves to an array"},
		{"null value", map[string]any{"X": map[string]any{"V": "<secret:DB#/nul>"}}, errors.ErrCodeBadState, "resolves to null"},
		{"descend into scalar", map[string]any{"X": map[string]any{"V": "<secret:DB#/pass/x>"}}, errors.ErrCodeBadState, "cannot descend into a string"},
		{"bad index", map[string]any{"X": map[string]any{"V": "<secret:DB#/list/x>"}}, errors.ErrCodeBadState, `"x" is not a valid array index`},
		{"leading zero index", map[string]any{"X": map[string]any{"V": "<secret:DB#/list/00>"}}, errors.ErrCodeBadState, `"00" is not a valid array index`},
		{"past the end", map[string]any{"X": map[string]any{"V": "<secret:DB#/list/->"}}, errors.ErrCodeBadState, `"-" is not a valid array index`},
		{"index out of range", map[string]any{"X": map[string]any{"V": "<secret:DB#/list/1>"}}, errors.ErrCodeBadState, "array index 1 is out of range"},
		{"not json", map[string]any{"X": map[string]any{"V": "<secret:NOT_JSON#/a>"}}, errors.ErrCodeBadState, "secret: NOT_JSON is not valid JSON"},
		{"top-level array", map[string]any{"X": map[string]any{"V": "<secret:ARRAY#/0>"}}, errors.ErrCodeBadState, "secret: ARRAY is not a JSON object"},
		{"top-level scalar", map[string]any{"X": map[string]any{"V": "<secret:SCALAR#/a>"}}, errors.ErrCodeBadState, "secret: SCALAR is not a JSON object"},
		{"trailing data", map[string]any{"X": map[string]any{"V": "<secret:TRAILING#/a>"}}, errors.ErrCodeBadState, "has trailing data"},
		{"top-level grouped", map[string]any{"V": "<secret#>"}, errors.ErrCodeBadState, "a grouped secret needs a parent key"},
		{"pointer without slash", map[string]any{"X": map[string]any{"V": "<secret:DB#pass>"}}, errors.ErrCodeBadState, "must start with '/'"},
		{"empty verbatim id", map[string]any{"X": map[string]any{"V": "<secret:>"}}, errors.ErrCodeBadState, "provide a custom key or remove ':'"},
		{"unknown form", map[string]any{"X": map[string]any{"V": "<secret!>"}}, errors.ErrCodeBadState, "invalid secret value: <secret!>"},
		{"manager error plain", map[string]any{"X": map[string]any{"V": "<secret:NOPE>"}}, errors.ErrCodeUnknown, "could not load secret: NOPE"},
		{"manager error grouped", map[string]any{"X": map[string]any{"V": "<secret:NOPE#/a>"}}, errors.ErrCodeUnknown, "could not load secret: NOPE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newFakeSecretsManager(values)
			err := recoverPanicError(t, func() { loadSecrets(sm)(tt.conf) })
			require.Error(t, err)
			require.True(t, errors.IsCode(err, tt.code), "want code %s, got: %v", tt.code, err)
			msg := err.Error()
			require.Contains(t, msg, tt.errContains)
			require.NotContains(t, msg, leak, "error must not leak secret content")
		})
	}
}

func TestResolveJSONPointer_RFC6901Examples(t *testing.T) {
	// The example document and pointers from RFC 6901 section 5.
	doc := parseJSONForTest(t, `{
		"foo": ["bar", "baz"],
		"": 0,
		"a/b": 1,
		"c%d": 2,
		"e^f": 3,
		"g|h": 4,
		"i\\j": 5,
		"k\"l": 6,
		" ": 7,
		"m~n": 8
	}`)

	tests := []struct {
		pointer string
		want    any
	}{
		{"/foo", []any{"bar", "baz"}},
		{"/foo/0", "bar"},
		{"/", num("0")},
		{"/a~1b", num("1")},
		{"/c%d", num("2")},
		{"/e^f", num("3")},
		{"/g|h", num("4")},
		{"/i\\j", num("5")},
		{"/k\"l", num("6")},
		{"/ ", num("7")},
		{"/m~0n", num("8")},
	}
	for _, tt := range tests {
		t.Run(tt.pointer, func(t *testing.T) {
			got, err := resolveJSONPointer(doc, tt.pointer)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestJSONPointerTokenEscaping(t *testing.T) {
	for _, raw := range []string{"plain", "a/b", "a~b", "a~/b", "~1", "~0", "~01", "/", "~", ""} {
		escaped := escapeJSONPointerToken(raw)
		require.NotContains(t, escaped, "/", "escaped token must be a single reference token")
		require.Equal(t, raw, unescapeJSONPointerToken(escaped), "round trip of %q", raw)
	}
	// Order of operations matters: "~01" is an escaped '~' followed by a literal '1', not an escaped '/'.
	require.Equal(t, "~1", unescapeJSONPointerToken("~01"))
	require.Equal(t, "/", unescapeJSONPointerToken("~1"))
}

func recoverPanicError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		var ok bool
		err, ok = r.(error)
		require.True(t, ok, "panic value is not an error: %v", r)
	}()
	fn()
	return nil
}

func parseJSONForTest(t *testing.T, s string) any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var doc any
	require.NoError(t, dec.Decode(&doc))
	return doc
}

func num(s string) any {
	return json.Number(s)
}
