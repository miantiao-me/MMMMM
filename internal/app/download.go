package app

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo"

	"github.com/miantiao-me/MMMMM/internal/local"
	"github.com/miantiao-me/MMMMM/internal/music"
	"github.com/miantiao-me/MMMMM/internal/player"
)

// saveSongAs asks the listener where to save a song with the native dialog,
// suggesting name in their Downloads folder, and returns the path, or "" if
// they didn't pick one. It blocks, so it runs on a goroutine; tests replace it.
var saveSongAs = func(a *app, name string) string {
	dir, _ := mygo.App.Path(mygo.PathDownloads)
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:            a.win,
		Title:             T("Download"),
		DefaultPath:       filepath.Join(dir, name),
		ButtonLabel:       T("Download"),
		CreateDirectories: true,
	})
	if err != nil {
		return ""
	}
	return path
}

// canDownload reports whether tr can be saved to a file of the listener's: a
// song its source streams for the app to play, rather than one already on disk
// or one that plays elsewhere.
func (a *app) canDownload(tr *music.Track) bool {
	lib := a.trackLib(tr)
	return lib != nil && lib.id != local.ID && lib.can("stream") &&
		lib.src.Info().Playback == music.PlaybackStream && !tr.Unavailable
}

// download saves a copy of tr where the listener picks, at the quality the
// settings say, from the cache if it's kept there; toasts say how it goes.
func (a *app) download(tr music.Track) {
	lib := a.trackLib(&tr)
	quality := a.settings.Quality
	go func() {
		song, err := a.pl.FindSong(lib.src, tr, quality)
		if err != nil {
			a.post(func() { a.actionFailed(lib, err, T("Couldn't download this song")) })
			return
		}
		name := songFileName(&tr, song.Type)
		if a.phone != nil {
			a.exportMobileSong(lib, tr, song, name)
			return
		}
		path := saveSongAs(a, name)
		if path == "" {
			song.Close()
			return
		}
		a.post(func() { a.toast = T("Downloading…") })
		err = song.SaveTo(path)
		a.post(func() {
			if err != nil {
				a.actionFailed(lib, err, T("Couldn't download this song"))
				return
			}
			a.toast = Tf("Downloaded %s", tr.Name)
		})
	}()
}

// exportMobileSong hands a complete file to the system's document picker.
func (a *app) exportMobileSong(lib *library, tr music.Track, song *player.Song, name string) {
	defer song.Close()
	dir, err := os.MkdirTemp("", "mmmmm-export-")
	var destination string
	if err == nil {
		defer os.RemoveAll(dir)
		a.post(func() { a.toast = T("Downloading…") })
		path := filepath.Join(dir, name)
		if err = song.SaveTo(path); err == nil {
			destination, err = mygo.Dialog.SaveCopy(path, name)
		}
	}
	a.post(func() {
		if err != nil {
			a.actionFailed(lib, err, T("Couldn't download this song"))
		} else if destination != "" {
			a.toast = Tf("Downloaded %s", tr.Name)
		}
	})
}

// songFileName names the file of a song of type ext as its credits read,
// "Artists - Title.ext", with what no file name may hold taken out.
func songFileName(tr *music.Track, ext string) string {
	name := tr.Name
	if artists := tr.ArtistNames(); artists != "" {
		name = artists + " - " + name
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, name)
	// Most systems take 255 bytes, with room left for the extension.
	for len(name) > 200 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	// Windows drops a trailing dot or space, and a leading dot hides the file.
	name = strings.Trim(name, ". ")
	if name == "" {
		name = T("Song")
	}
	return name + "." + ext
}
