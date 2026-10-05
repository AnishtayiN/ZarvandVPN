package pm.zarvand.vpn

import android.app.Activity
import android.content.Intent
import io.flutter.plugin.common.EventChannel

object TunnelManager {
    private var eventSink: EventChannel.EventSink? = null
    val streamHandler = object : EventChannel.StreamHandler {
        override fun onListen(args: Any?, sink: EventChannel.EventSink?) {
            eventSink = sink
        }
        override fun onCancel(args: Any?) {
            eventSink = null
        }
    }

    fun emit(state: String) {
        android.os.Handler(android.os.Looper.getMainLooper()).post {
            eventSink?.success(state)
        }
    }

    fun startService(activity: Activity, intent: Intent) {
        if (android.os.Build.VERSION.SDK_INT >= 29) {
            activity.startForegroundService(intent)
        } else {
            activity.startService(intent)
        }
    }

    fun stop(activity: Activity) {
        activity.stopService(Intent(activity, TunnelVpnService::class.java))
        emit("disconnected")
    }
}
