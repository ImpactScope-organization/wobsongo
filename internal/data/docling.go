package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/model"
)

// ProcessedDocument is the parsed, unfiltered output of an external document processor.
type ProcessedDocument struct {
	// Title is the document-level title, if the processor could detect one.
	Title string

	// PageCount is the total number of pages in the document.
	PageCount int

	// Chunks contains every extracted element, unfiltered. Callers decide
	// which layout types to keep.
	Chunks []model.ParsedChunk
}

// DocumentProcessor defines the contract for extracting structured content from a document.
type DocumentProcessor interface {
	// ProcessFromURL fetches the document at documentURL, parses its layout,
	// and returns all extracted chunks and document-level metadata, unfiltered.
	ProcessFromURL(ctx context.Context, documentURL string) (*ProcessedDocument, error)
}
