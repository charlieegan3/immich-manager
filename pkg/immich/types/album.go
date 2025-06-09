package types

// Album represents an Immich album
type Album struct {
	ID          string      `json:"id"`
	Name        string      `json:"albumName"`
	SharedUsers []AlbumUser `json:"sharedUsers"`
}
