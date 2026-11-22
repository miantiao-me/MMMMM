package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"github.com/miantiao-me/MMMMM/internal/i18n"
	"github.com/miantiao-me/MMMMM/internal/local"
	"github.com/miantiao-me/MMMMM/internal/music"
)

var iconMobileLibraryActions = strokeIcon(`<circle cx="12" cy="12" r="10"/><path d="M6.5 12h.01M12 12h.01M17.5 12h.01"/>`)

func (a *app) mobileAlbumListRoute() (route, bool) {
	if a.route.page == pageAlbum && len(a.history) > 0 {
		r := a.history[len(a.history)-1]
		if r.src == a.route.src && (r.page == mobileAlbums || r.page == mobileLocalAlbumsSearch || r.page == pageSearch && r.kind == music.SearchAlbums) {
			return r, true
		}
	}
	return route{}, false
}

func (a *app) mobileAlbumList(c *ui.Context, r route, width float32) {
	p := a.pal
	list := ui.Column(c.Key(r)).MinWidth(0).FillHeight().Clip().Border(1, p.hairline)
	if width > 0 {
		list.Width(width).Shrink(0)
	} else {
		list.Grow(1)
	}
	list.Children(func() {
		ui.Text(c, T("Albums")).FontSize(21).FontWeight(700).Margin(16, 16, 10, 16).SingleLine()
		switch r.page {
		case pageSearch:
			a.mobileResults(c, r)
		case mobileAlbums, mobileLocalAlbumsSearch:
			var query string
			if filter := a.phone.filters[r]; filter != nil {
				query = strings.ToLower(strings.TrimSpace(*filter))
			}
			a.mobileCatalog(c, r, query)
		}
	})
}

func (a *app) mobileScreen(c *ui.Context, showSidebarButton bool) {
	m := a.phone
	r := a.route
	ui.Column(c.Key(r)).Grow(1).MinHeight(0).Children(func() {
		a.mobileNavigation(c, r, showSidebarButton)
		switch r.page {
		case pageHome:
			a.mobileHome(c)
		case mobileAlbums, mobileSongs, mobilePlaylists, mobileSearch, pageSearch, mobileLocalAlbumsSearch, mobileLocalSongsSearch:
			a.mobileLibrary(c, r)
		case mobileSettings, mobileSources, mobileSource:
			a.mobileSettingsScreen(c, r)
		case pageAlbum, pagePlaylist, pageLiked:
			a.mobileDetail(c, r)
		case pageBrowse:
			d := a.browses[r.id]
			if d == nil || d.page == nil {
				var message string
				if d != nil {
					message = d.err
				}
				a.mobileLoading(c, message, func() { a.loadBrowse(a.library, r.id, true) })
			} else {
				a.mobileBrowse(c, d.page)
			}
		default:
			a.content(c)
		}
	})
	if len(a.history) > 0 && !m.wide {
		edge := ui.Box(c.Key("back-edge")).Absolute().Top(0).Bottom(0).Left(0).Width(12)
		edge.Draggable(&m.backDrag, ui.DraggableOptions{
			Axis: ui.Horizontal, Bounds: &ui.Rect{W: 100},
			Snap: func(p ui.Point) ui.Point {
				if p.X > 56 {
					a.back()
				}
				return ui.Point{}
			},
		})
	}
}

func (a *app) mobileNavigation(c *ui.Context, r route, showSidebarButton bool) {
	p := a.pal
	var title string
	for _, tab := range mobileTabs {
		if tab.page == r.page {
			title = T(tab.name)
			break
		}
	}
	switch r.page {
	case pageHome:
		title = a.builtinTitle("home")
	case mobileSources:
		title = T("Music Sources")
	case mobileSource:
		if lib := a.mobileSourceLibrary(r.id); lib != nil {
			title = lib.name()
		}
	case mobileSearch, pageSearch, mobileLocalAlbumsSearch, mobileLocalSongsSearch:
		title = T("Search")
	case pageBrowse:
		if d := a.browses[r.id]; d != nil && d.page != nil {
			title = d.page.Title
		}
	}
	height := float32(52)
	if a.phone.compactHeight {
		height = 44
	}
	ui.Row(c).Height(height).Shrink(0).PaddingX(16).Gap(10).AlignItems(ui.Center).Children(func() {
		if showSidebarButton {
			a.mobileSidebarButton(c.Key("show-sidebar"), T("Show Sidebar"), false)
		}
		if len(a.history) > 0 {
			mobileIcon(c.Key("back"), iconBack, 22, p.text, T("Back")).
				Material(glass.Glass{}).Background(p.bg.Alpha(0.86)).Border(1, p.hairline).OnClick(a.back)
		}
		if a.phone.compactHeight && title != "" {
			ui.Text(c, title).Grow(1).MinWidth(0).FontSize(21).FontWeight(700).SingleLine()
		} else {
			ui.Box(c).Grow(1)
		}
		switch r.page {
		case mobileAlbums, mobileSongs:
			ui.Row(c).Radius(24).Gap(6).Shrink(0).Material(glass.Glass{}).Background(p.bg.Alpha(0.86)).Border(1, p.hairline).Children(func() {
				if a.id == local.ID && mobileIcon(c.Key("import"), iconPlus, 21, p.text, T("Import Music…")).Clicked() {
					a.importMobileMusic()
				}
				mobileIcon(c.Key("library-actions"), iconMobileLibraryActions, 21, p.text, T("Library actions")).Menu(func(menu *ui.Menu) {
					if menu.Item(T("Music Sources")).Chosen() {
						a.openSourceSettings()
					}
					menu.Separator()
					for _, item := range a.mobileCollections() {
						if menu.Item(item.label).Chosen() {
							a.open(item.route)
						}
					}
				})
			})
			if a.phone.wide && a.id == local.ID {
				label, page := mobileLocalSearchTarget(r.page)
				a.mobileLocalSearchButton(c, label, page, true)
			}
		case mobilePlaylists:
			if a.can("createPlaylist") && mobileIcon(c.Key("create-playlist"), iconPlus, 23, p.accent, T("New Playlist")).Clicked() {
				a.newPlaylist(nil)
			}
		case pagePlaylist:
			if pl := a.playlists[r.id]; pl != nil {
				mine := a.editable(pl.ID)
				if mine != nil || a.savable(pl) {
					mobileIcon(c.Key("playlist-actions"), iconMobileLibraryActions, 21, p.accent, T("More")).Menu(func(menu *ui.Menu) {
						if mine != nil {
							a.playlistMenu(menu, mine)
						}
						if a.savable(pl) {
							label := TC("Save", "library")
							if a.isSaved(pl) {
								label = T("Saved")
							}
							if menu.Item(label).Chosen() {
								a.toggleSaved(*pl)
							}
						}
					})
				}
			}
		case mobileSources:
			b := mobileIcon(c.Key("add-source"), iconPlus, 23, p.accent, T("Install a Plugin"))
			b.Menu(func(menu *ui.Menu) {
				if menu.Item(T("From GitHub…")).Disabled(a.installer.busy).Chosen() {
					a.askForRepo()
				}
				if menu.Item(T("From a Zip…")).Disabled(a.installer.busy).Chosen() {
					a.installFromDisk(false)
				}
			})
		}
	})
	if title != "" && !a.phone.compactHeight {
		ui.Text(c, title).FontSize(34).FontWeight(700).Margin(0, 20, 10, 20).SingleLine()
	}
}

func (a *app) mobileLibrary(c *ui.Context, r route) {
	p, m, lib := a.pal, a.phone, a.library
	if lib.id == local.ID && (r.page == mobileAlbums || r.page == mobileSongs) {
		if !m.wide {
			label, page := mobileLocalSearchTarget(r.page)
			a.mobileLocalSearchButton(c, label, page, false)
		}
		a.mobileCatalog(c, r, "")
		return
	}
	filter := m.filters[r]
	if filter == nil {
		filter = new(string)
		*filter = r.query
		m.filters[r] = filter
	}
	search := r.page == mobileSearch || r.page == pageSearch
	label, placeholder := T("Search"), T("Search")
	if search {
		label = T("Search Music")
	}
	if lib.id == local.ID {
		switch r.page {
		case mobileLocalAlbumsSearch:
			placeholder = T("Search Local Library")
		case mobileLocalSongsSearch:
			placeholder = T("Search Local Songs")
		}
	}
	ui.Row(c).Height(44).Margin(0, 16, 12, 16).Radius(22).PaddingX(14).Gap(6).
		Background(p.sidebar).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, iconSearch).Size(18, 18).TextColor(p.muted)
		field := ui.TextInput(c.Key("query"), filter).Placeholder(placeholder).Label(label).
			Grow(1).MinWidth(0).Height(36).Border(0, ui.Transparent).Background(ui.Transparent).FocusRing(false)
		if r.page == mobileLocalAlbumsSearch || r.page == mobileLocalSongsSearch {
			field.Bind(&m.searchField)
		}
		if search && field.Submitted() && strings.TrimSpace(*filter) != "" {
			a.searchFor(route{page: pageSearch, src: a.id, kind: a.mobileSearchKind(), query: strings.TrimSpace(*filter)})
		}
		if *filter != "" && iconButton(c.Key("clear"), iconX, 13, p.muted, T("Clear")).Clicked() {
			*filter = ""
			if r.page == pageSearch {
				a.show(route{page: mobileSearch, src: a.id})
				a.history = nil
			}
		}
	})
	query := strings.ToLower(strings.TrimSpace(*filter))
	switch r.page {
	case mobileAlbums, mobileSongs, mobileLocalAlbumsSearch, mobileLocalSongsSearch:
		a.mobileCatalog(c, r, query)
	case mobilePlaylists:
		a.mobilePlaylistList(c, r, query)
	case mobileSearch, pageSearch:
		ui.Row(c).Padding(0, 16, 8, 16).Gap(6).Children(func() {
			for _, k := range searchKinds {
				if k.needs != "" && !lib.can(k.needs) {
					continue
				}
				bg := p.sidebar
				if k.kind == a.mobileSearchKind() {
					bg = p.accent.Alpha(0.16)
				}
				if pillButton(c.Key(k.kind), nil, T(k.label), bg, p.text).PaddingX(9).Clicked() {
					a.searched.kind = k.kind
					if q := strings.TrimSpace(*filter); q != "" {
						a.searchFor(route{page: pageSearch, src: a.id, kind: k.kind, query: q})
					}
				}
			}
		})
		if r.page == pageSearch {
			a.mobileResults(c, r)
		}
	}
}

func mobileLocalSearchTarget(page pageKind) (string, pageKind) {
	if page == mobileAlbums {
		return T("Search Local Library"), mobileLocalAlbumsSearch
	}
	return T("Search Local Songs"), mobileLocalSongsSearch
}

func (a *app) mobileLocalSearchButton(c *ui.Context, label string, page pageKind, toolbar bool) {
	p := a.pal
	b := ui.ButtonBase(c.Key("search")).Label(label).Radius(22).PaddingX(14).Gap(6).
		AlignItems(ui.Center).Justify(ui.Start).Background(p.sidebar).FocusRing(false)
	if toolbar {
		b.Height(44).Width(242).MinWidth(0).Shrink(1).Material(glass.Glass{}).
			Background(p.bg.Alpha(0.86)).Border(1, p.hairline)
	} else {
		b.Height(42).Margin(0, 20, 10, 20)
	}
	b.OnClick(func() {
		r := route{page: page, src: a.id}
		delete(a.phone.filters, r)
		delete(a.phone.scroll, r)
		a.open(r)
		a.phone.searchField.Focus(c)
	}).Children(func() {
		ui.Icon(c, iconSearch).Size(18, 18).TextColor(p.text)
		ui.Text(c, label).FontSize(16).TextColor(p.muted).SingleLine().MinWidth(0)
	})
}

func (a *app) mobileSearchKind() music.SearchKind {
	if a.searched.kind != "" {
		return a.searched.kind
	}
	return music.SearchTracks
}

func (a *app) mobileCatalogPage(lib *library, icon string) string {
	if lib.can("browse") {
		for _, nav := range lib.src.Info().Navigation {
			if nav.Icon == icon && nav.Page != "" {
				return nav.Page
			}
		}
	}
	return ""
}

func (a *app) mobileCatalog(c *ui.Context, r route, query string) {
	p, lib := a.pal, a.library
	albums := r.page == mobileAlbums || r.page == mobileLocalAlbumsSearch
	var sections []music.Section
	icon := "note"
	if albums {
		icon = "album"
	}
	if page := a.mobileCatalogPage(lib, icon); page != "" {
		d := lib.browses[page]
		if d == nil {
			a.loadBrowse(lib, page, false)
			d = lib.browses[page]
		}
		if d.page == nil {
			a.mobileLoading(c, d.err, func() { a.loadBrowse(lib, page, true) })
			return
		}
		sections = d.page.Sections
	}
	if sections == nil && lib.home != nil {
		sections = lib.home.Sections
	}
	if sections == nil && lib.homeLoading {
		a.mobileLoading(c, "", nil)
		return
	}
	if albums {
		var items []music.Album
		for _, sec := range sections {
			items = append(items, sec.Albums...)
		}
		if lib.can("savedAlbums") {
			if !lib.savedAlbumsPaged.loaded && !lib.savedAlbumsPaged.loading && lib.savedAlbumsPaged.err == "" {
				a.loadSavedAlbums(lib)
			}
			for _, album := range lib.savedAlbums {
				found := false
				for _, item := range items {
					found = found || item.ID == album.ID
				}
				if !found {
					items = append(items, album)
				}
			}
		}
		items = filterMobileItems(items, query, func(item music.Album) string { return item.Name + " " + artistNames(item.Artists) })
		if len(items) > 0 {
			a.mobileList(c, r, len(items), func(i int) { a.mobileAlbumRow(c, &items[i]) })
			return
		}
	} else {
		var tracks []music.Track
		for _, sec := range sections {
			tracks = append(tracks, sec.Tracks...)
		}
		tracks = filterMobileItems(tracks, query, func(tr music.Track) string { return tr.Name + " " + tr.ArtistNames() + " " + tr.Album.Name })
		if len(tracks) > 0 {
			a.mobileList(c, r, len(tracks), func(i int) { a.mobileSongRow(c, tracks, i, "", true) })
			return
		}
	}
	if query != "" {
		a.mobileEmpty(c, iconSearch, T("No Results"), "", nil)
	} else if lib.id == local.ID {
		icon, title, detail := iconMobileSongs, T("No Songs Yet"), T("Imported songs appear here")
		if albums {
			icon, title, detail = iconMobileAlbums, T("No Albums Yet"), T("Imported albums appear here")
		}
		ui.Column(c.Key("local-empty")).Grow(1).MinHeight(0).Padding(20).Center().Gap(10).Children(func() {
			ui.Icon(c, icon).Size(46, 46).TextColor(p.faint.Alpha(0.58))
			ui.Text(c, title).FontSize(18).FontWeight(600).TextColor(p.muted)
			ui.Text(c, detail).FontSize(15).TextColor(p.faint)
		})
	} else if lib.homeErr != "" {
		a.mobileLoading(c, lib.homeErr, func() { a.loadHome(lib) })
	} else if lib.can("search") && lib.id != local.ID {
		a.mobileEmpty(c, iconSearch, T("No Results"), T("Search"), func() { a.selectMobileTab(mobileSearch) })
	} else {
		a.mobileEmpty(c, iconNote, T("No Music"), "", nil)
	}
}

func filterMobileItems[T any](all []T, query string, text func(T) string) []T {
	if query == "" {
		return all
	}
	var found []T
	for _, item := range all {
		if strings.Contains(strings.ToLower(text(item)), query) {
			found = append(found, item)
		}
	}
	return found
}

func (a *app) mobilePlaylistList(c *ui.Context, r route, query string) {
	playlists := append(append([]music.Playlist(nil), a.mine...), a.saved...)
	if a.can("liked") {
		playlists = append([]music.Playlist{{Name: T("Liked Songs")}}, playlists...)
	}
	playlists = filterMobileItems(playlists, query, func(pl music.Playlist) string { return pl.Name })
	if len(playlists) == 0 {
		a.mobileEmpty(c, iconPlaylist, T("No Playlists"), "", nil)
		return
	}
	a.mobileList(c, r, len(playlists), func(i int) {
		pl := playlists[i]
		if pl.ID == "" {
			a.mobileMediaRow(c.Key("liked"), pl.Name, "", "", true, func() { a.open(route{page: pageLiked}) })
			return
		}
		a.mobilePlaylistRow(c, &pl)
	})
}

func (a *app) mobileResults(c *ui.Context, r route) {
	d := a.searches[r]
	if d == nil || d.res.Len() == 0 && (d.page.loading || d.page.err != "") {
		var message string
		if d != nil {
			message = d.page.err
		}
		a.mobileLoading(c, message, func() { a.loadSearch(r, true) })
		return
	}
	count := len(d.res.Tracks)
	switch r.kind {
	case music.SearchAlbums:
		count = len(d.res.Albums)
	case music.SearchPlaylists:
		count = len(d.res.Playlists)
	case music.SearchArtists:
		count = len(d.res.Artists)
	case music.SearchShows:
		count = len(d.res.Shows)
	}
	rows := count
	if d.page.more {
		rows++
	}
	if rows == 0 {
		a.mobileEmpty(c, iconSearch, T("No Results"), "", nil)
		return
	}
	a.mobileList(c, r, rows, func(i int) {
		if i == count {
			a.moreResults(a.library, r, d)
			return
		}
		switch r.kind {
		case music.SearchAlbums:
			a.mobileAlbumRow(c, &d.res.Albums[i])
		case music.SearchPlaylists:
			a.mobilePlaylistRow(c, &d.res.Playlists[i])
		case music.SearchShows:
			show := d.res.Shows[i]
			a.mobileMediaRow(c.Key(show.ID), show.Name, show.Publisher, show.ImageURL, true, func() { a.open(route{page: pageShow, id: show.ID}) })
		case music.SearchArtists:
			artist := d.res.Artists[i]
			a.mobileMediaRow(c.Key(artist.ID), artist.Name, "", artist.ImageURL, true, func() { a.open(route{page: pageArtist, id: artist.ID}) })
		default:
			a.mobileSongRow(c, d.res.Tracks, i, "", true)
		}
	})
}

func (a *app) mobileList(c *ui.Context, r route, count int, row func(int)) ui.Element {
	state := a.phone.scroll[r]
	if state == nil {
		state = new(ui.ListState)
		a.phone.scroll[r] = state
	}
	return ui.List(c.Key("rows"), state, count, row).Grow(1).MinHeight(0).Padding(0, 8, 12, 8)
}

func (a *app) mobileMediaRow(c *ui.Context, title, sub, cover string, disclosure bool, goTo func()) ui.Element {
	p := a.pal
	b := ui.ButtonBase(c).Label(title).Height(60).PaddingX(12).Gap(14).AlignItems(ui.Center).Radius(10).FocusRing(false)
	if b.Pressed() {
		b.Background(p.accent.Alpha(0.1))
	}
	b.Children(func() {
		a.cover(c, cover, 104, 44, 6)
		mobileLabels(c, title, sub, p.text, p.muted, 16)
		if disclosure {
			ui.Icon(c, iconForward).Size(13, 13).TextColor(p.faint)
		}
	})
	if b.Clicked() {
		goTo()
	}
	return b
}

func (a *app) mobileAlbumRow(c *ui.Context, album *music.Album) {
	var details []string
	if name := artistNames(album.Artists); name != "" {
		details = append(details, name)
	}
	if album.TrackCount > 0 {
		details = append(details, i18n.Songs(album.TrackCount))
	}
	if album.Duration > 0 {
		details = append(details, clock(time.Duration(album.Duration)*time.Millisecond))
	}
	r := route{page: pageAlbum, src: a.id, id: album.ID}
	selected := r == a.route
	b := a.mobileMediaRow(c.Key(album.ID), album.Name, strings.Join(details, " · "), album.ImageURL, !selected, func() {
		if _, split := a.mobileAlbumListRoute(); a.phone.wide && split {
			a.forward = nil
			a.show(r)
		} else {
			a.open(r)
		}
	})
	if selected {
		b.Checked(true).Background(a.pal.accent.Alpha(0.1))
	}
}

func (a *app) mobilePlaylistRow(c *ui.Context, pl *music.Playlist) {
	a.mobileMediaRow(c.Key(pl.ID), pl.Name, playlistLine(pl), pl.ImageURL, true,
		func() { a.open(route{page: pagePlaylist, id: pl.ID}) })
}

func (a *app) mobileSongRow(c *ui.Context, tracks []music.Track, i int, playlist string, covers bool) {
	tr, p := &tracks[i], a.pal
	b := ui.ButtonBase(c.Key(i)).Label(tr.Name).Height(56).PaddingX(12).Gap(12).
		AlignItems(ui.Center).Radius(10).FocusRing(false).ContextMenu(a.songItems(tr, playlist))
	if b.Pressed() {
		b.Background(p.text.Alpha(0.06))
	}
	if tr.Unavailable {
		b.Opacity(0.45)
	}
	b.Children(func() {
		if covers {
			a.cover(c, tr.Album.ImageURL, 96, 44, 6)
		} else {
			ui.Text(c, strconv.Itoa(i+1)).Width(26).TextAlign(ui.Center).FontSize(14).FontFeatures("tnum").TextColor(p.faint)
		}
		subtitle := tr.ArtistNames()
		if covers && tr.Album.Name != "" {
			if subtitle != "" {
				subtitle += " · "
			}
			subtitle += tr.Album.Name
		}
		mobileLabels(c, tr.Name, subtitle, p.text, p.muted, 16)
		if tr.Duration > 0 {
			ui.Text(c, clock(tr.Length())).FontSize(13).TextColor(p.faint).FontFeatures("tnum").SingleLine()
		}
	})
	if b.Clicked() && !tr.Unavailable {
		a.pl.PlayTrack(tracks, i)
	}
}

func mobileLabels(c *ui.Context, title, sub string, fg, dim ui.Color, size float32) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
		ui.Text(c, title).FontSize(size).TextColor(fg).SingleLine()
		if sub != "" {
			ui.Text(c, sub).FontSize(13).TextColor(dim).SingleLine()
		}
	})
}

func (a *app) mobileEmpty(c *ui.Context, icon *ui.SVG, title, action string, run func()) {
	p := a.pal
	ui.Column(c).Grow(1).MinHeight(0).Padding(20).Center().Gap(12).Children(func() {
		ui.Icon(c, icon).Size(35, 35).TextColor(p.faint)
		ui.Text(c, title).FontSize(20).FontWeight(600).TextAlign(ui.Center)
		if action != "" && pillButton(c.Key("empty-action"), nil, action, p.sidebar, p.accent).Clicked() {
			run()
		}
	})
}

func (a *app) mobileLoading(c *ui.Context, message string, retry func()) {
	if message != "" {
		if a.library.needsSetup || a.library.needsSignIn && a.library.signedOut() {
			ui.Column(c).Grow(1).MinHeight(0).Padding(20).Center().Gap(12).Children(func() {
				ui.Text(c, message).TextColor(a.pal.muted).TextAlign(ui.Center)
				a.remedy(c)
			})
		} else {
			a.mobileEmpty(c, iconMusicSource, message, T("Try Again"), retry)
		}
		return
	}
	ui.Column(c).Grow(1).Center().Children(func() { ui.Spinner(c) })
}

func (a *app) mobileDetail(c *ui.Context, r route) {
	lib := a.library
	var title, sub, image, playlist string
	var tracks []music.Track
	switch r.page {
	case pageLiked:
		if !lib.likedPaged.loaded {
			a.mobileLoading(c, lib.likedPaged.err, func() { a.loadLikedPage(lib) })
			return
		}
		title, sub, tracks = a.builtinTitle("liked"), i18n.Songs(len(lib.likedList)), lib.likedList
	case pageAlbum:
		d := lib.albums[r.id]
		if d == nil {
			a.loadAlbum(lib, r.id)
			d = lib.albums[r.id]
		}
		if d.album == nil {
			a.mobileLoading(c, d.err, func() { a.loadAlbum(lib, r.id) })
			return
		}
		album := d.album.Album
		title, sub, image, tracks = album.Name, artistNames(album.Artists), album.ImageURL, d.album.Tracks
	default:
		pl := lib.playlists[r.id]
		if pl == nil {
			a.mobileLoading(c, lib.plErr[r.id], func() { a.loadPlaylist(lib, r.id, false) })
			return
		}
		title, sub, image, tracks, playlist = pl.Name, playlistLine(pl), pl.ImageURL, pl.Tracks, pl.ID
	}
	rows := len(tracks) + 1
	if r.page == pageLiked && (lib.likedPaged.more || lib.likedPaged.err != "") {
		rows++
	}
	a.mobileList(c, r, rows, func(i int) {
		if i > len(tracks) {
			a.pageFoot(c, &lib.likedPaged, func() { a.loadLikedPage(lib) })
			return
		}
		if i > 0 {
			a.mobileSongRow(c, tracks, i-1, playlist, r.page != pageAlbum)
			return
		}
		ui.Column(c.Key("detail-header")).Center().Padding(10, 16, 14, 16).Gap(9).Children(func() {
			width, _ := c.Size()
			limit := float32(220)
			if a.phone.wide || a.phone.compactHeight {
				limit = 132
			}
			size := min(limit, max(0, width-48))
			if r.page == pageLiked {
				ui.Box(c).Size(size, size).Radius(12).Center().Background(a.pal.accent).Children(func() {
					ui.Icon(c, iconHeartFill).Size(74, 74).TextColor(white)
				})
			} else {
				a.cover(c, image, 600, size, 10)
			}
			ui.Text(c, title).FontSize(24).FontWeight(700).TextAlign(ui.Center)
			if sub != "" {
				ui.Text(c, sub).FontSize(16).TextColor(a.pal.muted).TextAlign(ui.Center)
			}
			if len(tracks) > 0 {
				ui.Row(c).Gap(10).Justify(ui.Center).Children(func() { a.playButtons(c, tracks) })
			}
		})
	})
}
