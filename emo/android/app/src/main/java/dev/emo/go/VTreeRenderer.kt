package dev.emo.go

import androidx.compose.foundation.Image
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
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import coil.compose.AsyncImage
import org.json.JSONArray
import org.json.JSONObject

/**
 * RenderedTree renders a vtree (JSON) as Jetpack Compose.
 *
 * The vtree is the JSON-serialised dsl.Element tree emitted by the Go dev
 * server. Each node has:
 *   - id:        stable element ID
 *   - kind:      "scaffold" | "column" | "row" | "text" | "button" | ...
 *   - text:      string for Text/Button
 *   - props:     map of prop name → value (style, source, etc.)
 *   - children:  array of child elements
 *   - handlers:  array of {event, token} references
 *
 * Click handlers call back into EmoClient.sendEvent(token, "click") which
 * forwards the event over the WebSocket to the Go dev server.
 */
@Composable
fun RenderedTree(root: JSONObject, client: EmoClient) {
    Box(modifier = Modifier.fillMaxSize()) {
        RenderElement(root, client)
    }
}

@Composable
fun RenderElement(el: JSONObject, client: EmoClient) {
    val kind = el.optString("kind")
    when (kind) {
        "scaffold" -> ScaffoldRender(el, client)
        "column"   -> ColumnRender(el, client)
        "row"      -> RowRender(el, client)
        "view"     -> BoxRender(el, client)
        "text"     -> TextRender(el)
        "button"   -> ButtonRender(el, client)
        "textField","input" -> TextFieldRender(el, client)
        "image"    -> ImageRender(el)
        "spacer"   -> Spacer(Modifier.height(8.dp))
        "divider"  -> HorizontalDivider()
        // --- New native UI elements (emo 0.1.2) ---
        "webView"  -> WebViewRender(el)
        "safeAreaView" -> SafeAreaViewRender(el, client)
        "scrollView" -> ScrollViewRender(el, client)
        "switch"   -> SwitchRender(el, client)
        "slider"   -> SliderRender(el, client)
        "activityIndicator" -> Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        "picker"   -> PickerRender(el, client)
        "list"     -> ColumnRender(el, client)
        "card"     -> CardRender(el, client)
        "checkbox" -> CheckboxRender(el, client)
        "radioButton" -> RadioButtonRender(el, client)
        "icon"     -> IconRender(el)
        "fab"      -> FloatingActionButton(onClick = {}) { Text("+") }
        "progress" -> LinearProgressIndicator(modifier = Modifier.fillMaxWidth().padding(16.dp))
        "tabBar"   -> TabRowRender(el, client)
        "bottomNav" -> ColumnRender(el, client)
        "topBar"   -> TopBarRender(el)
        else -> Text("[unknown kind: $kind]", color = Color.Red)
    }
}

@Composable
fun ScaffoldRender(el: JSONObject, client: EmoClient) {
    val children = el.optJSONArray("children")
    Scaffold { padding ->
        Box(modifier = Modifier.padding(padding)) {
            if (children != null && children.length() > 0) {
                RenderElement(children.getJSONObject(0), client)
            }
        }
    }
}

@Composable
fun ColumnRender(el: JSONObject, client: EmoClient) {
    val spacing = (el.optJSONObject("props")?.opt("spacing") as? Number)?.toDouble() ?: 0.0
    val padding = (el.optJSONObject("props")?.opt("padding") as? Number)?.toDouble() ?: 0.0
    val bg = el.optJSONObject("props")?.optString("background")

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (padding > 0) Modifier.padding(padding.dp) else Modifier)
            .then(if (bg != null && bg.startsWith("#")) Modifier.background(parseColor(bg)) else Modifier)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(spacing.dp),
    ) {
        EachChild(el) { child -> RenderElement(child, client) }
    }
}

@Composable
fun RowRender(el: JSONObject, client: EmoClient) {
    val spacing = (el.optJSONObject("props")?.opt("spacing") as? Number)?.toDouble() ?: 0.0
    val padding = (el.optJSONObject("props")?.opt("padding") as? Number)?.toDouble() ?: 0.0

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (padding > 0) Modifier.padding(padding.dp) else Modifier),
        horizontalArrangement = Arrangement.spacedBy(spacing.dp),
    ) {
        EachChild(el) { child -> RenderElement(child, client) }
    }
}

@Composable
fun BoxRender(el: JSONObject, client: EmoClient) {
    Box(modifier = Modifier.fillMaxWidth()) {
        EachChild(el) { child -> RenderElement(child, client) }
    }
}

@Composable
fun TextRender(el: JSONObject) {
    val props = el.optJSONObject("props")
    val text = el.optString("text")
    val size = (props?.opt("fontSize") as? Number)?.toDouble() ?: 14.0
    val weight = props?.optString("fontWeight") ?: "normal"
    val color = props?.optString("color")
    Text(
        text = text,
        fontSize = size.sp,
        fontWeight = if (weight == "bold") FontWeight.Bold else FontWeight.Normal,
        color = if (color != null && color.startsWith("#")) parseColor(color) else Color.Unspecified,
    )
}

@Composable
fun ButtonRender(el: JSONObject, client: EmoClient) {
    val label = el.optString("text")
    val clickToken = handlerToken(el, "click")
    Button(
        onClick = {
            if (clickToken != null) client.sendEvent(clickToken, "click")
        },
    ) {
        Text(label)
    }
}

@Composable
fun TextFieldRender(el: JSONObject, client: EmoClient) {
    val placeholder = el.optJSONObject("props")?.optString("placeholder") ?: ""
    val changeToken = handlerToken(el, "change")
    var value by remember { mutableStateOf("") }
    TextField(
        value = value,
        onValueChange = { v ->
            value = v
            if (changeToken != null) client.sendEvent(changeToken, "change", v)
        },
        placeholder = { Text(placeholder) },
    )
}

@Composable
fun ImageRender(el: JSONObject) {
    val src = el.optJSONObject("props")?.optString("source") ?: ""
    AsyncImage(model = src, contentDescription = null, modifier = Modifier.wrapContentSize())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

@Composable
private fun EachChild(el: JSONObject, render: @Composable (JSONObject) -> Unit) {
    val children = el.optJSONArray("children") ?: return
    for (i in 0 until children.length()) {
        render(children.getJSONObject(i))
    }
}

// ---------------------------------------------------------------------------
// New native UI element renderers (emo 0.1.2)
// ---------------------------------------------------------------------------

/**
 * WebView — embeds a web page using Android's WebView via AndroidView.
 * Source URL comes from props.source.
 */
@Composable
fun WebViewRender(el: JSONObject) {
    val src = el.optJSONObject("props")?.optString("source") ?: ""
    AndroidView(
        factory = { ctx ->
            android.webkit.WebView(ctx).apply {
                settings.javaScriptEnabled = true
                loadUrl(src)
            }
        },
        modifier = Modifier.fillMaxWidth().height(300.dp)
    )
}

/**
 * SafeAreaView — adds padding for status bar and navigation bar.
 */
@Composable
fun SafeAreaViewRender(el: JSONObject, client: EmoClient) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .windowInsetsPadding(WindowInsets.statusBars)
            .windowInsetsPadding(WindowInsets.navigationBars)
    ) {
        EachChild(el) { child -> RenderElement(child, client) }
    }
}

/**
 * ScrollView — vertically scrolling container.
 */
@Composable
fun ScrollViewRender(el: JSONObject, client: EmoClient) {
    val spacing = (el.optJSONObject("props")?.opt("spacing") as? Number)?.toDouble() ?: 0.0
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(spacing.dp),
    ) {
        EachChild(el) { child -> RenderElement(child, client) }
    }
}

/**
 * Switch — toggle on/off.
 */
@Composable
fun SwitchRender(el: JSONObject, client: EmoClient) {
    val checked = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
    val changeToken = handlerToken(el, "change")
    var state by remember { mutableStateOf(checked) }
    Switch(
        checked = state,
        onCheckedChange = { v ->
            state = v
            if (changeToken != null) client.sendEvent(changeToken, "change", v)
        }
    )
}

/**
 * Slider — range input 0..1.
 */
@Composable
fun SliderRender(el: JSONObject, client: EmoClient) {
    val value = (el.optJSONObject("props")?.opt("value") as? Number)?.toFloat() ?: 0.5f
    val changeToken = handlerToken(el, "change")
    var state by remember { mutableStateOf(value) }
    Slider(
        value = state,
        onValueChange = { v ->
            state = v
            if (changeToken != null) client.sendEvent(changeToken, "change", v.toDouble())
        }
    )
}

/**
 * Picker — dropdown menu.
 */
@Composable
fun PickerRender(el: JSONObject, client: EmoClient) {
    val optionsArr = el.optJSONObject("props")?.optJSONArray("options")
    val options = mutableListOf<String>()
    if (optionsArr != null) {
        for (i in 0 until optionsArr.length()) {
            options.add(optionsArr.getString(i))
        }
    }
    var expanded by remember { mutableStateOf(false) }
    var selected by remember { mutableStateOf(if (options.isNotEmpty()) options[0] else "") }
    Box {
        OutlinedButton(onClick = { expanded = true }) { Text(selected) }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { opt ->
                DropdownMenuItem(
                    text = { Text(opt) },
                    onClick = {
                        selected = opt
                        expanded = false
                        val changeToken = handlerToken(el, "change")
                        if (changeToken != null) client.sendEvent(changeToken, "change", opt)
                    }
                )
            }
        }
    }
}

/**
 * Card — Material Design card.
 */
@Composable
fun CardRender(el: JSONObject, client: EmoClient) {
    Card(modifier = Modifier.fillMaxWidth().padding(8.dp)) {
        Column(modifier = Modifier.padding(16.dp)) {
            EachChild(el) { child -> RenderElement(child, client) }
        }
    }
}

/**
 * Checkbox — boolean checkbox.
 */
@Composable
fun CheckboxRender(el: JSONObject, client: EmoClient) {
    val checked = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
    val changeToken = handlerToken(el, "change")
    var state by remember { mutableStateOf(checked) }
    Checkbox(
        checked = state,
        onCheckedChange = { v ->
            state = v
            if (changeToken != null) client.sendEvent(changeToken, "change", v)
        }
    )
}

/**
 * RadioButton — radio button.
 */
@Composable
fun RadioButtonRender(el: JSONObject, client: EmoClient) {
    val selected = (el.optJSONObject("props")?.opt("value") as? Boolean) ?: false
    val clickToken = handlerToken(el, "click")
    RadioButton(
        selected = selected,
        onClick = {
            if (clickToken != null) client.sendEvent(clickToken, "click")
        }
    )
}

/**
 * Icon — Material icon (renders name as text placeholder).
 */
@Composable
fun IconRender(el: JSONObject) {
    val name = el.optJSONObject("props")?.optString("name") ?: "info"
    Text("[$name]", fontSize = 24.sp, color = Color.Gray)
}

/**
 * TabRow — horizontal tab bar.
 */
@Composable
fun TabRowRender(el: JSONObject, client: EmoClient) {
    val children = el.optJSONArray("children") ?: return
    val tabCount = children.length()
    if (tabCount == 0) return
    var selected by remember { mutableStateOf(0) }
    Column {
        TabRow(selectedTabIndex = selected) {
            for (i in 0 until tabCount) {
                val child = children.getJSONObject(i)
                val title = child.optString("text", "Tab $i")
                Tab(
                    selected = selected == i,
                    onClick = { selected = i },
                    text = { Text(title) }
                )
            }
        }
        RenderElement(children.getJSONObject(selected), client)
    }
}

/**
 * TopBar — top app bar.
 */
@Composable
fun TopBarRender(el: JSONObject) {
    val title = el.optJSONObject("props")?.optString("title") ?: el.optString("text", "")
    Text(
        title,
        fontSize = 20.sp,
        fontWeight = FontWeight.Bold,
        modifier = Modifier.fillMaxWidth().padding(16.dp)
    )
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
