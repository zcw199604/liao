package io.github.a7413498.liao.android.core.network

import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Test

class NetworkStackUrlTest {
    @Test
    fun `normalize api base url should append api suffix consistently`() {
        val expected = "http://127.0.0.1:8080/api/"

        assertEquals(expected, normalizeApiBaseUrl("http://127.0.0.1:8080").toString())
        assertEquals(expected, normalizeApiBaseUrl("http://127.0.0.1:8080/").toString())
        assertEquals(expected, normalizeApiBaseUrl("http://127.0.0.1:8080/api").toString())
        assertEquals(expected, normalizeApiBaseUrl("http://127.0.0.1:8080/api/").toString())
    }

    @Test
    fun `normalize api base url should preserve nested path`() {
        assertEquals(
            "https://example.com/root/api/",
            normalizeApiBaseUrl("https://example.com/root").toString(),
        )
    }

    @Test
    fun `build web socket url should switch scheme and trim api suffix`() {
        val wsUrl = buildWebSocketUrl(
            api = "http://127.0.0.1:8080/api/".toHttpUrl(),
            token = "abc 123",
        )

        assertEquals("ws://127.0.0.1:8080/ws?token=abc%20123", wsUrl)
    }

    @Test
    fun `build web socket url should preserve nested path on https`() {
        val wsUrl = buildWebSocketUrl(
            api = "https://example.com/root/api/".toHttpUrl(),
            token = "token",
        )

        assertEquals("wss://example.com/root/ws?token=token", wsUrl)
    }

    @Test
    fun `resolve dynamic api url should keep query and append relative segments`() {
        val resolved = resolveDynamicApiUrl(
            dynamicBase = "http://10.0.2.2:8080/api/".toHttpUrl(),
            originalUrl = "http://placeholder/api/douyin/detail/fetch?foo=bar&x=1".toHttpUrl(),
        )

        assertEquals("http://10.0.2.2:8080/api/douyin/detail/fetch?foo=bar&x=1", resolved.toString())
    }
}
