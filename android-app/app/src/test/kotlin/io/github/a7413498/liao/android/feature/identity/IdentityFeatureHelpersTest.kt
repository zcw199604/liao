package io.github.a7413498.liao.android.feature.identity

import org.junit.Assert.assertEquals
import org.junit.Test

class IdentityFeatureHelpersTest {
    @Test
    fun `identity intro text should distinguish create and edit mode`() {
        assertEquals("可创建新身份，或选择已有身份进入聊天。", identityIntroText(null))
        assertEquals("可创建新身份，或选择已有身份进入聊天。", identityIntroText(""))
        assertEquals("当前处于编辑模式，保存后会同步刷新列表与当前会话。", identityIntroText("identity-1"))
    }

    @Test
    fun `identity action labels should match current mode`() {
        assertEquals("创建身份", identityPrimaryActionLabel(null))
        assertEquals("保存编辑", identityPrimaryActionLabel("identity-1"))
        assertEquals("快速创建", identitySecondaryActionLabel(null))
        assertEquals("取消编辑", identitySecondaryActionLabel("identity-1"))
    }
}
