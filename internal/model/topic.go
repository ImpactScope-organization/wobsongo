package model

import (
	"time"

	"github.com/google/uuid"
)

// Topic is one entry of the closed topic list. The claim analyzer assigns
// exactly one active topic to every analysis, and the dashboard uses topics
// to group and filter claims and narratives.
// Topics are never deleted, because claims and narratives reference them. An
// admin takes a topic out of use by setting IsActive to false.
type Topic struct {
	// ID is the unique identifier of the topic.
	ID uuid.UUID `json:"id" format:"uuid"`

	// Name is the topic's display name. It is unique across all topics.
	Name string `json:"name"`

	// Description explains what the topic covers. It is included in the
	// analyzer prompt so the model can classify claims accurately.
	Description string `json:"description"`

	// IsActive reports whether the analyzer may still assign this topic to
	// new claims. Existing claims and narratives keep an inactive topic.
	IsActive bool `json:"is_active"`

	// CreatedAt is the time the topic was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the time the topic was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}
