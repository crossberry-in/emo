package dev.emo.go.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * DiscoveryScreen — shows the auto-detect animation while scanning,
 * then displays a carousel of discovered dev servers.
 */
@Composable
fun DiscoveryScreen(discovery: ServerDiscovery) {
    val servers by discovery.discoveredServers.collectAsState()
    val isScanning = servers.isEmpty()

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp)
    ) {
        Spacer(Modifier.height(40.dp))

        Text(
            if (isScanning) "Searching for dev servers…" else "Found ${servers.size} server(s)",
            fontSize = 24.sp,
            fontWeight = FontWeight.Bold,
            color = Color.White
        )
        Spacer(Modifier.height(8.dp))
        Text(
            if (isScanning) "Make sure 'emo start' is running on your computer"
            else "Tap a project to connect",
            fontSize = 14.sp,
            color = Color(0xFF9999BB)
        )

        Spacer(Modifier.height(32.dp))

        if (isScanning) {
            // Scanning animation — pulsing dots.
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.Center
            ) {
                CircularProgressIndicator(color = Color(0xFF6C47FF))
            }
        } else {
            // Carousel of discovered servers.
            LazyRow(
                horizontalArrangement = Arrangement.spacedBy(16.dp),
                contentPadding = PaddingValues(horizontal = 4.dp)
            ) {
                items(servers) { server ->
                    ServerCard(server) { discovery.connect(server) }
                }
            }
        }

        Spacer(Modifier.weight(1f))

        // Manual connect button.
        OutlinedButton(
            onClick = { discovery.showManualConnect() },
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.White)
        ) {
            Text("Enter URL manually")
        }

        Spacer(Modifier.height(8.dp))

        // Virtual Preview button — opens the phone-frame mockup.
        OutlinedButton(
            onClick = { discovery.showVirtualPreview() },
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color(0xFF9B6FFF))
        ) {
            Text("📱 Virtual Preview")
        }
    }
}

/**
 * ServerCard — a carousel card for a discovered dev server.
 */
@Composable
fun ServerCard(server: DiscoveredServer, onClick: () -> Unit) {
    Card(
        modifier = Modifier
            .width(280.dp)
            .height(180.dp)
            .clickable { onClick() },
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = Color(0xFF252540))
    ) {
        Column(
            modifier = Modifier.padding(20.dp).fillMaxSize(),
            verticalArrangement = Arrangement.SpaceBetween
        ) {
            // Project icon (purple circle with "e")
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .background(
                        Brush.radialGradient(listOf(Color(0xFF6C47FF), Color(0xFF9B6FFF))),
                        RoundedCornerShape(12.dp)
                    ),
                contentAlignment = Alignment.Center
            ) {
                Text("e", fontSize = 24.sp, fontWeight = FontWeight.Bold, color = Color.White)
            }

            Column {
                Text(
                    server.name,
                    fontSize = 16.sp,
                    fontWeight = FontWeight.Bold,
                    color = Color.White
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    "${server.lanIP}:${server.port}",
                    fontSize = 12.sp,
                    color = Color(0xFF7777AA)
                )
                Text(
                    "Project: ${server.projectId}",
                    fontSize = 11.sp,
                    color = Color(0xFF666688)
                )
            }
        }
    }
}

/**
 * ManualConnectScreen — enter a dev server URL manually.
 */
@Composable
fun ManualConnectScreen(discovery: ServerDiscovery) {
    var url by remember { mutableStateOf("ws://192.168.1.10:7575/ws") }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center
    ) {
        Text("Connect manually", fontSize = 24.sp, fontWeight = FontWeight.Bold, color = Color.White)
        Spacer(Modifier.height(24.dp))

        OutlinedTextField(
            value = url,
            onValueChange = { url = it },
            label = { Text("Dev server URL") },
            modifier = Modifier.fillMaxWidth(),
            colors = OutlinedTextFieldDefaults.colors(
                focusedTextColor = Color.White,
                unfocusedTextColor = Color.White,
                focusedBorderColor = Color(0xFF6C47FF),
                unfocusedBorderColor = Color(0xFF444466)
            )
        )

        Spacer(Modifier.height(16.dp))

        Button(
            onClick = {
                discovery.connect(DiscoveredServer(url, "manual", "", 7575, "Manual connection"))
            },
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF6C47FF))
        ) {
            Text("Connect", color = Color.White)
        }

        Spacer(Modifier.height(16.dp))

        TextButton(onClick = { discovery.showDiscovery() }) {
            Text("Back to auto-detect", color = Color(0xFF9B6FFF))
        }
    }
}

/**
 * ConnectedScreen — renders the vtree from the dev server.
 */
@Composable
fun ConnectedScreen(discovery: ServerDiscovery) {
    val vtree by discovery.vtree.collectAsState()
    val error by discovery.error.collectAsState()

    Surface(modifier = Modifier.fillMaxSize(), color = Color.White) {
        when {
            error != null -> {
                Column(
                    modifier = Modifier.fillMaxSize().padding(32.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center
                ) {
                    Text("Connection error", fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Color(0xFFB00020))
                    Spacer(Modifier.height(8.dp))
                    Text(error!!, color = Color(0xFFB00020))
                    Spacer(Modifier.height(24.dp))
                    Button(onClick = { discovery.showDiscovery() }) {
                        Text("Back to discovery")
                    }
                }
            }
            vtree != null -> {
                VTreeRenderer.RenderTree(vtree!!, discovery)
            }
            else -> {
                Column(
                    modifier = Modifier.fillMaxSize(),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center
                ) {
                    CircularProgressIndicator(color = Color(0xFF6C47FF))
                    Spacer(Modifier.height(16.dp))
                    Text("Connecting…", color = Color(0xFF666688))
                }
            }
        }
    }
}
