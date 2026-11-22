# AGENTS.md

## Project and references

MMMMM is a native MyGo music player with shared library, plugin and playback code across desktop and mobile. The website builds separately.

- [MyGo Music](https://github.com/egoist/mygo-music) is the upstream app and the reference for existing playback and plugin behaviour. See [README.md](README.md) for the public plugin API.
- [MuseAmp](https://github.com/Lakr233/MuseAmp) is the mobile UI and design reference. See [LICENSE](LICENSE) and [LICENSE-MuseAmp](LICENSE-MuseAmp) for attribution.
- [MyGo](https://github.com/egoist/mygo) provides the native UI and build tool. Inspect the pinned source with `go list -m -f '{{.Dir}}' github.com/egoist/mygo` and follow the [MyGo maintenance skill](.agents/skills/mygo-maintenance/SKILL.md) for UI lifetime, threading and upgrades.
- Use the [Apple HIG skill](.agents/skills/apple-hig/SKILL.md) for Apple platform UI work. The project provides `/duo` in Pi for iPhone Duo adaptation. [HIGAgentSkills 2026-09-27](https://github.com/justinwetch/HIGAgentSkills/tree/2206ff935b4affa9ad249a5399a3e8e29f5f6edd)

## Build and verify

- Use the Go version in `go.mod` and run `go mod download` from the repository root. MyGo needs Bun on `PATH` to evaluate `mygo.config.ts`.
- Run `go tool mygo dev` for desktop development. On Apple Silicon, `go tool mygo build -platform darwin/arm64 -sign - -skip-dmg -skip-notarize` builds a local ad hoc signed app in `build/`.
- Use the [mobile build commands](README.md#build) for iOS and Android. iOS requires Xcode with the target SDK. For Android, set `ANDROID_HOME` if MyGo cannot find the SDK.
- After Go changes, run `go build -o /dev/null .` and `go test ./internal/...` as CI does. Run `go tool mygo vet .` after changing MyGo UI or callbacks.
- Run one UI test with `go test ./internal/app -run '^TestMobileAlbumSplitPreservesSearch$' -count=5`, replacing the anchored test name as needed.
- After changing Swift or Kotlin sources in `internal/app/mobile/` or their `internal/app/mygo-plugin.json` manifest, build the affected mobile target. Go tests do not compile those sources. Duo fold detection requires Swift 6.4 and the iOS 27.1 SDK or later, and runs on iOS 27.1 or later. Check screen transitions and controls using [duo-notes.md](duo-notes.md).

## Compatibility and releases

- Keep `app.agili.mmmmm` as the application ID unless a migration is requested. Keep `mygo-music://` for existing plugin install links and `mygo-music-dev://` for development builds.
- Keep generated projects and artifacts in `.mygo/` and `build/` out of Git.
- Check [.github/workflows/release.yml](.github/workflows/release.yml) before release work. Pushing a `v*` tag triggers the release workflow. Create release tags or run the release workflow only when requested. Keep signing credentials in the keychain or CI secrets.

## Tests

Only write a test for behaviour that can actually break. If the code under test could change without the test noticing, or the test would fail on a harmless rewording, don't write it.

**Write tests for:**

- What input does: clicks, keys, drags, gestures and scrolling, and the state or effect they lead to (seeking when the progress bar is let go, Space after opening Now Playing, back and forward, flinging the cover, liking a song, leaving a station, switching source).
- Logic that doesn't need a view: parsing lyrics, formatting times and counts, the player's queue, seeking, fading and cache, decoding streams and images.
- A regression you just fixed. Say in the test's comment what went wrong.
- Frame budgets, in `perf_test.go`, so idle views don't start asking for frames they don't need.

**Don't write tests that:**

- Check that a list of labels shows on screen. That repeats the view code and breaks whenever the wording changes.
- Check default settings or the order of rows in a form.
- Test MyGo itself, for example that `AnimateTo` or a spring moves from 0 to 1.
- Depend on the wall clock. `ui.NewTester` starts its clock at the current time, so anything with a phase taken from the clock passes or fails depending on when the test runs. Also, `FramePending` ignores frames waiting on `After` or `Painter.After`.

**How:**

- Build views with `ui.NewTester` and set up the app with `newTestApp` (in `internal/app/helpers_test.go`). Find elements by their accessibility label, then `Click`, `RightClick`, `Press`/`Move`/`Release` or `Key`, and check the app's state afterwards.
- Swap out sources with `fakeSource` (in `internal/app/helpers_test.go`), setting the methods the test needs and deleting from `caps` what it should hide. Swap the clipboard and dialogs through package variables such as `copyText` and `pickFolders`, and restore the real one with `defer`.
- See what comes back from a source's goroutine with `posts` and `runPost`.
- Look at visuals instead of asserting on them. The `MM_FRAMES` tests save PNGs into that directory and skip when it isn't set. Check pixels only for coverage, such as whether a layer still covers the window.
- Start each test with a one-sentence comment saying the behaviour it checks, in the style of the code.
- Run `go test ./internal/...` before committing, and run a new UI test with `-count=5`. Fix or delete a flaky test; don't retry it.
