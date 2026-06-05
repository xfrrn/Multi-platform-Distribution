package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type uploadProgressStore struct {
	mu    sync.Mutex
	items map[string]uploadProgressSnapshot
}

type uploadProgressSnapshot struct {
	ID        string    `json:"id"`
	Phase     string    `json:"phase"`
	Loaded    int64     `json:"loaded"`
	Total     int64     `json:"total"`
	Percent   int       `json:"percent"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newUploadProgressStore() *uploadProgressStore {
	return &uploadProgressStore{items: make(map[string]uploadProgressSnapshot)}
}

func (s *uploadProgressStore) update(id string, patch uploadProgressSnapshot) {
	if id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.items[id]
	current.ID = id
	if patch.Phase != "" {
		current.Phase = patch.Phase
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	if patch.Error != "" {
		current.Error = patch.Error
	}
	current.Loaded = patch.Loaded
	current.Total = patch.Total
	current.Percent = patch.Percent
	current.UpdatedAt = time.Now()
	s.items[id] = current
	s.cleanupLocked()
}

func (s *uploadProgressStore) get(id string) (uploadProgressSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	s.cleanupLocked()
	return item, ok
}

func (s *uploadProgressStore) cleanupLocked() {
	cutoff := time.Now().Add(-30 * time.Minute)
	for id, item := range s.items {
		if item.UpdatedAt.Before(cutoff) {
			delete(s.items, id)
		}
	}
}

func (h *ArtifactHandler) UploadProgress(c *gin.Context) {
	id := c.Param("uploadId")
	progress, ok := h.uploads.get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload progress not found"})
		return
	}
	c.JSON(http.StatusOK, progress)
}
