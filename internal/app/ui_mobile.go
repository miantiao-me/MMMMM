package app

import (
	"runtime"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"

	"github.com/miantiao-me/MMMMM/internal/i18n"
	"github.com/miantiao-me/MMMMM/internal/local"
	"github.com/miantiao-me/MMMMM/internal/music"
)

const (
	mobileAlbums pageKind = -1 - iota
	mobileSongs
	mobilePlaylists
	mobileSettings
	mobileSources
	mobileSource
	mobileSearch
	mobileLocalAlbumsSearch
	mobileLocalSongsSearch
)

type mobileTab struct {
	page pageKind
	name string
	icon *ui.SVG
}

var iconMusicSource = strokeIcon(`<rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/>`)

var (
	iconMobileAlbums    = fillIcon(`<rect x="4" y="7" width="16" height="16" rx="3"/><path d="M7 5.5A2.5 2.5 0 0 1 9.5 3h5A2.5 2.5 0 0 1 17 5.5ZM8.5 2A1.5 1.5 0 0 1 10 0.5h4A1.5 1.5 0 0 1 15.5 2Z"/>`)
	iconMobileSongs     = fillIcon(`<path d="M10 18V4l9-2v4l-6 1.5V18a4.5 4.5 0 1 1-3-4.2z"/>`)
	iconMobilePlaylists = fillIcon(`<rect x="1" y="5" width="11" height="1.8" rx="0.9"/><rect x="1" y="9" width="11" height="1.8" rx="0.9"/><rect x="1" y="13" width="9" height="1.8" rx="0.9"/><path d="M15 17V4l8-2v4l-5 1.4V17a4 4 0 1 1-3-3.87z"/>`)
	iconMobileSettings  = fillIcon(`<path fill-rule="evenodd" d="M9 2h6l.6 3 2 .9 2.7-1.4 3 5.2-2.2 2v2.4l2.2 2-3 5.2-2.7-1.4-2 .9-.6 3H9l-.6-3-2-.9-2.7 1.4-3-5.2 2.2-2v-2.4l-2.2-2 3-5.2 2.7 1.4 2-.9L9 2zm3 7a4 4 0 1 0 0 8 4 4 0 0 0 0-8z"/>`)
	iconMobileSidebar   = strokeIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16M5.5 8h1M5.5 11h1M5.5 14h1"/>`)
)

var mobileTabs = []mobileTab{
	{mobileAlbums, N("Albums"), iconMobileAlbums},
	{mobileSongs, N("Songs"), iconMobileSongs},
	{mobilePlaylists, N("Playlist"), iconMobilePlaylists},
	{mobileSettings, N("Settings"), iconMobileSettings},
}

type mobileStack struct {
	route   route
	history []route
}

type mobileState struct {
	tab           pageKind
	stacks        map[pageKind]mobileStack
	scroll        map[route]*ui.ListState
	filters       map[route]*string
	searchField   ui.Handle
	pager         mobile.PageState
	popup         ui.Point
	miniDrag      ui.Point
	miniPage      ui.Point
	backDrag      ui.Point
	wide          bool
	compactHeight bool
	sidebarHidden bool
	importing     bool
	playerBars    bool
	audioRoute    ui.Rect
	fold          mobileFold
	mediaArtwork  *[]byte
	mediaUpdates  chan mobileMediaState
}

type mobileFold struct {
	Width, Height float32
	Region        ui.Rect
	CompactHeight bool
	RegularWidth  bool
}

type mobileSplit struct {
	first, gap float32
	sideBySide bool
}

func (f mobileFold) split(w, h, top, right, bottom, left float32) mobileSplit {
	if f.Width != w || f.Height != h {
		return mobileSplit{}
	}
	r := f.Region
	if r.W > 0 && r.X > left && r.X+r.W < w-right && r.Y <= top && r.Y+r.H >= h-bottom {
		return mobileSplit{first: r.X - left, gap: r.W, sideBySide: true}
	}
	if r.H > 0 && r.Y > top && r.Y+r.H < h-bottom && r.X <= left && r.X+r.W >= w-right {
		return mobileSplit{first: r.Y - top, gap: r.H}
	}
	return mobileSplit{}
}

func newMobileState() *mobileState {
	return &mobileState{
		tab: mobileAlbums, stacks: make(map[pageKind]mobileStack),
		scroll: make(map[route]*ui.ListState), filters: make(map[route]*string),
	}
}

func runMobile() error {
	a := &app{
		phone: newMobileState(), signInOver: new(atomic.Pointer[mygo.Window]),
		settingText: make(map[string]*string), lyricAt: -1, queueAt: -1, cacheUsed: -1,
	}
	mygo.App.WhenReady(func() {
		a.settings = loadSettings()
		a.settings.applyAppearance()
		mygo.SetOrientations(mygo.OrientationAny)
		i18n.SetLanguage(i18n.Resolve(a.settings.Language, systemLanguages()))
		a.makeSources()
		a.initPlayer()
		if a.settings.PlayerStyle != "" {
			go retroBitmaps()
		}
		a.initMobileMedia()
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title: mygo.App.Name(), BackgroundColor: "light-dark(#ffffff, #000000)",
			Content: ui.View(a.mobileView),
		})
		a.syncMobileMedia()
		a.boot()
		for _, link := range a.links {
			a.openLink(link)
		}
		a.links = nil
	})
	mygo.App.OnSaveState(func() []byte {
		if a.pl != nil {
			a.saveSession()
		}
		return nil
	})
	mygo.App.OnOpenURL(func(link string) {
		if a.win == nil {
			a.links = append(a.links, link)
		} else {
			a.openLink(link)
		}
	})
	return mygo.App.Run()
}

func (a *app) selectMobileTab(tab pageKind) {
	m := a.phone
	if m.tab == tab {
		if len(a.history) == 0 {
			if list := m.scroll[a.route]; list != nil {
				list.ScrollTo(0, ui.Start)
			}
			return
		}
		a.history = nil
		a.show(route{page: tab, src: a.id})
		return
	}
	if tab == mobileSettings {
		a.measureCache()
	}
	m.stacks[m.tab] = mobileStack{route: a.route, history: a.history}
	m.tab = tab
	a.nowPlaying = false
	stack := m.stacks[tab]
	if stack.route.src != a.id {
		stack = mobileStack{route: route{page: tab, src: a.id}}
	}
	a.history, a.forward = stack.history, nil
	a.show(stack.route)
}

func (a *app) selectMobileSource(lib *library) {
	if lib == a.library {
		return
	}
	a.activate(lib)
	m := a.phone
	a.searched = route{}
	clear(m.stacks)
	clear(m.scroll)
	clear(m.filters)
	a.history, a.forward = nil, nil
	m.tab = mobileAlbums
	a.show(route{page: m.tab, src: lib.id})
}

func (a *app) mobileHasSearch() bool {
	return a.id != local.ID && a.can("search")
}

// mobileCollections keeps the source's full pages alongside the catalog tabs.
func (a *app) mobileCollections() []sidebarItem {
	var items []sidebarItem
	home := false
	for _, item := range a.sidebarItems() {
		switch item.route.page {
		case pageHome:
			home = true
		case pageLiked:
			continue
		case pageBrowse:
			if a.id == local.ID {
				continue
			}
		}
		items = append(items, item)
	}
	if !home {
		items = append([]sidebarItem{{iconHome, T("Home"), route{page: pageHome}}}, items...)
	}
	return items
}

func (a *app) mobileView(c *ui.Context) {
	t := *c.Theme()
	p := paletteFor(&t)
	p.bg, p.sidebar, p.text, p.accent = white, ui.Hex("#f2f2f7"), ui.Hex("#000000"), ui.Hex("#e42319")
	if p.dark {
		p.bg, p.sidebar, p.text, p.accent = ui.Hex("#000000"), ui.Hex("#1c1c1e"), white, ui.Hex("#ff453a")
	}
	a.pal = p
	useAccent(&t, p)
	t.Text, t.TextMuted, t.Background = p.text, p.muted, p.bg
	c.SetTheme(&t)
	m := a.phone
	w, h := c.Size()
	top, right, bottom, left := c.SafeArea()
	width, height := w-left-right, h-top-bottom
	fold := m.fold.split(w, h, top, right, bottom, left)
	m.wide = fold.sideBySide || (fold.gap == 0 && (width >= 720 || m.fold.RegularWidth && width >= 680))
	m.compactHeight = height < 500 || m.fold.CompactHeight
	tr := a.pl.Current()
	if tr == nil {
		a.nowPlaying = false
	}
	if m.tab == mobileSearch && !a.mobileHasSearch() {
		a.selectMobileTab(mobileAlbums)
	}
	if a.win != nil && m.playerBars != a.nowPlaying {
		m.playerBars = a.nowPlaying
		style := mygo.BarStyleAuto
		if m.playerBars {
			style = mygo.BarStyleLight
		}
		mygo.SystemBars.SetStyle(style)
		if runtime.GOOS == "ios" {
			mygo.SystemBars.SetHidden(m.playerBars)
		}
	}
	if runtime.GOOS == "ios" && a.win != nil {
		var picker ui.Rect
		if a.nowPlaying && a.npPane != paneComments && m.popup.Y == 0 {
			picker = mobileRoutePicker(left, top, width, fold)
		}
		if m.audioRoute != picker {
			m.audioRoute = picker
			a.syncMobileMedia()
		}
	}
	if len(a.history) > 0 && !a.nowPlaying {
		c.OnShortcut(0, ui.KeyBack, a.back)
		c.OnShortcut(0, ui.KeyEscape, a.back)
	}
	hideSidebar := m.wide && !fold.sideBySide && m.sidebarHidden
	albumList, showAlbums := a.mobileAlbumListRoute()
	showAlbums = showAlbums && m.wide && (fold.sideBySide || width >= 760)
	const duoSidebarWidth = 164
	ui.Row(c).Fill().Clip().Background(p.bg).Children(func() {
		if fold.sideBySide {
			ui.Row(c.Key("leading-pane")).Width(fold.first).Shrink(0).FillHeight().Clip().Children(func() {
				a.mobileSidebar(c, min(duoSidebarWidth, fold.first), false)
				if showAlbums && fold.first >= duoSidebarWidth+180 {
					a.mobileAlbumList(c, albumList, 0)
				}
			})
			ui.Box(c).Width(fold.gap).Shrink(0)
		} else if m.wide {
			if !hideSidebar {
				a.mobileSidebar(c, 196, true)
			}
			if showAlbums {
				a.mobileAlbumList(c, albumList, min(340, width*0.32))
			}
		}
		ui.Column(c.Key("main")).Grow(1).MinWidth(0).FillHeight().Clip().Children(func() {
			if fold.gap > 0 && !fold.sideBySide {
				ui.Column(c).Height(fold.first).Shrink(0).Clip().Children(func() { a.mobileScreen(c, false) })
				ui.Box(c).Height(fold.gap).Shrink(0)
				ui.Box(c).Grow(1)
			} else {
				a.mobileScreen(c, hideSidebar)
			}
			if tr != nil {
				a.mobileMiniPlayer(c, tr)
			}
			if !m.wide {
				a.mobileTabBar(c)
			}
		})
	})
	ui.Overlay(c.Key("player-overlay"), func() {
		if a.nowPlaying {
			a.mobileNowPlaying(c, tr, w, h, top, right, bottom, left, fold)
		}
	})
	a.signInDialog(c, p, a.signIn.here)
	a.playlistNameDialog(c)
	a.deletePlaylistDialog(c)
	a.installDialog(c, p)
	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	a.askLikes()
}

func (a *app) mobileSidebar(c *ui.Context, width float32, collapsible bool) {
	p := a.pal
	bg := ui.Hex("#ece7e7")
	if p.dark {
		bg = p.sidebar
	}
	heading := func(label string) {
		ui.Text(c, label).FontSize(14).TextColor(p.muted).Margin(16, 10, 6, 10).SingleLine()
	}
	item := func(c *ui.Context, label string, icon *ui.SVG, selected bool, open func()) {
		fill := ui.Transparent
		if selected {
			fill = p.text.Alpha(0.09)
		}
		ui.ButtonBase(c).Label(label).Height(44).PaddingX(10).Gap(8).Radius(16).Background(fill).
			AlignItems(ui.Center).Justify(ui.Start).TextColor(p.text).Checked(selected).OnClick(open).Children(func() {
			ui.Icon(c, icon).Size(21, 21).TextColor(p.accent).Shrink(0)
			ui.Text(c, label).FontSize(15).SingleLine().MinWidth(0)
		})
	}
	openPlaylist := func(r route) {
		if a.phone.tab != mobilePlaylists {
			a.selectMobileTab(mobilePlaylists)
		}
		a.open(r)
	}
	ui.Column(c.Key("sidebar")).Width(width).Shrink(0).FillHeight().Background(bg).Padding(0, 8, 10, 8).Children(func() {
		if collapsible {
			a.mobileSidebarButton(c.Key("hide"), T("Hide Sidebar"), true).AlignSelf(ui.End).Margin(4, 0, 4, 0)
		}
		ui.Scroll(c.Key("collections")).Grow(1).MinHeight(0).Gap(1).Children(func() {
			heading(T("Library"))
			for _, tab := range mobileTabs[:3] {
				label := T(tab.name)
				selected := tab.page == a.phone.tab
				if tab.page == mobilePlaylists {
					label = T("All Playlists")
					selected = a.route.page == mobilePlaylists
				}
				if a.route.page == pageHome || a.route.page == pageBrowse {
					selected = false
				}
				item(c.Key(tab.page), label, tab.icon, selected, func() { a.selectMobileTab(tab.page) })
			}
			if a.mobileHasSearch() {
				item(c.Key("search"), T("Search"), iconSearch, a.phone.tab == mobileSearch, func() { a.selectMobileTab(mobileSearch) })
			}
			if a.can("liked") || len(a.mine)+len(a.saved) > 0 {
				heading(T("Playlists"))
				if a.can("liked") {
					item(c.Key("liked"), T("Liked Songs"), iconHeartFill, a.route.page == pageLiked, func() { openPlaylist(route{page: pageLiked}) })
				}
				for _, list := range [][]music.Playlist{a.mine, a.saved} {
					for _, pl := range list {
						item(c.Key(pl.ID), pl.Name, iconMobilePlaylists, a.route.page == pagePlaylist && a.route.id == pl.ID,
							func() { openPlaylist(route{page: pagePlaylist, id: pl.ID}) })
					}
				}
			}
			if a.id != local.ID {
				heading(a.library.name())
				for _, entry := range a.mobileCollections() {
					selected := a.route.page == entry.route.page && a.route.id == entry.route.id
					item(c.Key(entry.route), entry.label, entry.icon, selected, func() {
						if a.phone.tab != mobileAlbums && a.phone.tab != mobileSongs {
							a.selectMobileTab(mobileAlbums)
						}
						a.open(entry.route)
					})
				}
			}
			ui.Box(c).Height(12)
			item(c.Key("settings"), T("Settings"), iconMobileSettings, a.phone.tab == mobileSettings, func() { a.selectMobileTab(mobileSettings) })
		})
	})
}

func (a *app) mobileSidebarButton(c *ui.Context, label string, hide bool) ui.Element {
	p := a.pal
	return mobileIcon(c, iconMobileSidebar, 22, p.text, label).
		Material(glass.Glass{}).Background(p.bg.Alpha(0.86)).Border(1, p.hairline).
		OnClick(func() { a.phone.sidebarHidden = hide })
}

func (a *app) mobileTabBar(c *ui.Context) {
	p := a.pal
	height, iconSize, margin := float32(64), float32(24), float32(20)
	if a.phone.compactHeight {
		height, iconSize, margin = 48, 19, 10
	}
	ui.Row(c.Key("tab-bar")).Height(height).Shrink(0).Margin(2, margin, 3, margin).Gap(8).Children(func() {
		ui.Row(c.Key("tabs")).Grow(1).MinWidth(0).Padding(3).Radius(height/2).
			Material(glass.Glass{}).Background(p.bg.Alpha(0.86)).Border(1, p.hairline).Children(func() {
			for _, tab := range mobileTabs {
				label := T(tab.name)
				selected := tab.page == a.phone.tab
				fg := p.text
				b := ui.ButtonBase(c.Key(tab.page)).Label(label).Grow(1).MinWidth(0).Column().Center().Gap(3).Radius(28).Checked(selected)
				if selected {
					fg = p.accent
					b.Background(p.text.Alpha(0.08))
				}
				b.TextColor(fg).OnClick(func() { a.selectMobileTab(tab.page) }).Children(func() {
					ui.Icon(c, tab.icon).Size(iconSize, iconSize)
					ui.Text(c, label).FontSize(11).FontWeight(600).SingleLine()
				})
			}
		})
		if a.mobileHasSearch() {
			selected := a.phone.tab == mobileSearch
			bg, fg := p.bg.Alpha(0.86), p.text
			if selected {
				bg, fg = p.sidebar, p.accent
			}
			ui.ButtonBase(c.Key("search")).Label(T("Search")).Size(height, height).Shrink(0).Radius(height/2).Center().Checked(selected).
				Material(glass.Glass{}).Background(bg).Border(1, p.hairline).OnClick(func() { a.selectMobileTab(mobileSearch) }).Children(func() {
				ui.Icon(c, iconSearch).Size(25, 25).TextColor(fg)
			})
		}
	})
}

func mobileIcon(c *ui.Context, icon *ui.SVG, size float32, color ui.Color, label string) ui.Element {
	b := iconButton(c, icon, size, color, label).MinWidth(44).MinHeight(44).FocusRing(false)
	return b
}
