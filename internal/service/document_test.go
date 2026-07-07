package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
)

func TestDocumentService_Create(t *testing.T) {
	var created *model.Document
	repo := &mockrepo.DocumentRepoerMock{
		CreateFunc: func(_ context.Context, entity *model.Document) error {
			created = entity
			return nil
		},
	}
	svc := service.NewDocumentService(repo)

	req := &dto.CreateDocumentDTO{
		SHA256:          "abc123",
		FileKey:         "docs/fake.pdf",
		Title:           "A Fake Document",
		Filename:        "fake.pdf",
		Filetype:        "application/pdf",
		Filesize:        1024,
		PageCount:       10,
		PublisherName:   "Fake Press",
		PublicationYear: 2020,
	}

	doc, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID == uuid.Nil {
		t.Error("expected a generated document ID")
	}
	if doc.Title != req.Title || doc.SHA256 != req.SHA256 || string(doc.FileURL) != req.FileKey {
		t.Errorf("document fields do not match request: %+v", doc)
	}
	if created != doc {
		t.Error("expected the same document passed to repo.Create to be returned")
	}
}

func TestDocumentService_Create_PropagatesRepoError(t *testing.T) {
	repo := &mockrepo.DocumentRepoerMock{
		CreateFunc: func(_ context.Context, _ *model.Document) error { return data.ErrInternal },
	}
	svc := service.NewDocumentService(repo)

	_, err := svc.Create(context.Background(), &dto.CreateDocumentDTO{})
	if !errors.Is(err, data.ErrInternal) {
		t.Errorf("expected data.ErrInternal, got %v", err)
	}
}

func TestDocumentService_GetByID(t *testing.T) {
	id := uuid.New()
	want := &model.Document{ID: id, Title: "Found"}
	repo := &mockrepo.DocumentRepoerMock{
		GetByIDFunc: func(_ context.Context, gotID uuid.UUID) (*model.Document, error) {
			if gotID != id {
				t.Errorf("expected repo.GetByID called with %s, got %s", id, gotID)
			}
			return want, nil
		},
	}
	svc := service.NewDocumentService(repo)

	got, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestDocumentService_GetByID_PropagatesNotFound(t *testing.T) {
	repo := &mockrepo.DocumentRepoerMock{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*model.Document, error) {
			return nil, data.ErrNotFound
		},
	}
	svc := service.NewDocumentService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, data.ErrNotFound) {
		t.Errorf("expected data.ErrNotFound, got %v", err)
	}
}

func TestDocumentService_List(t *testing.T) {
	want := &dto.PaginationResults[model.Document]{
		Page: 1, PerPage: 20, TotalItems: 1,
		Items: []model.Document{{ID: uuid.New()}},
	}
	repo := &mockrepo.DocumentRepoerMock{
		PaginateFunc: func(_ context.Context, _ data.SupportsPagination) (*dto.PaginationResults[model.Document], error) {
			return want, nil
		},
	}
	svc := service.NewDocumentService(repo)

	got, err := svc.List(context.Background(), &dto.PaginationDTO{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestDocumentService_Update(t *testing.T) {
	id := uuid.New()
	existing := &model.Document{ID: id, Title: "Old Title", PublisherName: "Old Press"}
	var updated *model.Document
	repo := &mockrepo.DocumentRepoerMock{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*model.Document, error) { return existing, nil },
		UpdateFunc: func(_ context.Context, entity *model.Document) error {
			updated = entity
			return nil
		},
	}
	svc := service.NewDocumentService(repo)

	req := &dto.UpdateDocumentDTO{
		Title:           "New Title",
		PublisherName:   "New Press",
		PublicationYear: 2021,
	}
	doc, err := svc.Update(context.Background(), id, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Title != req.Title || doc.PublisherName != req.PublisherName ||
		doc.PublicationYear != req.PublicationYear {
		t.Errorf("expected fields to be updated, got %+v", doc)
	}
	if updated != existing {
		t.Error("expected repo.Update to be called with the mutated existing document")
	}
}

func TestDocumentService_Update_NotFoundNeverCallsUpdate(t *testing.T) {
	updateCalled := false
	repo := &mockrepo.DocumentRepoerMock{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*model.Document, error) { return nil, data.ErrNotFound },
		UpdateFunc: func(_ context.Context, _ *model.Document) error {
			updateCalled = true
			return nil
		},
	}
	svc := service.NewDocumentService(repo)

	_, err := svc.Update(
		context.Background(),
		uuid.New(),
		&dto.UpdateDocumentDTO{Title: "New Title"},
	)
	if !errors.Is(err, data.ErrNotFound) {
		t.Errorf("expected data.ErrNotFound, got %v", err)
	}
	if updateCalled {
		t.Error("expected repo.Update not to be called when GetByID fails")
	}
}

func TestDocumentService_Delete(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.DocumentRepoerMock{
		DeleteFunc: func(_ context.Context, gotID uuid.UUID) error {
			if gotID != id {
				t.Errorf("expected repo.Delete called with %s, got %s", id, gotID)
			}
			return nil
		},
	}
	svc := service.NewDocumentService(repo)

	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocumentService_Delete_PropagatesRepoError(t *testing.T) {
	repo := &mockrepo.DocumentRepoerMock{
		DeleteFunc: func(_ context.Context, _ uuid.UUID) error { return data.ErrForbidden },
	}
	svc := service.NewDocumentService(repo)

	err := svc.Delete(context.Background(), uuid.New())
	if !errors.Is(err, data.ErrForbidden) {
		t.Errorf("expected data.ErrForbidden, got %v", err)
	}
}
