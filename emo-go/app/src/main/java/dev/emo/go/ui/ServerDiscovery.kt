package dev.emo.go.ui

import android.content.Context
import android.net.wifi.WifiManager
import android.text.format.Formatter
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import okhttp3.*
import okhttp3.WebSocket
import org.json.JSONObject
import java.net.Inet4Address
import java.net.NetworkInterface
import java.util.concurrent.TimeUnit

/**
 * Screen — the app's navigation state.
 */
enum class Screen {
    Splash, Discovering, Connected, ManualConnect
}

/**
 * DiscoveredServer — a dev server found on the LAN.
 */
data class DiscoveredServer(
    val url: String,
    val projectId: String,
    val lanIP: String,
    val port: Int,
    val name: String = "emo project"
)

/**
 * ServerDiscovery — auto-detects emo dev servers on the LAN.
 *
 * Scans all IPs on the local network (e.g., 192.168.1.1-254) on port 7575
 * and checks each for the /manifest endpoint. If the response contains an
 * "emo" project ID, the server is added to the discovered list.
 *
 * Also manages the WebSocket connection to the selected server.
 */
class ServerDiscovery {

    private val _currentScreen = MutableStateFlow(Screen.Splash)
    val currentScreen: StateFlow<Screen> = _currentScreen

    private val _discoveredServers = MutableStateFlow<List<DiscoveredServer>>(emptyList())
    val discoveredServers: StateFlow<List<DiscoveredServer>> = _discoveredServers

    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Idle)
    val connectionState: StateFlow<ConnectionState> = _connectionState

    private val _vtree = MutableStateFlow<JSONObject?>(null)
    val vtree: StateFlow<JSONObject?> = _vtree

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    private var ws: WebSocket? = null
    private var scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private var scanJob: Job? = null

    /**
     * startDiscovery — begins scanning the LAN for dev servers.
     */
    fun startDiscovery() {
        _currentScreen.value = Screen.Discovering
        _discoveredServers.value = emptyList()

        scanJob?.cancel()
        scanJob = scope.launch(Dispatchers.IO) {
            val lanIP = getLanIP()
            if (lanIP == null) {
                _error.value = "Not connected to WiFi"
                return@launch
            }

            // Scan common ports.
            val ports = listOf(7575, 7576, 7577, 8080, 3000)
            val baseIP = lanIP.substringBeforeLast(".")

            // Scan 1-254 in parallel (limited concurrency).
            val jobs = mutableListOf<Deferred<Unit>>()

            for (i in 1..254) {
                val ip = "$baseIP.$i"
                for (port in ports) {
                    jobs.add(async {
                        try {
                            tryServer(ip, port)
                        } catch (e: Exception) {
                            // Not reachable — skip.
                        }
                    })
                }
            }

            jobs.awaitAll()
        }
    }

    /**
     * tryServer — checks if a given IP:port is an emo dev server.
     */
    private suspend fun tryServer(ip: String, port: Int) {
        val url = "http://$ip:$port/manifest"
        try {
            val client = OkHttpClient.Builder()
                .connectTimeout(500, TimeUnit.MILLISECONDS)
                .readTimeout(500, TimeUnit.MILLISECONDS)
                .build()

            val req = Request.Builder().url(url).build()
            client.newCall(req).execute().use { resp ->
                if (resp.isSuccessful) {
                    val body = resp.body?.string() ?: return
                    val json = JSONObject(body)
                    val projectId = json.optString("projectId", "")
                    val lanIP = json.optString("lanIP", ip)
                    if (projectId.isNotEmpty()) {
                        val server = DiscoveredServer(
                            url = "ws://$ip:$port/ws",
                            projectId = projectId,
                            lanIP = lanIP,
                            port = port,
                            name = "emo project ($projectId)"
                        )
                        val current = _discoveredServers.value.toMutableList()
                        if (current.none { it.projectId == projectId }) {
                            current.add(server)
                            _discoveredServers.value = current
                        }
                    }
                }
            }
        } catch (e: Exception) {
            // Not an emo server or not reachable — skip.
        }
    }

    /**
     * connect — opens a WebSocket to a discovered server.
     */
    fun connect(server: DiscoveredServer) {
        _connectionState.value = ConnectionState.Connecting
        _currentScreen.value = Screen.Connected

        val client = OkHttpClient.Builder()
            .pingInterval(15, TimeUnit.SECONDS)
            .build()

        val req = Request.Builder().url(server.url).build()
        client.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                ws = webSocket
                _connectionState.value = ConnectionState.Connected
                _error.value = null

                // Send handshake.
                val hs = JSONObject().apply {
                    put("kind", "handshake")
                    put("ts", System.currentTimeMillis())
                    val payload = JSONObject().apply {
                        put("client", "emo-go-android")
                        put("device", android.os.Build.MODEL)
                        put("android", "API ${android.os.Build.VERSION.SDK_INT}")
                        put("appVer", "0.2.0")
                    }
                    put("payload", payload)
                }
                webSocket.send(hs.toString())
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                _connectionState.value = ConnectionState.Disconnected
                _vtree.value = null
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                _connectionState.value = ConnectionState.Error(t.message ?: "Connection failed")
                _error.value = t.message
                // Auto-reconnect after 3 seconds.
                scope.launch {
                    delay(3000)
                    if (_currentScreen.value == Screen.Connected) {
                        connect(server)
                    }
                }
            }
        })
    }

    private fun handleMessage(raw: String) {
        val msg = try { JSONObject(raw) } catch (e: Exception) { return }
        when (msg.optString("kind")) {
            "hello" -> { /* server greeting */ }
            "vtree" -> {
                val payload = msg.optJSONObject("payload")
                val tree = payload?.opt("root") as? JSONObject
                if (tree != null) {
                    _vtree.value = tree
                    _error.value = null
                }
            }
            "error" -> {
                val payload = msg.optJSONObject("payload")
                _error.value = payload?.optString("message") ?: "Unknown error"
            }
        }
    }

    fun sendEvent(token: String, event: String, value: Any? = null) {
        val ws = this.ws ?: return
        val msg = JSONObject().apply {
            put("kind", "event")
            put("ts", System.currentTimeMillis())
            val payload = JSONObject().apply {
                put("token", token)
                put("event", event)
                if (value != null) put("value", value)
            }
            put("payload", payload)
        }
        ws.send(msg.toString())
    }

    fun showManualConnect() { _currentScreen.value = Screen.ManualConnect }
    fun showDiscovery() { startDiscovery() }

    fun stop() {
        scanJob?.cancel()
        ws?.close(1000, "client disconnect")
        ws = null
        scope.cancel()
    }

    private fun getLanIP(): String? {
        try {
            val interfaces = NetworkInterface.getNetworkInterfaces()
            for (intf in interfaces) {
                if (!intf.isUp || intf.isLoopback) continue
                val addrs = intf.inetAddresses
                while (addrs.hasMoreElements()) {
                    val addr = addrs.nextElement()
                    if (addr is Inet4Address && !addr.isLoopbackAddress) {
                        return addr.hostAddress
                    }
                }
            }
        } catch (e: Exception) {
            // Fall through.
        }
        return null
    }
}

/**
 * ConnectionState — tracks the WebSocket connection status.
 */
sealed class ConnectionState {
    object Idle : ConnectionState()
    object Connecting : ConnectionState()
    object Connected : ConnectionState()
    object Disconnected : ConnectionState()
    data class Error(val message: String) : ConnectionState()
}
