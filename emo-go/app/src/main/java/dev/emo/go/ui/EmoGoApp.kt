package dev.emo.go.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import dev.emo.go.update.UpdateChecker
import dev.emo.go.update.UpdateState

/**
 * EmoGoApp — the root composable for the emo Go app.
 *
 * Screens:
 *   1. Splash screen (logo + "emo Go" title)
 *   2. Server discovery (auto-detect dev servers on LAN)
 *   3. Carousel of recent servers
 *   4. Connected — render vtree from dev server
 */
@Composable
fun EmoGoApp(
    discovery: ServerDiscovery,
    initialUrl: String?,
    projectId: String
) {
    val screen by discovery.currentScreen.collectAsState()
    val updateState by UpdateChecker.updateState.collectAsState()

    Surface(
        modifier = Modifier.fillMaxSize(),
        color = Color(0xFF1A1A2E)
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            when (screen) {
                Screen.Splash -> SplashScreen(onTimeout = { discovery.startDiscovery() })
                Screen.Discovering -> DiscoveryScreen(discovery)
                Screen.Connected -> ConnectedScreen(discovery)
                Screen.ManualConnect -> ManualConnectScreen(discovery)
                Screen.VirtualPreview -> VirtualPreviewScreen(discovery)
            }

            // Update notification overlay
            if (updateState is UpdateState.UpdateAvailable) {
                UpdateBanner(updateState as UpdateState.UpdateAvailable)
            }

            // Footer: "Built with ♥️ by crossberry"
            Footer()
        }
    }
}

/**
 * SplashScreen — logo + title for 2 seconds.
 */
@Composable
fun SplashScreen(onTimeout: () -> Unit) {
    LaunchedEffect(Unit) {
        kotlinx.coroutines.delay(2000)
        onTimeout()
    }

    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        // emo logo (purple circle with "e")
        Box(
            modifier = Modifier
                .size(120.dp)
                .background(
                    Brush.radialGradient(listOf(Color(0xFF6C47FF), Color(0xFF9B6FFF))),
                    RoundedCornerShape(28.dp)
                ),
            contentAlignment = Alignment.Center
        ) {
            Text("e", fontSize = 64.sp, fontWeight = FontWeight.Bold, color = Color.White)
        }
        Spacer(Modifier.height(24.dp))
        Text("emo Go", fontSize = 32.sp, fontWeight = FontWeight.Bold, color = Color.White)
        Spacer(Modifier.height(8.dp))
        Text(
            "Live preview for emo apps",
            fontSize = 14.sp,
            color = Color(0xFF9999BB)
        )
    }
}

/**
 * Footer — "Built with ♥️ by crossberry" at the bottom.
 */
@Composable
fun Footer() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.BottomCenter
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 16.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                "Built with ",
                fontSize = 12.sp,
                color = Color(0xFF7777AA)
            )
            Text("♥", fontSize = 14.sp, color = Color(0xFFFF4466))
            Text(
                " by ",
                fontSize = 12.sp,
                color = Color(0xFF7777AA)
            )
            Text(
                "crossberry",
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                color = Color(0xFF6C47FF)
            )
        }
    }
}

/**
 * UpdateBanner — shows when a new version is available.
 * Tapping it triggers the update (downloads + installs new APK).
 */
@Composable
fun UpdateBanner(state: UpdateState.UpdateAvailable) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.TopCenter
    ) {
        Card(
            modifier = Modifier
                .padding(16.dp)
                .fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = Color(0xFF6C47FF))
        ) {
            Row(
                modifier = Modifier.padding(16.dp).fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text("Update available", color = Color.White, fontWeight = FontWeight.Bold)
                    Text("v${state.version} — tap to update", color = Color(0xFFCCCCFF), fontSize = 12.sp)
                }
                Button(
                    onClick = { state.onUpdate() },
                    colors = ButtonDefaults.buttonColors(containerColor = Color.White)
                ) {
                    Text("Update", color = Color(0xFF6C47FF), fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}
