package pm.zarvand.vpn

import android.os.Build
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val methodCh = "zarvand/tunnel"
    private val eventCh = "zarvand/tunnel_state"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, methodCh)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "start" -> {
                        val domains = call.argument<List<String>>("domains") ?: emptyList()
                        val key = call.argument<String>("key") ?: ""
                        val server = call.argument<String>("server") ?: ""
                        TunnelManager.start(this, domains, key, server)
                        result.success(null)
                    }
                    "stop" -> {
                        TunnelManager.stop(this)
                        result.success(null)
                    }
                    else -> result.notImplemented()
                }
            }
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, eventCh)
            .setStreamHandler(TunnelManager.streamHandler)
    }
}
