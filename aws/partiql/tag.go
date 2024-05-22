package partiql

import "strings"

type tagValue struct {
	Name   string
	Ignore bool
	Squash bool
	Bag    bool
}

func parseTag(tagStr string) tagValue {
	name, opts, _ := strings.Cut(tagStr, ",")
	return tagValue{
		Name:   name,
		Ignore: name == "-",
		Squash: strings.Contains(opts, "squash"),
		Bag:    strings.Contains(opts, "bag"),
	}
}
