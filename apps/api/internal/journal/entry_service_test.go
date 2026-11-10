package journal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

var entryTestNow = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

// helloDoc is a small Tiptap document: two paragraphs, one word in bold.
const helloDoc = `{"type":"doc","content":[` +
	`{"type":"paragraph","content":[{"type":"text","text":"Hello "},{"type":"text","marks":[{"type":"bold"}],"text":"world"}]},` +
	`{"type":"paragraph","content":[{"type":"text","text":"Second line."}]}]}`

// calls records the order the fakes were used in, e.g. "s3.put,mongo.create".
type calls []string

func (c *calls) add(name string) { *c = append(*c, name) }
func (c calls) String() string   { return strings.Join(c, ",") }

type fakeEntryRepo struct {
	log     *calls
	found   Entry
	findErr error
	created Entry
	updated EntryUpdate
	listed  ListQuery
}

func (f *fakeEntryRepo) CreateEntry(_ context.Context, e Entry) (Entry, error) {
	f.log.add("mongo.create")
	f.created = e
	return e, nil
}

func (f *fakeEntryRepo) FindEntry(_ context.Context, _, _ string) (Entry, error) {
	f.log.add("mongo.find")
	return f.found, f.findErr
}

func (f *fakeEntryRepo) ListEntries(_ context.Context, _ string, q ListQuery) (EntryPage, error) {
	f.log.add("mongo.list")
	f.listed = q
	return EntryPage{}, nil
}

func (f *fakeEntryRepo) UpdateEntry(_ context.Context, _, _ string, upd EntryUpdate, _ time.Time) (Entry, error) {
	f.log.add("mongo.update")
	f.updated = upd
	return f.found, nil
}

func (f *fakeEntryRepo) DeleteEntry(context.Context, string, string, time.Time) error {
	f.log.add("mongo.delete")
	return nil
}

type fakeJournalFinder struct{ err error }

func (f fakeJournalFinder) FindJournal(_ context.Context, ownerID string) (Journal, error) {
	if f.err != nil {
		return Journal{}, f.err
	}
	return Journal{ID: "journal-1", OwnerID: ownerID}, nil
}

type fakeBodies struct {
	log     *calls
	putKey  string
	putBody string
	putErr  error
	stored  string
}

func (f *fakeBodies) PutBody(_ context.Context, key string, body []byte) error {
	f.log.add("s3.put")
	f.putKey, f.putBody = key, string(body)
	return f.putErr
}

func (f *fakeBodies) GetBody(_ context.Context, _ string) ([]byte, error) {
	f.log.add("s3.get")
	return []byte(f.stored), nil
}

// newFakes returns a call log shared by a fake repository and a fake body store.
func newFakes() (*calls, *fakeEntryRepo, *fakeBodies) {
	log := &calls{}
	return log, &fakeEntryRepo{log: log}, &fakeBodies{log: log}
}

// newTestEntryService wires the fakes together with a fixed clock and entry ID.
func newTestEntryService(repo *fakeEntryRepo, journals fakeJournalFinder, bodies *fakeBodies) *EntryService {
	s := NewEntryService(repo, journals, bodies, func() string { return "entry-1" })
	s.now = func() time.Time { return entryTestNow }
	return s
}

func validCreateInput() CreateEntryInput {
	return CreateEntryInput{EntryDate: "2026-11-10", Body: json.RawMessage(helloDoc)}
}

func TestCreateEntryStoresBodyThenMetadata(t *testing.T) {
	log, repo, bodies := newFakes()
	svc := newTestEntryService(repo, fakeJournalFinder{}, bodies)

	got, err := svc.CreateEntry(context.Background(), "user-1", CreateEntryInput{
		Title:     "  First day  ",
		EntryDate: "2026-11-10",
		Mood:      "Excited",
		Tags:      []string{"Work", "work", " milestone "},
		Body:      json.RawMessage(helloDoc),
	})
	if err != nil {
		t.Fatalf("CreateEntry() error = %v", err)
	}

	if log.String() != "s3.put,mongo.create" {
		t.Errorf("calls = %s, want s3.put,mongo.create (body first)", log)
	}
	wantKey := "users/user-1/entries/entry-1.json"
	if bodies.putKey != wantKey {
		t.Errorf("body key = %q, want %q", bodies.putKey, wantKey)
	}

	e := repo.created
	for _, c := range []struct{ name, got, want string }{
		{"ID", e.ID, "entry-1"},
		{"JournalID", e.JournalID, "journal-1"},
		{"OwnerID", e.OwnerID, "user-1"},
		{"Title", e.Title, "First day"},
		{"Mood", e.Mood, "excited"},
		{"Tags", strings.Join(e.Tags, ","), "work,milestone"},
		{"BodyKey", e.BodyKey, wantKey},
		{"Excerpt", e.Excerpt, "Hello world Second line."},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if e.WordCount != 4 {
		t.Errorf("WordCount = %d, want 4", e.WordCount)
	}
	if !e.CreatedAt.Equal(entryTestNow) || !e.UpdatedAt.Equal(entryTestNow) {
		t.Errorf("CreatedAt/UpdatedAt = %v/%v, want %v", e.CreatedAt, e.UpdatedAt, entryTestNow)
	}
	if string(got.Body) != helloDoc {
		t.Errorf("returned body = %s, want the body that was sent", got.Body)
	}
}

func TestCreateEntryStopsWhenBodyFails(t *testing.T) {
	log, repo, bodies := newFakes()
	bodies.putErr = errors.New("s3 unavailable")

	_, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).CreateEntry(context.Background(), "user-1", validCreateInput())
	if err == nil {
		t.Fatal("CreateEntry() error = nil, want the S3 failure")
	}
	if log.String() != "s3.put" {
		t.Errorf("calls = %s, want only s3.put: no metadata without a body", log)
	}
}

func TestCreateEntryNeedsAJournal(t *testing.T) {
	log, repo, bodies := newFakes()

	_, err := newTestEntryService(repo, fakeJournalFinder{err: ErrNotFound}, bodies).CreateEntry(context.Background(), "user-1", validCreateInput())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
	if len(*log) != 0 {
		t.Errorf("calls = %s, want none", log)
	}
}

func TestCreateEntryAcceptsEdgeDates(t *testing.T) {
	for _, date := range []string{"1900-01-01", "2026-11-11"} { // earliest allowed, and "tomorrow"
		_, repo, bodies := newFakes()
		in := validCreateInput()
		in.EntryDate = date
		if _, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).CreateEntry(context.Background(), "user-1", in); err != nil {
			t.Errorf("CreateEntry(entryDate %s) error = %v, want nil", date, err)
		}
	}
}

func TestCreateEntryValidation(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		mutate func(*CreateEntryInput)
	}{
		{"missing date", "entryDate", func(in *CreateEntryInput) { in.EntryDate = "" }},
		{"wrong date format", "entryDate", func(in *CreateEntryInput) { in.EntryDate = "10/11/2026" }},
		{"impossible date", "entryDate", func(in *CreateEntryInput) { in.EntryDate = "2026-02-30" }},
		{"two days ahead", "entryDate", func(in *CreateEntryInput) { in.EntryDate = "2026-11-12" }},
		{"title too long", "title", func(in *CreateEntryInput) { in.Title = strings.Repeat("a", 201) }},
		{"control character", "title", func(in *CreateEntryInput) { in.Title = "a\x00b" }},
		{"unknown mood", "mood", func(in *CreateEntryInput) { in.Mood = "hangry" }},
		{"tag with a space", "tags", func(in *CreateEntryInput) { in.Tags = []string{"two words"} }},
		{"too many tags", "tags", func(in *CreateEntryInput) { in.Tags = manyTags(21) }},
		{"missing body", "body", func(in *CreateEntryInput) { in.Body = nil }},
		{"body is null", "body", func(in *CreateEntryInput) { in.Body = json.RawMessage(`null`) }},
		{"body is not a doc", "body", func(in *CreateEntryInput) { in.Body = json.RawMessage(`{"type":"paragraph"}`) }},
		{"body is a string", "body", func(in *CreateEntryInput) { in.Body = json.RawMessage(`"hello"`) }},
		{"body too deep", "body", func(in *CreateEntryInput) { in.Body = deepDoc(60) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, repo, bodies := newFakes()
			in := validCreateInput()
			tt.mutate(&in)

			_, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).CreateEntry(context.Background(), "user-1", in)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Field != tt.field {
				t.Fatalf("error = %v, want a ValidationError on %q", err, tt.field)
			}
			if len(*log) != 0 {
				t.Errorf("calls = %s, want none: invalid input must not reach S3 or MongoDB", log)
			}
		})
	}
}

func TestEntryReturnsBody(t *testing.T) {
	log, repo, bodies := newFakes()
	repo.found = Entry{ID: "entry-1", OwnerID: "user-1", BodyKey: BodyKey("user-1", "entry-1")}
	bodies.stored = helloDoc

	got, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).Entry(context.Background(), "user-1", "entry-1")
	if err != nil {
		t.Fatalf("Entry() error = %v", err)
	}
	if string(got.Body) != helloDoc {
		t.Errorf("Body = %s, want helloDoc", got.Body)
	}
	if log.String() != "mongo.find,s3.get" {
		t.Errorf("calls = %s, want mongo.find,s3.get", log)
	}
}

func TestEntryRefusesBodyOutsideOwnersPrefix(t *testing.T) {
	log, repo, bodies := newFakes()
	repo.found = Entry{ID: "entry-1", OwnerID: "user-1", BodyKey: "users/user-2/entries/entry-1.json"}

	if _, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).Entry(context.Background(), "user-1", "entry-1"); err == nil {
		t.Fatal("Entry() error = nil, want a refusal")
	}
	if log.String() != "mongo.find" {
		t.Errorf("calls = %s, want mongo.find only: S3 must not be read", log)
	}
}

func TestUpdateEntryChecksOwnershipFirst(t *testing.T) {
	log, repo, bodies := newFakes()
	repo.findErr = ErrNotFound // another user's entry looks exactly like a missing one

	_, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).UpdateEntry(context.Background(), "user-1", "someone-elses-entry", UpdateEntryInput{Body: json.RawMessage(helloDoc)})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
	if log.String() != "mongo.find" {
		t.Errorf("calls = %s, want mongo.find only: S3 must not be written", log)
	}
}

func TestUpdateEntryBodyRefreshesExcerpt(t *testing.T) {
	log, repo, bodies := newFakes()
	repo.found = Entry{ID: "entry-1", OwnerID: "user-1", BodyKey: BodyKey("user-1", "entry-1")}
	newBody := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Just three words"}]}]}`

	got, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).UpdateEntry(context.Background(), "user-1", "entry-1", UpdateEntryInput{Body: json.RawMessage(newBody)})
	if err != nil {
		t.Fatalf("UpdateEntry() error = %v", err)
	}
	if log.String() != "mongo.find,s3.put,mongo.update" {
		t.Errorf("calls = %s, want mongo.find,s3.put,mongo.update", log)
	}
	if bodies.putKey != BodyKey("user-1", "entry-1") {
		t.Errorf("body key = %q, want the entry's existing key", bodies.putKey)
	}
	if repo.updated.Excerpt == nil || *repo.updated.Excerpt != "Just three words" {
		t.Errorf("Excerpt update = %v, want \"Just three words\"", repo.updated.Excerpt)
	}
	if repo.updated.WordCount == nil || *repo.updated.WordCount != 3 {
		t.Errorf("WordCount update = %v, want 3", repo.updated.WordCount)
	}
	if string(got.Body) != newBody {
		t.Errorf("Body = %s, want the new body", got.Body)
	}
}

func TestUpdateEntryMetadataOnlyKeepsBody(t *testing.T) {
	log, repo, bodies := newFakes()
	repo.found = Entry{ID: "entry-1", OwnerID: "user-1", BodyKey: BodyKey("user-1", "entry-1")}
	bodies.stored = helloDoc
	title := "New title"

	got, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).UpdateEntry(context.Background(), "user-1", "entry-1", UpdateEntryInput{Title: &title})
	if err != nil {
		t.Fatalf("UpdateEntry() error = %v", err)
	}
	if log.String() != "mongo.find,mongo.update,s3.get" {
		t.Errorf("calls = %s, want mongo.find,mongo.update,s3.get (no S3 write)", log)
	}
	if repo.updated.Excerpt != nil {
		t.Error("Excerpt changed without a new body")
	}
	if string(got.Body) != helloDoc {
		t.Errorf("Body = %s, want the stored body", got.Body)
	}
}

func TestUpdateEntryNeedsAField(t *testing.T) {
	log, repo, bodies := newFakes()

	_, err := newTestEntryService(repo, fakeJournalFinder{}, bodies).UpdateEntry(context.Background(), "user-1", "entry-1", UpdateEntryInput{})
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("error = %v, want a ValidationError", err)
	}
	if len(*log) != 0 {
		t.Errorf("calls = %s, want none", log)
	}
}

func TestListEntriesCleansQuery(t *testing.T) {
	_, repo, bodies := newFakes()
	svc := newTestEntryService(repo, fakeJournalFinder{}, bodies)

	if _, err := svc.ListEntries(context.Background(), "user-1", ListInput{Tag: " Work ", Limit: 500}); err != nil {
		t.Fatalf("ListEntries() error = %v", err)
	}
	if repo.listed.Tag != "work" || repo.listed.Limit != 100 || repo.listed.Cursor != nil {
		t.Errorf("query = %+v, want tag work, limit 100, no cursor", repo.listed)
	}

	for _, in := range []ListInput{{Tag: "not valid!"}, {Cursor: "garbage"}} {
		var verr *ValidationError
		if _, err := svc.ListEntries(context.Background(), "user-1", in); !errors.As(err, &verr) {
			t.Errorf("ListEntries(%+v) error = %v, want a ValidationError", in, err)
		}
	}
}

func TestSummarise(t *testing.T) {
	excerpt, words := Summarise("  one\n\ntwo   three ")
	if excerpt != "one two three" || words != 3 {
		t.Errorf("Summarise() = %q, %d; want \"one two three\", 3", excerpt, words)
	}

	excerpt, words = Summarise(strings.Repeat("word ", 100))
	if words != 100 {
		t.Errorf("words = %d, want 100", words)
	}
	if !strings.HasSuffix(excerpt, "…") || utf8.RuneCountInString(excerpt) > excerptRunes+1 {
		t.Errorf("excerpt = %q (%d runes), want at most %d runes ending in …", excerpt, utf8.RuneCountInString(excerpt), excerptRunes+1)
	}
}

func manyTags(n int) []string {
	tags := make([]string, n)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag%d", i)
	}
	return tags
}

// deepDoc nests depth blockquotes inside a doc.
func deepDoc(depth int) json.RawMessage {
	return json.RawMessage(`{"type":"doc","content":[` +
		strings.Repeat(`{"type":"blockquote","content":[`, depth) +
		strings.Repeat(`]}`, depth) + `]}`)
}
