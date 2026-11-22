package dev.mygo

import android.graphics.Rect
import android.view.View
import androidx.core.content.ContextCompat
import androidx.core.util.Consumer
import androidx.window.java.layout.WindowInfoTrackerCallbackAdapter
import androidx.window.layout.FoldingFeature
import androidx.window.layout.WindowInfoTracker
import androidx.window.layout.WindowLayoutInfo
import org.json.JSONObject

internal class MusicFoldObserver(private val view: View) : View.OnAttachStateChangeListener {
    private var tracker: WindowInfoTrackerCallbackAdapter? = null
    private var fold: FoldingFeature? = null
    private var last = ""
    private val listener = Consumer<WindowLayoutInfo> { info ->
        fold = info.displayFeatures.filterIsInstance<FoldingFeature>().firstOrNull { it.isSeparating }
        publish()
    }
    private val layout = View.OnLayoutChangeListener { _, _, _, _, _, _, _, _, _ -> publish() }

    init {
        view.addOnAttachStateChangeListener(this)
        view.addOnLayoutChangeListener(layout)
        if (view.isAttachedToWindow) onViewAttachedToWindow(view)
    }

    fun close() {
        onViewDetachedFromWindow(view)
        view.removeOnAttachStateChangeListener(this)
        view.removeOnLayoutChangeListener(layout)
    }

    override fun onViewAttachedToWindow(v: View) {
        val activity = PluginHost.activity ?: return
        val windowTracker = WindowInfoTrackerCallbackAdapter(WindowInfoTracker.getOrCreate(activity))
        tracker = windowTracker
        windowTracker.addWindowLayoutInfoListener(activity, ContextCompat.getMainExecutor(activity), listener)
    }

    override fun onViewDetachedFromWindow(v: View) {
        tracker?.removeWindowLayoutInfoListener(listener)
        tracker = null
        fold = null
        publish()
    }

    private fun publish() {
        val region = JSONObject()
        val event = JSONObject().put("region", region)
        fold?.let { feature ->
            val location = IntArray(2)
            view.getLocationInWindow(location)
            val bounds = Rect(feature.bounds)
            bounds.offset(-location[0], -location[1])
            val density = HostImpl.density.toDouble()
            // A crease may be a line. Keep a little space between the two panes there too.
            val margin = (6 * density).toInt()
            if (bounds.width() == 0) bounds.inset(-margin, 0)
            if (bounds.height() == 0) bounds.inset(0, -margin)
            if (bounds.intersect(0, 0, view.width, view.height)) {
                event.put("width", view.width / density).put("height", view.height / density)
                region.put("x", bounds.left / density).put("y", bounds.top / density)
                    .put("w", bounds.width() / density).put("h", bounds.height() / density)
            }
        }
        val signature = event.toString()
        if (signature == last) return
        last = signature
        PluginHost.emit("musicMedia", "fold", event)
    }
}
