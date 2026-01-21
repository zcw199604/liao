package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (a *App) buildRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(a.corsMiddleware)

	// WebSocket 入口（握手 token 通过 query 参数校验）
	r.Get("/ws", a.handleWebSocket)

	// 静态上传文件
	r.Handle("/upload/*", a.uploadFileServer())
	r.Handle("/lsp/*", a.lspFileServer())

	// API
	r.Route("/api", func(api chi.Router) {
		api.Use(a.jwtMiddleware)

		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/login", a.handleAuthLogin)
			ar.Get("/verify", a.handleAuthVerify)
		})

		// Identity
		api.Get("/getIdentityList", a.handleGetIdentityList)
		api.Post("/createIdentity", a.handleCreateIdentity)
		api.Post("/quickCreateIdentity", a.handleQuickCreateIdentity)
		api.Post("/updateIdentity", a.handleUpdateIdentity)
		api.Post("/updateIdentityId", a.handleUpdateIdentityID)
		api.Post("/deleteIdentity", a.handleDeleteIdentity)
		api.Post("/selectIdentity", a.handleSelectIdentity)

		// Local Favorite
		api.Route("/favorite", func(fr chi.Router) {
			fr.Post("/add", a.handleFavoriteAdd)
			fr.Post("/remove", a.handleFavoriteRemove)
			fr.Post("/removeById", a.handleFavoriteRemoveByID)
			fr.Get("/listAll", a.handleFavoriteListAll)
			fr.Get("/check", a.handleFavoriteCheck)
		})

		// Upstream HTTP proxy + upload
		api.Post("/getHistoryUserList", a.handleGetHistoryUserList)
		api.Post("/getFavoriteUserList", a.handleGetFavoriteUserList)
		api.Post("/reportReferrer", a.handleReportReferrer)
		api.Post("/getMessageHistory", a.handleGetMessageHistory)
		api.Get("/getImgServer", a.handleGetImgServer)
		api.Post("/updateImgServer", a.handleUpdateImgServer)
		api.Post("/uploadMedia", a.handleUploadMedia)
		api.Post("/uploadImage", a.handleUploadImage)
		api.Post("/checkDuplicateMedia", a.handleCheckDuplicateMedia)
		api.Get("/getCachedImages", a.handleGetCachedImages)
		api.Post("/toggleFavorite", a.handleToggleFavorite)
		api.Post("/cancelFavorite", a.handleCancelFavorite)

		// Media history
		api.Post("/recordImageSend", a.handleRecordImageSend)
		api.Get("/getUserUploadHistory", a.handleGetUserUploadHistory)
		api.Get("/getUserSentImages", a.handleGetUserSentImages)
		api.Get("/getUserUploadStats", a.handleGetUserUploadStats)
		api.Get("/getChatImages", a.handleGetChatImages)
		api.Post("/reuploadHistoryImage", a.handleReuploadHistoryImage)
		api.Get("/getAllUploadImages", a.handleGetAllUploadImages)
		api.Post("/deleteMedia", a.handleDeleteMedia)
		api.Post("/batchDeleteMedia", a.handleBatchDeleteMedia)
		api.Post("/repairMediaHistory", a.handleRepairMediaHistory)

		// Video extract（视频抽帧任务）
		api.Post("/uploadVideoExtractInput", a.handleUploadVideoExtractInput)
		api.Post("/cleanupVideoExtractInput", a.handleCleanupVideoExtractInput)
		api.Get("/probeVideo", a.handleProbeVideo)
		api.Post("/createVideoExtractTask", a.handleCreateVideoExtractTask)
		api.Get("/getVideoExtractTaskList", a.handleGetVideoExtractTaskList)
		api.Get("/getVideoExtractTaskDetail", a.handleGetVideoExtractTaskDetail)
		api.Post("/cancelVideoExtractTask", a.handleCancelVideoExtractTask)
		api.Post("/continueVideoExtractTask", a.handleContinueVideoExtractTask)
		api.Post("/deleteVideoExtractTask", a.handleDeleteVideoExtractTask)

		// 抖音下载（TikTokDownloader Web API）
		api.Route("/douyin", func(dr chi.Router) {
			dr.Post("/detail", a.handleDouyinDetail)
			dr.Get("/download", a.handleDouyinDownload)
			dr.Head("/download", a.handleDouyinDownload)
			dr.Post("/import", a.handleDouyinImport)
		})

		// mtPhoto 相册
		api.Get("/getMtPhotoAlbums", a.handleGetMtPhotoAlbums)
		api.Get("/getMtPhotoAlbumFiles", a.handleGetMtPhotoAlbumFiles)
		api.Get("/getMtPhotoThumb", a.handleGetMtPhotoThumb)
		api.Get("/downloadMtPhotoOriginal", a.handleDownloadMtPhotoOriginal)
		api.Get("/resolveMtPhotoFilePath", a.handleResolveMtPhotoFilePath)
		api.Post("/importMtPhotoMedia", a.handleImportMtPhotoMedia)

		// System config（全局配置：所有用户共用）
		api.Get("/getSystemConfig", a.handleGetSystemConfig)
		api.Post("/updateSystemConfig", a.handleUpdateSystemConfig)
		api.Post("/resolveImagePort", a.handleResolveImagePort)

		// System（依赖 WebSocket 管理器，在后续阶段补齐实现）
		api.Post("/deleteUpstreamUser", a.handleDeleteUpstreamUser)
		api.Get("/getConnectionStats", a.handleGetConnectionStats)
		api.Post("/disconnectAllConnections", a.handleDisconnectAllConnections)
		api.Get("/getForceoutUserCount", a.handleGetForceoutUserCount)
		api.Post("/clearForceoutUsers", a.handleClearForceoutUsers)
	})

	// 前端静态资源 + SPA 回退
	r.Handle("/*", a.spaHandler())

	return r
}
