// Package types provides Immich API data types.
package types

// Album represents an Immich album.
type Album struct {
	ID         string      `json:"id"`
	Name       string      `json:"albumName"`
	AlbumUsers []AlbumUser `json:"albumUsers"`
}
