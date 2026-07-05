package dev.emo.go

import android.util.Log
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import okhttp3.*
import org.json.JSONArray
import org.json.JSONObject

/**
 * EmoClient manages the WebSocket connection to the emo dev server and the
 * current render state.
 *
 * State machine:
 *   idle → connecting → connected (tree visible) → ...
 *   any → error
 *
 * The server pushes vtree messages; this client stores the latest tree in
 * [state] so Compose can re-render. Event handlers from the rendered UI call
 * back into [sendEvent] to forward clicks/changes to the Go dev server.
 */
class EmoClient {

    data class State(
        val connecting: Boolean = false,
        val connected: Boolean = false,
        val tree: JSONObject? = null,
        val error: String? = null,
    )

    private val _state = MutableStateFlow(State())
    val state: StateFlow<State> = _state

    private var ws: WebSocket? = null
    private var serverUrl: String? = null
    private var projectId: String = "unknown"
    private var retryMs: Long = 1000

    fun connect(url: String, projectId: String) {
        if (_state.value.connected) return
        this.serverUrl = url
        this.projectId = projectId
        _state.value = _state.value.copy(connecting = true, error = null)

        val client = OkHttpClient.Builder()
            .pingInterval(java.time.Duration.ofSeconds(15))
            .build()

        val req = Request.Builder().url(url).build()
        client.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Log.i(TAG, "connected to $url")
                ws = webSocket
                retryMs = 1000
                _state.value = _state.value.copy(connecting = false, connected = true, error = null)

                // Send handshake.
                val hs = JSONObject().apply {
                    put("kind", "handshake")
                    put("ts", System.currentTimeMillis())
                    val payload = JSONObject().apply {
                        put("client", "emo-go-android")
                        put("device", android.os.Build.MODEL)
                        put("android", "API ${android.os.Build.VERSION.SDK_INT}")
                        put("appVer", "0.1.0")
                    }
                    put("payload", payload)
                }
                webSocket.send(hs.toString())
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Log.w(TAG, "ws closed: $code $reason")
                _state.value = _state.value.copy(connected = false, tree = null)
                scheduleReconnect()
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "ws failure", t)
                _state.value = _state.value.copy(connecting = false, connected = false, error = t.message)
                scheduleReconnect()
            }
        })
    }

    fun disconnect() {
        ws?.close(1000, "client disconnect")
        ws = null
        _state.value = State()
    }

    private fun scheduleReconnect() {
        val url = serverUrl ?: return
        val delay = retryMs
        retryMs = (retryMs * 2).coerceAtMost(30_000)
        Thread {
            Thread.sleep(delay)
            if (!_state.value.connected) {
                Log.i(TAG, "reconnecting to $url…")
                connect(url, projectId)
            }
        }.start()
    }

    private fun handleMessage(raw: String) {
        val msg = try {
            JSONObject(raw)
        } catch (e: Exception) {
            Log.e(TAG, "bad message: $raw", e); return
        }
        val kind = msg.optString("kind")
        when (kind) {
            "hello" -> {
                Log.i(TAG, "server hello: ${msg.optJSONObject("payload")}")
            }
            "vtree" -> {
                val payload = msg.optJSONObject("payload")
                val tree = payload?.opt("root") as? JSONObject
                if (tree != null) {
                    _state.value = _state.value.copy(tree = tree, error = null)
                    Log.i(TAG, "vtree received (${payload.optString("reason")})")
                }
            }
            "patch" -> {
                // Apply ops to current tree. For simplicity we replace the
                // whole tree with the server's current snapshot encoded in
                // payload.hash — the server always follows a patch with a
                // fresh vtree for now.
                Log.i(TAG, "patch received")
            }
            "toast" -> {
                val payload = msg.optJSONObject("payload")
                Log.i(TAG, "toast: ${payload?.optString("text")}")
            }
            "log" -> {
                val payload = msg.optJSONObject("payload")
                Log.i("emo:log", "[${payload?.optString("level")}] ${payload?.optString("msg")}")
            }
            "error" -> {
                val payload = msg.optJSONObject("payload")
                _state.value = _state.value.copy(error = payload?.optString("message") ?: "unknown error")
            }
            "reload" -> {
                Log.i(TAG, "full reload requested")
                // We could restart the activity here; for now we just clear
                // the tree and let the next vtree message repaint.
                _state.value = _state.value.copy(tree = null)
            }
            "plugin" -> {
                // Result of a plugin invocation we requested. Forward to
                // whoever is waiting (handled by EmoPluginBridge in a real app).
                Log.i(TAG, "plugin result: $msg")
            }
            else -> Log.w(TAG, "unknown message kind: $kind")
        }
    }

    /**
     * Send a user event (button click, text change, ...) back to the Go dev
     * server. The server dispatches it to the registered handler via its
     * opaque token.
     */
    fun sendEvent(token: String, event: String, value: Any? = null, elementId: String? = null) {
        val ws = this.ws ?: return
        val msg = JSONObject().apply {
            put("kind", "event")
            put("ts", System.currentTimeMillis())
            val payload = JSONObject().apply {
                put("token", token)
                put("event", event)
                if (value != null) put("value", value)
                if (elementId != null) put("element", elementId)
            }
            put("payload", payload)
        }
        ws.send(msg.toString())
    }

    companion object {
        private const val TAG = "emo:client"
    }
}
