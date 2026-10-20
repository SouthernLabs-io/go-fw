package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/southernlabs-io/go-fw/errors"
)

const secretRefPrefix = "<secret"

// secretRef is a parsed <secret…> placeholder.
//
//	<secret>            whole secret, id = full config path (transformed by the manager)
//	<secret:ID>         whole secret, id = ID verbatim
//	<secret#>           JSON field named after the config key, id = parent config path
//	<secret#/ptr>       JSON Pointer /ptr, id = parent config path
//	<secret:ID#>        JSON field named after the config key, id = ID verbatim
//	<secret:ID#/ptr>    JSON Pointer /ptr, id = ID verbatim
type secretRef struct {
	raw      string
	verbatim bool
	id       string
	// pointer is an RFC 6901 JSON Pointer into the secret. Empty selects the whole secret.
	pointer string
}

func (r secretRef) grouped() bool {
	return r.pointer != ""
}

// parseSecretRef parses value found at config key `key` under parentPath (dotted, "" at the top level).
// ok is false when value is not a secret placeholder at all.
func parseSecretRef(value, parentPath, key string) (ref secretRef, ok bool, err error) {
	if !strings.HasPrefix(value, secretRefPrefix) || !strings.HasSuffix(value, ">") {
		return secretRef{}, false, nil
	}
	ref.raw = value
	rest := value[len(secretRefPrefix) : len(value)-1]

	switch {
	case rest == "":
		ref.id = joinConfigPath(parentPath, key)
		return ref, true, nil

	case rest[0] == ':':
		id, pointer, hasPointer := strings.Cut(rest[1:], "#")
		if id == "" {
			return ref, true, errors.NewBadStatef(
				"invalid secret value: %s, for key: %s, provide a custom key or remove ':'", value, key)
		}
		ref.verbatim = true
		ref.id = id
		if hasPointer {
			if err = ref.setPointer(pointer, key); err != nil {
				return ref, true, err
			}
		}
		return ref, true, nil

	case rest[0] == '#':
		if parentPath == "" {
			return ref, true, errors.NewBadStatef(
				"invalid secret value: %s, for key: %s, a grouped secret needs a parent key, use <secret:ID#...> instead",
				value, key)
		}
		ref.id = parentPath
		if err = ref.setPointer(rest[1:], key); err != nil {
			return ref, true, err
		}
		return ref, true, nil

	default:
		return ref, true, errors.NewBadStatef("invalid secret value: %s, for key: %s", value, key)
	}
}

// setPointer stores the JSON Pointer that follows '#'. An empty pointer selects the member named
// after the config key.
func (r *secretRef) setPointer(pointer, key string) error {
	if pointer == "" {
		r.pointer = "/" + escapeJSONPointerToken(key)
		return nil
	}
	if pointer[0] != '/' {
		return errors.NewBadStatef(
			"invalid secret value: %s, for key: %s, the JSON Pointer after '#' must start with '/'", r.raw, key)
	}
	r.pointer = pointer
	return nil
}

func joinConfigPath(parentPath, key string) string {
	if parentPath == "" {
		return key
	}
	return parentPath + "." + key
}

// secretResolver resolves parsed placeholders through a SecretsManager, fetching and parsing each
// secret at most once per instance so that every field of a grouped secret costs a single call.
type secretResolver struct {
	ctx     context.Context
	sm      SecretsManager
	raw     map[string]string
	objects map[string]map[string]any
}

func newSecretResolver(ctx context.Context, sm SecretsManager) *secretResolver {
	return &secretResolver{
		ctx:     ctx,
		sm:      sm,
		raw:     make(map[string]string),
		objects: make(map[string]map[string]any),
	}
}

func (r *secretResolver) resolve(ref secretRef, key string) (string, error) {
	if !ref.grouped() {
		return r.fetch(ref)
	}
	obj, err := r.object(ref)
	if err != nil {
		return "", err
	}
	val, err := resolveJSONPointer(obj, ref.pointer)
	if err != nil {
		return "", errors.NewBadStatef(
			"could not resolve JSON Pointer %s in secret: %s, for key: %s, error: %w", ref.pointer, ref.id, key, err)
	}
	return secretScalar(val, ref, key)
}

func (r *secretResolver) memoKey(ref secretRef) string {
	if ref.verbatim {
		return "id:" + ref.id
	}
	return "key:" + ref.id
}

func (r *secretResolver) fetch(ref secretRef) (string, error) {
	memo := r.memoKey(ref)
	if v, ok := r.raw[memo]; ok {
		return v, nil
	}
	var v string
	var err error
	if ref.verbatim {
		v, err = r.sm.GetSecretVerbatim(r.ctx, ref.id)
	} else {
		v, err = r.sm.GetSecret(r.ctx, ref.id)
	}
	if err != nil {
		return "", errors.NewUnknownf("could not load secret: %s, error: %w", ref.id, err)
	}
	r.raw[memo] = v
	return v, nil
}

// object returns the secret parsed as a JSON object. Decode errors are deliberately not wrapped:
// encoding/json quotes the offending input, which would leak secret material into logs.
func (r *secretResolver) object(ref secretRef) (map[string]any, error) {
	memo := r.memoKey(ref)
	if obj, ok := r.objects[memo]; ok {
		return obj, nil
	}
	raw, err := r.fetch(ref)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var doc any
	if err = dec.Decode(&doc); err != nil {
		return nil, errors.NewBadStatef("secret: %s is not valid JSON", ref.id)
	}
	if _, err = dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.NewBadStatef("secret: %s has trailing data after the JSON value", ref.id)
	}
	obj, ok := doc.(map[string]any)
	if !ok {
		return nil, errors.NewBadStatef("secret: %s is not a JSON object", ref.id)
	}
	r.objects[memo] = obj
	return obj, nil
}

// secretScalar converts the value a Pointer resolved to into the string form the config map expects.
// Numbers keep their literal text so the later weakly-typed decode fills int fields exactly as an
// environment variable would.
func secretScalar(val any, ref secretRef, key string) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", errors.NewBadStatef(
			"JSON Pointer %s in secret: %s, for key: %s, resolves to %s, a config value needs a string, number or bool",
			ref.pointer, ref.id, key, jsonTypeName(val))
	}
}

func jsonTypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// resolveJSONPointer evaluates an RFC 6901 JSON Pointer against doc. pointer must start with '/'.
func resolveJSONPointer(doc any, pointer string) (any, error) {
	cur := doc
	for _, token := range strings.Split(pointer[1:], "/") {
		token = unescapeJSONPointerToken(token)
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[token]
			if !ok {
				return nil, errors.NewBadStatef("member %q not found", token)
			}
			cur = v
		case []any:
			idx, ok := parseJSONPointerIndex(token)
			if !ok {
				return nil, errors.NewBadStatef("%q is not a valid array index", token)
			}
			if idx >= len(node) {
				return nil, errors.NewBadStatef("array index %d is out of range", idx)
			}
			cur = node[idx]
		default:
			return nil, errors.NewBadStatef("cannot descend into a %s with token %q", jsonTypeName(cur), token)
		}
	}
	return cur, nil
}

// parseJSONPointerIndex accepts only the canonical decimal form required by RFC 6901: "0" or digits
// without a leading zero. The "-" (past-the-end) token is not addressable for reads.
func parseJSONPointerIndex(token string) (int, bool) {
	if token == "" || (len(token) > 1 && token[0] == '0') {
		return 0, false
	}
	for _, c := range token {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	idx, err := strconv.Atoi(token)
	return idx, err == nil
}

func escapeJSONPointerToken(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

func unescapeJSONPointerToken(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
}
