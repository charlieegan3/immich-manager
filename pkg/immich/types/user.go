// Package types provides Immich API data types.
package types

// User represents an Immich user.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// AlbumUser represents a user shared with an album.
type AlbumUser struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Role string `json:"role"`
}
