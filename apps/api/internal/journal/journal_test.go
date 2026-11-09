package journal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeRepo is an in-memory Repository that records what was called.
type fakeRepo struct {
	users    map[string]User
	journals map[string]Journal
	calls    []string
	findErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[string]User{}, journals: map[string]Journal{}}
}

func (f *fakeRepo) FindUser(_ context.Context, id string) (User, error) {
	f.calls = append(f.calls, "FindUser")
	if f.findErr != nil {
		return User{}, f.findErr
	}
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeRepo) EnsureUser(_ context.Context, u User) (User, error) {
	f.calls = append(f.calls, "EnsureUser")
	if existing, ok := f.users[u.ID]; ok {
		return existing, nil
	}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeRepo) EnsureJournal(_ context.Context, j Journal) error {
	f.calls = append(f.calls, "EnsureJournal")
	if _, ok := f.journals[j.OwnerID]; !ok {
		f.journals[j.OwnerID] = j
	}
	return nil
}

func (f *fakeRepo) UpdateUser(_ context.Context, id string, in UpdateMeInput) (User, error) {
	f.calls = append(f.calls, "UpdateUser")
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	if in.DisplayName != nil {
		u.DisplayName = *in.DisplayName
	}
	f.users[id] = u
	return u, nil
}

type fakeEmails struct {
	email string
	err   error
	calls int
}

func (f *fakeEmails) Email(context.Context, string) (string, error) {
	f.calls++
	return f.email, f.err
}

func TestMeCreatesUserAndJournalOnFirstCall(t *testing.T) {
	repo := newFakeRepo()
	emails := &fakeEmails{email: "rishe@example.com"}
	svc := NewService(repo, emails)
	svc.now = func() time.Time { return time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC) }

	u, err := svc.Me(context.Background(), "user-123", "token")
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if u.ID != "user-123" || u.Email != "rishe@example.com" || u.DisplayName != "rishe" {
		t.Errorf("Me() = %+v", u)
	}

	j, ok := repo.journals["user-123"]
	if !ok {
		t.Fatal("journal was not created")
	}
	if j.Title != "rishe's Journal" || j.Visibility != "private" || j.Theme.Paper != "dotted" {
		t.Errorf("journal = %+v", j)
	}

	// Journal must be created before the user (see the comment in Me).
	want := "FindUser,EnsureJournal,EnsureUser"
	if got := strings.Join(repo.calls, ","); got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
}

func TestMeReturnsExistingUserWithoutLookups(t *testing.T) {
	repo := newFakeRepo()
	repo.users["user-123"] = User{ID: "user-123", DisplayName: "Rishi"}
	emails := &fakeEmails{email: "rishe@example.com"}

	u, err := NewService(repo, emails).Me(context.Background(), "user-123", "token")
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if u.DisplayName != "Rishi" {
		t.Errorf("DisplayName = %q, want %q", u.DisplayName, "Rishi")
	}
	if emails.calls != 0 {
		t.Errorf("email lookups = %d, want 0", emails.calls)
	}
}

func TestMeFailsSafely(t *testing.T) {
	t.Run("email lookup fails: nothing is written", func(t *testing.T) {
		repo := newFakeRepo()
		_, err := NewService(repo, &fakeEmails{err: errors.New("cognito down")}).Me(context.Background(), "user-123", "token")
		if err == nil {
			t.Fatal("Me(): expected an error, got nil")
		}
		if len(repo.users) != 0 || len(repo.journals) != 0 {
			t.Error("Me() wrote data despite failing")
		}
	})

	t.Run("database error is returned, not treated as a new user", func(t *testing.T) {
		repo := newFakeRepo()
		repo.findErr = errors.New("connection refused")
		emails := &fakeEmails{email: "a@b.c"}
		if _, err := NewService(repo, emails).Me(context.Background(), "user-123", "token"); err == nil {
			t.Fatal("Me(): expected an error, got nil")
		}
		if emails.calls != 0 {
			t.Error("Me() looked up the email after a database error")
		}
	})
}

func ptr[T any](v T) *T { return &v }

func TestValidateUpdate(t *testing.T) {
	tests := []struct {
		name       string
		in         UpdateMeInput
		wantErr    bool
		wantName   string
		wantHandle string
	}{
		{name: "nothing to update", in: UpdateMeInput{}, wantErr: true},
		{name: "display name trimmed", in: UpdateMeInput{DisplayName: ptr("  Rishi  ")}, wantName: "Rishi"},
		{name: "empty display name", in: UpdateMeInput{DisplayName: ptr("   ")}, wantErr: true},
		{name: "display name too long", in: UpdateMeInput{DisplayName: ptr(strings.Repeat("a", 51))}, wantErr: true},
		{name: "display name 50 emoji is fine", in: UpdateMeInput{DisplayName: ptr(strings.Repeat("✍", 50))}, wantName: strings.Repeat("✍", 50)},
		{name: "display name with newline", in: UpdateMeInput{DisplayName: ptr("Ri\nshi")}, wantErr: true},
		{name: "handle lowercased", in: UpdateMeInput{Handle: ptr(" Rishi_01 ")}, wantHandle: "rishi_01"},
		{name: "handle too short", in: UpdateMeInput{Handle: ptr("ab")}, wantErr: true},
		{name: "handle too long", in: UpdateMeInput{Handle: ptr(strings.Repeat("a", 31))}, wantErr: true},
		{name: "handle with space", in: UpdateMeInput{Handle: ptr("ri shi")}, wantErr: true},
		{name: "handle starting with dash", in: UpdateMeInput{Handle: ptr("-rishi")}, wantErr: true},
		{name: "handle with slash", in: UpdateMeInput{Handle: ptr("../admin")}, wantErr: true},
		{name: "reserved handle", in: UpdateMeInput{Handle: ptr("Admin")}, wantErr: true},
		{name: "ai opt-in only", in: UpdateMeInput{AIOptIn: ptr(true)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := ValidateUpdate(tc.in)
			if tc.wantErr {
				var verr *ValidationError
				if !errors.As(err, &verr) {
					t.Fatalf("error = %v, want a *ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if tc.wantName != "" && (out.DisplayName == nil || *out.DisplayName != tc.wantName) {
				t.Errorf("DisplayName = %v, want %q", out.DisplayName, tc.wantName)
			}
			if tc.wantHandle != "" && (out.Handle == nil || *out.Handle != tc.wantHandle) {
				t.Errorf("Handle = %v, want %q", out.Handle, tc.wantHandle)
			}
		})
	}
}

func TestDefaultDisplayName(t *testing.T) {
	tests := map[string]string{
		"rishe@example.com":                "rishe",
		"@example.com":                     "Writer",
		strings.Repeat("a", 60) + "@x.com": strings.Repeat("a", 50),
	}
	for email, want := range tests {
		if got := defaultDisplayName(email); got != want {
			t.Errorf("defaultDisplayName(%q) = %q, want %q", email, got, want)
		}
	}
}
