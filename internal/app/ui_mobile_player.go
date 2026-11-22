package app

import (
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"

	"github.com/miantiao-me/MMMMM/internal/lyrics"
	"github.com/miantiao-me/MMMMM/internal/music"
	"github.com/miantiao-me/MMMMM/internal/player"
)

func (a *app) mobileMiniPlayer(c *ui.Context, tr *music.Track) {
	m, p := a.phone, a.pal
	height, cover, margin := float32(60), float32(44), float32(20)
	if m.compactHeight {
		height, cover, margin = 48, 36, 10
	}
	bar := ui.Box(c.Key("mini-player")).Height(height).Shrink(0).Margin(0, margin, 4, margin).Radius(height/2).Clip().
		Material(glass.Glass{}).Background(p.bg.Alpha(0.88)).Border(1, p.hairline)
	bar.Draggable(&m.miniDrag, ui.DraggableOptions{
		Axis: ui.Vertical, Bounds: &ui.Rect{Y: -76, H: 76},
		Snap: func(point ui.Point) ui.Point {
			if point.Y < -32 {
				a.openMobilePlayer()
			}
			return ui.Point{}
		},
	})
	bar.Children(func() {
		ui.Row(c).Fill().PaddingX(8).AlignItems(ui.Center).Children(func() {
			open := ui.ButtonBase(c.Key("open")).Label(T("Now Playing")).Grow(1).MinWidth(0).
				Height(height - 4).AlignItems(ui.Center).Gap(9).FocusRing(false).ContextMenu(a.songItems(tr, ""))
			open.Draggable(&m.miniPage, ui.DraggableOptions{
				Axis: ui.Horizontal, Bounds: &ui.Rect{X: -68, W: 136},
				Snap: func(point ui.Point) ui.Point {
					if point.X < -34 {
						a.pl.Next()
					} else if point.X > 34 {
						a.pl.Prev()
					}
					return ui.Point{}
				},
			})
			open.Children(func() {
				a.cover(c, tr.Album.ImageURL, 96, cover, 9)
				mobileLabels(c, tr.Name, tr.ArtistNames(), p.text, p.muted, 15)
			})
			if open.Clicked() {
				a.openMobilePlayer()
			}
			play := iconPlay
			label := T("Play")
			if a.pl.Playing() {
				play, label = iconPause, T("Pause")
			}
			if mobileIcon(c.Key("play"), play, 22, p.text, label).Clicked() {
				a.pl.TogglePlay()
			}
			if mobileIcon(c.Key("next"), iconNext, 23, p.text, T("Next")).Clicked() {
				a.pl.Next()
			}
		})
		ui.Box(c).Absolute().Left(14).Right(14).Bottom(0).Height(2).Draw(func(paint *ui.Painter, rect ui.Rect) {
			dur := a.pl.Duration()
			if dur > 0 {
				rect.W *= clamp01(float32(a.pl.Position().Seconds() / dur.Seconds()))
				paint.Fill(rect, p.accent, 1)
			}
			if a.pl.Playing() {
				paint.After(250 * time.Millisecond)
			}
		})
	})
}

func (a *app) openMobilePlayer() {
	a.phone.pager.Page = 1
	a.phone.popup = ui.Point{}
	a.queueAt, a.lyricAt = -1, -2
	a.npPane = paneLyrics
	a.nowPlaying = true
}

const mobilePlayerHeaderHeight = 44

func mobileRoutePicker(left, top, width float32, fold mobileSplit) ui.Rect {
	if fold.sideBySide {
		width = fold.first
	}
	return ui.Rect{X: left + width - 56, Y: top, W: 44, H: 44}
}

func (a *app) mobileNowPlaying(c *ui.Context, tr *music.Track, w, h, top, right, bottom, left float32, fold mobileSplit) {
	m := a.phone
	width, height := w-left-right, h-top-bottom
	wide := m.wide || (fold.gap == 0 && width >= 540 && m.compactHeight)
	bg, end := ui.Hex("#7d7d7e"), ui.Hex("#777879")
	if wide {
		bg, end = ui.Hex("#21222a"), ui.Hex("#0a0a0c")
	}
	art := a.art.get(sized(tr.Album.ImageURL, 96), false)
	if art != nil && !art.failed {
		bg = art.tint.Mix(ui.Hex("#121216"), 0.46)
		end = bg.Mix(ui.Hex("#000000"), 0.7)
	}
	offscreen := ui.Motion{YPercent: 100, Opacity: 1}
	sheet := ui.Box(c.Key("now-playing")).Absolute().Left(0).Right(0).Top(0).Bottom(0).Clip().Modal().
		LinearGradient(ui.LinearGradient{Angle: 180, From: bg, To: end}).
		Transition(ui.ElementTransition{Position: true, Duration: 280 * time.Millisecond, Enter: &offscreen, Exit: &offscreen})
	sheet.Draggable(&m.popup, ui.DraggableOptions{
		Axis: ui.Vertical, Bounds: &ui.Rect{H: h},
		Snap: func(point ui.Point) ui.Point {
			if point.Y > 90 {
				a.nowPlaying = false
			}
			return ui.Point{}
		},
	})
	sheet.Children(func() {
		c.OnWindowShortcut(0, ui.KeyBack, a.back)
		c.OnWindowShortcut(0, ui.KeyEscape, a.back)
		if a.settings.Fog {
			a.backdrop(c, art, bg)
		}
		ui.Column(c).Fill().Padding(top, right, bottom, left).TextColor(white).Children(func() {
			headerWidth := width
			if fold.sideBySide {
				headerWidth = fold.first
			}
			ui.Row(c.Key("player-header")).Width(headerWidth).Height(mobilePlayerHeaderHeight).Shrink(0).
				AlignItems(ui.Center).Justify(ui.Center).PaddingX(56).Children(func() {
				ui.ButtonBase(c.Key("close")).Label(T("Close Now Playing")).Width(84).Height(44).Center().FocusRing(false).
					OnClick(func() { a.nowPlaying = false }).Children(func() {
					ui.Box(c).Size(64, 5).Radius(2.5).Background(white.Alpha(0.4))
				})
			})
			if a.npPane == paneComments {
				commentsWidth := width
				if fold.sideBySide {
					commentsWidth = fold.first
				}
				comments := ui.Column(c.Key("comment-page")).Width(commentsWidth).Grow(1).MinHeight(0).PaddingX(24)
				if fold.gap > 0 && !fold.sideBySide {
					comments.Grow(0).Height(max(0, fold.first-mobilePlayerHeaderHeight))
				}
				comments.Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						mobileIcon(c.Key("comments-back"), iconBack, 21, white, T("Back")).OnClick(a.back)
						ui.Text(c, T("Comments")).FontSize(25).FontWeight(700)
					})
					a.commentsPane(c, tr)
				})
				return
			}
			if wide {
				gap, padding := float32(16), float32(20)
				paneWidth := max(0, (width-gap)/2)
				if fold.sideBySide {
					paneWidth, gap = fold.first, fold.gap
				}
				if m.compactHeight {
					padding = 12
				}
				ui.Row(c.Key("wide")).Grow(1).MinHeight(0).Gap(gap).Children(func() {
					ui.Column(c.Key("artwork-panel")).Width(paneWidth).Shrink(0).FillHeight().Justify(ui.Center).Gap(12).Children(func() {
						if size := min(height*0.6, height-265); size > 0 {
							a.mobileArtwork(c, tr, paneWidth, size)
						}
						a.mobilePlayerControls(c, tr, height)
					})
					ui.Column(c.Key("list-panel")).Grow(1).MinWidth(0).FillHeight().PaddingX(padding).Children(func() {
						a.mobilePlayerList(c, tr, m.pager.Page == 0, height)
					})
				})
				return
			}
			pageHeight := height - mobilePlayerHeaderHeight
			if fold.gap > 0 {
				pageHeight = max(0, fold.first-mobilePlayerHeaderHeight)
			}
			requestedPage := -1
			pages := mobile.PageView(c.Key("pages"), &m.pager, 3, func(i int) {
				ui.Column(c).Fill().Children(func() {
					if i == 1 {
						ui.Column(c).Grow(1).MinHeight(0).Justify(ui.Center).Gap(12).Children(func() {
							artHeight := pageHeight - 240
							if fold.gap > 0 {
								artHeight = pageHeight - 16
							}
							if artHeight > 0 {
								a.mobileArtwork(c, tr, width, artHeight)
							}
							if fold.gap == 0 {
								a.mobileCompactControls(c, tr, height)
							}
						})
						if fold.gap == 0 {
							requestedPage = a.mobileCompactActions(c, tr)
						}
					} else {
						ui.Column(c).Fill().PaddingX(16).Children(func() {
							a.mobilePlayerList(c, tr, i == 0, pageHeight)
						})
					}
				})
			}).MinHeight(0)
			if fold.gap > 0 {
				pages.Height(pageHeight).Shrink(0)
				ui.Box(c).Height(fold.gap).Shrink(0)
				ui.Column(c).Grow(1).MinHeight(0).Justify(ui.Center).Children(func() {
					a.mobileCompactControls(c, tr, height-fold.first-fold.gap)
					requestedPage = a.mobileCompactActions(c, tr)
				})
			} else {
				pages.Grow(1)
			}
			// PageView writes the settled swipe position after building its pages.
			if requestedPage >= 0 {
				m.pager.Page = requestedPage
				c.AnimationFrame()
			}
		})
	})
}

func (a *app) mobileCompactControls(c *ui.Context, tr *music.Track, height float32) {
	pl := a.pl
	dim := white.Alpha(0.8)
	ui.Column(c.Key("compact-controls")).WidthPercent(100).MaxWidth(488).AlignSelf(ui.Center).PaddingX(32).Children(func() {
		ui.Text(c, tr.Name).FontWeight(700).FontSize(24).TextColor(white).TextAlign(ui.Center).SingleLine()
		ui.Text(c, tr.ArtistNames()).FontSize(15).TextColor(dim).TextAlign(ui.Center).SingleLine()
		gap := min(22, max(4, (height-300)*0.06))
		ui.Box(c).Height(gap)
		a.scrubber(c.Key("progress"), &a.npScrub, white.Alpha(0.14), white, 24, white)
		ui.Row(c).Children(func() {
			ui.Text(c, clock(pl.Position())).FontSize(13).FontFeatures("tnum").TextColor(dim)
			ui.Box(c).Grow(1)
			ui.Text(c, "−"+clock(max(0, pl.Duration()-pl.Position()))).FontSize(13).FontFeatures("tnum").TextColor(dim)
		})
		if pl.Playing() {
			c.After(500 * time.Millisecond)
		}
		a.mobileTransport(c, 33)
	})
}

func (a *app) mobileTransport(c *ui.Context, size float32) {
	pl := a.pl
	ui.Row(c.Key("transport")).Height(max(44, size+16)).AlignItems(ui.Center).Justify(ui.SpaceAround).Children(func() {
		if mobileIcon(c.Key("prev"), iconPrev, size, white, T("Previous")).Clicked() {
			pl.Prev()
		}
		if playButton(c.Key("play"), pl.Playing(), size+9, white).MinWidth(44).MinHeight(44).Clicked() {
			pl.TogglePlay()
		}
		if mobileIcon(c.Key("next"), iconNext, size, white, T("Next")).Clicked() {
			pl.Next()
		}
	})
}

func (a *app) mobileCompactActions(c *ui.Context, tr *music.Track) int {
	requestedPage := -1
	ui.Row(c.Key("auxiliary")).Height(44).Shrink(0).PaddingX(24).Gap(8).AlignItems(ui.Center).Children(func() {
		lines, _ := a.pl.Lyrics()
		lyric := ""
		if i := lyrics.Index(lines, a.pl.Position()); i >= 0 {
			lyric = lines[i].Text
		}
		if lyric != "" {
			b := ui.ButtonBase(c.Key("current-lyric")).Grow(1).MinWidth(0).MinHeight(44).Justify(ui.Start).
				FocusRing(false).Label(T("Lyrics")).Children(func() {
				ui.Text(c, lyric).FontSize(13).TextColor(white.Alpha(0.8)).SingleLine()
			})
			if b.Clicked() {
				requestedPage = 2
			}
		} else {
			ui.Box(c).Grow(1)
		}
		a.likeButton(c.Key("like"), tr, 20, white)
		more := mobileIcon(c.Key("more"), iconMore, 21, white, T("More"))
		more.Menu(func(menu *ui.Menu) {
			if menu.Item(T("Play Queue")).Chosen() {
				requestedPage = 0
			}
			if menu.Item(T("Lyrics")).Chosen() {
				requestedPage = 2
			}
			a.mobileCommentsItem(menu, tr)
			menu.Separator()
			a.songItems(tr, "")(menu)
		})
	})
	return requestedPage
}

func (a *app) mobileCommentsItem(menu *ui.Menu, tr *music.Track) {
	if a.trackLib(tr).can("comments") && menu.Item(T("Comments")).Chosen() {
		a.npPane = paneComments
	}
}

func (a *app) mobilePlayerList(c *ui.Context, tr *music.Track, queue bool, height float32) {
	if queue {
		pl := a.pl
		ui.Row(c).Margin(8, 0, 4, 0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, T("Play Queue")).FontSize(25).FontWeight(700).TextColor(white).Grow(1)
			shuffle := mobileIcon(c.Key("shuffle"), iconShuffle, 21, white, T("Shuffle")).Checked(pl.Shuffle())
			if shuffle.Clicked() {
				pl.ToggleShuffle()
			}
			mode := pl.Repeat()
			icon := iconRepeat
			if mode == player.RepeatOne {
				icon = iconRepeat1
			}
			if mobileIcon(c.Key("repeat"), icon, 21, white, T("Repeat")).Checked(mode != player.RepeatOff).Clicked() {
				pl.CycleRepeat()
			}
		})
		a.queuePane(c)
	} else {
		ui.Text(c, T("Lyrics")).FontSize(25).FontWeight(700).TextColor(white).Margin(14, 0, 8, 0)
		if a.settings.HotComments && a.trackLib(tr).can("comments") {
			a.commentsFor(tr)
		}
		a.lyricsPane(c, tr, height)
	}
}

func (a *app) mobileArtwork(c *ui.Context, tr *music.Track, width, limit float32) {
	size := max(0, min(512, width*0.8, width-64, limit))
	box := ui.Box(c).Size(size, size).Shrink(0).AlignSelf(ui.Center).Center()
	if style := a.settings.PlayerStyle; style != "" {
		artwork := a.art.get(sized(tr.Album.ImageURL, 900), false)
		tint := ui.Hex("#787882")
		if artwork != nil && !artwork.failed {
			tint = artwork.tint
		}
		box.Children(func() { a.retroPlayer(c, style, tr, artwork, size, tint) })
	} else if tr.Album.ImageURL == "" && !a.phone.wide {
		box.Children(func() {
			ui.Icon(c, iconMobileSongs).Size(min(32, size), min(32, size)).TextColor(ui.Hex("#424448").Alpha(0.38))
		})
	} else {
		box.Radius(28).Shadow(0, 12, 26, -6, ui.RGBA(0, 0, 0, 0.38)).Children(func() {
			a.cover(c, tr.Album.ImageURL, 900, size, 28)
		})
	}
}

func (a *app) mobilePlayerControls(c *ui.Context, tr *music.Track, height float32) {
	pl, m := a.pl, a.phone
	dim := white.Alpha(0.63)
	ui.Column(c).WidthPercent(100).MaxWidth(488).AlignSelf(ui.Center).Padding(2, 26, 5, 26).Gap(5).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				ui.Text(c, tr.Name).FontWeight(700).FontSize(22).TextColor(white).SingleLine()
				ui.Text(c, tr.ArtistNames()).FontSize(16).TextColor(dim).SingleLine()
			})
			a.likeButton(c.Key("like"), tr, 20, white)
			more := mobileIcon(c.Key("more"), iconMore, 22, white, T("More"))
			more.Menu(func(menu *ui.Menu) {
				a.mobileCommentsItem(menu, tr)
				a.songItems(tr, "")(menu)
			})
		})
		a.scrubber(c.Key("progress"), &a.npScrub, dim.Alpha(0.35), white.Alpha(0.78), 24, white)
		ui.Row(c).Children(func() {
			ui.Text(c, clock(pl.Position())).FontSize(12).FontFeatures("tnum").TextColor(dim)
			ui.Box(c).Grow(1)
			ui.Text(c, "−"+clock(max(0, pl.Duration()-pl.Position()))).FontSize(12).FontFeatures("tnum").TextColor(dim)
		})
		if pl.Playing() {
			c.After(500 * time.Millisecond)
		}
		size := float32(36)
		if height < 430 {
			size = 27
		}
		a.mobileTransport(c, size)
		ui.Row(c.Key("panels")).Height(44).AlignItems(ui.Center).Justify(ui.Center).Gap(4).Children(func() {
			page := 2
			if m.pager.Page == 0 {
				page = 0
			}
			for _, item := range []struct {
				name string
				page int
			}{{T("Lyrics"), 2}, {T("Play Queue"), 0}} {
				selected := page == item.page
				bg := white.Alpha(0.08)
				if selected {
					bg = white.Alpha(0.22)
				}
				if pillButton(c.Key(item.page), nil, item.name, bg, white).Height(36).Checked(selected).Clicked() {
					m.pager.Page = item.page
				}
			}
		})
	})
}
