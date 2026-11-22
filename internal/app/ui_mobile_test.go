package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/miantiao-me/MMMMM/internal/local"
	"github.com/miantiao-me/MMMMM/internal/music"
)

// Home stays reachable from the library menu, with Back through its collections to Albums.
func TestMobileHomeOpensCollection(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.library.home = &music.Browse{Sections: []music.Section{{Title: "For You", Links: []music.Link{
		{Title: "New Releases", Page: "releases"},
	}}}}
	a.library.browses["releases"] = &browseData{page: &music.Browse{Title: "New Releases"}}

	tt := ui.NewTester(a.mobileView, 402, 874)
	if err := tt.Click("Library actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Home"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("New Releases"); err != nil {
		t.Fatal(err)
	}
	if a.phone.tab != mobileAlbums || a.route.page != pageBrowse || a.route.id != "releases" {
		t.Fatalf("opened tab %v, route %+v", a.phone.tab, a.route)
	}
	tt.Key(0, ui.KeyBack)
	if a.route.page != pageHome {
		t.Fatalf("Back should return to Home, got %+v", a.route)
	}
	tt.Key(0, ui.KeyBack)
	if a.route.page != mobileAlbums {
		t.Fatalf("Back should return to Albums, got %+v", a.route)
	}
}

// Local search filters on a separate page and Back restores the complete library.
func TestMobileLocalSearchReturnsToUnfilteredSongs(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	lib := addFake(a, newFakeSource(local.ID))
	lib.src.(*fakeSource).info.Navigation = []music.NavItem{{Title: "Songs", Page: "songs", Icon: "note"}}
	lib.browses["songs"] = &browseData{page: &music.Browse{Sections: []music.Section{{Tracks: []music.Track{
		song("1", "First song"), song("2", "Second song"),
	}}}}}
	a.library = lib
	a.route = route{page: mobileSongs, src: local.ID}
	a.phone.tab = mobileSongs

	tt := ui.NewTester(a.mobileView, 402, 874)
	if err := tt.Click("Search Local Songs"); err != nil {
		t.Fatal(err)
	}
	tt.Type("First")
	if a.route.page != mobileLocalSongsSearch {
		t.Fatalf("search didn't open a separate page: %+v", a.route)
	}
	if _, found := tt.Find("Second song"); found {
		t.Fatal("search still shows the nonmatching song")
	}
	tt.Key(0, ui.KeyBack)
	if _, found := tt.Find("Second song"); !found {
		t.Fatal("Back didn't restore the complete song list")
	}
}

// The standalone search tab submits a query to the active plugin's catalog.
func TestMobileSearchQueriesPlugin(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.phone.tab, a.route.page = mobileSettings, mobileSettings
	ch := posts(a)
	fake(a).search = func(query string, kind music.SearchKind, _ int) (*music.SearchResult, error) {
		if query == "Slow Tide" && kind == music.SearchTracks {
			return &music.SearchResult{Tracks: []music.Track{song("1", "Slow Tide")}}, nil
		}
		return &music.SearchResult{}, nil
	}
	tt := ui.NewTester(a.mobileView, 402, 874)
	if err := tt.Click("Search"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Search Music"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Slow Tide")
	tt.Key(0, ui.KeyEnter)
	r := a.route
	if r.page != pageSearch || r.query != "Slow Tide" {
		t.Fatalf("submitted search opened %+v", r)
	}
	until(t, ch, func() bool { return !a.searches[r].page.loading })
	if got := a.searches[r].res.Tracks; len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("expected the plugin's search result, got %+v", got)
	}
}

// The library menu opens a plugin collection that shares the Albums tab's icon.
func TestMobileOpensAdditionalPluginCollection(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	fake(a).info.Navigation = []music.NavItem{
		{Title: "Albums", Page: "albums", Icon: "album"},
		{Title: "New Releases", Page: "releases", Icon: "album"},
	}
	a.library.browses["albums"] = &browseData{page: &music.Browse{Title: "Albums"}}
	a.library.browses["releases"] = &browseData{page: &music.Browse{Title: "New Releases"}}

	tt := ui.NewTester(a.mobileView, 402, 874)
	if err := tt.Click("Library actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("New Releases"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.route.page != pageBrowse || a.route.id != "releases" {
		t.Fatalf("opened %+v, want releases browse page", a.route)
	}
}

// The library actions and tab navigation stay on opposite sides of the fold as the window changes orientation.
func TestMobileLibraryAvoidsFold(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.phone.fold = mobileFold{Width: 951, Height: 669, Region: ui.Rect{X: 455.5, W: 40, H: 669}}
	tt := ui.NewTester(a.mobileView, 951, 669)
	settings, ok := tt.Find("Settings")
	if !ok || settings.X+settings.W > 455.5 {
		t.Fatalf("Settings must fit to the left of the fold, got %+v", settings)
	}
	actions, ok := tt.Find("Library actions")
	if !ok || actions.X < 495.5 {
		t.Fatalf("Library actions must be to the right of the fold, got %+v", actions)
	}

	a.phone.fold = mobileFold{Width: 669, Height: 951, Region: ui.Rect{Y: 455.5, W: 669, H: 40}}
	tt.SetSize(669, 951)
	actions, ok = tt.Find("Library actions")
	if !ok || actions.Y+actions.H > 455.5 {
		t.Fatalf("Library actions must fit above the fold, got %+v", actions)
	}
	settings, ok = tt.Find("Settings")
	if !ok || settings.Y < 495.5 {
		t.Fatalf("Settings must be below the fold, got %+v", settings)
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if a.route.page != mobileSettings {
		t.Fatalf("Settings button opened %+v", a.route)
	}
}

// Selecting another album on the inner display keeps the catalog and Back returns to the original search on the outer display.
func TestMobileAlbumSplitPreservesSearch(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	lib := addFake(a, newFakeSource(local.ID))
	a.activate(lib)
	a.route = route{page: mobileAlbums, src: lib.id}
	albums := []music.Album{{ID: "1", Name: "Live at Dawn"}, {ID: "2", Name: "Live at Night"}, {ID: "3", Name: "Studio"}}
	lib.home = &music.Browse{Sections: []music.Section{{Albums: albums}}}
	for _, album := range albums {
		lib.albums[album.ID] = &albumData{album: &music.AlbumPage{Album: album, Tracks: []music.Track{song(album.ID, "Track "+album.ID)}}}
	}

	tt := ui.NewTester(a.mobileView, 402, 874)
	if err := tt.Click("Search Local Library"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Live")
	search := a.route
	if err := tt.Click("Live at Dawn"); err != nil {
		t.Fatal(err)
	}
	a.phone.fold = mobileFold{Width: 951, Height: 669, Region: ui.Rect{X: 455.5, W: 40, H: 669}}
	tt.SetSize(951, 669)
	album, found := tt.Find("Live at Night")
	if !found || album.X+album.W > 455.5 {
		t.Fatalf("the filtered catalog must remain beside the album, got %+v", album)
	}
	if _, found := tt.Find("Studio"); found {
		t.Fatal("the catalog lost its filter")
	}
	if err := tt.Click("Live at Night"); err != nil {
		t.Fatal(err)
	}
	track, found := tt.Find("Track 2")
	if !found || track.X < 495.5 {
		t.Fatalf("the selected album's tracks must be across the fold, got %+v", track)
	}
	tt.SetSize(402, 874)
	if _, found := tt.Find("Track 2"); !found {
		t.Fatal("switching to the outer display lost the selected album")
	}
	tt.Key(0, ui.KeyBack)
	if a.route != search || *a.phone.filters[search] != "Live" {
		t.Fatalf("Back lost the original search, got %+v", a.route)
	}
}

// A large bottom inset leaves the primary navigation inside the usable window.
func TestMobileTabsWithLargeBottomInset(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	tt := ui.NewTester(a.mobileView, 402, 580)
	tt.SetInsets(24, 0, 220, 0)
	settings, found := tt.Find("Settings")
	if !found || settings.Y+settings.H > 360 {
		t.Fatalf("Settings must be above the bottom inset, got %+v", settings)
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if a.route.page != mobileSettings {
		t.Fatalf("Settings opened %+v", a.route)
	}
}

// The queue stays selected across a book fold and transport stays below a table fold.
func TestMobilePlayerPreservesQueueAcrossFold(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.pl.Cue([]music.Track{song("1", "First song"), song("2", "Second song")})
	a.openMobilePlayer()
	tt := ui.NewTester(a.mobileView, 402, 874)
	settled(tt)
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Play Queue"); err != nil {
		t.Fatal(err)
	}
	settled(tt)

	a.phone.fold = mobileFold{Width: 951, Height: 669, Region: ui.Rect{X: 455.5, W: 40, H: 669}}
	tt.SetSize(951, 669)
	settled(tt)
	shuffle, found := tt.Find("Shuffle")
	if !found || shuffle.X < 495.5 {
		t.Fatalf("the queue controls must be to the right of the fold, got %+v", shuffle)
	}
	a.phone.fold = mobileFold{Width: 669, Height: 951, Region: ui.Rect{Y: 455.5, W: 669, H: 40}}
	tt.SetSize(669, 951)
	settled(tt)
	shuffle, found = tt.Find("Shuffle")
	if !found || shuffle.Y+shuffle.H > 455.5 {
		t.Fatalf("the queue controls must be above the fold, got %+v", shuffle)
	}
	next, found := tt.Find("Next")
	if !found || next.Y < 495.5 {
		t.Fatalf("the transport must stay below the fold, got %+v", next)
	}
	if err := tt.Click("Shuffle"); err != nil {
		t.Fatal(err)
	}
	if !a.pl.Shuffle() || a.phone.pager.Page != 0 || a.pl.Current().ID != "1" {
		t.Fatal("resizing disturbed the current track or the selected queue")
	}
}

// System Back closes the player before it moves through the library history.
func TestMobileBackClosesNowPlaying(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.pl.Cue([]music.Track{song("1", "Sample track")})
	a.open(route{page: mobileSongs})
	a.openMobilePlayer()
	tt := ui.NewTester(a.mobileView, 402, 874)
	tt.Key(0, ui.KeyEscape)
	if a.nowPlaying || a.route.page != mobileSongs {
		t.Fatalf("Back should close the player above Songs, got player=%v, route=%+v", a.nowPlaying, a.route)
	}
	tt.Key(0, ui.KeyBack)
	if a.route.page != mobileAlbums {
		t.Fatalf("Back should return to Albums, got %+v", a.route)
	}
}

// The mini player opens Now Playing, where the queue menu leads to the shuffle control.
func TestMobileMiniPlayerOpensQueue(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.pl.Cue([]music.Track{song("1", "First song"), song("2", "Second song")})
	tt := ui.NewTester(a.mobileView, 402, 874)
	if err := tt.Click("Now Playing"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Play Queue"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if err := tt.Click("Shuffle"); err != nil {
		t.Fatal(err)
	}
	if !a.nowPlaying || a.phone.pager.Page != 0 || !a.pl.Shuffle() {
		t.Fatal("the queue did not turn on shuffle")
	}
}

// The compact player's heart likes the current song through its source.
func TestMobileNowPlayingLikesTrack(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.account = &music.Account{ID: "1", Name: "Alex"}
	calls := make(chan string, 1)
	fake(a).like = func(id string, like bool) error {
		if like {
			calls <- id
		}
		return nil
	}
	a.pl.Cue([]music.Track{song("1", "Sample track")})
	a.openMobilePlayer()
	tt := ui.NewTester(a.mobileView, 402, 874)
	settled(tt)
	if err := tt.Click("Like"); err != nil {
		t.Fatal(err)
	}
	if !a.isLiked(a.pl.Current()) {
		t.Fatal("the playing song is still unliked")
	}
	if id := wait(t, calls, "the like"); id != "1" {
		t.Fatalf("liked track %q, want 1", id)
	}
}

// Reopening the tablet sidebar restores navigation, including direct access to playlists.
func TestMobileSidebarOpensPlaylist(t *testing.T) {
	a := newTestApp()
	a.phone = newMobileState()
	a.route.page = mobileAlbums
	a.mine = []music.Playlist{{ID: "morning", Name: "Morning"}}
	a.library.playlists["morning"] = &a.mine[0]
	tt := ui.NewTester(a.mobileView, 1080, 820)
	if err := tt.Click("Hide Sidebar"); err != nil {
		t.Fatal(err)
	}
	if _, found := tt.Find("Morning"); found {
		t.Fatal("the playlist remained available after hiding the sidebar")
	}
	if err := tt.Click("Show Sidebar"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Morning"); err != nil {
		t.Fatal(err)
	}
	if a.phone.tab != mobilePlaylists || a.route.page != pagePlaylist || a.route.id != "morning" {
		t.Fatalf("opened tab %v, route %+v", a.phone.tab, a.route)
	}
	tt.Key(0, ui.KeyBack)
	if a.route.page != mobilePlaylists {
		t.Fatalf("Back should return to all playlists, got %+v", a.route)
	}
}
