package db

import "github.com/jinzhu/gorm"

type Track struct {
	gorm.Model
	TrackId    string `gorm:"unique_index:idx_playlist_track_id"`
	PlaylistId uint   `gorm:"TYPE:integer REFERENCES playlists;unique_index:idx_playlist_track_id"`
	Name       string

	// Artist           string
	// Album            string
}
