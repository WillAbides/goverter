package goverter

import (
	"fmt"
	"sort"
	"strings"
)

type methodIndexEntry[T any] struct {
	Def  *methodDefinition
	Item *T
}

type methodIndexID struct {
	sig    signature
	idx    int
	update bool
}

func newMethodIndex[T any]() *methodIndex[T] {
	return &methodIndex[T]{
		Exact: map[signature][]methodIndexEntry[T]{},
	}
}

type methodIndex[T any] struct {
	Exact  map[signature][]methodIndexEntry[T]
	Update []*T
}

func (l *methodIndex[T]) GetAll() []*T {
	var items []*T
	for _, exacts := range l.Exact {
		for _, exact := range exacts {
			items = append(items, exact.Item)
		}
	}
	return append(items, l.Update...)
}

func (l *methodIndex[T]) RegisterOverrideOverlapping(t *T, def *methodDefinition) {
	newEntry := methodIndexEntry[T]{Def: def, Item: t}
	for i, entry := range l.Exact[def.Signature] {
		if satisfiesContext(entry.Def.Context, def.Context) || satisfiesContext(def.Context, entry.Def.Context) {
			l.Exact[def.Signature][i] = newEntry
			return
		}
	}

	l.Exact[def.Signature] = append(l.Exact[def.Signature], newEntry)
}

func (l *methodIndex[T]) RegisterUpdate(t *T) (methodIndexID, error) {
	l.Update = append(l.Update, t)
	return methodIndexID{update: true, idx: len(l.Update) - 1}, nil
}

func (l *methodIndex[T]) Register(t *T, def *methodDefinition) (methodIndexID, error) {
	for _, entry := range l.Exact[def.Signature] {
		if err := checkMethodOverlap(entry.Def, def); err != nil {
			return methodIndexID{}, err
		}
		if err := checkMethodOverlap(def, entry.Def); err != nil {
			return methodIndexID{}, err
		}
	}

	newEntry := methodIndexEntry[T]{Def: def, Item: t}
	l.Exact[def.Signature] = append(l.Exact[def.Signature], newEntry)
	return methodIndexID{sig: def.Signature, idx: len(l.Exact[def.Signature]) - 1}, nil
}

func checkMethodOverlap(left, right *methodDefinition) error {
	if satisfiesContext(left.Context, right.Context) {
		return fmt.Errorf("Overlapping signatures found. All sources and contexts of this method\n    %s%s\n\nare contained in method\n    %s%s\n\nGoverter doesn't know which method to use when all contexts of the second method are available.\nRemove one of the methods to prevent this ambiguity.", left.ID, left.ArgDebug("        "), right.ID, right.ArgDebug("        "))
	}
	return nil
}

func (l *methodIndex[T]) ByID(id methodIndexID) *T {
	if id.update {
		return l.Update[id.idx]
	}
	return l.Exact[id.sig][id.idx].Item
}

func (l *methodIndex[T]) Has(sig signature) bool {
	_, ok := l.Exact[sig]
	return ok
}

func (l *methodIndex[T]) Get(sig signature, m map[string]*xType) (*T, error) {
	hits, ok := l.Exact[sig]
	if !ok {
		return nil, nil
	}

	for _, hit := range hits {
		if satisfiesContext(hit.Def.Context, m) {
			return hit.Item, nil
		}
	}

	return nil, satisfiedError(sig, m, hits)
}

func satisfiedError[T any](sig signature, available map[string]*xType, hits []methodIndexEntry[T]) error {
	var hitStrings []string
	for _, hit := range hits {
		hitStrings = append(hitStrings, fmt.Sprintf("%s:\n    %s", hit.Def.ID, strings.Join(availableContextDebug(hit.Def.Context, available), "\n    ")))
	}
	return fmt.Errorf(`Found custom functions(s) converting %s to %s
but not all required context params are available in the current method.

%s`, sig.Source, sig.Target, strings.Join(hitStrings, "\n\n"))
}

func availableContextDebug(required, available map[string]*xType) []string {
	var lines []string

	use := usageFromMap(available)
	for key := range required {
		_, ok := available[key]
		use.Used(key)

		prefix := "[available] "
		if !ok {
			prefix = "[missing]   "
		}
		lines = append(lines, prefix+key)
	}

	for _, x := range use.Unused() {
		lines = append(lines, "[unused]    "+x)
	}

	sort.Strings(lines)

	return lines
}

func satisfiesContext(required, m map[string]*xType) bool {
	for key := range required {
		if _, ok := m[key]; !ok {
			return false
		}
	}
	return true
}

func usageFromMap[V any](value map[string]V) UsageChecker {
	m := map[string]struct{}{}

	for key := range value {
		m[key] = struct{}{}
	}

	return m
}
