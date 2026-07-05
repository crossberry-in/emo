package dev.emo.go

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import android.util.Log
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import okhttp3.*
import okhttp3.ws.WebSocket
import org.json.JSONObject
import org.json.JSONArray
import java.lang.ref.WeakReference

/**
 * emo Go — the Expo Go equivalent for the emo framework.
 *
 * This Activity:
 *   1. Reads the dev server URL from Intent extras (emo_server) or shows a
 *      manual connect screen.
 *   2. Opens a WebSocket to the dev server.
 *   3. Receives vtree messages and renders them as native Jetpack Compose UI.
 *   4. Sends user events (click, change, ...) back to the dev server, which
 *      dispatches them to the Go handlers.
 *
 * The app is intentionally tiny: it knows nothing about your specific project.
 * It's a generic vtree renderer, exactly like Expo Go is a generic JS bundle
 * runner.
 */
class MainActivity : Activity() {

    private val client = EmoClient()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Read dev server URL from the launch intent (set by `emo go` via adb
        // am start --es emo_server ws://...). If absent, let the user type it.
        val serverUrl = intent?.getStringExtra("emo_server")
        val projectId = intent?.getStringExtra("emo_project") ?: "unknown"

        setContent {
            MaterialTheme {
                EmoRootScreen(client, serverUrl, projectId)
            }
        }

        // Auto-connect if URL was provided.
        if (serverUrl != null) {
            client.connect(serverUrl, projectId)
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        client.disconnect()
    }
}

/**
 * Top-level screen: shows either a connect form or the rendered vtree.
 */
@Composable
fun EmoRootScreen(client: EmoClient, initialUrl: String?, projectId: String) {
    var url by remember { mutableStateOf(initialUrl ?: "") }
    val state by client.state.collectAsState()

    Surface(modifier = Modifier.fillMaxSize()) {
        when {
            state.error != null -> ErrorOverlay(state.error!!)
            state.tree != null -> RenderedTree(state.tree!!, client)
            state.connecting -> ConnectingScreen()
            else -> ConnectForm(url = url, onUrlChange = { url = it }, onConnect = {
                client.connect(url, projectId)
            })
        }
    }
}

/**
 * Connect form shown when no URL was passed via Intent extras.
 */
@Composable
fun ConnectForm(url: String, onUrlChange: (String) -> Unit, onConnect: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("emo Go", fontSize = 32.sp, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(8.dp))
        Text("Connect to your emo dev server", color = Color.Gray)
        Spacer(Modifier.height(32.dp))
        OutlinedTextField(
            value = url,
            onValueChange = onUrlChange,
            label = { Text("Dev server URL") },
            placeholder = { Text("ws://192.168.1.10:7575/ws") },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(16.dp))
        Button(onClick = onConnect, modifier = Modifier.fillMaxWidth()) {
            Text("Connect")
        }
    }
}

@Composable
fun ConnectingScreen() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            CircularProgressIndicator()
            Spacer(Modifier.height(16.dp))
            Text("Connecting to dev server…")
        }
    }
}

@Composable
fun ErrorOverlay(message: String) {
    Box(modifier = Modifier.fillMaxSize().background(Color(0xFFFFE0E0)), contentAlignment = Alignment.Center) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.padding(32.dp),
        ) {
            Text("⚠️ emo error", fontSize = 24.sp, fontWeight = FontWeight.Bold, color = Color(0xFFB00020))
            Spacer(Modifier.height(16.dp))
            Text(message, color = Color(0xFFB00020))
        }
    }
}
