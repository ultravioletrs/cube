package guardrails

import "time"

type Metadata map[string]any

type PageMetadata struct {
	Total    uint64    `json:"total,omitempty" db:"total"`
	Offset   uint64    `json:"offset,omitempty" db:"offset"`
	Limit    uint64    `json:"limit,omitempty" db:"limit"`
	Name     string    `json:"name,omitempty"`
	Order    string    `json:"order,omitempty"`
	Dir      string    `json:"dir,omitempty"`
	User     string    `json:"user,omitempty"`
	Category string    `json:"category,omitempty"`
	From     time.Time `json:"-"`
	To       time.Time `json:"-"`
}
