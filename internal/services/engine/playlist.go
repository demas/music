package engine

import (
	"time"

	"github.com/demas/music/internal/services/datastore/repository"

	"github.com/demas/music/internal/services/musicservices"

	"github.com/demas/music/internal/models/core"
	"go.uber.org/zap"
)

type PlaylistDownloader struct {
	Engine         *Engine
	DataRepository *repository.Repository
	Logger         *zap.SugaredLogger

	CurrentPlaylist    *core.Playlist
	MusicService       musicservices.IMusicService
	MusicRepository    musicservices.IMasterDataRepository
	TotalSingles       uint
	TotalAlbums        uint
	PlaylistWasUpdated bool
}

// Download playlist from music service
func (e *Engine) PlaylistDownloader() *PlaylistDownloader {
	return &PlaylistDownloader{
		Engine:          e,
		DataRepository:  e.DataRepository,
		Logger:          zap.NewExample().Sugar(),
		MusicRepository: musicservices.NewMusicRepository(),
		TotalSingles:    0,
		TotalAlbums:     0}
}

func (d *PlaylistDownloader) createRelease(album *core.Album, track *core.Track) {
	_, err := d.DataRepository.ReleaseRepository.Store(&core.Release{
		AlbumId:    album.Id,
		PlaylistId: d.CurrentPlaylist.Id,
		SyncDate:   time.Now(),
	})

	if err != nil {
		d.Logger.With(zap.Error(err)).Errorw("не удалось сохранить релиз",
			"Track.ServiceId", track.TrackId,
			"Track.PlaylistId", track.PlaylistId,
			"Track.AlbumId", track.ServiceAlbumId)
		return
	}

	if album.AlbumType == "album" {
		d.TotalAlbums += 1
	} else if album.AlbumType == "single" {
		d.TotalSingles += 1
	}
}

func (d *PlaylistDownloader) processTrack(track *core.Track) {

	err := track.Validate()
	if err != nil {
		d.Logger.With(zap.Error(err)).Errorw("incorrect track")
		return
	}

	if !track.MasterData {
		masterData := &MasterData{
			Engine:          d.Engine,
			Logger:          d.Logger,
			MusicRepository: d.MusicRepository,
		}

		err = masterData.findArtistAndAlbum(track)
		if err != nil {
			return
		}
	}

	_, err = d.DataRepository.TrackRepository.GetByPlaylistAndTrackId(track.PlaylistId, track.TrackId)
	if err != nil {

		// artist
		artist, err := d.Engine.returnOrCreateArtist(d.MusicRepository, track.ServiceArtistId)
		if err != nil {
			switch e := err.(type) {
			case *DownloadError:
				d.Logger.With(zap.Error(e)).Errorw("не удалось получить исполнителя на музыкальном сервисе",
					"Track.ServiceId", track.TrackId,
					"Track.PlaylistId", track.PlaylistId,
					"Track.ArtistId", track.ServiceArtistId)
			case *StoreError:
				d.Logger.With(zap.Error(err)).Errorw("не удалось сохранить исполнителя",
					"Track.ServiceId", track.TrackId,
					"Track.PlaylistId", track.PlaylistId,
					"Track.ArtistId", track.ServiceArtistId)
			}
			return
		}

		// album
		album, newAlbum, err := d.Engine.returnOrCreateAlbum(d.MusicRepository, track.ServiceAlbumId, artist.Id)
		if err != nil {
			switch e := err.(type) {
			case *DownloadError:
				d.Logger.With(zap.Error(e)).Errorw("не удалось получить альбом на музыкальном сервисе",
					"Track.ServiceId", track.TrackId,
					"Track.PlaylistId", track.PlaylistId,
					"Track.AlbumId", track.ServiceAlbumId)
			case *StoreError:
				d.Logger.With(zap.Error(err)).Errorw("не удалось сохранить альбом",
					"Track.ServiceId", track.TrackId,
					"Track.PlaylistId", track.PlaylistId,
					"Track.AlbumId", track.ServiceAlbumId)
			}
			return
		}
		album.ArtistId = artist.Id

		if newAlbum && isItNewRelease(album.ReleaseDate) {
			d.createRelease(album, track)
		}

		track.ArtistId = artist.Id
		track.AlbumId = album.Id

		_, err = d.DataRepository.TrackRepository.Store(track)
		if err != nil {
			d.Logger.With(zap.Error(err)).Errorw("не удалось сохранить трек")
		}

		d.PlaylistWasUpdated = true

		return
	}

	return
}

func (d *PlaylistDownloader) Download(playlistId uint) *DownloadResult {

	var err error
	defer func() {
		_ = d.Logger.Sync()
	}()

	d.CurrentPlaylist, err = d.DataRepository.PlaylistRepository.GetById(playlistId)
	if err != nil {
		d.Logger.Errorw("Cannot find playlist",
			"PlaylistId", playlistId)
		return DownloadTrackError()
	}

	d.MusicService = musicservices.NewMusicService(d.CurrentPlaylist.Service)
	servicePlaylist, tracks, err := d.MusicService.DownloadPlaylist(d.CurrentPlaylist.PlaylistId)
	if err != nil {
		d.Logger.With(zap.Error(err)).Error(err)
	}

	// TODO: refactore - PlaylistWasUpdated изменяется как то слишком неявно
	d.PlaylistWasUpdated = false
	for _, track := range tracks {
		track.PlaylistId = playlistId
		d.processTrack(track)
	}

	d.CurrentPlaylist.Name = servicePlaylist.Name
	d.CurrentPlaylist.Description = servicePlaylist.Description
	if d.PlaylistWasUpdated {
		t := time.Now()
		d.CurrentPlaylist.LastChanged = &t
	}

	_, err = d.DataRepository.PlaylistRepository.Update(playlistId, d.CurrentPlaylist)
	if err != nil {
		d.Logger.With(zap.Error(err)).Error("не удалось обновить плейлист")
	}

	return &DownloadResult{Downloaded: true, Album: d.TotalAlbums, Single: d.TotalSingles}
}
