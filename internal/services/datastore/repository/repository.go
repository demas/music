package repository

import (
	"github.com/demas/music/internal/services/datastore"
	"github.com/jinzhu/gorm"
)

type Repository struct {
	AlbumRepository    datastore.IAlbumRepository
	ArtistRepository   datastore.IArtistRepository
	PlaylistRepository datastore.IPlaylistRepository
	ReleaseRepository  datastore.IReleaseRepository
	TrackRepository    datastore.ITrackRepository
}

func NewRepository(dbHandler *gorm.DB) *Repository {
	return &Repository{
		AlbumRepository:    datastore.NewAlbumRepository(dbHandler),
		ArtistRepository:   datastore.NewArtistRepository(dbHandler),
		PlaylistRepository: datastore.NewPlaylistRepository(dbHandler),
		ReleaseRepository:  datastore.NewReleaseRepository(dbHandler),
		TrackRepository:    datastore.NewTrackRepository(dbHandler),
	}
}
