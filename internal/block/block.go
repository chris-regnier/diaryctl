// Package block provides the Block data structure for diary entries.
// Blocks are atomic content units with timestamps and flat key-value attributes.
package block

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// Block represents an atomic content unit within a day.
// Blocks are immutably ordered by CreatedAt timestamp within each day.
type Block struct {
	// ID is an 8-character lowercase alphanumeric nanoid that uniquely identifies this block
	ID string

	// Content is the text content of the block (must be non-empty)
	Content string

	// CreatedAt is the immutable timestamp that determines the block's position
	// within a day. Blocks are always ordered by CreatedAt ascending.
	CreatedAt time.Time

	// UpdatedAt tracks when the block was last modified
	UpdatedAt time.Time

	// Attributes is a flat map of key-value pairs for metadata
	// Examples: type=note, tags=work,personal, mood=happy
	Attributes map[string]string
}

const (
	// idAlphabet is the character set used for generating block IDs
	idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	// idLength is the length of generated block IDs
	idLength = 8
)

var (
	// idPattern validates that an ID matches the expected format
	idPattern = regexp.MustCompile(`^[a-z0-9]{8}$`)

	// ErrInvalidID indicates that a block ID doesn't match the required format
	ErrInvalidID = errors.New("invalid block ID: must be 8 lowercase alphanumeric characters")

	// ErrEmptyContent indicates that a block's content is empty
	ErrEmptyContent = errors.New("block content must not be empty")
)

// NewID generates a new 8-character lowercase alphanumeric block ID
// using the nanoid algorithm with a custom alphabet.
// The generated ID has high entropy and is suitable for use as a unique identifier.
// Panics if ID generation fails, as this is a critical system failure.
func NewID() string {
	// Use custom alphabet for lowercase alphanumeric characters
	id, err := gonanoid.Generate(idAlphabet, idLength)
	if err != nil {
		// ID generation failure is critical - this should never happen with valid alphabet/length
		// If it does happen, we must panic as continuing could create data corruption
		panic(fmt.Sprintf("critical: failed to generate block ID: %v", err))
	}
	return id
}

// ValidateID checks whether the given string is a valid block ID.
// A valid ID must be exactly 8 characters long and contain only
// lowercase letters (a-z) and digits (0-9).
//
// Returns nil if the ID is valid, or ErrInvalidID if it's not.
func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return ErrInvalidID
	}
	return nil
}

// ValidateContent checks whether the given content string is valid.
// Content must be non-empty (after trimming whitespace is optional,
// but here we check for any non-empty string as per spec).
//
// Returns nil if the content is valid, or ErrEmptyContent if it's empty.
func ValidateContent(content string) error {
	if content == "" {
		return ErrEmptyContent
	}
	return nil
}
