// Package journal holds Inkwell's domain types and the rules for users and journals.
package journal

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	// ErrNotFound means the record doesn't exist (or isn't the caller's).
	ErrNotFound = errors.New("not found")
	// ErrHandleTaken means another user already has that handle.
	ErrHandleTaken = errors.New("handle already taken")
)

// User is one person. ID is their Cognito "sub", so no separate mapping is needed.
type User struct {
	ID          string    `bson:"_id" json:"id"`
	Email       string    `bson:"email" json:"email"`
	DisplayName string    `bson:"displayName" json:"displayName"`
	Handle      *string   `bson:"handle,omitempty" json:"handle"` // nil until the user picks one
	AIOptIn     bool      `bson:"aiOptIn" json:"aiOptIn"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}

// Theme is how a journal looks.
type Theme struct {
	Paper    string `bson:"paper" json:"paper"`
	Font     string `bson:"font" json:"font"`
	Accent   string `bson:"accent" json:"accent"`
	CoverKey string `bson:"coverKey,omitempty" json:"coverKey,omitempty"`
}

// Journal is a user's notebook. MongoDB assigns its _id; the store fills in ID.
type Journal struct {
	ID         string    `bson:"-" json:"id"`
	OwnerID    string    `bson:"ownerId" json:"ownerId"`
	Title      string    `bson:"title" json:"title"`
	Visibility string    `bson:"visibility" json:"visibility"`
	Theme      Theme     `bson:"theme" json:"theme"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
}

// UpdateMeInput is what PATCH /api/me accepts. A nil field means "leave unchanged".
type UpdateMeInput struct {
	DisplayName *string `json:"displayName"`
	Handle      *string `json:"handle"`
	AIOptIn     *bool   `json:"aiOptIn"`
}

// ValidationError explains why input was rejected. Its message is safe to show the user.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// Repository stores users and journals.
type Repository interface {
	FindUser(ctx context.Context, id string) (User, error)
	EnsureUser(ctx context.Context, u User) (User, error)
	EnsureJournal(ctx context.Context, j Journal) error
	UpdateUser(ctx context.Context, id string, in UpdateMeInput) (User, error)
	FindJournal(ctx context.Context, ownerID string) (Journal, error)
	UpdateJournal(ctx context.Context, ownerID string, in UpdateJournalInput) (Journal, error)
}

// EmailLookup finds the email address belonging to an access token's user.
type EmailLookup interface {
	Email(ctx context.Context, accessToken string) (string, error)
}

// Service applies Inkwell's rules for users and journals.
type Service struct {
	repo   Repository
	emails EmailLookup
	now    func() time.Time
}

// NewService creates a Service.
func NewService(repo Repository, emails EmailLookup) *Service {
	return &Service{repo: repo, emails: emails, now: time.Now}
}

// Me returns the current user, creating the user and their journal on first sign-in.
func (s *Service) Me(ctx context.Context, userID, accessToken string) (User, error) {
	u, err := s.repo.FindUser(ctx, userID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return User{}, err
	}

	email, err := s.emails.Email(ctx, accessToken)
	if err != nil {
		return User{}, fmt.Errorf("look up email: %w", err)
	}

	now := storedTime(s.now())
	name := defaultDisplayName(email)

	// Journal first, then user: if anything fails in between, the next call retries both,
	// and "user exists" always implies "journal exists".
	if err := s.repo.EnsureJournal(ctx, NewJournal(userID, name, now)); err != nil {
		return User{}, fmt.Errorf("create journal: %w", err)
	}
	u, err = s.repo.EnsureUser(ctx, User{ID: userID, Email: email, DisplayName: name, CreatedAt: now})
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UpdateMe validates in and applies it to the current user.
func (s *Service) UpdateMe(ctx context.Context, userID string, in UpdateMeInput) (User, error) {
	clean, err := ValidateUpdate(in)
	if err != nil {
		return User{}, err
	}
	return s.repo.UpdateUser(ctx, userID, clean)
}

// storedTime is t as MongoDB stores it: UTC, to the millisecond. Using it for every timestamp
// means a value in a create response matches the same value read back later.
func storedTime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Millisecond)
}

// NewJournal returns a user's first journal with the default theme.
func NewJournal(ownerID, displayName string, now time.Time) Journal {
	return Journal{
		OwnerID:    ownerID,
		Title:      displayName + "'s Journal",
		Visibility: "private",
		Theme:      Theme{Paper: "dotted", Font: "serif", Accent: "#2f5d50"},
		CreatedAt:  now,
	}
}

const (
	maxDisplayNameRunes = 50
	defaultName         = "Writer"
)

var (
	handlePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,29}$`)
	reservedHandles = map[string]bool{
		"admin": true, "api": true, "healthz": true, "inkwell": true,
		"j": true, "root": true, "support": true, "www": true,
	}
)

// ValidateUpdate checks and normalises PATCH /api/me input.
func ValidateUpdate(in UpdateMeInput) (UpdateMeInput, error) {
	if in.DisplayName == nil && in.Handle == nil && in.AIOptIn == nil {
		return UpdateMeInput{}, &ValidationError{Message: "provide at least one of displayName, handle or aiOptIn"}
	}

	out := UpdateMeInput{AIOptIn: in.AIOptIn}

	if in.DisplayName != nil {
		name := strings.TrimSpace(*in.DisplayName)
		switch {
		case name == "":
			return UpdateMeInput{}, &ValidationError{Field: "displayName", Message: "must not be empty"}
		case utf8.RuneCountInString(name) > maxDisplayNameRunes:
			return UpdateMeInput{}, &ValidationError{Field: "displayName", Message: fmt.Sprintf("must be at most %d characters", maxDisplayNameRunes)}
		case strings.IndexFunc(name, unicode.IsControl) >= 0:
			return UpdateMeInput{}, &ValidationError{Field: "displayName", Message: "must not contain control characters"}
		}
		out.DisplayName = &name
	}

	if in.Handle != nil {
		handle := strings.ToLower(strings.TrimSpace(*in.Handle))
		switch {
		case !handlePattern.MatchString(handle):
			return UpdateMeInput{}, &ValidationError{Field: "handle", Message: "must be 3-30 characters: lowercase letters, numbers, _ or -, starting with a letter or number"}
		case reservedHandles[handle]:
			return UpdateMeInput{}, &ValidationError{Field: "handle", Message: "is reserved"}
		}
		out.Handle = &handle
	}

	return out, nil
}

// defaultDisplayName uses the part of the email before "@", e.g. "rishe@example.com" -> "rishe".
func defaultDisplayName(email string) string {
	local, _, _ := strings.Cut(email, "@")
	local = strings.TrimSpace(local)
	if local == "" {
		return defaultName
	}
	if utf8.RuneCountInString(local) > maxDisplayNameRunes {
		local = string([]rune(local)[:maxDisplayNameRunes])
	}
	return local
}
