package storage

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// local disk storage: STORAGE_DIR/<org_id>/<doc_id>/<file>
type Local struct {
	root string
}

func NewLocal(root string) *Local { return &Local{root: root} }

func (l *Local) Save(orgID, docID uuid.UUID, fileName string, data []byte) (string, error) {
	dir := filepath.Join(l.root, orgID.String(), docID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// keep only base name - avoid path traversal
	path := filepath.Join(dir, filepath.Base(fileName))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (l *Local) Read(path string) ([]byte, error) { return os.ReadFile(path) }

func (l *Local) Delete(path string) error { return os.Remove(path) }
