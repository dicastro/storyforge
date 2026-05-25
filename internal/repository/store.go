// Package repository provides filesystem-based persistence for storyforge content.
// There is no database — all data lives in YAML files under the content/ directory.
package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dicastro/storyforge/internal/model"
	"gopkg.in/yaml.v3"
)

// Store is the entry point for all content operations.
type Store struct {
	contentRoot string
}

// NewStore creates a Store rooted at the given directory.
func NewStore(contentRoot string) *Store {
	return &Store{contentRoot: contentRoot}
}

// ---- Sagas -------------------------------------------------------------------

// SagasDir returns the path to the sagas directory.
func (s *Store) SagasDir() string {
	return filepath.Join(s.contentRoot, "sagas")
}

// StandaloneDir returns the path to standalone books.
func (s *Store) StandaloneDir() string {
	return filepath.Join(s.contentRoot, "standalone")
}

// GlobalCharactersDir returns the path to cross-saga characters.
func (s *Store) GlobalCharactersDir() string {
	return filepath.Join(s.contentRoot, "characters")
}

// ListSagaIDs returns the id of every saga found on disk.
func (s *Store) ListSagaIDs() ([]string, error) {
	return listSubdirectories(s.SagasDir())
}

// LoadSaga reads saga.yaml and all its character files.
func (s *Store) LoadSaga(sagaID string) (*model.Saga, error) {
	sagaDir := filepath.Join(s.SagasDir(), sagaID)
	sagaFile := filepath.Join(sagaDir, "saga.yaml")

	saga, err := decodeYAML[model.Saga](sagaFile)
	if err != nil {
		return nil, fmt.Errorf("loading saga %q: %w", sagaID, err)
	}

	for _, charID := range saga.Characters {
		char, err := s.loadSagaCharacter(sagaID, charID)
		if err != nil {
			return nil, err
		}
		saga.ResolvedCharacters = append(saga.ResolvedCharacters, char)
	}

	return saga, nil
}

func (s *Store) loadSagaCharacter(sagaID, charID string) (*model.Character, error) {
	path := filepath.Join(s.SagasDir(), sagaID, "characters", charID+".yaml")
	char, err := decodeYAML[model.Character](path)
	if err != nil {
		return nil, fmt.Errorf("loading character %q for saga %q: %w", charID, sagaID, err)
	}
	return char, nil
}

// ---- Books -------------------------------------------------------------------

// ListBookIDsForSaga returns the id of every book folder inside a saga.
func (s *Store) ListBookIDsForSaga(sagaID string) ([]string, error) {
	booksDir := filepath.Join(s.SagasDir(), sagaID, "books")
	return listSubdirectories(booksDir)
}

// LoadBook reads a book.yaml and resolves saga + character references.
func (s *Store) LoadBook(sagaID, bookID string) (*model.Book, error) {
	bookDir := filepath.Join(s.SagasDir(), sagaID, "books", bookID)
	bookFile := filepath.Join(bookDir, "book.yaml")

	book, err := decodeYAML[model.Book](bookFile)
	if err != nil {
		return nil, fmt.Errorf("loading book %q/%q: %w", sagaID, bookID, err)
	}
	book.RootPath = bookDir

	saga, err := s.LoadSaga(sagaID)
	if err != nil {
		return nil, err
	}
	book.Saga = saga

	charMap := make(map[string]*model.Character)
	for _, c := range saga.ResolvedCharacters {
		charMap[c.ID] = c
	}
	for _, ref := range book.Characters {
		if _, exists := charMap[ref.ID]; !exists && ref.Name != "" {
			charMap[ref.ID] = &model.Character{
				ID:           ref.ID,
				Name:         ref.Name,
				Role:         ref.Role,
				Description:  ref.Description,
				VisualPrompt: ref.VisualPrompt,
			}
		}
	}
	for _, c := range charMap {
		book.ResolvedCharacters = append(book.ResolvedCharacters, c)
	}

	return book, nil
}

// LoadStandaloneBook reads a book from the standalone/ directory.
func (s *Store) LoadStandaloneBook(bookID string) (*model.Book, error) {
	bookDir := filepath.Join(s.StandaloneDir(), bookID)
	bookFile := filepath.Join(bookDir, "book.yaml")

	book, err := decodeYAML[model.Book](bookFile)
	if err != nil {
		return nil, fmt.Errorf("loading standalone book %q: %w", bookID, err)
	}
	book.RootPath = bookDir
	return book, nil
}

// ListAllBooks returns every book across sagas and standalone collections.
func (s *Store) ListAllBooks() ([]*model.Book, error) {
	var books []*model.Book

	sagaIDs, err := s.ListSagaIDs()
	if err != nil {
		return nil, err
	}
	for _, sagaID := range sagaIDs {
		bookIDs, err := s.ListBookIDsForSaga(sagaID)
		if err != nil {
			return nil, err
		}
		for _, bookID := range bookIDs {
			book, err := s.LoadBook(sagaID, bookID)
			if err != nil {
				return nil, err
			}
			books = append(books, book)
		}
	}

	standaloneIDs, err := listSubdirectories(s.StandaloneDir())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, id := range standaloneIDs {
		book, err := s.LoadStandaloneBook(id)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

// ---- Helpers -----------------------------------------------------------------

func decodeYAML[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	var v T
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", path, err)
	}
	return &v, nil
}

func listSubdirectories(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}