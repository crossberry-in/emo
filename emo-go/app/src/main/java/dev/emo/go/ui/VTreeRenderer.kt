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
    fun RenderTree(root: JSONObject, client: ServerDiscovery) {
        Box(modifier = Modifier.fillMaxSize()) {
            RenderElement(root, client)
        }
    }

    @Composable
    fun RenderElement(el: JSONObject, client: ServerDiscovery) {
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
                Button(onClick = { if (tok != null) client.sendEvent(tok, "click") }) {
                    Text(el.optString("text"))
                }
            }
            "textField", "input" -> {
                val ph = el.optJSONObject("props")?.optString("placeholder") ?: ""
                val tok = handlerToken(el, "change")
                var v by remember { mutableStateOf("") }
                TextField(value = v, onValueChange = { nv -> v = nv; if (tok != null) client.sendEvent(tok, "change", nv) },
                    placeholder = { Text(ph) })
            }
            "webView" -> {
                val src = el.optJSONObject("props")?.optString("source") ?: ""
                AndroidView(factory = { ctx ->
                    android.webkit.WebView(ctx).apply {
                        settings.javaScriptEnabled = true
                        loadUrl(src)
                    }
                }, modifier = Modifier.fillMaxWidth().height(300.dp))
            }
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
                Switch(checked = s, onCheckedChange = { nv -> s = nv; if (t != null) client.sendEvent(t, "change", nv) })
            }
            "slider" -> {
                val v = (el.optJSONObject("props")?.opt("value") as? Number)?.toFloat() ?: 0.5f
                val t = handlerToken(el, "change")
                var s by remember { mutableStateOf(v) }
                Slider(value = s, onValueChange = { nv -> s = nv; if (t != null) client.sendEvent(t, "change", nv.toDouble()) })
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
                Checkbox(checked = s, onCheckedChange = { nv -> s = nv; if (t != null) client.sendEvent(t, "change", nv) })
            }
            "radioButton" -> {
                val v = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
                val t = handlerToken(el, "click")
                RadioButton(selected = v, onClick = { if (t != null) client.sendEvent(t, "click") })
            }
            "icon" -> {
                val n = el.optJSONObject("props")?.optString("name") ?: "info"
                Text("[$n]", fontSize = 24.sp, color = Color.Gray)
            }
            else -> Text("[${el.optString("kind")}]", color = Color.Red)
        }
    }

    @Composable
    private fun renderChildren(el: JSONObject, client: ServerDiscovery) {
        val children = el.optJSONArray("children") ?: return
        for (i in 0 until children.length()) {
            RenderElement(children.getJSONObject(i), client)
        }
    }

    @Composable
    private fun renderChild(el: JSONObject, index: Int, client: ServerDiscovery) {
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
