/*
 * Compose 根导航负责串起登录、身份、会话、聊天、收藏与设置页面。
 * 当前版本增加应用级协调器，统一处理启动恢复、会话失效和 WebSocket 绑定。
 */
package io.github.a7413498.liao.android.app

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.navArgument
import io.github.a7413498.liao.android.feature.auth.LoginScreen
import io.github.a7413498.liao.android.feature.auth.LoginViewModel
import io.github.a7413498.liao.android.feature.chatlist.ChatListScreen
import io.github.a7413498.liao.android.feature.chatlist.ChatListViewModel
import io.github.a7413498.liao.android.feature.chatroom.ChatRoomScreen
import io.github.a7413498.liao.android.feature.chatroom.ChatRoomViewModel
import io.github.a7413498.liao.android.feature.douyin.DouyinScreen
import io.github.a7413498.liao.android.feature.douyin.DouyinViewModel
import io.github.a7413498.liao.android.feature.favorites.GlobalFavoritesScreen
import io.github.a7413498.liao.android.feature.favorites.GlobalFavoritesViewModel
import io.github.a7413498.liao.android.feature.identity.IdentityScreen
import io.github.a7413498.liao.android.feature.identity.IdentityViewModel
import io.github.a7413498.liao.android.feature.media.MediaLibraryScreen
import io.github.a7413498.liao.android.feature.media.MediaLibraryViewModel
import io.github.a7413498.liao.android.feature.mtphoto.MtPhotoScreen
import io.github.a7413498.liao.android.feature.mtphoto.MtPhotoViewModel
import io.github.a7413498.liao.android.feature.settings.SettingsScreen
import io.github.a7413498.liao.android.feature.settings.SettingsViewModel
import io.github.a7413498.liao.android.feature.videoextract.VideoExtractCreateScreen
import io.github.a7413498.liao.android.feature.videoextract.VideoExtractCreateViewModel
import io.github.a7413498.liao.android.feature.videoextract.VideoExtractTaskCenterScreen
import io.github.a7413498.liao.android.feature.videoextract.VideoExtractTaskCenterViewModel

private const val MT_PHOTO_IMPORT_LOCAL_PATH = "mtphoto_import_local_path"
private const val MT_PHOTO_IMPORT_LOCAL_FILENAME = "mtphoto_import_local_filename"
private const val DOUYIN_IMPORT_LOCAL_PATH = "douyin_import_local_path"
private const val DOUYIN_IMPORT_LOCAL_FILENAME = "douyin_import_local_filename"

object LiaoRoute {
    const val LOGIN = "login"
    const val IDENTITY = "identity"
    const val CHAT_LIST = "chatList"
    const val CHAT_ROOM = "chatRoom/{peerId}/{peerName}"
    const val SETTINGS = "settings"
    const val FAVORITES = "favorites"
    const val MEDIA_LIBRARY = "mediaLibrary"
    const val DOUYIN = "douyin"
    const val VIDEO_EXTRACT_CREATE = "videoExtractCreate"
    const val VIDEO_EXTRACT_TASKS = "videoExtractTasks"
    const val DOUYIN_CHAT_IMPORT = "douyinChatImport"
    const val MTPHOTO = "mtphoto"
    const val MTPHOTO_ROUTE = "mtphoto?folderId={folderId}&folderName={folderName}"
    const val MTPHOTO_CHAT_IMPORT = "mtphotoChatImport"

    fun chatRoom(peerId: String, peerName: String): String =
        "chatRoom/${peerId}/${java.net.URLEncoder.encode(peerName, Charsets.UTF_8.name())}"

    fun mtPhoto(folderId: Long? = null, folderName: String = ""): String {
        if (folderId == null || folderId <= 0) return MTPHOTO
        return "$MTPHOTO?folderId=$folderId&folderName=${java.net.URLEncoder.encode(folderName, Charsets.UTF_8.name())}"
    }
}

@Composable
fun LiaoApp() {
    val navController = rememberNavController()
    val coordinatorViewModel: AppCoordinatorViewModel = hiltViewModel()
    val appState by coordinatorViewModel.uiState.collectAsStateWithLifecycle()
    val backStackEntry by navController.currentBackStackEntryAsState()

    val activePeerId = if (backStackEntry?.destination?.route == LiaoRoute.CHAT_ROOM) {
        backStackEntry?.arguments?.getString("peerId")
    } else {
        null
    }

    LaunchedEffect(activePeerId) {
        coordinatorViewModel.setActiveChatPeer(activePeerId)
    }

    LaunchedEffect(appState.sessionExpiredMessage) {
        if (!appState.sessionExpiredMessage.isNullOrBlank()) {
            navController.navigate(LiaoRoute.LOGIN) {
                popUpTo(0)
            }
            coordinatorViewModel.consumeSessionExpiredMessage()
        }
    }

    if (!appState.launchResolved) {
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.Center,
        ) {
            CircularProgressIndicator(modifier = Modifier.wrapContentSize())
        }
        return
    }

    NavHost(navController = navController, startDestination = appState.launchRoute) {
        composable(LiaoRoute.LOGIN) {
            val viewModel: LoginViewModel = hiltViewModel()
            LoginScreen(
                viewModel = viewModel,
                onLoginSuccess = { hasCurrentSession ->
                    navController.navigate(if (hasCurrentSession) LiaoRoute.CHAT_LIST else LiaoRoute.IDENTITY) {
                        popUpTo(LiaoRoute.LOGIN) { inclusive = true }
                    }
                }
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
                onOpenGlobalFavorites = { navController.navigate(LiaoRoute.FAVORITES) },
                onOpenChat = { peerId, peerName -> navController.navigate(LiaoRoute.chatRoom(peerId, peerName)) }
            )
        }
        composable(
            route = LiaoRoute.CHAT_ROOM,
            arguments = listOf(
                navArgument("peerId") { type = NavType.StringType },
                navArgument("peerName") { type = NavType.StringType },
            )
        ) { entry ->
            val viewModel: ChatRoomViewModel = hiltViewModel()
            val peerId = entry.arguments?.getString("peerId").orEmpty()
            val peerName = java.net.URLDecoder.decode(
                entry.arguments?.getString("peerName").orEmpty(),
                Charsets.UTF_8.name(),
            )
            val importedLocalPath by entry.savedStateHandle.getStateFlow<String?>(MT_PHOTO_IMPORT_LOCAL_PATH, null).collectAsStateWithLifecycle()
            val importedLocalFilename by entry.savedStateHandle.getStateFlow<String?>(MT_PHOTO_IMPORT_LOCAL_FILENAME, null).collectAsStateWithLifecycle()
            val importedDouyinLocalPath by entry.savedStateHandle.getStateFlow<String?>(DOUYIN_IMPORT_LOCAL_PATH, null).collectAsStateWithLifecycle()
            val importedDouyinLocalFilename by entry.savedStateHandle.getStateFlow<String?>(DOUYIN_IMPORT_LOCAL_FILENAME, null).collectAsStateWithLifecycle()

            LaunchedEffect(importedLocalPath, importedLocalFilename) {
                val localPath = importedLocalPath
                if (!localPath.isNullOrBlank()) {
                    viewModel.importMtPhotoMedia(localPath = localPath, localFilename = importedLocalFilename.orEmpty())
                    entry.savedStateHandle[MT_PHOTO_IMPORT_LOCAL_PATH] = null
                    entry.savedStateHandle[MT_PHOTO_IMPORT_LOCAL_FILENAME] = null
                }
            }

            LaunchedEffect(importedDouyinLocalPath, importedDouyinLocalFilename) {
                val localPath = importedDouyinLocalPath
                if (!localPath.isNullOrBlank()) {
                    viewModel.importDouyinMedia(localPath = localPath, localFilename = importedDouyinLocalFilename.orEmpty())
                    entry.savedStateHandle[DOUYIN_IMPORT_LOCAL_PATH] = null
                    entry.savedStateHandle[DOUYIN_IMPORT_LOCAL_FILENAME] = null
                }
            }

            ChatRoomScreen(
                peerId = peerId,
                peerName = peerName,
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onOpenMtPhoto = { navController.navigate(LiaoRoute.MTPHOTO_CHAT_IMPORT) },
                onOpenDouyin = { navController.navigate(LiaoRoute.DOUYIN_CHAT_IMPORT) },
            )
        }
        composable(LiaoRoute.SETTINGS) {
            val viewModel: SettingsViewModel = hiltViewModel()
            SettingsScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onOpenGlobalFavorites = { navController.navigate(LiaoRoute.FAVORITES) },
                onOpenMediaLibrary = { navController.navigate(LiaoRoute.MEDIA_LIBRARY) },
                onOpenMtPhoto = { navController.navigate(LiaoRoute.mtPhoto()) },
                onOpenDouyin = { navController.navigate(LiaoRoute.DOUYIN) },
                onOpenVideoExtract = { navController.navigate(LiaoRoute.VIDEO_EXTRACT_CREATE) },
                onOpenVideoExtractTasks = { navController.navigate(LiaoRoute.VIDEO_EXTRACT_TASKS) },
                onLoggedOut = {
                    navController.navigate(LiaoRoute.LOGIN) {
                        popUpTo(0)
                    }
                }
            )
        }
        composable(LiaoRoute.VIDEO_EXTRACT_CREATE) {
            val viewModel: VideoExtractCreateViewModel = hiltViewModel()
            VideoExtractCreateScreen(
                onBack = { navController.popBackStack() },
                onOpenTaskCenter = { navController.navigate(LiaoRoute.VIDEO_EXTRACT_TASKS) },
                viewModel = viewModel,
            )
        }
        composable(LiaoRoute.VIDEO_EXTRACT_TASKS) {
            val viewModel: VideoExtractTaskCenterViewModel = hiltViewModel()
            VideoExtractTaskCenterScreen(
                onBack = { navController.popBackStack() },
                onOpenCreate = { navController.navigate(LiaoRoute.VIDEO_EXTRACT_CREATE) },
                viewModel = viewModel,
            )
        }
        composable(LiaoRoute.MEDIA_LIBRARY) {
            val viewModel: MediaLibraryViewModel = hiltViewModel()
            MediaLibraryScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onOpenMtPhotoFolder = { folderId, folderName ->
                    navController.navigate(LiaoRoute.mtPhoto(folderId = folderId, folderName = folderName))
                },
            )
        }
        composable(LiaoRoute.DOUYIN) {
            val viewModel: DouyinViewModel = hiltViewModel()
            DouyinScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
            )
        }
        composable(LiaoRoute.DOUYIN_CHAT_IMPORT) {
            val viewModel: DouyinViewModel = hiltViewModel()
            DouyinScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onImportToChat = { imported ->
                    navController.previousBackStackEntry?.savedStateHandle?.set(DOUYIN_IMPORT_LOCAL_PATH, imported.localPath)
                    navController.previousBackStackEntry?.savedStateHandle?.set(DOUYIN_IMPORT_LOCAL_FILENAME, imported.localFilename)
                    navController.popBackStack()
                },
            )
        }
        composable(
            route = LiaoRoute.MTPHOTO_ROUTE,
            arguments = listOf(
                navArgument("folderId") {
                    type = NavType.LongType
                    defaultValue = -1L
                },
                navArgument("folderName") {
                    type = NavType.StringType
                    defaultValue = ""
                },
            ),
        ) { entry ->
            val viewModel: MtPhotoViewModel = hiltViewModel()
            val initialFolderId = entry.arguments?.getLong("folderId")?.takeIf { it > 0L }
            val initialFolderName = java.net.URLDecoder.decode(
                entry.arguments?.getString("folderName").orEmpty(),
                Charsets.UTF_8.name(),
            )
            MtPhotoScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                initialFolderId = initialFolderId,
                initialFolderName = initialFolderName,
            )
        }
        composable(LiaoRoute.MTPHOTO_CHAT_IMPORT) {
            val viewModel: MtPhotoViewModel = hiltViewModel()
            MtPhotoScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onImportToChat = { imported ->
                    navController.previousBackStackEntry?.savedStateHandle?.set(MT_PHOTO_IMPORT_LOCAL_PATH, imported.localPath)
                    navController.previousBackStackEntry?.savedStateHandle?.set(MT_PHOTO_IMPORT_LOCAL_FILENAME, imported.localFilename)
                    navController.popBackStack()
                },
            )
        }
        composable(LiaoRoute.FAVORITES) {
            val viewModel: GlobalFavoritesViewModel = hiltViewModel()
            GlobalFavoritesScreen(
                viewModel = viewModel,
                onBack = { navController.popBackStack() },
                onOpenChat = { peerId: String, peerName: String ->
                    navController.navigate(LiaoRoute.chatRoom(peerId, peerName))
                },
            )
        }
    }
}
