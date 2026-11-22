package dev.mygo

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.media.MediaMetadata
import android.media.session.MediaSession
import android.media.session.PlaybackState
import android.os.IBinder
import android.util.Base64
import android.view.View
import org.json.JSONObject

class MusicMediaPlugin : MyGoPlugin {
    private var foldWindow: View? = null
    private var foldObserver: MusicFoldObserver? = null

    override fun call(method: String, args: JSONObject, reply: PluginReply) {
        try {
            when (method) {
                "configure" -> Unit
                "update" -> {
                    val view = HostImpl.windows[args.optLong("window")]?.view
                    if (view !== foldWindow) {
                        foldObserver?.close()
                        foldWindow = view
                        foldObserver = view?.let(::MusicFoldObserver)
                    }
                    MusicMedia.update(PluginHost.context, args)
                }
                else -> {
                    reply.failure(PluginFailure("NoMethod", method))
                    return
                }
            }
            reply.success(null)
        } catch (error: Exception) {
            reply.failure(PluginFailure("MediaSession", error.message.orEmpty()))
        }
    }
}

private object MusicMedia {
    const val channel = "musicPlayback"
    const val notificationID = 1
    var info = JSONObject()
        private set
    var service: MusicPlaybackService? = null
    private var session: MediaSession? = null
    private var artwork: Bitmap? = null
    private var starting = false

    val playing: Boolean get() = info.optBoolean("playing")

    private fun command(name: String, position: Double = 0.0) {
        PluginHost.emit("musicMedia", "command", JSONObject().put("name", name).put("position", position))
    }

    fun update(context: Context, args: JSONObject) {
        info = args
        if (args.has("artwork")) {
            val data = Base64.decode(args.optString("artwork"), Base64.DEFAULT)
            artwork = if (data.isEmpty()) null else BitmapFactory.decodeByteArray(data, 0, data.size)
        }
        if (!args.optBoolean("present")) {
            session?.release()
            session = null
            service?.stopSelf()
            manager(context).cancel(notificationID)
            return
        }
        val media = session ?: MediaSession(context, channel).apply {
            setCallback(object : MediaSession.Callback() {
                override fun onPlay() = command("play")
                override fun onPause() = command("pause")
                override fun onSkipToNext() = command("next")
                override fun onSkipToPrevious() = command("previous")
                override fun onSeekTo(pos: Long) = command("seek", pos / 1000.0)
                override fun onStop() = command("pause")
            })
            isActive = true
            session = this
        }
        val duration = (args.optDouble("duration", 0.0) * 1000).toLong()
        val position = (args.optDouble("position", 0.0) * 1000).toLong()
        media.setMetadata(MediaMetadata.Builder()
            .putString(MediaMetadata.METADATA_KEY_TITLE, args.optString("title"))
            .putString(MediaMetadata.METADATA_KEY_ARTIST, args.optString("artist"))
            .putString(MediaMetadata.METADATA_KEY_ALBUM, args.optString("album"))
            .putLong(MediaMetadata.METADATA_KEY_DURATION, duration)
            .putBitmap(MediaMetadata.METADATA_KEY_ALBUM_ART, artwork).build())
        var actions = PlaybackState.ACTION_PLAY or PlaybackState.ACTION_PAUSE or PlaybackState.ACTION_PLAY_PAUSE or
            PlaybackState.ACTION_SKIP_TO_NEXT or PlaybackState.ACTION_SKIP_TO_PREVIOUS
        if (duration > 0) actions = actions or PlaybackState.ACTION_SEEK_TO
        media.setPlaybackState(PlaybackState.Builder().setActions(actions)
            .setState(if (playing) PlaybackState.STATE_PLAYING else PlaybackState.STATE_PAUSED,
                position, if (playing) 1f else 0f).build())
        val activeService = service
        if (activeService != null) {
            activeService.refresh()
        } else if (playing && !starting) {
            starting = true
            try {
                context.startForegroundService(Intent(context, MusicPlaybackService::class.java))
            } catch (error: Exception) {
                starting = false
                throw error
            }
        }
    }

    fun started() { starting = false }

    fun manager(context: Context): NotificationManager = context.getSystemService(NotificationManager::class.java)

    fun notification(context: Context): Notification {
        val manager = manager(context)
        manager.createNotificationChannel(NotificationChannel(channel, "Playback", NotificationManager.IMPORTANCE_LOW))
        val app = context.packageManager.getLaunchIntentForPackage(context.packageName)?.let {
            PendingIntent.getActivity(context, 0, it, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT)
        }
        fun action(icon: Int, label: String, name: String): Notification.Action {
            val intent = Intent(context, MusicPlaybackService::class.java).setAction(name)
            val pending = PendingIntent.getService(context, 0, intent, PendingIntent.FLAG_IMMUTABLE)
            return Notification.Action.Builder(android.graphics.drawable.Icon.createWithResource(context, icon), label, pending).build()
        }
        val style = Notification.MediaStyle().setShowActionsInCompactView(0, 1, 2)
        session?.let { style.setMediaSession(it.sessionToken) }
        return Notification.Builder(context, channel)
            .setSmallIcon(android.R.drawable.ic_media_play)
            .setContentTitle(info.optString("title"))
            .setContentText(info.optString("artist"))
            .setLargeIcon(artwork)
            .setContentIntent(app)
            .setOnlyAlertOnce(true)
            .setStyle(style)
            .addAction(action(android.R.drawable.ic_media_previous, "Previous", "previous"))
            .addAction(if (playing) action(android.R.drawable.ic_media_pause, "Pause", "pause")
                       else action(android.R.drawable.ic_media_play, "Play", "play"))
            .addAction(action(android.R.drawable.ic_media_next, "Next", "next"))
            .build()
    }

    fun transport(name: String) {
        if (session != null) command(name)
    }
}

class MusicPlaybackService : Service() {
    private var foreground = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        MusicMedia.service = this
        MusicMedia.started()
        // A foreground-service start must be acknowledged even if playback paused in transit.
        if (intent?.action == null && !foreground) {
            startForeground(MusicMedia.notificationID, MusicMedia.notification(this))
            foreground = true
        }
        intent?.action?.let(MusicMedia::transport)
        refresh()
        return START_NOT_STICKY
    }

    fun refresh() {
        if (!MusicMedia.info.optBoolean("present")) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return
        }
        val notification = MusicMedia.notification(this)
        if (MusicMedia.playing) {
            startForeground(MusicMedia.notificationID, notification)
            foreground = true
        } else {
            if (foreground) stopForeground(STOP_FOREGROUND_DETACH)
            foreground = false
            MusicMedia.manager(this).notify(MusicMedia.notificationID, notification)
        }
    }

    override fun onDestroy() {
        MusicMedia.service = null
        super.onDestroy()
    }
}
