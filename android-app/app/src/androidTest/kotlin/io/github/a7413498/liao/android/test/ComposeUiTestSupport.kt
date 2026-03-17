package io.github.a7413498.liao.android.test

import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.test.junit4.ComposeContentTestRule
import io.github.a7413498.liao.android.app.theme.LiaoTheme

fun ComposeContentTestRule.setLiaoTestContent(content: @Composable () -> Unit) {
    setContent {
        LiaoTheme {
            Surface {
                content()
            }
        }
    }
}
