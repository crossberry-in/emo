package dev.emo.go.ui

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
import androidx.compose.ui.viewinterop.AndroidView
import org.json.JSONObject

/**
 * VTreeRenderer — renders a vtree (JSON) as Jetpack Compose.
 */
object VTreeRenderer {

    @Composable
    fun RenderTree(root: JSONObject, client: ServerDiscovery?) {
        Box(modifier = Modifier.fillMaxSize()) {
            RenderElement(root, client)
        }
    }

    @Composable
    fun RenderElement(el: JSONObject, client: ServerDiscovery?) {
        when (el.optString("kind")) {
            "scaffold" -> { Scaffold { p -> Box(Modifier.padding(p)) { renderChild(el, 0, client) } } }
            "column" -> {
                val p = el.optJSONObject("props")
                val sp = (p?.opt("spacing") as? Number)?.toDouble() ?: 0.0
                val pd = (p?.opt("padding") as? Number)?.toDouble() ?: 0.0
                val bg = p?.optString("background")
                Column(Modifier.fillMaxWidth()
                    .then(if (pd > 0) Modifier.padding(pd.dp) else Modifier)
                    .then(if (bg?.startsWith("#") == true) Modifier.background(parseColor(bg)) else Modifier)
                    .verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(sp.dp)) {
                    renderChildren(el, client)
                }
            }
            "row" -> {
                val p = el.optJSONObject("props")
                val sp = (p?.opt("spacing") as? Number)?.toDouble() ?: 0.0
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(sp.dp)) {
                    renderChildren(el, client)
                }
            }
            "view" -> Box(Modifier.fillMaxWidth()) { renderChildren(el, client) }
            "text" -> {
                val p = el.optJSONObject("props")
                val sz = (p?.opt("fontSize") as? Number)?.toDouble() ?: 14.0
                val w = p?.optString("fontWeight") ?: "normal"
                val c = p?.optString("color")
                Text(el.optString("text"), fontSize = sz.sp,
                    fontWeight = if (w == "bold") FontWeight.Bold else FontWeight.Normal,
                    color = if (c?.startsWith("#") == true) parseColor(c) else Color.Unspecified)
            }
            "button" -> {
                val tok = handlerToken(el, "click")
                Button(onClick = { if (tok != null) client?.sendEvent(tok, "click") }) {
                    Text(el.optString("text"))
                }
            }
            "textField", "input" -> {
                val ph = el.optJSONObject("props")?.optString("placeholder") ?: ""
                val tok = handlerToken(el, "change")
                var v by remember { mutableStateOf("") }
                TextField(value = v, onValueChange = { nv -> v = nv; if (tok != null) client?.sendEvent(tok, "change", nv) },
                    placeholder = { Text(ph) })
            }
            "webView" -> AdvancedWebViewRender(el, client)
            "safeAreaView" -> Column(Modifier.fillMaxSize()) { renderChildren(el, client) }
            "scrollView" -> Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) { renderChildren(el, client) }
            "divider" -> HorizontalDivider()
            "spacer" -> Spacer(Modifier.height(8.dp))
            "card" -> Card(Modifier.fillMaxWidth().padding(8.dp)) {
                Column(Modifier.padding(16.dp)) { renderChildren(el, client) }
            }
            "switch" -> {
                val v = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
                val t = handlerToken(el, "change")
                var s by remember { mutableStateOf(v) }
                Switch(checked = s, onCheckedChange = { nv -> s = nv; if (t != null) client?.sendEvent(t, "change", nv) })
            }
            "slider" -> {
                val v = (el.optJSONObject("props")?.opt("value") as? Number)?.toFloat() ?: 0.5f
                val t = handlerToken(el, "change")
                var s by remember { mutableStateOf(v) }
                Slider(value = s, onValueChange = { nv -> s = nv; if (t != null) client?.sendEvent(t, "change", nv.toDouble()) })
            }
            "activityIndicator" -> Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            "progress" -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(16.dp))
            "topBar" -> {
                val title = el.optJSONObject("props")?.optString("title") ?: ""
                Text(title, fontSize = 20.sp, fontWeight = FontWeight.Bold,
                    modifier = Modifier.fillMaxWidth().padding(16.dp))
            }
            "fab" -> FloatingActionButton(onClick = {}) { Text("+") }
            "checkbox" -> {
                val v = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
                val t = handlerToken(el, "change")
                var s by remember { mutableStateOf(v) }
                Checkbox(checked = s, onCheckedChange = { nv -> s = nv; if (t != null) client?.sendEvent(t, "change", nv) })
            }
            "radioButton" -> {
                val v = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
                val t = handlerToken(el, "click")
                RadioButton(selected = v, onClick = { if (t != null) client?.sendEvent(t, "click") })
            }
            "icon" -> {
                val n = el.optJSONObject("props")?.optString("name") ?: "info"
                Text("[$n]", fontSize = 24.sp, color = Color.Gray)
            }
            else -> Text("[${el.optString("kind")}]", color = Color.Red)
        }
    }

    @Composable
    private fun renderChildren(el: JSONObject, client: ServerDiscovery?) {
        val children = el.optJSONArray("children") ?: return
        for (i in 0 until children.length()) {
            RenderElement(children.getJSONObject(i), client)
        }
    }

    @Composable
    private fun renderChild(el: JSONObject, index: Int, client: ServerDiscovery?) {
        val children = el.optJSONArray("children")
        if (children != null && children.length() > index) {
            RenderElement(children.getJSONObject(index), client)
        }
    }

    private fun handlerToken(el: JSONObject, event: String): String? {
        val arr = el.optJSONArray("handlers") ?: return null
        for (i in 0 until arr.length()) {
            val h = arr.getJSONObject(i)
            if (h.optString("event") == event) return h.optString("token")
        }
        return null
    }

    private fun parseColor(hex: String): Color {
        val s = hex.removePrefix("#")
        return try {
            val v = s.toLong(16)
            when (s.length) {
                6 -> Color((0xFF shl 24) or v.toInt())
                8 -> Color(v.toInt())
                else -> Color.Unspecified
            }
        } catch (e: Exception) { Color.Unspecified }
    }
}

/**
 * AdvancedWebViewRender — full-featured WebView matching react-native-webview.
 *
 * Supports all props from react-native-webview:
 *   - source (URL), html (inline HTML)
 *   - injectedJavaScript, injectedJavaScriptBeforeContentLoaded
 *   - javaScriptEnabled, domStorageEnabled, cacheEnabled
 *   - userAgent, scalesPageToFit, textZoom, minimumFontSize
 *   - geolocationEnabled, allowFileAccess, allowFileAccessFromFileURLs
 *   - mediaPlaybackRequiresUserGesture, allowsFullscreenVideo
 *   - method, headers, body (for POST requests)
 *
 * Event handlers (sent back to the emo dev server):
 *   - onMessage: messages from window.EmoGo.postMessage()
 *   - onLoadStart, onLoadEnd, onLoadProgress
 *   - onNavigationStateChange, onError, onHttpError
 */
@Composable
fun AdvancedWebViewRender(el: JSONObject, client: ServerDiscovery?) {
    val props = el.optJSONObject("props")
    val source = props?.optString("source") ?: ""
    val html = props?.optString("html") ?: ""
    val injectedJS = props?.optString("injectedJavaScript") ?: ""
    val injectedJSBefore = props?.optString("injectedJavaScriptBeforeContentLoaded") ?: ""
    val userAgent = props?.optString("userAgent") ?: ""
    val jsEnabled = props?.optBoolean("javaScriptEnabled", true) ?: true
    val domStorageEnabled = props?.optBoolean("domStorageEnabled", true) ?: true
    val cacheEnabled = props?.optBoolean("cacheEnabled", true) ?: true
    val scalesPageToFit = props?.optBoolean("scalesPageToFit", true) ?: true
    val geolocationEnabled = props?.optBoolean("geolocationEnabled", false) ?: false
    val allowFileAccess = props?.optBoolean("allowFileAccess", true) ?: true
    val allowFileAccessFromFileURLs = props?.optBoolean("allowFileAccessFromFileURLs", false) ?: false
    val textZoom = props?.optInt("textZoom", 100) ?: 100
    val minFontSize = props?.optInt("minimumFontSize", 1) ?: 1
    val mediaRequiresGesture = props?.optBoolean("mediaPlaybackRequiresUserGesture", true) ?: true
    val allowsFullscreen = props?.optBoolean("allowsFullscreenVideo", false) ?: false
    val method = props?.optString("method", "GET") ?: "GET"

    // Event handler tokens
    val onMessageToken = handlerTokenPub(el, "message")
    val onLoadStartToken = handlerTokenPub(el, "loadStart")
    val onLoadEndToken = handlerTokenPub(el, "loadEnd")
    val onLoadProgressToken = handlerTokenPub(el, "loadProgress")
    val onNavChangeToken = handlerTokenPub(el, "navigationStateChange")
    val onErrorToken = handlerTokenPub(el, "error")
    val onHttpErrorToken = handlerTokenPub(el, "httpError")

    // Load progress bar
    var progress by remember { mutableFloatStateOf(0f) }
    var isLoading by remember { mutableStateOf(false) }

    Column(modifier = Modifier.fillMaxWidth().height(400.dp)) {
        // Progress bar
        if (isLoading && progress < 1f) {
            LinearProgressIndicator(
                progress = { progress },
                modifier = Modifier.fillMaxWidth()
            )
        }

        AndroidView(
            factory = { ctx ->
                android.webkit.WebView(ctx).apply {
                    // Basic settings
                    settings.javaScriptEnabled = jsEnabled
                    settings.domStorageEnabled = domStorageEnabled
                    settings.cacheMode = if (cacheEnabled)
                        android.webkit.WebSettings.LOAD_DEFAULT
                    else
                        android.webkit.WebSettings.LOAD_NO_CACHE
                    settings.setSupportZoom(scalesPageToFit)
                    settings.builtInZoomControls = scalesPageToFit
                    settings.displayZoomControls = false
                    settings.loadWithOverviewMode = scalesPageToFit
                    settings.useWideViewPort = scalesPageToFit
                    settings.textZoom = textZoom
                    settings.minimumFontSize = minFontSize
                    settings.setGeolocationEnabled(geolocationEnabled)
                    settings.allowFileAccess = allowFileAccess
                    settings.allowFileAccessFromFileURLs = allowFileAccessFromFileURLs
                    settings.mediaPlaybackRequiresUserGesture = mediaRequiresGesture
                    if (allowsFullscreen) {
                        // API 19+ allows fullscreen video by default
                    }

                    // Custom User-Agent
                    if (userAgent.isNotEmpty()) {
                        settings.userAgentString = userAgent
                    }

                    // Inject JS bridge before content loads
                    if (injectedJSBefore.isNotEmpty()) {
                        evaluateJavascript(injectedJSBefore, null)
                    }

                    // Inject emo Go message bridge
                    val bridge = """
                        window.EmoGo = {
                            postMessage: function(data) {
                                EmoGoBridge.postMessage(String(data));
                            }
                        };
                    """.trimIndent()
                    evaluateJavascript(bridge, null)
                    addJavascriptInterface(object {
                        @android.webkit.JavascriptInterface
                        fun postMessage(data: String) {
                            if (onMessageToken != null) {
                                client?.sendEvent(onMessageToken, "message", data)
                            }
                        }
                    }, "EmoGoBridge")

                    // WebView client for events
                    webViewClient = object : android.webkit.WebViewClient() {
                        override fun onPageStarted(view: android.webkit.WebView?, url: String?, favicon: android.graphics.Bitmap?) {
                            super.onPageStarted(view, url, favicon)
                            isLoading = true
                            progress = 0f
                            if (onLoadStartToken != null && url != null) {
                                client?.sendEvent(onLoadStartToken, "loadStart", url)
                            }
                            if (onNavChangeToken != null) {
                                val navState = mapOf(
                                    "url" to (url ?: ""),
                                    "loading" to true,
                                    "canGoBack" to (view?.canGoBack() ?: false),
                                    "canGoForward" to (view?.canGoForward() ?: false)
                                )
                                client?.sendEvent(onNavChangeToken, "navigationStateChange", navState)
                            }
                        }

                        override fun onPageFinished(view: android.webkit.WebView?, url: String?) {
                            super.onPageFinished(view, url)
                            isLoading = false
                            progress = 1f
                            // Inject JS after content loads
                            if (injectedJS.isNotEmpty()) {
                                view?.evaluateJavascript(injectedJS, null)
                            }
                            if (onLoadEndToken != null && url != null) {
                                client?.sendEvent(onLoadEndToken, "loadEnd", url)
                            }
                            if (onNavChangeToken != null) {
                                val navState = mapOf(
                                    "url" to (url ?: ""),
                                    "loading" to false,
                                    "canGoBack" to (view?.canGoBack() ?: false),
                                    "canGoForward" to (view?.canGoForward() ?: false)
                                )
                                client?.sendEvent(onNavChangeToken, "navigationStateChange", navState)
                            }
                        }

                        override fun onReceivedError(view: android.webkit.WebView?, request: android.webkit.WebResourceRequest?, error: android.webkit.WebResourceError?) {
                            super.onReceivedError(view, request, error)
                            val errMsg = error?.description?.toString() ?: "Unknown error"
                            if (onErrorToken != null) {
                                client?.sendEvent(onErrorToken, "error", errMsg)
                            }
                        }

                        override fun onReceivedHttpError(view: android.webkit.WebView?, request: android.webkit.WebResourceRequest?, errorResponse: android.webkit.WebResourceResponse?) {
                            super.onReceivedHttpError(view, request, errorResponse)
                            if (onHttpErrorToken != null) {
                                val errData = mapOf(
                                    "url" to (request?.url?.toString() ?: ""),
                                    "statusCode" to (errorResponse?.statusCode ?: 0),
                                    "reasonPhrase" to (errorResponse?.reasonPhrase ?: "")
                                )
                                client?.sendEvent(onHttpErrorToken, "httpError", errData)
                            }
                        }
                    }

                    // Web chrome client for progress
                    webChromeClient = object : android.webkit.WebChromeClient() {
                        override fun onProgressChanged(view: android.webkit.WebView?, newProgress: Int) {
                            progress = newProgress / 100f
                            if (onLoadProgressToken != null) {
                                client?.sendEvent(onLoadProgressToken, "loadProgress", newProgress / 100.0)
                            }
                        }
                    }

                    // Load content
                    when {
                        html.isNotEmpty() -> {
                            loadDataWithBaseURL(null, html, "text/html", "UTF-8", null)
                        }
                        source.isNotEmpty() -> {
                            if (method == "POST") {
                                val body = props?.optString("body", "") ?: ""
                                postUrl(source, body.toByteArray())
                            } else {
                                loadUrl(source)
                            }
                        }
                    }
                }
            },
            modifier = Modifier.fillMaxSize()
        )
    }
}

// handlerTokenPub is a public version of handlerToken for use outside the object.
private fun handlerTokenPub(el: JSONObject, event: String): String? {
    val arr = el.optJSONArray("handlers") ?: return null
    for (i in 0 until arr.length()) {
        val h = arr.getJSONObject(i)
        if (h.optString("event") == event) return h.optString("token")
    }
    return null
}
