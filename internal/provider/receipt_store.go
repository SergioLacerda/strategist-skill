package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReceiptNonceStore claims a nonce once for a mission. Implementations must
// retain a claim for at least the receipt validity window.
type ReceiptNonceStore interface {
	Claim(missionID, nonce string, issuedAt time.Time) error
}

// FileReceiptNonceStore stores only a hash of mission and nonce. O_EXCL makes
// competing claims atomic across processes without recording a raw nonce.
type FileReceiptNonceStore struct {
	Root string
}

// Claim records a nonce or rejects an existing claim as a replay.
func (s FileReceiptNonceStore) Claim(missionID, nonce string, _ time.Time) error {
	if s.Root == "" {
		return fmt.Errorf("receipt nonce storage is unavailable")
	}
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return fmt.Errorf("create receipt nonce storage: %w", err)
	}
	sum := sha256.Sum256([]byte(missionID + "\x00" + nonce))
	path := filepath.Join(s.Root, hex.EncodeToString(sum[:]))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: path is a sha256 hex digest of missionID+nonce joined to s.Root, not raw user input
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("invocation receipt nonce was replayed")
	}
	if err != nil {
		return fmt.Errorf("claim invocation receipt nonce: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close invocation receipt nonce file: %w", err)
	}
	return nil
}

// MemoryReceiptNonceStore is a hermetic test implementation.
type MemoryReceiptNonceStore struct {
	mu     sync.Mutex
	claims map[string]struct{}
}

// NewMemoryReceiptNonceStore returns an empty in-memory replay store.
func NewMemoryReceiptNonceStore() *MemoryReceiptNonceStore {
	return &MemoryReceiptNonceStore{claims: map[string]struct{}{}}
}

// Claim records a nonce once in memory.
func (s *MemoryReceiptNonceStore) Claim(missionID, nonce string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := missionID + "\x00" + nonce
	if _, found := s.claims[key]; found {
		return fmt.Errorf("invocation receipt nonce was replayed")
	}
	s.claims[key] = struct{}{}
	return nil
}
