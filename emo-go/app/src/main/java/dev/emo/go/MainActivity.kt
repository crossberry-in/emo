package dev.emo.go

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.*
import dev.emo.go.ui.EmoGoApp
import dev.emo.go.ui.ServerDiscovery
import dev.emo.go.update.UpdateChecker

/**
 * emo Go — the Expo Go equivalent for the emo framework.
 *
 * Features:
 *   - Logo + title splash screen
 *   - Auto-detect dev servers on LAN (port 7575)
 *   - Carousel of recently connected projects
 *   - In-app update notifications
 *   - VTree rendering as Jetpack Compose
 *   - Built with ♥️ by crossberry
 */
class MainActivity : ComponentActivity() {

    private val discovery = ServerDiscovery()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Check for app updates in background.
        UpdateChecker.checkForUpdates(this)

        setContent {
            EmoGoApp(
                discovery = discovery,
                initialUrl = intent?.getStringExtra("emo_server"),
                projectId = intent?.getStringExtra("emo_project") ?: "unknown"
            )
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        discovery.stop()
    }
}
