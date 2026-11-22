package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/miantiao-me/MMMMM/internal/local"
	"github.com/miantiao-me/MMMMM/internal/music"
)

func (a *app) mobileHome(c *ui.Context) {
	lib := a.library
	if lib.home != nil && len(lib.home.Sections) > 0 {
		a.mobileBrowse(c, lib.home)
		return
	}
	if lib.homeLoading || !lib.started || lib.accountUnknown() {
		a.mobileLoading(c, "", nil)
		return
	}
	var page music.Browse
	if len(lib.recent) > 0 {
		page.Sections = append(page.Sections, music.Section{Title: T("Recently Played"), Tracks: lib.recent[:min(8, len(lib.recent))]})
	}
	if len(lib.mine)+len(lib.saved) > 0 {
		lists := append(append([]music.Playlist(nil), lib.mine...), lib.saved...)
		page.Sections = append(page.Sections, music.Section{Title: T("Playlists"), Playlists: lists})
	}
	switch {
	case len(page.Sections) > 0:
		a.mobileBrowse(c, &page)
	case lib.homeErr != "":
		a.mobileLoading(c, lib.homeErr, func() { a.loadHome(lib) })
	case lib.id == local.ID:
		a.mobileEmpty(c, iconMusicSource, T("No Music"), T("Import Music…"), a.importMobileMusic)
	case lib.signedOut():
		a.mobileEmpty(c, iconUser, Tf("Sign in to %s", lib.name()), T("Sign In"), func() { a.promptSignIn(lib) })
	case lib.can("search"):
		a.mobileEmpty(c, iconNote, T("Nothing here yet"), T("Search"), func() { a.selectMobileTab(mobileSearch) })
	default:
		a.mobileEmpty(c, iconNote, T("Nothing here yet"), "", nil)
	}
}

func (a *app) mobileBrowse(c *ui.Context, page *music.Browse) {
	r := a.route
	rows := browseLayout(page, false, 1)
	header := page.Description != "" || page.Subtitle != "" || page.ImageURL != ""
	count := len(rows)
	if header {
		count++
	}
	if count == 0 {
		a.mobileEmpty(c, iconNote, T("Nothing here yet"), "", nil)
		return
	}
	a.mobileList(c, r, count, func(i int) {
		if header {
			if i == 0 {
				ui.Column(c).Padding(6, 12, 12, 12).Gap(6).Children(func() {
					if page.ImageURL != "" {
						a.cover(c, page.ImageURL, 160, 72, 12)
					}
					if page.Subtitle != "" {
						ui.Text(c, page.Subtitle).FontSize(15).TextColor(a.pal.muted)
					}
					if page.Description != "" {
						ui.Text(c, page.Description).FontSize(15)
					}
				})
				return
			}
			i--
		}
		entry := rows[i]
		section := &page.Sections[entry.section]
		at := entry.from
		switch entry.kind {
		case rowTitle:
			ui.Row(c).Padding(14, 12, 3, 12).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, section.Title).FontSize(22).FontWeight(700).Grow(1).MinWidth(0)
				if section.MorePage != "" && pillButton(c.Key("more"), nil, T("See All"), a.pal.sidebar, a.pal.accent).Clicked() {
					switch section.MorePage {
					case a.mobileCatalogPage(a.library, "album"):
						a.selectMobileTab(mobileAlbums)
					case a.mobileCatalogPage(a.library, "note"):
						a.selectMobileTab(mobileSongs)
					default:
						a.open(route{page: pageBrowse, id: section.MorePage})
					}
				}
			})
		case rowEmpty:
			ui.Text(c, section.Empty).FontSize(15).TextColor(a.pal.muted).Padding(8, 12, 12, 12)
		case rowTrack:
			a.mobileSongRow(c, section.Tracks, at, "", true)
		case rowAlbums:
			a.mobileAlbumRow(c, &section.Albums[at])
		case rowPlaylists:
			a.mobilePlaylistRow(c, &section.Playlists[at])
		case rowArtists:
			item := section.Artists[at]
			a.mobileMediaRow(c, item.Name, "", item.ImageURL, true, func() { a.open(route{page: pageArtist, id: item.ID}) })
		case rowShows:
			item := section.Shows[at]
			a.mobileMediaRow(c, item.Name, item.Publisher, item.ImageURL, true, func() { a.open(route{page: pageShow, id: item.ID}) })
		case rowStations:
			item := &section.Stations[at]
			a.mobileMediaRow(c, item.Name, item.Description, item.ImageURL, false, func() { a.playStation(a.library, item) })
		case rowLinks:
			item := section.Links[at]
			a.mobileMediaRow(c, item.Title, item.Subtitle, item.ImageURL, true, func() { a.open(route{page: pageBrowse, id: item.Page}) })
		}
	})
}
