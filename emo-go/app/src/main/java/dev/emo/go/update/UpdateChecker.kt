package dev.emo.go.update

import android.content.Context
import android.content.Intent
import android.net.Uri
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * UpdateChecker — checks GitHub for new emo Go releases.
 *
 * On app launch, fetches the latest release from
 * https://api.github.com/repos/crossberry-in/emo-go/releases/latest
 * and compares the version. If newer, shows an UpdateBanner.
 *
 * Tapping "Update" opens the release page in the browser, where the
 * user can download the new APK.
 */
object UpdateChecker {

    private val _updateState = MutableStateFlow<UpdateState>(UpdateState.Idle)
    val updateState: StateFlow<UpdateState> = _updateState

    private val currentVersion = "0.2.0" // BuildConfig.VERSION_NAME would be better
    private val repo = "crossberry-in/emo-go"

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())

    fun checkForUpdates(context: Context) {
        scope.launch(Dispatchers.IO) {
            try {
                val client = OkHttpClient.Builder()
                    .connectTimeout(5, TimeUnit.SECONDS)
                    .readTimeout(5, TimeUnit.SECONDS)
                    .build()

                val req = Request.Builder()
                    .url("https://api.github.com/repos/$repo/releases/latest")
                    .header("Accept", "application/vnd.github.v3+json")
                    .build()

                client.newCall(req).execute().use { resp ->
                    if (!resp.isSuccessful) return@launch
                    val body = resp.body?.string() ?: return@launch
                    val json = JSONObject(body)
                    val tagName = json.optString("tag_name", "") // e.g. "v0.2.1"
                    val version = tagName.removePrefix("v")
                    val htmlUrl = json.optString("html_url", "")
                    val releaseNotes = json.optString("body", "")

                    if (isNewer(version, currentVersion)) {
                        _updateState.value = UpdateState.UpdateAvailable(
                            version = version,
                            downloadUrl = htmlUrl,
                            releaseNotes = releaseNotes,
                            onUpdate = {
                                // Open the release page in the browser.
                                val intent = Intent(Intent.ACTION_VIEW, Uri.parse(htmlUrl))
                                intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                                context.startActivity(intent)
                            }
                        )
                    }
                }
            } catch (e: Exception) {
                // Silently fail — update check is non-critical.
            }
        }
    }

    /**
     * isNewer — returns true if `remote` is a newer version than `current`.
     * Simple lexicographic comparison on dot-separated numeric parts.
     */
    private fun isNewer(remote: String, current: String): Boolean {
        val r = remote.split(".").map { it.toIntOrNull() ?: 0 }
        val c = current.split(".").map { it.toIntOrNull() ?: 0 }
        for (i in 0 until maxOf(r.size, c.size)) {
            val rv = r.getOrElse(i) { 0 }
            val cv = c.getOrElse(i) { 0 }
            if (rv > cv) return true
            if (rv < cv) return false
        }
        return false
    }
}

/**
 * UpdateState — the state of the update check.
 */
sealed class UpdateState {
    object Idle : UpdateState()
    data class UpdateAvailable(
        val version: String,
        val downloadUrl: String,
        val releaseNotes: String,
        val onUpdate: () -> Unit
    ) : UpdateState()
    object UpToDate : UpdateState()
}
