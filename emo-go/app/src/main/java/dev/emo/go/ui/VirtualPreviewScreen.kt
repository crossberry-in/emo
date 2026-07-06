package dev.emo.go.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * VirtualPreviewScreen — renders the live vtree inside a phone-frame mockup.
 *
 * This is emo Go's built-in virtual device: a phone-shaped frame that shows
 * exactly what your app looks like on a real device, without needing a real
 * device or emulator. The vtree updates live as you edit .em files.
 *
 * Features:
 *   - Phone frame with rounded corners and status bar
 *   - Live vtree rendering inside the frame
 *   - Back button to return to discovery
 *   - "Built with ♥️ by crossberry" footer
 */
@Composable
fun VirtualPreviewScreen(discovery: ServerDiscovery) {
    val vtree by discovery.vtree.collectAsState()
    val error by discovery.error.collectAsState()

    Column(
        modifier = Modifier.fillMaxSize().background(Color(0xFF1A1A2E)),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Top bar with back button
        Row(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextButton(onClick = { discovery.showDiscovery() }) {
                Text("← Back", color = Color(0xFF9B6FFF))
            }
            Spacer(Modifier.weight(1f))
            Text(
                "Virtual Preview",
                color = Color.White,
                fontWeight = FontWeight.Bold,
                fontSize = 16.sp
            )
            Spacer(Modifier.weight(1f))
            Spacer(Modifier.width(48.dp))
        }

        // Phone frame
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.Center
        ) {
            PhoneFrame(vtree = vtree, error = error)
        }
    }
}

/**
 * PhoneFrame — a phone-shaped mockup that contains the live vtree.
 *
 * Renders a 360x640 dp phone frame with:
 *   - Rounded corners and dark border (like a real phone)
 *   - Status bar at top (time, battery)
 *   - Content area that renders the vtree
 *   - Home indicator at bottom
 */
@Composable
fun PhoneFrame(vtree: org.json.JSONObject?, error: String?) {
    Column(
        modifier = Modifier
            .width(360.dp)
            .height(640.dp)
            .clip(RoundedCornerShape(32.dp))
            .background(Color.Black)
            .border(4.dp, Color(0xFF333344), RoundedCornerShape(32.dp))
    ) {
        // Status bar
        Row(
            modifier = Modifier.fillMaxWidth().background(Color(0xFF1A1A2E)).padding(horizontal = 16.dp, vertical = 6.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text("9:41", color = Color.White, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("📶", fontSize = 10.sp)
                Spacer(Modifier.width(4.dp))
                Text("🔋", fontSize = 10.sp)
            }
        }

        // Content area
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.White)
        ) {
            when {
                error != null -> {
                    Column(
                        modifier = Modifier.fillMaxSize().padding(24.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.Center
                    ) {
                        Text("⚠️", fontSize = 48.sp)
                        Spacer(Modifier.height(16.dp))
                        Text("Connection Error", fontWeight = FontWeight.Bold, color = Color(0xFFB00020))
                        Spacer(Modifier.height(8.dp))
                        Text(error, color = Color(0xFFB00020), fontSize = 12.sp)
                    }
                }
                vtree != null -> {
                    // Render the live vtree inside the phone frame
                    Box(modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                        VTreeRenderer.RenderTree(vtree!!, null)
                    }
                }
                else -> {
                    Column(
                        modifier = Modifier.fillMaxSize(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.Center
                    ) {
                        CircularProgressIndicator(color = Color(0xFF6C47FF), strokeWidth = 2.dp)
                        Spacer(Modifier.height(12.dp))
                        Text("Waiting for vtree…", color = Color(0xFF9999BB), fontSize = 12.sp)
                    }
                }
            }
        }
    }
}
