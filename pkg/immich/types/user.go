package types

// User represents an Immich user
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// AlbumUser represents a user shared with an album
type AlbumUser struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}