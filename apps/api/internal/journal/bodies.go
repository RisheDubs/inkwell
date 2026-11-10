package journal

import "context"

// MaxBodyBytes caps one entry's Tiptap JSON body: comfortably more than 10,000 words of formatted text.
const MaxBodyBytes = 256 << 10

// BodyStore keeps entry bodies outside MongoDB. Keys always come from BodyKey, never from a request.
type BodyStore interface {
	PutBody(ctx context.Context, key string, body []byte) error
	GetBody(ctx context.Context, key string) ([]byte, error)
}

// BodyKey is where an entry's body lives: users/{ownerID}/entries/{entryID}.json.
// It uses the Cognito sub, never a name or email, because names change and are personal data.
func BodyKey(ownerID, entryID string) string {
	return "users/" + ownerID + "/entries/" + entryID + ".json"
}
