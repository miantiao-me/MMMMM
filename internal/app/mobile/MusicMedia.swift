import AVFoundation
import AVKit
import MediaPlayer
import OSLog
import UIKit

final class MusicMediaPlugin: NSObject, MyGoPlugin, AVRoutePickerViewDelegate {
    private let logger = Logger(subsystem: Bundle.main.bundleIdentifier ?? "app.agili.mmmmm", category: "Audio output")
    private var routePicker: AVRoutePickerView?
    private weak var foldObserver: UIView?
    private var configured = false
    private var playing = false
    private var artwork: MPMediaItemArtwork?
    private var resumeAfterInterruption = false

    func call(_ method: String, _ args: [String: Any], _ reply: @escaping PluginReply) {
        switch method {
        case "configure":
            do {
                try AVAudioSession.sharedInstance().setCategory(.playback)
                configureCommands()
                reply(.success(nil))
            } catch {
                reply(.failure(PluginFailure("AudioSession", error.localizedDescription)))
            }
        case "update":
            update(args)
            reply(.success(nil))
        default:
            reply(.failure(PluginFailure("NoMethod", method)))
        }
    }

    private func configureCommands() {
        guard !configured else { return }
        configured = true
        let center = MPRemoteCommandCenter.shared()
        for (command, name) in [(center.playCommand, "play"), (center.pauseCommand, "pause"),
                                (center.togglePlayPauseCommand, "toggle"),
                                (center.nextTrackCommand, "next"), (center.previousTrackCommand, "previous")] {
            command.addTarget { _ in
                Self.command(name)
                return .success
            }
        }
        center.changePlaybackPositionCommand.addTarget { event in
            guard let event = event as? MPChangePlaybackPositionCommandEvent else { return .commandFailed }
            Self.command("seek", event.positionTime)
            return .success
        }
        NotificationCenter.default.addObserver(self, selector: #selector(interrupted),
                                               name: AVAudioSession.interruptionNotification, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(routeChanged),
                                               name: AVAudioSession.routeChangeNotification, object: nil)
    }

    @MainActor private func showRoutePicker(in view: UIView?, at frame: CGRect) {
        guard let view, !frame.isEmpty else {
            routePicker?.isHidden = true
            return
        }
        let picker = routePicker ?? AVRoutePickerView()
        if picker.superview !== view {
            picker.delegate = self
            picker.tintColor = .white
            picker.activeTintColor = .white
            picker.prioritizesVideoDevices = false
            picker.accessibilityLabel = "AirPlay"
            view.addSubview(picker)
            routePicker = picker
        }
        picker.frame = frame
        picker.isHidden = false
    }

    func routePickerViewWillBeginPresentingRoutes(_ routePickerView: AVRoutePickerView) {
        logger.notice("Presenting audio routes")
    }

    private func update(_ args: [String: Any]) {
        let window = args["window"] as? NSNumber
        let rect = args["routePicker"] as? [String: Double] ?? [:]
        let frame = CGRect(x: rect["X"] ?? 0, y: rect["Y"] ?? 0, width: rect["W"] ?? 0, height: rect["H"] ?? 0)
        DispatchQueue.main.async {
            let view = window.flatMap { MyGoWindows.controller($0)?.view }
            if let view, self.foldObserver?.superview !== view {
                self.foldObserver?.removeFromSuperview()
                self.foldObserver = MusicFoldObserver(in: view)
            }
            self.showRoutePicker(in: view, at: frame)
        }
        if let encoded = args["artwork"] as? String {
            artwork = Data(base64Encoded: encoded).flatMap { UIImage(data: $0) }.map { image in
                MPMediaItemArtwork(boundsSize: image.size) { _ in image }
            }
        }
        let present = args["present"] as? Bool ?? false
        playing = present && (args["playing"] as? Bool ?? false)
        let commands = MPRemoteCommandCenter.shared()
        commands.playCommand.isEnabled = present
        commands.pauseCommand.isEnabled = present
        commands.togglePlayPauseCommand.isEnabled = present
        commands.nextTrackCommand.isEnabled = present
        commands.previousTrackCommand.isEnabled = present
        let duration = args["duration"] as? Double ?? 0
        commands.changePlaybackPositionCommand.isEnabled = present && duration > 0
        let nowPlaying = MPNowPlayingInfoCenter.default()
        guard present else {
            nowPlaying.nowPlayingInfo = nil
            nowPlaying.playbackState = .stopped
            return
        }
        var info: [String: Any] = [
            MPMediaItemPropertyTitle: args["title"] as? String ?? "",
            MPMediaItemPropertyArtist: args["artist"] as? String ?? "",
            MPMediaItemPropertyAlbumTitle: args["album"] as? String ?? "",
            MPNowPlayingInfoPropertyElapsedPlaybackTime: args["position"] as? Double ?? 0,
            MPNowPlayingInfoPropertyPlaybackRate: playing ? 1.0 : 0.0,
            MPNowPlayingInfoPropertyMediaType: MPNowPlayingInfoMediaType.audio.rawValue,
        ]
        if duration > 0 { info[MPMediaItemPropertyPlaybackDuration] = duration }
        if let artwork { info[MPMediaItemPropertyArtwork] = artwork }
        nowPlaying.nowPlayingInfo = info
        nowPlaying.playbackState = playing ? .playing : .paused
    }

    @objc private func interrupted(_ notification: Notification) {
        guard let info = notification.userInfo,
              let value = info[AVAudioSessionInterruptionTypeKey] as? UInt,
              let type = AVAudioSession.InterruptionType(rawValue: value) else { return }
        DispatchQueue.main.async {
            switch type {
            case .began:
                self.resumeAfterInterruption = self.playing
                if self.playing { Self.command("pause") }
            case .ended:
                let options = AVAudioSession.InterruptionOptions(rawValue: info[AVAudioSessionInterruptionOptionKey] as? UInt ?? 0)
                if self.resumeAfterInterruption && options.contains(.shouldResume) {
                    Self.command("play")
                }
                self.resumeAfterInterruption = false
            @unknown default: break
            }
        }
    }

    @objc private func routeChanged(_ notification: Notification) {
        guard let value = notification.userInfo?[AVAudioSessionRouteChangeReasonKey] as? UInt,
              AVAudioSession.RouteChangeReason(rawValue: value) == .oldDeviceUnavailable else { return }
        DispatchQueue.main.async {
            if self.playing { Self.command("pause") }
        }
    }

    private static func command(_ name: String, _ position: Double = 0) {
        PluginHost.emit("musicMedia", "command", ["name": name, "position": position])
    }
}
