package little_reef.internal.lockscreen

import android.Manifest
import android.app.Activity
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.content.res.ColorStateList
import android.content.res.Configuration
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.drawable.Icon
import android.media.MediaMetadata
import android.media.session.MediaSession
import android.media.session.PlaybackState
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.os.PowerManager
import android.os.SystemClock
import android.provider.Settings
import android.widget.RemoteViews
import dev.mygo.*

// The round on Android's lock screen: an ongoing notification, visible on the
// lock screen, whose chronometer counts down to the round's end by itself,
// with a button to pause or resume; posted by a foreground service, which
// keeps the app's Go clock running while the phone is locked.

fun makeTimer(): Timer = AndroidTimer()

// A channel's importance cannot rise once made: the first one was silent
// (IMPORTANCE_LOW), which HyperOS and others hide on the lock screen, so
// the countdown has a channel of its own now and the old one goes.
private const val CHANNEL = "reef-countdown"
private const val OLD_CHANNEL = "reef-timer"
// Android 16 promotes an ongoing notification that asks for it to a Live
// Update: a chip in the status bar, the top of the lock screen, and on
// HyperOS 3 the Super Island. NotificationCompat.setRequestPromotedOngoing
// sets this extra; the module sets it itself, without AndroidX.
private const val EXTRA_REQUEST_PROMOTED = "android.requestPromotedOngoing"
private const val NOTIFICATION = 4110
private const val EXTRA_ACTION = "little_reef.lockscreen.action"
private const val ASK_NOTIFICATIONS = 4111

/** Shared between the timer Go holds, the service and the button receiver. */
internal object TimerState {
    @Volatile var notification: Notification? = null
    @Volatile var listener: ((Action) -> kotlin.Unit)? = null
    @Volatile var serviceRunning = false
}

private class AndroidTimer : Timer {
    private val context: Context get() = Native.context
    private var askedPermission = false

    override fun show(r: Round) {
        val ctx = context
        askNotificationPermission()
        ensureChannel(ctx)
        val n = notification(ctx, r)
        TimerState.notification = n
        if (TimerState.serviceRunning) {
            ctx.getSystemService(NotificationManager::class.java).notify(NOTIFICATION, n)
        } else {
            ctx.startForegroundService(Intent(ctx, TimerService::class.java))
        }
    }

    override fun hide() {
        val ctx = context
        MediaCard.release()
        TimerState.notification = null
        ctx.stopService(Intent(ctx, TimerService::class.java))
        ctx.getSystemService(NotificationManager::class.java).cancel(NOTIFICATION)
    }

    override fun onAction(f: (Action) -> kotlin.Unit): AutoCloseable {
        TimerState.listener = f
        return NativeCloser { if (TimerState.listener === f) TimerState.listener = null }
    }

    override fun backgroundRestricted(): Boolean {
        val ctx = context
        val power = ctx.getSystemService(PowerManager::class.java)
        return !power.isIgnoringBatteryOptimizations(ctx.packageName)
    }

    override fun allowBackground() {
        val ctx = context
        val ask = Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, Uri.parse("package:${ctx.packageName}"))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            ctx.startActivity(ask)
        } catch (e: Exception) {
            // Some systems have no such dialog: open the list instead.
            ctx.startActivity(Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
        }
    }

    override fun notificationsBlocked(): Boolean {
        val ctx = context
        val manager = ctx.getSystemService(NotificationManager::class.java)
        if (!manager.areNotificationsEnabled()) return true
        val channel = manager.getNotificationChannel(CHANNEL) ?: return false
        return channel.importance == NotificationManager.IMPORTANCE_NONE
    }

    override fun openNotificationSettings() {
        val ctx = context
        val intent = Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS)
            .putExtra(Settings.EXTRA_APP_PACKAGE, ctx.packageName)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            ctx.startActivity(intent)
        } catch (e: Exception) {
            ctx.startActivity(Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:${ctx.packageName}")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
        }
    }

    private fun askNotificationPermission() {
        if (askedPermission || Build.VERSION.SDK_INT < 33) return
        val activity = Native.activity ?: return
        if (activity.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return
        askedPermission = true
        activity.runOnUiThread { activity.requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), ASK_NOTIFICATIONS) }
    }
}

private fun ensureChannel(ctx: Context) {
    val manager = ctx.getSystemService(NotificationManager::class.java)
    if (manager.getNotificationChannel(CHANNEL) != null) return
    manager.deleteNotificationChannel(OLD_CHANNEL)
    // Shown on the lock screen, yet quiet: no sound, no vibration, and the
    // notification alerts only once anyway.
    val channel = NotificationChannel(CHANNEL, "倒數計時", NotificationManager.IMPORTANCE_DEFAULT).apply {
        description = "專注與休息的倒數，顯示在鎖定畫面與狀態列上"
        lockscreenVisibility = Notification.VISIBILITY_PUBLIC
        setSound(null, null)
        enableVibration(false)
        setShowBadge(false)
    }
    manager.createNotificationChannel(channel)
}

private fun notification(ctx: Context, r: Round): Notification {
    val open = ctx.packageManager.getLaunchIntentForPackage(ctx.packageName)
        ?.addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP)
    val tap = open?.let { PendingIntent.getActivity(ctx, 0, it, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT) }
    val toggle = if (r.running) Action.ACTION_PAUSE else Action.ACTION_RESUME
    val toggleIntent = PendingIntent.getBroadcast(
        ctx, toggle.ordinal,
        Intent(ctx, ActionReceiver::class.java).putExtra(EXTRA_ACTION, toggle.value),
        PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
    )
    // Ending asks first, in the app: an activity brings it to the front,
    // which a broadcast may not do.
    val endIntent = PendingIntent.getActivity(
        ctx, Action.ACTION_END.ordinal, Intent(ctx, EndActivity::class.java),
        PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
    )
    val b = Notification.Builder(ctx, CHANNEL)
        .setSmallIcon(R.drawable.reef_timer)
        .setContentTitle(r.title)
        .setOngoing(true)
        .setOnlyAlertOnce(true)
        .setVisibility(Notification.VISIBILITY_PUBLIC)
        .setCategory(Notification.CATEGORY_STOPWATCH)
    if (tap != null) b.setContentIntent(tap)
    if (Build.VERSION.SDK_INT >= 31) b.setForegroundServiceBehavior(Notification.FOREGROUND_SERVICE_IMMEDIATE)
    if (r.style == Style.STYLE_MEDIA) {
        // The system's media card: its buttons and progress bar come from
        // the session's playback state, which the system moves on itself.
        val session = MediaCard.update(ctx, r)
        val toggleIcon = Icon.createWithResource(ctx, if (r.running) R.drawable.reef_pause else R.drawable.reef_play)
        b.setStyle(Notification.MediaStyle().setMediaSession(session.sessionToken).setShowActionsInCompactView(0, 1))
            .addAction(Notification.Action.Builder(toggleIcon, if (r.running) r.pause else r.resume, toggleIntent).build())
            .addAction(Notification.Action.Builder(Icon.createWithResource(ctx, R.drawable.reef_stop), r.end, endIntent).build())
            .setContentText(r.text)
            .setShowWhen(false)
        return b.build()
    }
    MediaCard.release()
    if (r.style == Style.STYLE_LARGE) {
        // The app's own layout: large digits and buttons of icons. Android
        // promotes no notification with a custom view to a Live Update.
        b.setStyle(Notification.DecoratedCustomViewStyle())
            .setCustomContentView(largeView(ctx, r, R.layout.reef_countdown, toggleIntent, endIntent))
            .setCustomBigContentView(largeView(ctx, r, R.layout.reef_countdown_big, toggleIntent, endIntent))
            .setShowWhen(false)
        return b.build()
    }
    // Android's own templates draw a button's text alone; the icons go with
    // them for systems, and surfaces such as HyperOS's Super Island, that
    // draw them.
    val toggleIcon = Icon.createWithResource(ctx, if (r.running) R.drawable.reef_pause else R.drawable.reef_play)
    b.addAction(Notification.Action.Builder(toggleIcon, if (r.running) r.pause else r.resume, toggleIntent).build())
        .addAction(Notification.Action.Builder(Icon.createWithResource(ctx, R.drawable.reef_stop), r.end, endIntent).build())
    if (r.running) {
        // The system counts down to the end itself, with the app asleep.
        b.setUsesChronometer(true)
            .setChronometerCountDown(true)
            .setShowWhen(true)
            .setWhen(r.endsAt.toEpochMilli())
            .setContentText(r.text)
    } else {
        b.setShowWhen(false).setContentText("${clock(r.remaining.inWholeSeconds)} · ${r.text}")
    }
    if (Build.VERSION.SDK_INT >= 36) {
        b.addExtras(Bundle().apply { putBoolean(EXTRA_REQUEST_PROMOTED, true) })
        // The status bar chip counts down by itself while running; paused,
        // it says so.
        if (!r.running) b.setShortCriticalText(r.title)
    }
    return b.build()
}

private fun clock(seconds: Long): String {
    val s = seconds.coerceAtLeast(0)
    return String.format("%02d:%02d", s / 60, s % 60)
}

/** largeView fills one of the app's layouts: its chronometer counts down by itself. */
private fun largeView(ctx: Context, r: Round, layout: Int, toggle: PendingIntent, end: PendingIntent): RemoteViews {
    val v = RemoteViews(ctx.packageName, layout)
    v.setTextViewText(R.id.reef_title, r.title)
    v.setTextViewText(R.id.reef_text, r.text)
    val left = if (r.running) r.endsAt.toEpochMilli() - System.currentTimeMillis() else r.remaining.inWholeMilliseconds
    // A count-down chronometer shows base minus now, in the elapsed clock.
    v.setChronometer(R.id.reef_clock, SystemClock.elapsedRealtime() + left.coerceAtLeast(0), null, r.running)
    v.setChronometerCountDown(R.id.reef_clock, true)
    v.setImageViewResource(R.id.reef_toggle, if (r.running) R.drawable.reef_pause else R.drawable.reef_play)
    tintIcons(v)
    v.setContentDescription(R.id.reef_toggle, if (r.running) r.pause else r.resume)
    v.setOnClickPendingIntent(R.id.reef_toggle, toggle)
    v.setContentDescription(R.id.reef_end, r.end)
    v.setOnClickPendingIntent(R.id.reef_end, end)
    return v
}

// The icons' colors: dark on a light notification, white on a dark one. A
// theme attribute in the layout resolves against the app's theme, not the
// shade's, which drew them black on the dark lock screen.
private const val ICON_ON_LIGHT = 0xDE000000.toInt()
private const val ICON_ON_DARK = 0xFFFFFFFF.toInt()

private fun tintIcons(v: RemoteViews) {
    for (id in intArrayOf(R.id.reef_toggle, R.id.reef_end)) {
        if (Build.VERSION.SDK_INT >= 31) {
            // The shade picks the color for its own light or dark mode.
            v.setColorStateList(id, "setImageTintList", ColorStateList.valueOf(ICON_ON_LIGHT), ColorStateList.valueOf(ICON_ON_DARK))
        } else {
            val night = Native.context.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK == Configuration.UI_MODE_NIGHT_YES
            v.setInt(id, "setColorFilter", if (night) ICON_ON_DARK else ICON_ON_LIGHT)
        }
    }
}

/**
 * The round as a media session, for the media card: play and pause are the
 * round's resume and pause, a custom action ends it, and the progress bar
 * runs over the round's length. Headset buttons reach it too.
 *
 * A media control may not bring the app to the front, so the end button
 * asks to confirm itself: pressed, it turns into a check mark that ends the
 * round when pressed within a few seconds, and turns back after them.
 */
internal object MediaCard {
    private const val END = "little_reef.end"
    private const val CONFIRM = "little_reef.confirm_end"
    private const val CONFIRM_MS = 5000L
    private val main = Handler(Looper.getMainLooper())
    private val unconfirm = Runnable { confirming = false; publish() }
    private var session: MediaSession? = null
    private var art: Bitmap? = null
    @Volatile private var round: Round? = null
    @Volatile private var confirming = false

    fun update(ctx: Context, r: Round): MediaSession {
        val s = session ?: MediaSession(ctx, "little-reef-round").also { made ->
            // Go calls in on a thread of its own, which has no looper: the
            // session's callbacks come on the main thread's.
            made.setCallback(object : MediaSession.Callback() {
                override fun onPlay() { TimerState.listener?.invoke(Action.ACTION_RESUME) }
                override fun onPause() { TimerState.listener?.invoke(Action.ACTION_PAUSE) }
                override fun onCustomAction(action: String, extras: Bundle?) {
                    main.removeCallbacks(unconfirm)
                    when (action) {
                        END -> {
                            confirming = true
                            publish()
                            main.postDelayed(unconfirm, CONFIRM_MS)
                        }
                        CONFIRM -> {
                            confirming = false
                            publish()
                            TimerState.listener?.invoke(Action.ACTION_END_NOW)
                        }
                    }
                }
            }, main)
            ctx.packageManager.getLaunchIntentForPackage(ctx.packageName)?.let {
                made.setSessionActivity(PendingIntent.getActivity(ctx, 2, it, PendingIntent.FLAG_IMMUTABLE))
            }
            made.isActive = true
            session = made
        }
        val total = r.total.inWholeMilliseconds.coerceAtLeast(1)
        s.setMetadata(
            MediaMetadata.Builder()
                .putString(MediaMetadata.METADATA_KEY_TITLE, r.title)
                .putString(MediaMetadata.METADATA_KEY_ARTIST, r.text)
                .putLong(MediaMetadata.METADATA_KEY_DURATION, total)
                .putBitmap(MediaMetadata.METADATA_KEY_ART, artwork(ctx))
                .build(),
        )
        round = r
        publish()
        return s
    }

    /** publish sets the playback state: the progress, and the end button as it is now. */
    private fun publish() {
        val s = session ?: return
        val r = round ?: return
        val total = r.total.inWholeMilliseconds.coerceAtLeast(1)
        val left = if (r.running) r.endsAt.toEpochMilli() - System.currentTimeMillis() else r.remaining.inWholeMilliseconds
        val position = (total - left).coerceIn(0, total)
        val end = if (confirming) {
            PlaybackState.CustomAction.Builder(CONFIRM, r.confirmEnd, R.drawable.reef_check).build()
        } else {
            PlaybackState.CustomAction.Builder(END, r.end, R.drawable.reef_stop).build()
        }
        s.setPlaybackState(
            PlaybackState.Builder()
                .setActions(PlaybackState.ACTION_PLAY or PlaybackState.ACTION_PAUSE or PlaybackState.ACTION_PLAY_PAUSE)
                .addCustomAction(end)
                .setState(
                    if (r.running) PlaybackState.STATE_PLAYING else PlaybackState.STATE_PAUSED,
                    position, if (r.running) 1f else 0f, SystemClock.elapsedRealtime(),
                )
                .build(),
        )
    }

    fun release() {
        main.removeCallbacks(unconfirm)
        confirming = false
        round = null
        session?.let {
            it.isActive = false
            it.release()
        }
        session = null
    }

    /** The app's icon, as the card's artwork. */
    private fun artwork(ctx: Context): Bitmap {
        art?.let { return it }
        val icon = ctx.packageManager.getApplicationIcon(ctx.packageName)
        val size = 256
        val bitmap = Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        icon.setBounds(0, 0, size, size)
        icon.draw(canvas)
        art = bitmap
        return bitmap
    }
}

/** Opens the app to end the round: Go asks to confirm there. */
class EndActivity : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        TimerState.listener?.invoke(Action.ACTION_END)
        packageManager.getLaunchIntentForPackage(packageName)
            ?.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP)
            ?.let { startActivity(it) }
        finish()
    }
}

/** Keeps the app running, and its notification shown, while a round is on. */
class TimerService : Service() {
    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val n = TimerState.notification
        if (n == null) {
            stopSelf()
            return START_NOT_STICKY
        }
        if (Build.VERSION.SDK_INT >= 34) {
            startForeground(NOTIFICATION, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
        } else {
            startForeground(NOTIFICATION, n)
        }
        TimerState.serviceRunning = true
        return START_NOT_STICKY
    }

    override fun onDestroy() {
        TimerState.serviceRunning = false
        super.onDestroy()
    }
}

/** Hands the notification's button to Go. */
class ActionReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val value = intent.getStringExtra(EXTRA_ACTION) ?: return
        val action = try { Action.of(value) } catch (e: Exception) { return }
        TimerState.listener?.invoke(action)
    }
}
