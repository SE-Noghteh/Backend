package valueobjects

// StoredObject identifies a private object without exposing a public URL.
type StoredObject struct {
	Key         string
	ContentType string
	Size        int64
}
