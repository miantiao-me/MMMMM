package app

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const mobileMediaID = "musicMedia"

type mobileMediaState struct {
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	Album       string  `json:"album"`
	Duration    float64 `json:"duration"`
	Position    float64 `json:"position"`
	Playing     bool    `json:"playing"`
	Present     bool    `json:"present"`
	Window      int     `json:"window,omitempty"`
	RoutePicker ui.Rect `json:"routePicker"`
	artwork     *[]byte
}

func (a *app) initMobileMedia() {
	a.phone.mediaUpdates = make(chan mobileMediaState, 1)
	a.pl.ArtworkChanged = func(data []byte) {
		a.phone.mediaArtwork = nil
		if len(data) > 0 {
			a.phone.mediaArtwork = &data
		}
		a.syncMobileMedia()
	}
	native := mygo.Native(mobileMediaID)
	native.On("fold", func(raw []byte) {
		var fold mobileFold
		if json.Unmarshal(raw, &fold) == nil {
			a.post(func() { a.phone.fold = fold })
		}
	})
	native.On("command", func(raw []byte) {
		var command struct {
			Name     string
			Position float64
		}
		if json.Unmarshal(raw, &command) != nil {
			return
		}
		a.post(func() {
			switch command.Name {
			case "play", "pause":
				if (command.Name == "play") != a.pl.Playing() {
					a.pl.TogglePlay()
				}
			case "toggle":
				a.pl.TogglePlay()
			case "next":
				a.pl.Next()
			case "previous":
				a.pl.Prev()
			case "seek":
				a.pl.Seek(time.Duration(command.Position * float64(time.Second)))
			}
		})
	})
	go func(updates <-chan mobileMediaState) {
		call := func(method string, state any) bool {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := native.Call(ctx, method, state, nil); err != nil {
				log.Printf("mobile media %s: %v", method, err)
				return false
			}
			return true
		}
		call("configure", nil)
		var sent *[]byte
		for state := range updates {
			update := struct {
				mobileMediaState
				Artwork *[]byte `json:"artwork,omitempty"`
			}{mobileMediaState: state}
			if state.artwork != sent {
				data := []byte{}
				if state.artwork != nil {
					if img, err := decodeImage(*state.artwork); err == nil {
						if _, jpeg, err := shrinkArt(img, 360); err == nil {
							data = jpeg
						}
					}
				}
				update.Artwork = &data
			}
			if call("update", update) {
				sent = state.artwork
			}
		}
	}(a.phone.mediaUpdates)
}

func (a *app) syncMobileMedia() {
	updates := a.phone.mediaUpdates
	if updates == nil {
		return
	}
	var state mobileMediaState
	if track := a.pl.Current(); track != nil {
		state = mobileMediaState{
			Title: track.Name, Artist: track.ArtistNames(), Album: track.Album.Name,
			Duration: a.pl.Duration().Seconds(), Position: a.pl.Position().Seconds(),
			Playing: a.pl.Playing(), Present: true, artwork: a.phone.mediaArtwork,
		}
	}
	if a.win != nil {
		state.Window = a.win.ID()
		state.RoutePicker = a.phone.audioRoute
	}
	// The system only needs the newest state if it is busy with an update.
	select {
	case <-updates:
	default:
	}
	updates <- state
}
