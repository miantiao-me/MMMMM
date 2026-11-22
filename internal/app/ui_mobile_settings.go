package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/miantiao-me/MMMMM/internal/local"
)

func (a *app) mobileSourceLibrary(id string) *library {
	for _, lib := range a.allLibs {
		if lib.id == id {
			return lib
		}
	}
	return nil
}

func (a *app) mobileSettingsScreen(c *ui.Context, r route) {
	ui.Scroll(c.Key("settings-scroll")).Grow(1).MinHeight(0).Padding(8, 16, 20, 16).Children(func() {
		ui.Column(c).Gap(22).Children(func() {
			switch r.page {
			case mobileSettings:
				a.mobileSettingsHome(c)
			case mobileSources:
				a.mobileSettingsGroup(c, T("Music Sources"), func() {
					for _, lib := range a.allLibs {
						c := c.Key(lib.id)
						detail := ""
						if lib == a.library {
							detail = T("Active")
						} else if !a.pluginEnabled(lib) {
							detail = T("Disabled")
						}
						a.mobileSettingLink(c, lib.name(), detail, iconMusicSource, func() {
							a.open(route{page: mobileSource, id: lib.id})
						})
					}
				})
			case mobileSource:
				lib := a.mobileSourceLibrary(r.id)
				if lib != nil {
					a.mobileSourceSettings(c, lib)
				}
			}
		})
	})
}

func (a *app) mobileSettingsHome(c *ui.Context) {
	ui.TextInput(c.Key("settings-search"), &a.settingsQuery).Label(T("Search Settings")).Placeholder(T("Search Settings"))
	query := strings.ToLower(strings.TrimSpace(a.settingsQuery))
	library := T("Library")
	sources, importMusic := T("Music Sources"), T("Import Music…")
	showLibrary := settingsMatch(query, library)
	showSources := showLibrary || settingsMatch(query, sources, a.library.name())
	showImport := showLibrary || settingsMatch(query, importMusic)
	if showSources || showImport {
		a.mobileSettingsGroup(c.Key("library"), library, func() {
			if showSources {
				a.mobileSettingLink(c.Key("sources"), sources, a.library.name(), iconMusicSource,
					func() { a.open(route{page: mobileSources}) })
			}
			if showImport {
				a.mobileSettingLink(c.Key("import"), importMusic, "", iconFolder, a.importMobileMusic)
			}
		})
	}
	found := showSources || showImport
	for _, sec := range a.settingsSections(c, a.pal) {
		if !a.settingsFinds(sec, query) {
			continue
		}
		found = true
		all := settingsMatch(query, sec.title)
		a.mobileSettingsGroup(c.Key(sec.id), sec.title, func() {
			sec.build(func(label, desc string, control func()) {
				if all || settingsMatch(query, label, desc) {
					a.mobileSettingRow(c, label, desc, control)
				}
			}, func(text string, draw func()) {
				if all || settingsMatch(query, text) {
					draw()
				}
			})
		})
	}
	if !found {
		ui.Text(c, T("No settings found")).TextColor(a.pal.muted)
	}
	if query == "" {
		ui.Text(c, Tf("Current Version %s", mygo.App.Version())).FontSize(13).TextColor(a.pal.muted).TextAlign(ui.Center)
	}
}

func (a *app) mobileSourceSettings(c *ui.Context, lib *library) {
	a.mobileSettingsGroup(c.Key("active"), "", func() {
		if a.pluginEnabled(lib) && lib != a.library {
			a.mobileSettingLink(c, T("Use This Music Source"), "", iconMusicSource, func() { a.selectMobileSource(lib) })
		} else if lib == a.library {
			a.mobileSetting(c, T("Active Music Source"), nil)
		}
	})
	if lib.id == local.ID {
		a.mobileSettingsGroup(c.Key("local"), T("Library"), func() {
			a.mobileSettingLink(c.Key("import"), T("Import Music…"), "", iconFolder, a.importMobileMusic)
			if a.local != nil && !a.local.Scanning() {
				a.mobileSettingLink(c.Key("rescan"), T("Rescan"), "", iconRefresh, a.local.Rescan)
			}
			a.mobileSetting(c.Key("enabled"), T("Enabled"), func() { a.pluginSwitch(c, lib) })
		})
		return
	}
	a.mobileSettingsGroup(c.Key("plugin"), T("Plugin"), func() { a.pluginSettings(c, lib, a.pal) })
}

func (a *app) mobileSettingsGroup(c *ui.Context, title string, rows func()) {
	p := a.pal
	ui.Column(c).Gap(7).Children(func() {
		if title != "" {
			ui.Text(c, title).Margin(0, 14, 0, 14).FontSize(13).TextColor(p.muted)
		}
		ui.Column(c).Radius(12).Background(p.sidebar).Dividers(1, p.hairline).Clip().Children(rows)
	})
}

func (a *app) mobileSetting(c *ui.Context, title string, control func()) {
	a.mobileSettingRow(c, title, "", control)
}

func (a *app) mobileSettingRow(c *ui.Context, title, detail string, control func()) {
	ui.Row(c).MinHeight(48).Padding(8, 14).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
			ui.Text(c, title).FontSize(16)
			if detail != "" {
				ui.Text(c, detail).FontSize(13).TextColor(a.pal.muted)
			}
		})
		if control != nil {
			control()
		}
	})
}

func (a *app) mobileSettingLink(c *ui.Context, title, detail string, icon *ui.SVG, action func()) {
	p := a.pal
	b := ui.ButtonBase(c).Label(title).MinHeight(48).Padding(5, 14).Gap(10).AlignItems(ui.Center).FocusRing(false)
	if b.Pressed() {
		b.Background(p.accent.Alpha(0.10))
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(21, 21).TextColor(p.accent)
		ui.Text(c, title).FontSize(16).Grow(1).MinWidth(0).SingleLine()
		if detail != "" {
			ui.Text(c, detail).FontSize(14).TextColor(p.muted).MaxWidth(110).SingleLine()
		}
		ui.Icon(c, iconForward).Size(12, 12).TextColor(p.faint)
	})
	if b.Clicked() {
		action()
	}
}

func (a *app) importMobileMusic() {
	if a.phone.importing || a.local == nil {
		return
	}
	a.phone.importing = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.win, Multiple: true,
			Filters: []mygo.FileFilter{
				{Name: T("Music Files"), Extensions: []string{"mp3", "flac", "ogg", "wav", "m4a", "aac", "lrc"}},
			},
		})
		dir := userDataDir("imported-music")
		count := 0
		if err == nil && len(paths) > 0 {
			err = os.MkdirAll(dir, 0o700)
			if err == nil {
				for _, path := range paths {
					if copyErr := keepImportedMusic(dir, path); copyErr != nil {
						err = copyErr
					} else {
						count++
					}
				}
			}
		}
		if count > 0 {
			a.local.Rescan()
		}
		a.post(func() {
			a.phone.importing = false
			if err != nil {
				a.toast = Tf("Couldn't import music: %s", err.Error())
			} else if count > 0 {
				a.toast = T("Music imported")
			}
		})
	}()
}

func keepImportedMusic(dir, path string) error {
	name := filepath.Base(path)
	destination := filepath.Join(dir, name)
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("%s: %w", name, os.ErrExist)
		}
		return err
	}
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	pending, err := os.CreateTemp(dir, ".import-*")
	if err != nil {
		return err
	}
	defer os.Remove(pending.Name())
	_, err = io.Copy(pending, input)
	closeErr := pending.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(pending.Name(), destination)
}
