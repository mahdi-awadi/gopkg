// Package i18n provides shared internationalization helpers.
//
// LocalizedString is the project-standard JSONB i18n storage type used by
// airports, cities, countries, airlines, bus charter, and charter hotel.
package i18n

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// LocalizedString is a map of language code to localized text, stored as JSONB.
// Example: {"en": "Hilton Baghdad", "ar": "هيلتون بغداد", "ku": "..."}.
type LocalizedString map[string]string

// Scan implements sql.Scanner for reading JSONB from the database.
func (ls *LocalizedString) Scan(src interface{}) error {
	if src == nil {
		*ls = LocalizedString{}
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, ls)
	case string:
		return json.Unmarshal([]byte(v), ls)
	}
	return fmt.Errorf("unsupported type for LocalizedString: %T", src)
}

// Value implements driver.Valuer for writing JSONB to the database.
func (ls LocalizedString) Value() (driver.Value, error) {
	if ls == nil {
		return "{}", nil
	}
	return json.Marshal(ls)
}

// Get returns the value for the given language, falling back to English,
// then to the first non-empty value in the map. Returns "" if the map is
// empty or nil.
func (ls LocalizedString) Get(lang string) string {
	if ls == nil {
		return ""
	}
	if v, ok := ls[lang]; ok && v != "" {
		return v
	}
	if v, ok := ls["en"]; ok && v != "" {
		return v
	}
	for _, v := range ls {
		if v != "" {
			return v
		}
	}
	return ""
}
