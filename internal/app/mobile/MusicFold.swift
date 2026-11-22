import UIKit

@MainActor final class MusicFoldObserver: UIView {
    private var region = CGRect.zero
    private var windowSize = CGSize.zero
    private var compactHeight = false
    private var regularWidth = false
    private var updateScheduled = false

    init(in view: UIView) {
        super.init(frame: view.bounds)
        isUserInteractionEnabled = false
        autoresizingMask = [.flexibleWidth, .flexibleHeight]
        #if compiler(>=6.4)
        if #available(iOS 27.1, *) {
            addInteraction(UIHingeInteraction { [weak self] _, _ in self?.scheduleUpdate() })
        }
        #endif
        registerForTraitChanges([UITraitHorizontalSizeClass.self, UITraitVerticalSizeClass.self]) {
            (observer: MusicFoldObserver, _) in observer.scheduleUpdate()
        }
        view.addSubview(self)
        scheduleUpdate()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError() }

    override func layoutSubviews() {
        super.layoutSubviews()
        scheduleUpdate()
    }

    private func scheduleUpdate() {
        guard !updateScheduled else { return }
        updateScheduled = true
        // UIKit updates the reserved regions as the scene lays out after a hinge change.
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            self.updateScheduled = false
            self.publishDivision()
        }
    }

    private func publishDivision() {
        guard let view = superview else { return }
        var frame = CGRect.zero
        #if compiler(>=6.4)
        if #available(iOS 27.1, *) {
            frame = view.reservedRegions(kind: .division).first(where: \.isActive)?.frame ?? .zero
        }
        #endif
        let size = view.bounds.size
        let compact = traitCollection.verticalSizeClass == .compact
        let regular = traitCollection.horizontalSizeClass == .regular
        guard frame != region || size != windowSize || compact != compactHeight || regular != regularWidth else { return }
        region = frame
        windowSize = size
        compactHeight = compact
        regularWidth = regular
        PluginHost.emit("musicMedia", "fold", [
            "width": size.width, "height": size.height,
            "region": ["x": frame.minX, "y": frame.minY, "w": frame.width, "h": frame.height],
            "compactHeight": compact, "regularWidth": regular,
        ] as [String: Any])
    }
}
