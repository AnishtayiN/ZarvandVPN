package pm.zarvand.vpn

import android.content.Intent
import android.net.VpnService
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val methodCh = "arefdns/tunnel"
    private val eventCh = "arefdns/tunnel_state"
    private val REQ_VPN = 1001
    private var pendingResult: MethodChannel.Result? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, methodCh)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "start" -> {
                        val domains = call.argument<List<String>>("domains") ?: emptyList()
                        val key = call.argument<String>("key") ?: ""
                        val encMethod = call.argument<Int>("encryptionMethod") ?: 1
                        val protocol = call.argument<String>("protocol") ?: "SOCKS5"
                        val listenIp = call.argument<String>("listenIp") ?: "127.0.0.1"
                        val listenPort = call.argument<Int>("listenPort") ?: 18000
                        val localDns = call.argument<Boolean>("localDns") ?: false
                        val localDnsIp = call.argument<String>("localDnsIp") ?: "127.0.0.1"
                        val localDnsPort = call.argument<Int>("localDnsPort") ?: 53
                        val resolvers = call.argument<List<String>>("resolvers") ?: listOf("8.8.8.8")
                        val dup = call.argument<Int>("packetDuplication") ?: 3
                        val upComp = call.argument<Int>("uploadCompression") ?: 0
                        val downComp = call.argument<Int>("downloadCompression") ?: 0
                        val logLevel = call.argument<String>("logLevel") ?: "INFO"
                        val workers = call.argument<Int>("rxTxWorkers") ?: 4

                        val intent = Intent(this, TunnelVpnService::class.java).apply {
                            putExtra("domains", domains.toTypedArray())
                            putExtra("key", key)
                            putExtra("encMethod", encMethod)
                            putExtra("protocol", protocol)
                            putExtra("listenIp", listenIp)
                            putExtra("listenPort", listenPort)
                            putExtra("localDns", localDns)
                            putExtra("localDnsIp", localDnsIp)
                            putExtra("localDnsPort", localDnsPort)
                            putExtra("resolvers", resolvers.toTypedArray())
                            putExtra("dup", dup)
                            putExtra("upComp", upComp)
                            putExtra("downComp", downComp)
                            putExtra("logLevel", logLevel)
                            putExtra("workers", workers)
                        }
                        if (VpnService.prepare(this) != null) {
                            pendingResult = result
                            startActivityForResult(intent, REQ_VPN)
                        } else {
                            TunnelManager.startService(this, intent)
                            result.success(null)
                        }
                    }
                    "stop" -> {
                        TunnelManager.stop(this)
                        result.success(null)
                    }
                    "vpnPermission" -> result.success(VpnService.prepare(this) == null)
                    "requestVpnPermission" -> {
                        val i = VpnService.prepare(this)
                        if (i != null) {
                            pendingResult = result
                            startActivityForResult(i, REQ_VPN)
                        } else result.success(null)
                    }
                    else -> result.notImplemented()
                }
            }
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, eventCh)
            .setStreamHandler(TunnelManager.streamHandler)
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == REQ_VPN) {
            pendingResult?.success(null)
            pendingResult = null
            if (resultCode == RESULT_OK) TunnelManager.emit("permissionGranted")
            else TunnelManager.emit("permissionDenied")
        }
    }
}
