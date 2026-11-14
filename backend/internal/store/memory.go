package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"slices"
	"sync"
	"time"

	"phystech-answer/backend/internal/model"
)

type Memory struct {
	mu    sync.RWMutex
	docs  map[string]model.Document
	users map[string]model.User
	reads map[string]map[string]time.Time // docID -> userID -> readAt
}

func NewMemory() *Memory {
	return &Memory{
		docs:  map[string]model.Document{},
		users: map[string]model.User{},
		reads: map[string]map[string]time.Time{},
	}
}

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func (m *Memory) ListDocuments(_ context.Context) ([]model.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	docs := make([]model.Document, 0, len(m.docs))
	for _, d := range m.docs {
		d.Tags = slices.Clone(d.Tags)
		docs = append(docs, d)
	}
	slices.SortFunc(docs, func(a, b model.Document) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return docs, nil
}

func (m *Memory) GetDocument(_ context.Context, id string) (model.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	d, ok := m.docs[id]
	if !ok {
		return model.Document{}, ErrNotFound
	}
	d.Tags = slices.Clone(d.Tags)
	return d, nil
}

func (m *Memory) CreateDocument(_ context.Context, doc model.Document) (model.Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	doc.ID = newID("doc")
	doc.Version = 1
	doc.CreatedAt = now
	doc.UpdatedAt = now
	m.docs[doc.ID] = doc
	return doc, nil
}

func (m *Memory) UpdateDocument(_ context.Context, doc model.Document) (model.Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.docs[doc.ID]
	if !ok {
		return model.Document{}, ErrNotFound
	}
	doc.Version = existing.Version + 1
	doc.CreatedAt = existing.CreatedAt
	doc.UpdatedAt = time.Now()
	m.docs[doc.ID] = doc
	return doc, nil
}

func (m *Memory) DeleteDocument(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.docs[id]; !ok {
		return ErrNotFound
	}
	delete(m.docs, id)
	delete(m.reads, id)
	return nil
}

func (m *Memory) ListUsers(_ context.Context) ([]model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}
	slices.SortFunc(users, func(a, b model.User) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return users, nil
}

func (m *Memory) CreateUser(_ context.Context, user model.User) (model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user.ID = newID("usr")
	user.CreatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *Memory) MarkRead(_ context.Context, docID, userID string) (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.docs[docID]; !ok {
		return time.Time{}, ErrNotFound
	}
	if _, ok := m.users[userID]; !ok {
		return time.Time{}, ErrNotFound
	}
	readAt := time.Now()
	if m.reads[docID] == nil {
		m.reads[docID] = map[string]time.Time{}
	}
	m.reads[docID][userID] = readAt
	return readAt, nil
}

func (m *Memory) ReadStatus(_ context.Context, userID string) (map[string]time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]time.Time, len(m.reads))
	for docID, byUser := range m.reads {
		if t, ok := byUser[userID]; ok {
			status[docID] = t
		}
	}
	return status, nil
}
