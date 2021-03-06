package codegen

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var namespaceEscape = regexp.MustCompile("([/.])_?internal([/.]?)")

type Identifier struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (i Identifier) String() string {
	return i.GetQualifiedClasspath()
}

func (i *Identifier) GetIdentifier() Identifier {
	return *i
}

func (i Identifier) GetQualifiedClasspath() string {
	return i.Namespace + "." + i.Name
}

func (i Identifier) PackagePath() string {
	if i.Namespace == "" {
		Logger.Panicf("%+v has no namespace!", i)
	}
	if i.Name == "" {
		Logger.Panicf("%+v has no name!", i)
	}
	var p string
	if TypeRegistry.IsCyclic(i) {
		p = "conflictResolution"
	} else {
		p = i.Namespace
	}
	return FqcpToPackagePath(p)
}

func (i *Identifier) Receiver() string {
	return ReceiverName(i.Name)
}

func (i *Identifier) Resolve() ComplexType {
	return TypeRegistry.Resolve(*i)
}

func FqcpToPackagePath(fqcp string) string {
	fqcp = strings.Replace(namespaceEscape.ReplaceAllString(fqcp, "${1}_internal${2}"), ".", "/", -1)

	if PackagePrefix != "" {
		fqcp = filepath.Join(PackagePrefix, fqcp)
	}

	return fqcp
}

type IdentifierSet map[Identifier]bool

func (set IdentifierSet) Add(id Identifier) {
	set[id] = true
}

func (set IdentifierSet) AddAll(other IdentifierSet) {
	for id := range other {
		set[id] = true
	}
}

func (set IdentifierSet) Remove(id Identifier) {
	delete(set, id)
}

func (set IdentifierSet) Get(id Identifier) bool {
	return set[id]
}

func (set IdentifierSet) String() string {
	var classes []string
	for s := range set {
		classes = append(classes, s.GetQualifiedClasspath())
	}
	sort.Strings(classes)
	return "{" + strings.Join(classes, ", ") + "}"
}
