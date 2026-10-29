package model

import (
	"time"

	"github.com/google/uuid"
)

// NarrativeStatus describes how far a narrative's fact-check has progressed.
type NarrativeStatus string

const (
	// NarrativeStatusNeedsCheck means the narrative has not been checked yet.
	NarrativeStatusNeedsCheck NarrativeStatus = "needs_check"

	// NarrativeStatusAutoChecked means the system checked the narrative
	// against the knowledge base and found relevant evidence.
	NarrativeStatusAutoChecked NarrativeStatus = "auto_checked"

	// NarrativeStatusNotInKB means the knowledge base had no relevant
	// evidence (the newest claim's verdict is insufficient evidence).
	NarrativeStatusNotInKB NarrativeStatus = "not_in_kb"

	// NarrativeStatusExpertVerified means an expert reviewed the narrative.
	NarrativeStatusExpertVerified NarrativeStatus = "expert_verified"
)

const (
	// NarrativeHarmMin is the lowest harm level.
	NarrativeHarmMin = 1

	// NarrativeHarmMax is the highest harm level.
	NarrativeHarmMax = 5
)

// Narrative groups claims that share the same meaning within one topic.
type Narrative struct {
	// ID is the unique identifier of the narrative.
	ID uuid.UUID `json:"id" format:"uuid"`

	// TopicID is the topic the narrative belongs to. A claim is only matched
	// against narratives of its own topic.
	TopicID uuid.UUID `json:"topic_id" format:"uuid"`

	// Statement describes the narrative. An LLM writes it, deliberately broad,
	// when the narrative is created, and an admin may edit it afterwards.
	Statement string `json:"statement"`

	// TopicName is the name of the narrative's topic.
	TopicName string `json:"topic_name"`

	// Verdict is the verdict of the narrative's newest claim (supported,
	// contradicted, partially_supported, mixed or insufficient_evidence).
	// Nil while the narrative has no claim.
	Verdict *string `json:"verdict,omitempty"`

	// Harm is the harm level between NarrativeHarmMin and NarrativeHarmMax.
	// Until an expert validates it, the system estimates it from the severity
	// and high-risk flag of the newest claim. Nil while there is no estimate.
	Harm *int `json:"harm,omitempty"`

	// HarmValidated is true once an expert confirmed or set Harm. The system
	// then stops recomputing Harm.
	HarmValidated bool `json:"harm_validated"`

	// Status is the narrative's fact-check status.
	Status NarrativeStatus `json:"status"`

	// IsActive reports whether the narrative is still actively discussed,
	// based on activity observed within a defined time period: it is true
	// while a linked claim occurred inside the activity window. The system
	// sets it to true when a claim joins the narrative, and back to false when
	// the window passes without new claims.
	IsActive bool `json:"is_active"`

	// StatementEditedBy is the admin who last edited Statement. Nil while the
	// LLM-written statement is untouched.
	StatementEditedBy *uuid.UUID `json:"statement_edited_by,omitempty" format:"uuid"`

	// StatementEditedAt is when Statement was last edited by an admin.
	StatementEditedAt *time.Time `json:"statement_edited_at,omitempty"`

	// CreatedAt is the time the narrative was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the time the narrative was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// NarrativeClaim links one claim to one narrative. A claim may belong to
// several narratives and a narrative holds many claims. The pair
// (ClaimID, NarrativeID) is unique.
type NarrativeClaim struct {
	// ID is the unique identifier of the link.
	ID uuid.UUID `json:"id" format:"uuid"`

	// ClaimID is the linked claim.
	ClaimID uuid.UUID `json:"claim_id" format:"uuid"`

	// NarrativeID is the linked narrative.
	NarrativeID uuid.UUID `json:"narrative_id" format:"uuid"`

	// CreatedAt is the time the claim joined the narrative.
	CreatedAt time.Time `json:"created_at"`
}
