package proto

import (
	"sort"

	"github.com/samber/lo"
)

// FileFields will return a slice of fields for the model that are of type file
func (m *Model) FileFields() []*Field {
	return lo.Filter(m.Fields, func(f *Field, _ int) bool {
		return f.IsFile()
	})
}

// HasFiles checks if the model has any fields that are files
func (m *Model) HasFiles() bool {
	return len(m.FileFields()) > 0
}

// FieldNames provides a (sorted) list of the fields in the model of the given name.
func (m *Model) FieldNames() []string {
	names := lo.Map(m.Fields, func(x *Field, _ int) string {
		return x.Name
	})
	sort.Strings(names)
	return names
}

// ForeignKeyFields returns all the fields in the given model which have their ForeignKeyInfo
// populated.
func (m *Model) ForeignKeyFields() []*Field {
	return lo.Filter(m.Fields, func(f *Field, _ int) bool {
		return f.ForeignKeyInfo != nil
	})
}

// PrimaryKeyFieldName returns the name of the field in the given model,
// that is marked as being the model's primary key. (Or empty string).
func (m *Model) PrimaryKeyFieldName() string {
	field, _ := lo.Find(m.Fields, func(f *Field) bool {
		return f.PrimaryKey
	})
	if field != nil {
		return field.Name
	}
	return ""
}
