package io.github.a7413498.liao.android.feature.settings

import io.github.a7413498.liao.android.app.theme.LiaoThemePreference
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import org.junit.Assert.assertEquals
import org.junit.Test

class SettingsFeatureHelpersTest {
    @Test
    fun `websocket state display text should cover all supported states`() {
        assertEquals("Idle", WebSocketState.Idle.toDisplayText())
        assertEquals("Connecting", WebSocketState.Connecting.toDisplayText())
        assertEquals("Connected", WebSocketState.Connected.toDisplayText())
        assertEquals("Reconnecting", WebSocketState.Reconnecting.toDisplayText())
        assertEquals("Forceout", WebSocketState.Forceout(forbiddenUntilMillis = 123L).toDisplayText())
        assertEquals("Closed", WebSocketState.Closed.toDisplayText())
    }

    @Test
    fun `system config helpers should normalize defaults and ui fallbacks`() {
        assertEquals(defaultSystemConfig(), SystemConfigDtoFixtures.defaults)
        assertEquals("fixed", normalizeImagePortMode(" fixed "))
        assertEquals("probe", normalizeImagePortMode("PROBE"))
        assertEquals("real", normalizeImagePortMode(" real "))
        assertEquals("fixed", normalizeImagePortMode("unknown"))
        assertEquals("probe", " PROBE ".toUiMode())
        assertEquals("2048", 0L.toUiRealMinBytes())
        assertEquals("8192", 8192L.toUiRealMinBytes())
        assertEquals("10", 0.toUiMtPhotoThreshold())
        assertEquals("18", 18.toUiMtPhotoThreshold())
    }

    @Test
    fun `theme display label should align with preference`() {
        assertEquals("跟随系统", LiaoThemePreference.AUTO.toDisplayLabel())
        assertEquals("浅色", LiaoThemePreference.LIGHT.toDisplayLabel())
        assertEquals("深色", LiaoThemePreference.DARK.toDisplayLabel())
    }

    private object SystemConfigDtoFixtures {
        val defaults = io.github.a7413498.liao.android.core.network.SystemConfigDto(
            imagePortMode = "fixed",
            imagePortFixed = "9006",
            imagePortRealMinBytes = 2048,
            mtPhotoTimelineDeferSubfolderThreshold = 10,
        )
    }
}
