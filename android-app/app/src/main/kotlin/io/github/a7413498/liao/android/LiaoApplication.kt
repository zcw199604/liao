/*
 * 应用入口负责初始化 Hilt 依赖图。
 * Android 客户端的全局单例都从这里开始装配。
 */
package io.github.a7413498.liao.android

import android.app.Application
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class LiaoApplication : Application()
