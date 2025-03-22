package method

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jmattheis/goverter"
)

type methodIndexEntry[T any] struct {
	Def  *goverter.MethodDefinition
	Item *T
}

type MethodIndexID struct {
	sig    goverter.Signature
	idx    int
	update bool
}

func NewMethodIndex[T any]() *MethodIndex[T] {
	return &MethodIndex[T]{
		Exact: map[goverter.Signature][]methodIndexEntry[T]{},
	}
}

type MethodIndex[T any] struct {
	Exact  map[goverter.Signature][]methodIndexEntry[T]
	Update []*T
}

func (l *MethodIndex[T]) GetAll() []*T {
	var items []*T
	for _, exacts := range l.Exact {
		for _, exact := range exacts {
			items = append(items, exact.Item)
		}
	}
	return append(items, l.Update...)
}

func (l *MethodIndex[T]) RegisterOverrideOverlapping(t *T, def *goverter.MethodDefinition) {
	newEntry := methodIndexEntry[T]{Def: def, Item: t}
	for i, entry := range l.Exact[def.Signature] {
		if satisfiesContext(entry.Def.Context, def.Context) || satisfiesContext(def.Context, entry.Def.Context) {
			l.Exact[def.Signature][i] = newEntry
			return
		}
	}

	l.Exact[def.Signature] = append(l.Exact[def.Signature], newEntry)
}

func (l *MethodIndex[T]) RegisterUpdate(t *T) (MethodIndexID, error) {
	l.Update = append(l.Update, t)
	return MethodIndexID{update: true, idx: len(l.Update) - 1}, nil
}

func (l *MethodIndex[T]) Register(t *T, def *goverter.MethodDefinition) (MethodIndexID, error) {
	for _, entry := range l.Exact[def.Signature] {
		if err := checkMethodOverlap(entry.Def, def); err != nil {
			return MethodIndexID{}, err
		}
		if err := checkMethodOverlap(def, entry.Def); err != nil {
			return MethodIndexID{}, err
		}
	}

	newEntry := methodIndexEntry[T]{Def: def, Item: t}
	l.Exact[def.Signature] = append(l.Exact[def.Signature], newEntry)
	return MethodIndexID{sig: def.Signature, idx: len(l.Exact[def.Signature]) - 1}, nil
}

func checkMethodOverlap(left, right *goverter.MethodDefinition) error {
	if satisfiesContext(left.Context, right.Context) {
		return fmt.Errorf("Overlapping signatures found. All sources and contexts of this method\n    %s%s\n\nare contained in method\n    %s%s\n\nGoverter doesn't know which method to use when all contexts of the second method are available.\nRemove one of the methods to prevent this ambiguity.", left.ID, left.ArgDebug("        "), right.ID, right.ArgDebug("        "))
	}
	return nil
}

func (l *MethodIndex[T]) ByID(id MethodIndexID) *T {
	if id.update {
		return l.Update[id.idx]
	}
	return l.Exact[id.sig][id.idx].Item
}

func (l *MethodIndex[T]) Has(sig goverter.Signature) bool {
	_, ok := l.Exact[sig]
	return ok
}

func (l *MethodIndex[T]) Get(sig goverter.Signature, m map[string]*goverter.Type) (*T, error) {
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

func satisfiedError[T any](sig goverter.Signature, available map[string]*goverter.Type, hits []methodIndexEntry[T]) error {
	var hitStrings []string
	for _, hit := range hits {
		hitStrings = append(hitStrings, fmt.Sprintf("%s:\n    %s", hit.Def.ID, strings.Join(AvailableContextDebug(hit.Def.Context, available), "\n    ")))
	}
	return fmt.Errorf(`Found custom functions(s) converting %s to %s
but not all required context params are available in the current method.

%s`, sig.Source, sig.Target, strings.Join(hitStrings, "\n\n"))
}

func AvailableContextDebug(required, available map[string]*goverter.Type) []string {
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

func satisfiesContext(required, m map[string]*goverter.Type) bool {
	for key := range required {
		if _, ok := m[key]; !ok {
			return false
		}
	}
	return true
}

func usageFromMap[V any](value map[string]V) goverter.UsageChecker {
	m := map[string]struct{}{}

	for key := range value {
		m[key] = struct{}{}
	}

	return m
}
