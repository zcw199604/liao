/*
 * 主题文件集中定义 Android 客户端的基础配色与 Material3 外观。
 * 当前补齐深色 / 浅色 / 跟随系统三种主题偏好，并与 DataStore 持久化联动。
 */
package io.github.a7413498.liao.android.app.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable

private val LightColors = lightColorScheme()
private val DarkColors = darkColorScheme()

enum class LiaoThemePreference(val storageValue: String) {
    AUTO("auto"),
    LIGHT("light"),
    DARK("dark");

    companion object {
        fun fromStorage(raw: String?): LiaoThemePreference = when (raw?.trim()?.lowercase()) {
            AUTO.storageValue -> AUTO
            LIGHT.storageValue -> LIGHT
            DARK.storageValue -> DARK
            else -> DARK
        }
    }
}

@Composable
fun LiaoTheme(
    preference: LiaoThemePreference = LiaoThemePreference.DARK,
    content: @Composable () -> Unit,
) {
    val useDarkTheme = when (preference) {
        LiaoThemePreference.AUTO -> isSystemInDarkTheme()
        LiaoThemePreference.LIGHT -> false
        LiaoThemePreference.DARK -> true
    }
    MaterialTheme(
        colorScheme = if (useDarkTheme) DarkColors else LightColors,
        content = content,
    )
}
