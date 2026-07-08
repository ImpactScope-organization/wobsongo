// Package service defines business logic needed to handle
// data operations across the system. It clearly separates the
// transport layer and data leyer, by relying on the data package
// that defines the data storage layer as interfaces.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// DocumentService defines a set of available methods
// related to documents operations.
type DocumentService struct {
	repo data.DocumentRepoer
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(repo data.DocumentRepoer) *DocumentService {
	return &DocumentService{
		repo: repo,
	}
}

// Create ingests a new document.
func (s *DocumentService) Create(
	ctx context.Context,
	req *dto.CreateDocumentDTO,
) (*model.Document, error) {
	now := time.Now()
	doc := &model.Document{
		ID:              uuid.New(),
		CreatedAt:       now,
		ModifiedAt:      now,
		FileURL:         model.S3Key(req.FileKey),
		SHA256:          req.SHA256,
		Title:           req.Title,
		Filename:        req.Filename,
		Filetype:        req.Filetype,
		Filesize:        req.Filesize,
		PageCount:       req.PageCount,
		PublisherName:   req.PublisherName,
		PublicationYear: req.PublicationYear,
	}

	err := s.repo.WithTx(ctx, func(txRepo data.DocumentRepoer) error {
		if err := txRepo.Create(ctx, doc); err != nil {
			return err
		}
		return txRepo.Enqueue(ctx, queue.ParseDocumentDTO{
			DocumentID: doc.ID,
			FileKey:    string(doc.FileURL),
		})
	})
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// GetByID retrieves a document by its ID.
func (s *DocumentService) GetByID(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves a paginated list of documents.
func (s *DocumentService) List(
	ctx context.Context,
	pagination *dto.PaginationDTO,
) (*dto.PaginationResults[model.Document], error) {
	return s.repo.Paginate(ctx, pagination)
}

// Update updates a document's descriptive metadata.
func (s *DocumentService) Update(
	ctx context.Context,
	id uuid.UUID,
	req *dto.UpdateDocumentDTO,
) (*model.Document, error) {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	doc.Title = req.Title
	doc.PublisherName = req.PublisherName
	doc.PublicationYear = req.PublicationYear
	doc.ModifiedAt = time.Now()

	if err := s.repo.Update(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// Delete removes a document by its ID.
func (s *DocumentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
