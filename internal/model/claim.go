package model

import (
	"time"

	"github.com/google/uuid"
)

// ClaimFormat describes how a claim reached the system.
type ClaimFormat string

const (
	// ClaimFormatText means the claim came from a typed message.
	ClaimFormatText ClaimFormat = "text"

	// ClaimFormatAudio means the claim came from a voice message, after
	// transcription.
	ClaimFormatAudio ClaimFormat = "audio"

	// ClaimFormatVideo means the claim came from a video's transcript.
	ClaimFormatVideo ClaimFormat = "video"
)

// ClaimPlatformWhatsApp is the platform value of claims that came through
// the WhatsApp bot.
const ClaimPlatformWhatsApp = "whatsapp"

// ClaimCitation is one piece of knowledge-base evidence the judge cited for a
// claim's verdict. It is stored inside Claim.Citations as JSON.
type ClaimCitation struct {
	// Index is the 0-based number the judge's reasoning text references
	// inline as "[N]", linking the prose back to this entry.
	Index int `json:"index"`

	// Source is "chunk" for a passage of a document or "fact" for an atomic
	// knowledge fact.
	Source string `json:"source"`

	// DocumentID is the source document the evidence came from.
	DocumentID uuid.UUID `json:"document_id"`

	// ChunkID is the chunk the evidence traces back to.
	ChunkID uuid.UUID `json:"chunk_id"`

	// Text is the matched text: a chunk's text, or a fact's
	// subject-predicate-object.
	Text string `json:"text"`
}

// Claim is one normalized sub-claim taken from a user's message, stored with
// its verdict from the knowledge-base check. Claims are the dashboard's
// source of truth for volume, trend and platform breakdown, and are grouped
// into narratives.
type Claim struct {
	// ID is the unique identifier of the claim.
	ID uuid.UUID `json:"id" format:"uuid"`

	// MessageID is the conversation message the claim came from. It is nil if
	// that message was removed from the chat history, and the claim stays
	// usable because it keeps its own text, platform and timestamp.
	MessageID *uuid.UUID `json:"message_id,omitempty" format:"uuid"`

	// Text is the sub-claim produced by the analyzer.
	Text string `json:"text"`

	// TopicID is the topic the analyzer assigned to the claim.
	TopicID uuid.UUID `json:"topic_id" format:"uuid"`

	// Language is the language the claim was written in.
	Language Language `json:"language"`

	// Platform is where the claim came from, e.g. ClaimPlatformWhatsApp.
	Platform string `json:"platform"`

	// Format is how the claim reached the system (text, audio or video).
	Format ClaimFormat `json:"format"`

	// OccurredAt is when the claim appeared (the message time). The "newest
	// claim" of a narrative is the linked claim with the greatest OccurredAt.
	OccurredAt time.Time `json:"occurred_at"`

	// Verdict is how the claim relates to the knowledge base's evidence.
	Verdict Verdict `json:"verdict"`

	// Severity is how high-stakes the claim's subject matter is. The
	// dashboard derives a narrative's harm estimate from it.
	Severity Severity `json:"severity"`

	// HighRisk is true when the analyzer or the keyword fail-safe flagged the
	// claim as high-risk.
	HighRisk bool `json:"high_risk"`

	// HighRiskReason explains why the claim was flagged as high-risk. Empty
	// when HighRisk is false.
	HighRiskReason string `json:"high_risk_reason,omitempty"`

	// RecommendMedicalConsult reports whether the response told the user to
	// see a medical professional.
	RecommendMedicalConsult bool `json:"recommend_medical_consult"`

	// Reasoning is the judge's detailed reasoning, with inline "[N]"
	// references to Citations.
	Reasoning string `json:"reasoning,omitempty"`

	// BriefReasoning is a short version of Reasoning for list views.
	BriefReasoning string `json:"brief_reasoning,omitempty"`

	// Citations is the evidence the judge cited, stored as JSON.
	Citations []ClaimCitation `json:"citations"`

	// JudgedAt is when the claim was evaluated against the knowledge base.
	JudgedAt time.Time `json:"judged_at"`

	// CreatedAt is the time the claim record was created.
	CreatedAt time.Time `json:"created_at"`
}
