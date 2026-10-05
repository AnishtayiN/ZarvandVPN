package pm.zarvand.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import java.io.File

class TunnelVpnService : VpnService() {
    companion object {
        private const val CHANNEL_ID = "zarvand_vpn"
        private const val NOTIF_ID = 1
        private const val TUN_IP = "198.18.0.1"
        private const val TUN_DNS = "198.18.0.2"
    }

    private var tun: ParcelFileDescriptor? = null
    private var coreProcess: Process? = null
    private var tunProcess: Process? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startAsForeground()
        TunnelManager.emit("connecting")

        val domains = intent?.getStringArrayExtra("domains") ?: emptyArray()
        val key = intent?.getStringExtra("key") ?: ""
        val encMethod = intent?.getIntExtra("encMethod", 1) ?: 1
        val protocol = intent?.getStringExtra("protocol") ?: "SOCKS5"
        val listenIp = intent?.getStringExtra("listenIp") ?: "127.0.0.1"
        val listenPort = intent?.getIntExtra("listenPort", 18000) ?: 18000
        val localDns = intent?.getBooleanExtra("localDns", false) ?: false
        val localDnsIp = intent?.getStringExtra("localDnsIp") ?: "127.0.0.1"
        val localDnsPort = intent?.getIntExtra("localDnsPort", 53) ?: 53
        val resolvers = intent?.getStringArrayExtra("resolvers") ?: arrayOf("8.8.8.8")
        val dup = intent?.getIntExtra("dup", 3) ?: 3
        val upComp = intent?.getIntExtra("upComp", 0) ?: 0
        val downComp = intent?.getIntExtra("downComp", 0) ?: 0
        val logLevel = intent?.getStringExtra("logLevel") ?: "INFO"
        val workers = intent?.getIntExtra("workers", 4) ?: 4

        stopAll(emitState = false)

        val ok = startCore(domains, key, encMethod, protocol, listenIp, listenPort,
            localDns, localDnsIp, localDnsPort, resolvers, dup, upComp, downComp, logLevel, workers)
        if (!ok) {
            TunnelManager.emit("disconnected")
            return START_NOT_STICKY
        }
        val tunneled = startTunnel(listenIp, listenPort)
        if (!tunneled) {
            TunnelManager.emit("disconnected")
            return START_NOT_STICKY
        }
        TunnelManager.emit("connected")
        return START_STICKY
    }

    private fun startAsForeground() {
        val nm = getSystemService(NOTIFICATION_SERVICE) as NotificationManager
        if (Build.VERSION.SDK_INT >= 26) {
            nm.createNotificationChannel(
                NotificationChannel(CHANNEL_ID, "ZarvandVPN", NotificationManager.IMPORTANCE_LOW)
            )
        }
        val pi = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )
        val b = if (Build.VERSION.SDK_INT >= 26) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION") Notification.Builder(this)
        }
        b.setContentTitle("ZarvandVPN").setContentText("Tunnel active")
            .setSmallIcon(android.R.drawable.ic_lock_lock).setContentIntent(pi).setOngoing(true)
        startForeground(NOTIF_ID, b.build())
    }

    override fun onRevoke() {
        stopAll()
        super.onRevoke()
    }

    override fun onDestroy() {
        stopAll()
        super.onDestroy()
    }

    private fun stopAll(emitState: Boolean = true) {
        try { tunProcess?.destroy() } catch (_: Exception) {}
        try { coreProcess?.destroy() } catch (_: Exception) {}
        try { tun?.close() } catch (_: Exception) {}
        tun = null
        if (emitState) TunnelManager.emit("disconnected")
    }

    /** Writes full client_config.toml + resolvers file matching user settings. */
    private fun startCore(
        domains: Array<String>, key: String, encMethod: Int, protocol: String,
        listenIp: String, listenPort: Int, localDns: Boolean, localDnsIp: String,
        localDnsPort: Int, resolvers: Array<String>, dup: Int, upComp: Int,
        downComp: Int, logLevel: String, workers: Int
    ): Boolean {
        val nativeDir = applicationInfo.nativeLibraryDir
        val coreBin = File(nativeDir, "libzarvand.so")
        if (!coreBin.exists()) {
            TunnelManager.emit("log:core binary missing")
            return false
        }
        val cfgDir = File(filesDir, "core").apply { mkdirs() }
        val cfg = File(cfgDir, "client_config.toml")
        val resFile = File(cfgDir, "client_resolvers.txt")
        resFile.writeText(resolvers.joinToString("\n") { it.trim() } + "\n")

        val domainsList = domains.joinToString(", ") { "\"$it\"" }
        cfg.writeText(buildString {
            appendLine("# ZarvandVPN - generated config")
            appendLine("DOMAINS = [$domainsList]")
            appendLine("DATA_ENCRYPTION_METHOD = $encMethod")
            appendLine("ENCRYPTION_KEY = \"$key\"")
            appendLine("PROTOCOL_TYPE = \"$protocol\"")
            appendLine("LISTEN_IP = \"$listenIp\"")
            appendLine("LISTEN_PORT = $listenPort")
            appendLine("SOCKS5_AUTH = false")
            appendLine("LOCAL_DNS_ENABLED = $localDns")
            appendLine("LOCAL_DNS_IP = \"$localDnsIp\"")
            appendLine("LOCAL_DNS_PORT = $localDnsPort")
            appendLine("PACKET_DUPLICATION_COUNT = ${dup.coerceIn(1, 12)}")
            appendLine("SETUP_PACKET_DUPLICATION_COUNT = ${(dup + 1).coerceIn(1, 12)}")
            appendLine("UPLOAD_COMPRESSION_TYPE = $upComp")
            appendLine("DOWNLOAD_COMPRESSION_TYPE = $downComp")
            appendLine("RX_TX_WORKERS = ${workers.coerceIn(1, 16)}")
            appendLine("TUNNEL_PROCESS_WORKERS = ${workers.coerceIn(1, 16)}")
            appendLine("LOG_LEVEL = \"$logLevel\"")
        })

        val pb = ProcessBuilder(
            coreBin.absolutePath, "-config", cfg.absolutePath, "-resolvers", resFile.absolutePath
        )
        pb.directory(cfgDir)
        pb.redirectErrorStream(true)
        return try {
            coreProcess = pb.start()
            Thread {
                try {
                    coreProcess?.inputStream?.bufferedReader()?.forEachLine { line ->
                        TunnelManager.emit("log:$line")
                    }
                } catch (_: Exception) {}
            }.start()
            true
        } catch (e: Exception) {
            TunnelManager.emit("log:core start failed: ${e.message}")
            false
        }
    }

    /** Creates TUN and runs tun2socks (hev-socks5-tunnel) pointing at our local SOCKS5. */
    private fun startTunnel(listenIp: String, listenPort: Int): Boolean {
        val nativeDir = applicationInfo.nativeLibraryDir
        val tunBin = File(nativeDir, "libhevtunnel.so")
        if (!tunBin.exists()) {
            TunnelManager.emit("log:tun2socks binary missing")
            return false
        }
        val confDir = File(filesDir, "tun").apply { mkdirs() }
        val conf = File(confDir, "tun.conf")
        conf.writeText("""
tunnel:
  mtu: 8500
  ipv4: $TUN_IP

socks5:
  address: $listenIp
  port: $listenPort
  udp: 'udp'
  misc:
    task-stack-size: 24576
    udp_read_timeout: 5000
    limit-nofile: 65535
""".trimIndent())

        val builder = Builder()
            .setSession("ZarvandVPN")
            .setMtu(8500)
            .addAddress(TUN_IP, 32)
            .addRoute("0.0.0.0", 1)
            .addRoute("128.0.0.0", 1)
            .addDnsServer(TUN_DNS)
            .setBlocking(true)
        try { builder.addDisallowedApplication(packageName) } catch (_: Exception) {}
        tun = builder.establish() ?: run {
            TunnelManager.emit("log:VpnService.establish() failed")
            return false
        }
        val pb = ProcessBuilder(
            tunBin.absolutePath, "-c", conf.absolutePath, "-f", tun!!.detachFd().toString()
        )
        pb.redirectErrorStream(true)
        return try {
            tunProcess = pb.start()
            Thread {
                try {
                    tunProcess?.inputStream?.bufferedReader()?.forEachLine { line ->
                        TunnelManager.emit("log:$line")
                    }
                } catch (_: Exception) {}
            }.start()
            true
        } catch (e: Exception) {
            TunnelManager.emit("log:tun start failed: ${e.message}")
            false
        }
    }
}
