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
        var instance: TunnelVpnService? = null
    }

    private var tun: ParcelFileDescriptor? = null
    private var coreProcess: Process? = null
    private var tunProcess: Process? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        instance = this
        startAsForeground()

        val domains = intent?.getStringArrayExtra("domains") ?: emptyArray()
        val key = intent?.getStringExtra("key") ?: ""
        val server = intent?.getStringExtra("server") ?: ""

        TunnelManager.emit("connecting")
        startCore(domains, key, server)
        startTunnel()
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
        instance = null
        super.onDestroy()
    }

    private fun stopAll() {
        tunProcess?.destroy()
        coreProcess?.destroy()
        try { tun?.close() } catch (_: Exception) {}
        tun = null
        TunnelManager.emit("disconnected")
    }

    private fun startCore(domains: Array<String>, key: String, server: String) {
        val nativeDir = applicationInfo.nativeLibraryDir
        val cfgDir = File(filesDir, "core").apply { mkdirs() }
        val cfg = File(cfgDir, "client_config.toml")
        val domainsList = domains.joinToString(", ") { "\"$it\"" }
        val serverLine = if (server.isNotBlank()) "\nSERVERS = [\"$server\"]" else ""
        cfg.writeText(
            """
DOMAINS = [$domainsList]
DATA_ENCRYPTION_METHOD = 1
ENCRYPTION_KEY = "$key"
PROTOCOL_TYPE = "SOCKS5"
LISTEN_IP = "127.0.0.1"
LISTEN_PORT = 18000
$serverLine
""".trimIndent()
        )
        val pb = ProcessBuilder(File(nativeDir, "libzarvand.so").absolutePath, "-config", cfg.absolutePath)
        pb.redirectErrorStream(true)
        coreProcess = pb.start()
        Thread {
            coreProcess?.inputStream?.bufferedReader()?.forEachLine { line ->
                TunnelManager.emit("log")
            }
        }.start()
    }

    private fun startTunnel() {
        val nativeDir = applicationInfo.nativeLibraryDir
        val confDir = File(filesDir, "tun").apply { mkdirs() }
        val conf = File(confDir, "tun.conf")
        conf.writeText(
            """
tunnel:
  mtu: 8500
  ipv4: 198.18.0.1

socks5:
  address: 127.0.0.1
  port: 18000
  udp: 'udp'
  misc:
    udp_read_timeout: 5000
""".trimIndent()
        )
        val builder = Builder()
            .setSession("ZarvandVPN")
            .setMtu(8500)
            .addAddress("198.18.0.1", 32)
            .addRoute("0.0.0.0", 1)
            .addRoute("128.0.0.0", 1)
            .addDnsServer("198.18.0.2")
            .setBlocking(true)
        // bypass our own app traffic
        try { builder.addDisallowedApplication(packageName) } catch (_: Exception) {}
        tun = builder.establish()

        val pb = ProcessBuilder(
            File(nativeDir, "libhevtunnel.so").absolutePath, "-c", conf.absolutePath, "-f", tun!!.detachFd().toString()
        )
        pb.redirectErrorStream(true)
        tunProcess = pb.start()
    }
}
