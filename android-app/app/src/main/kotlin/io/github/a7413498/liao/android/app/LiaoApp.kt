/*
 * Compose 根导航负责串起登录、身份、会话、聊天与设置页面。
 * 当前版本优先保证主链路清晰，复杂弹层留到后续功能迭代。
 */
package io.github.a7413498.liao.android.app

import androidx.compose.runtime.Composable
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import io.github.a7413498.liao.android.feature.auth.LoginScreen
import io.github.a7413498.liao.android.feature.auth.LoginViewModel
import io.github.a7413498.liao.android.feature.chatlist.ChatListScreen
import io.github.a7413498.liao.android.feature.chatlist.ChatListViewModel
import io.github.a7413498.liao.android.feature.chatroom.ChatRoomScreen
import io.github.a7413498.liao.android.feature.chatroom.ChatRoomViewModel
import io.github.a7413498.liao.android.feature.identity.IdentityScreen
import io.github.a7413498.liao.android.feature.identity.IdentityViewModel
import io.github.a7413498.liao.android.feature.settings.SettingsScreen
import io.github.a7413498.liao.android.feature.settings.SettingsViewModel

object LiaoRoute {
    const val LOGIN = "login"
    const val IDENTITY = "identity"
    const val CHAT_LIST = "chatList"
    const val CHAT_ROOM = "chatRoom/{peerId}/{peerName}"
    const val SETTINGS = "settings"

    fun chatRoom(peerId: String, peerName: String): String =
        "chatRoom/${peerId}/${java.net.URLEncoder.encode(peerName, Charsets.UTF_8.name())}"
}

@Composable
fun LiaoApp() {
    val navController = rememberNavController()

    NavHost(navController = navController, startDestination = LiaoRoute.LOGIN) {
        composable(LiaoRoute.LOGIN) {
            val viewModel: LoginViewModel = hiltViewModel()
            LoginScreen(
                viewModel = viewModel,
                onLoginSuccess = { navController.navigate(LiaoRoute.IDENTITY) }
            )
        }
        composable(LiaoRoute.IDENTITY) {
            val viewModel: IdentityViewModel = hiltViewModel()
            IdentityScreen(
                viewModel = viewModel,
                onIdentitySelected = { navController.navigate(LiaoRoute.CHAT_LIST) }
            )
        }
        composable(LiaoRoute.CHAT_LIST) {
            val viewModel: ChatListViewModel = hiltViewModel()
            ChatListScreen(
                viewModel = viewModel,
                onOpenSettings = { navController.navigate(LiaoRoute.SETTINGS) },
                onOpenChat = { peerId, peerName -> navController.navigate(LiaoRoute.chatRoom(peerId, peerName)) }
            )
        }
        composable(
            route = LiaoRoute.CHAT_ROOM,
            arguments = listOf(
                navArgument("peerId") { type = NavType.StringType },
                navArgument("peerName") { type = NavType.StringType },
            )
        ) { backStackEntry ->
            val viewModel: ChatRoomViewModel = hiltViewModel()
            val peerId = backStackEntry.arguments?.getString("peerId").orEmpty()
            val peerName = java.net.URLDecoder.decode(
                backStackEntry.arguments?.getString("peerName").orEmpty(),
                Charsets.UTF_8.name()
            )
            ChatRoomScreen(
                peerId = peerId,
                peerName = peerName,
                viewModel = viewModel,
                onBack = { navController.popBackStack() }
            )
        }
        composable(LiaoRoute.SETTINGS) {
            val viewModel: SettingsViewModel = hiltViewModel()
            SettingsScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onLoggedOut = {
                    navController.navigate(LiaoRoute.LOGIN) {
                        popUpTo(0)
                    }
                }
            )
        }
    }
}
