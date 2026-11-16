package service

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var errPipelineBoom = errors.New("pipeline dependency unavailable")

type pipelineFailure struct {
	stage  string
	reason string
}

// pipelineStore is a data.VideoRepoer that keeps the transcription and
// analysis state of videos in memory.
type pipelineStore struct {
	data.VideoRepoer
	videos          map[uuid.UUID]*model.Video
	status          map[uuid.UUID]model.VideoStatus
	transcripts     map[uuid.UUID]string
	failures        map[uuid.UUID]pipelineFailure
	analyzeEnqueued []uuid.UUID
	txCalls         int

	getErr         error
	mediaErr       error
	transcribedErr error
	enqueueErr     error
	analyzedErr    error
	failErr        error
}

// newPipelineStore stores the videos, all in the given status.
func newPipelineStore(status model.VideoStatus, videos ...*model.Video) *pipelineStore {
	s := &pipelineStore{
		videos:      make(map[uuid.UUID]*model.Video),
		status:      make(map[uuid.UUID]model.VideoStatus),
		transcripts: make(map[uuid.UUID]string),
		failures:    make(map[uuid.UUID]pipelineFailure),
	}
	for _, v := range videos {
		v.ID = uuid.New()
		s.videos[v.ID] = v
		s.status[v.ID] = status
	}
	return s
}

// WithTx runs fn and, like a database transaction, undoes everything the fake
// stored when fn fails.
func (s *pipelineStore) WithTx(_ context.Context, fn func(data.VideoRepoer) error) error {
	s.txCalls++
	status, transcripts := maps.Clone(s.status), maps.Clone(s.transcripts)
	enqueued := slices.Clone(s.analyzeEnqueued)

	if err := fn(s); err != nil {
		s.status, s.transcripts, s.analyzeEnqueued = status, transcripts, enqueued
		return err
	}
	return nil
}

func (s *pipelineStore) GetVideoForTranscription(
	_ context.Context,
	id uuid.UUID,
) (*model.Video, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.status[id] != model.VideoStatusScreenedIn {
		return nil, data.ErrNotFound
	}
	return s.videos[id], nil
}

func (s *pipelineStore) SetVideoMediaURL(_ context.Context, id uuid.UUID, mediaURL string) error {
	if s.mediaErr != nil {
		return s.mediaErr
	}
	if s.status[id] == model.VideoStatusScreenedIn {
		s.videos[id].MediaURL = &mediaURL
	}
	return nil
}

func (s *pipelineStore) SetVideoTranscribed(
	_ context.Context,
	id uuid.UUID,
	transcript string,
) (bool, error) {
	if s.transcribedErr != nil {
		return false, s.transcribedErr
	}
	if s.status[id] != model.VideoStatusScreenedIn {
		return false, nil
	}
	s.status[id] = model.VideoStatusTranscribed
	s.transcripts[id] = transcript
	return true, nil
}

func (s *pipelineStore) EnqueueAnalyzeVideoJob(
	_ context.Context,
	id uuid.UUID,
) (bool, error) {
	if s.enqueueErr != nil {
		return false, s.enqueueErr
	}
	s.analyzeEnqueued = append(s.analyzeEnqueued, id)
	return true, nil
}

func (s *pipelineStore) GetVideoForAnalysis(
	_ context.Context,
	id uuid.UUID,
) (*model.Video, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.status[id] != model.VideoStatusTranscribed {
		return nil, data.ErrNotFound
	}
	return s.videos[id], nil
}

func (s *pipelineStore) SetVideoAnalyzed(
	_ context.Context,
	id uuid.UUID,
	status model.VideoStatus,
) (bool, error) {
	if s.analyzedErr != nil {
		return false, s.analyzedErr
	}
	if s.status[id] != model.VideoStatusTranscribed {
		return false, nil
	}
	s.status[id] = status
	return true, nil
}

func (s *pipelineStore) SetVideoFailed(
	_ context.Context,
	id uuid.UUID,
	fromStatus model.VideoStatus,
	stage, reason string,
) (bool, error) {
	if s.failErr != nil {
		return false, s.failErr
	}
	if s.status[id] != fromStatus {
		return false, nil
	}
	s.status[id] = model.VideoStatusFailed
	s.failures[id] = pipelineFailure{stage: stage, reason: reason}
	return true, nil
}

// pipelineMediaResolver is a data.VideoMediaResolver.
type pipelineMediaResolver struct {
	url   string
	err   error
	calls []string
}

func (r *pipelineMediaResolver) ResolveMediaURL(
	_ context.Context,
	videoURL string,
) (string, error) {
	r.calls = append(r.calls, videoURL)
	if r.err != nil {
		return "", r.err
	}
	return r.url, nil
}

// pipelineTranscriber is a data.SpeechTranscriber. onCall runs before it answers.
type pipelineTranscriber struct {
	transcript string
	err        error
	onCall     func()
	gotURLs    []string
	gotFormats []string
}

func (tr *pipelineTranscriber) TranscribeURL(
	_ context.Context,
	audioURL, format string,
) (string, error) {
	tr.gotURLs = append(tr.gotURLs, audioURL)
	tr.gotFormats = append(tr.gotFormats, format)
	if tr.onCall != nil {
		tr.onCall()
	}
	if tr.err != nil {
		return "", tr.err
	}
	return tr.transcript, nil
}

// pipelineClaimChecker is a ClaimChecker. onCheck runs before it answers.
type pipelineClaimChecker struct {
	result  *ClaimCheckResult
	err     error
	onCheck func()
	reqs    []*dto.CheckClaimDTO
}

func (c *pipelineClaimChecker) CheckClaim(
	_ context.Context,
	req *dto.CheckClaimDTO,
) (*ClaimCheckResult, error) {
	c.reqs = append(c.reqs, req)
	if c.onCheck != nil {
		c.onCheck()
	}
	if c.err != nil {
		return nil, c.err
	}
	return c.result, nil
}
