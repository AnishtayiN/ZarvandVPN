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
        eventSink?.success(state)
    }

    fun start(activity: Activity, domains: List<String>, key: String, server: String) {
        val intent = Intent(activity, TunnelVpnService::class.java)
        intent.putExtra("domains", domains.toTypedArray())
        intent.putExtra("key", key)
        intent.putExtra("server", server)
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
