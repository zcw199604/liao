package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const (
	upstreamHistoryURL  = "http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserList_Random"
	upstreamFavoriteURL = "http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserList_My"
	upstreamReportURL   = "http://v1.chat2019.cn/asmx/method.asmx/referrer_record"
	upstreamMsgURL      = "http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserMsgsPage"
	upstreamImgServer   = "http://v1.chat2019.cn/asmx/method.asmx/getImgServer"
)

func (a *App) handleGetHistoryUserList(w http.ResponseWriter, r *http.Request) {
	totalStart := time.Now()
	upstreamMs := int64(-1)
	enrichUserInfoMs := int64(-1)
	lastMsgMs := int64(-1)
	resultSize := -1
	var upstreamStatus int
	cacheEnabled := a.userInfoCache != nil

	_ = r.ParseForm()

	myUserID := defaultString(r.FormValue("myUserID"), "5be810d731d340f090b098392f9f0a31")
	vipcode := defaultString(r.FormValue("vipcode"), "")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	defer func() {
		slog.Info(
			"[timing] /api/getHistoryUserList",
			"myUserID", myUserID,
			"status", upstreamStatus,
			"size", resultSize,
			"upstreamMs", upstreamMs,
			"enrichUserInfoMs", enrichUserInfoMs,
			"lastMsgMs", lastMsgMs,
			"totalMs", time.Since(totalStart).Milliseconds(),
			"cacheEnabled", cacheEnabled,
		)
	}()

	slog.Info("获取历史用户列表请求", "myUserID", myUserID, "vipcode", vipcode, "serverPort", serverPort)

	form := url.Values{}
	form.Set("myUserID", myUserID)
	form.Set("vipcode", vipcode)
	form.Set("serverPort", serverPort)

	slog.Info("请求参数", "myUserID", myUserID, "vipcode", vipcode, "serverPort", serverPort)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
	}
	if cookieData != "" {
		headers["Cookie"] = cookieData
	}

	upstreamStart := time.Now()
	status, body, err := a.postForm(r.Context(), upstreamHistoryURL, form, headers)
	upstreamMs = time.Since(upstreamStart).Milliseconds()
	upstreamStatus = status
	if err != nil {
		slog.Error("调用上游接口失败", "api", "/api/getHistoryUserList", "error", err)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: "+err.Error()+"\"}")
		return
	}
	if status != http.StatusOK {
		slog.Error("调用上游接口失败", "api", "/api/getHistoryUserList", "status", status)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: upstream status "+fmt.Sprint(status)+"\"}")
		return
	}

	slog.Info("上游接口返回", "status", status, "bodyLength", len(body))
	slog.Debug("上游接口 body", "api", "/api/getHistoryUserList", "body", body)

	if cacheEnabled && strings.TrimSpace(body) != "" {
		var list []map[string]any
		if err := json.Unmarshal([]byte(body), &list); err != nil {
			slog.Error("解析上游历史用户列表失败", "error", err)
		} else {
			idKey := "id"
			if len(list) > 0 {
				if _, ok := list[0]["id"]; !ok {
					if _, ok := list[0]["UserID"]; ok {
						idKey = "UserID"
					} else if _, ok := list[0]["userid"]; ok {
						idKey = "userid"
					}
				}
			}

			enrichUserInfoMs, lastMsgMs = enrichUserListInPlace(a.userInfoCache, list, idKey, myUserID)

			resultSize = len(list)

			enhanced, marshalErr := json.Marshal(list)
			if marshalErr == nil {
				writeText(w, http.StatusOK, string(enhanced))
				return
			}

			slog.Error("增强历史用户列表失败", "error", marshalErr)
		}
	}

	writeText(w, http.StatusOK, body)
}

func (a *App) handleGetFavoriteUserList(w http.ResponseWriter, r *http.Request) {
	totalStart := time.Now()
	upstreamMs := int64(-1)
	enrichUserInfoMs := int64(-1)
	lastMsgMs := int64(-1)
	resultSize := -1
	var upstreamStatus int
	cacheEnabled := a.userInfoCache != nil

	_ = r.ParseForm()

	myUserID := defaultString(r.FormValue("myUserID"), "5be810d731d340f090b098392f9f0a31")
	vipcode := defaultString(r.FormValue("vipcode"), "")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	defer func() {
		slog.Info(
			"[timing] /api/getFavoriteUserList",
			"myUserID", myUserID,
			"status", upstreamStatus,
			"size", resultSize,
			"upstreamMs", upstreamMs,
			"enrichUserInfoMs", enrichUserInfoMs,
			"lastMsgMs", lastMsgMs,
			"totalMs", time.Since(totalStart).Milliseconds(),
			"cacheEnabled", cacheEnabled,
		)
	}()

	slog.Info("获取收藏用户列表请求", "myUserID", myUserID, "vipcode", vipcode, "serverPort", serverPort)

	form := url.Values{}
	form.Set("myUserID", myUserID)
	form.Set("vipcode", vipcode)
	form.Set("serverPort", serverPort)

	slog.Info("请求参数", "myUserID", myUserID, "vipcode", vipcode, "serverPort", serverPort)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
	}
	if cookieData != "" {
		headers["Cookie"] = cookieData
	}

	upstreamStart := time.Now()
	status, body, err := a.postForm(r.Context(), upstreamFavoriteURL, form, headers)
	upstreamMs = time.Since(upstreamStart).Milliseconds()
	upstreamStatus = status
	if err != nil {
		slog.Error("调用上游接口失败", "api", "/api/getFavoriteUserList", "error", err)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: "+err.Error()+"\"}")
		return
	}
	if status != http.StatusOK {
		slog.Error("调用上游接口失败", "api", "/api/getFavoriteUserList", "status", status)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: upstream status "+fmt.Sprint(status)+"\"}")
		return
	}

	slog.Info("上游接口返回", "status", status, "bodyLength", len(body))
	slog.Debug("上游接口 body", "api", "/api/getFavoriteUserList", "body", body)

	if cacheEnabled && strings.TrimSpace(body) != "" {
		var list []map[string]any
		if err := json.Unmarshal([]byte(body), &list); err != nil {
			slog.Error("解析上游收藏用户列表失败", "error", err)
		} else {
			idKey := "id"
			if len(list) > 0 {
				if _, ok := list[0]["id"]; !ok {
					if _, ok := list[0]["UserID"]; ok {
						idKey = "UserID"
					} else if _, ok := list[0]["userid"]; ok {
						idKey = "userid"
					}
				}
			}

			enrichUserInfoMs, lastMsgMs = enrichUserListInPlace(a.userInfoCache, list, idKey, myUserID)

			resultSize = len(list)

			enhanced, marshalErr := json.Marshal(list)
			if marshalErr == nil {
				writeText(w, http.StatusOK, string(enhanced))
				return
			}

			slog.Error("增强收藏用户列表失败", "error", marshalErr)
		}
	}

	writeText(w, http.StatusOK, body)
}

func (a *App) handleReportReferrer(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	referrerURL := r.FormValue("referrerUrl")
	currURL := r.FormValue("currUrl")
	userID := r.FormValue("userid")
	cookieData := r.FormValue("cookieData")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	slog.Info("上报访问记录", "referrerUrl", referrerURL, "currUrl", currURL, "userid", userID)
	slog.Info(
		"请求 Headers",
		"host", "v1.chat2019.cn",
		"origin", "http://v1.chat2019.cn",
		"referer", referer,
		"userAgent", userAgent,
		"cookiePresent", strings.TrimSpace(cookieData) != "",
		"cookieLength", len(cookieData),
	)

	form := url.Values{}
	form.Set("referrer_url", referrerURL)
	form.Set("curr_url", currURL)
	form.Set("userid", userID)

	slog.Info("上报参数", "referrer_url", referrerURL, "curr_url", currURL, "userid", userID)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
		"Cookie":     cookieData,
	}

	upstreamStart := time.Now()
	status, body, err := a.postForm(r.Context(), upstreamReportURL, form, headers)
	upstreamMs := time.Since(upstreamStart).Milliseconds()
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		slog.Error("上报访问记录失败", "status", status, "upstreamMs", upstreamMs, "error", err)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"上报失败: "+err.Error()+"\"}")
		return
	}

	slog.Info("上报接口返回", "status", status, "bodyLength", len(body), "upstreamMs", upstreamMs)
	slog.Debug("上报接口 body", "body", body)
	writeText(w, http.StatusOK, body)
}

func (a *App) handleGetMessageHistory(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	myUserID := r.FormValue("myUserID")
	userToID := r.FormValue("UserToID")
	isFirst := defaultString(r.FormValue("isFirst"), "1")
	firstTid := defaultString(r.FormValue("firstTid"), "0")
	vipcode := defaultString(r.FormValue("vipcode"), "")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	slog.Info("获取消息历史请求", "myUserID", myUserID, "UserToID", userToID, "isFirst", isFirst, "firstTid", firstTid)

	const defaultHistoryPageSize = 20
	conversationKey := generateConversationKey(strings.TrimSpace(myUserID), strings.TrimSpace(userToID))
	cacheEnabled := a.chatHistoryCache != nil && conversationKey != ""
	trimmedFirstTid := strings.TrimSpace(firstTid)
	isHistoryPage := trimmedFirstTid != "" && trimmedFirstTid != "0"

	var cachedMessages []map[string]any
	if cacheEnabled {
		cached, err := a.chatHistoryCache.GetMessages(r.Context(), conversationKey, firstTid, defaultHistoryPageSize)
		if err != nil {
			slog.Warn("读取Redis聊天记录失败(降级为仅上游)", "conversationKey", conversationKey, "error", err)
			cacheEnabled = false
		} else {
			cachedMessages = cached
		}
	}

	// 仅“历史翻页”（firstTid>0）允许在缓存足够时跳过上游；
	// 最新页（firstTid=0）必须请求上游以保证拿到最新消息（避免仅靠缓存漏消息）。
	if cacheEnabled && isHistoryPage && len(cachedMessages) >= defaultHistoryPageSize {
		resp := map[string]any{
			"code":          0,
			"contents_list": cachedMessages[:defaultHistoryPageSize],
		}
		if raw, err := json.Marshal(resp); err == nil {
			writeText(w, http.StatusOK, string(raw))
			return
		}
	}

	form := url.Values{}
	form.Set("myUserID", myUserID)
	form.Set("UserToID", userToID)
	form.Set("isFirst", isFirst)
	form.Set("firstTid", firstTid)
	form.Set("vipcode", vipcode)
	form.Set("serverPort", serverPort)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
	}
	if cookieData != "" {
		headers["Cookie"] = cookieData
	}

	slog.Info("请求参数", "myUserID", myUserID, "UserToID", userToID, "isFirst", isFirst, "firstTid", firstTid, "vipcode", vipcode, "serverPort", serverPort)

	upstreamStart := time.Now()
	status, body, err := a.postForm(r.Context(), upstreamMsgURL, form, headers)
	upstreamMs := time.Since(upstreamStart).Milliseconds()
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		slog.Error("获取消息历史失败", "status", status, "upstreamMs", upstreamMs, "error", err)

		if len(cachedMessages) > 0 {
			if a.userInfoCache != nil && isFirst == "1" && !isHistoryPage && len(cachedMessages) > 0 {
				first := cachedMessages[0]
				fromUserID := strings.TrimSpace(toString(first["id"]))
				toUserID := strings.TrimSpace(toString(first["toid"]))
				content := strings.TrimSpace(toString(first["content"]))
				tm := strings.TrimSpace(toString(first["time"]))
				tp := inferMessageType(content)
				if fromUserID != "" && toUserID != "" && content != "" && tm != "" {
					cacheFrom := fromUserID
					cacheTo := toUserID
					if myUserID != fromUserID && myUserID != toUserID {
						if userToID == fromUserID {
							cacheTo = myUserID
						} else if userToID == toUserID {
							cacheFrom = myUserID
						}
					}
					a.userInfoCache.SaveLastMessage(CachedLastMessage{
						FromUserID: cacheFrom,
						ToUserID:   cacheTo,
						Content:    content,
						Type:       tp,
						Time:       tm,
					})
				}
			}

			limit := defaultHistoryPageSize
			if len(cachedMessages) < limit {
				limit = len(cachedMessages)
			}
			resp := map[string]any{
				"code":          0,
				"contents_list": cachedMessages[:limit],
			}
			if raw, marshalErr := json.Marshal(resp); marshalErr == nil {
				writeText(w, http.StatusOK, string(raw))
				return
			}
		}

		writeText(w, http.StatusInternalServerError, "{\"error\":\"获取消息历史失败: "+err.Error()+"\"}")
		return
	}

	slog.Info("上游接口返回", "status", status, "bodyLength", len(body), "upstreamMs", upstreamMs)
	slog.Debug("上游接口 body", "api", "/api/getMessageHistory", "body", body)

	if strings.TrimSpace(body) == "" {
		writeText(w, http.StatusOK, body)
		return
	}

	needParse := a.userInfoCache != nil || a.chatHistoryCache != nil
	if !needParse {
		writeText(w, http.StatusOK, body)
		return
	}

	var root any
	if err := json.Unmarshal([]byte(body), &root); err != nil {
		writeText(w, http.StatusOK, body)
		return
	}

	// 新格式：包含 contents_list（合并 Redis 历史 + 回填缓存 + 写入最后消息缓存）
	if obj, ok := root.(map[string]any); ok {
		if contents, ok := obj["contents_list"]; ok {
			if arr, ok := contents.([]any); ok {
				upstreamList := make([]map[string]any, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						upstreamList = append(upstreamList, m)
					}
				}

				if a.userInfoCache != nil && isFirst == "1" && !isHistoryPage && len(upstreamList) > 0 {
					first := upstreamList[0]
					fromUserID := strings.TrimSpace(toString(first["id"]))
					toUserID := strings.TrimSpace(toString(first["toid"]))
					content := strings.TrimSpace(toString(first["content"]))
					tm := strings.TrimSpace(toString(first["time"]))
					tp := inferMessageType(content)

					if fromUserID != "" && toUserID != "" && content != "" && tm != "" {
						cacheFrom := fromUserID
						cacheTo := toUserID
						if myUserID != fromUserID && myUserID != toUserID {
							if userToID == fromUserID {
								cacheTo = myUserID
							} else if userToID == toUserID {
								cacheFrom = myUserID
							}
						}
						a.userInfoCache.SaveLastMessage(CachedLastMessage{
							FromUserID: cacheFrom,
							ToUserID:   cacheTo,
							Content:    content,
							Type:       tp,
							Time:       tm,
						})
					}
				}

				if cacheEnabled && len(upstreamList) > 0 {
					// best-effort 回填：用于弥补上游历史过期导致的缺口
					a.chatHistoryCache.SaveMessages(context.Background(), conversationKey, upstreamList)
				}

				merged := mergeHistoryMessages(upstreamList, cachedMessages, defaultHistoryPageSize)
				obj["contents_list"] = merged

				if enhanced, err := json.Marshal(obj); err == nil {
					writeText(w, http.StatusOK, string(enhanced))
					return
				}
			}

			// contents_list 存在但解析失败：保持兼容，返回原 body
			writeText(w, http.StatusOK, body)
			return
		}
	}

	// 旧格式：数组，增强用户信息后返回增强数组
	if arr, ok := root.([]any); ok && a.userInfoCache != nil {
		list := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				list = append(list, m)
			}
		}
		list = a.userInfoCache.BatchEnrichUserInfo(list, "userid")
		if enhanced, err := json.Marshal(list); err == nil {
			writeText(w, http.StatusOK, string(enhanced))
			return
		}
	}

	writeText(w, http.StatusOK, body)
}

func (a *App) handleGetImgServer(w http.ResponseWriter, r *http.Request) {
	slog.Info("获取图片服务器地址请求")
	urlWithTS := upstreamImgServer + "?_=" + fmt.Sprint(time.Now().UnixMilli())
	slog.Info("请求 URL", "url", urlWithTS)
	resp, err := a.httpClient.Get(urlWithTS)
	if err != nil {
		slog.Error("获取图片服务器失败", "error", err)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"获取图片服务器失败: "+err.Error()+"\"}")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("获取图片服务器失败", "status", resp.StatusCode, "upstreamStatus", resp.Status)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"获取图片服务器失败: upstream status "+resp.Status+"\"}")
		return
	}
	slog.Info("上游接口返回", "status", resp.StatusCode, "bodyLength", len(body))
	writeText(w, http.StatusOK, string(body))
}

func (a *App) handleUpdateImgServer(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	server := strings.TrimSpace(r.FormValue("server"))
	slog.Info("更新图片服务器", "server", server)
	a.imageServer.SetImgServerHost(server)
	if a.imagePortResolver != nil {
		a.imagePortResolver.ClearAll()
	}
	writeText(w, http.StatusOK, "{\"success\":true}")
}

func (a *App) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	totalStart := time.Now()
	if err := r.ParseMultipartForm(110 << 20); err != nil {
		slog.Error("上传媒体请求解析失败", "error", err)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"本地存储失败: "+err.Error()+"\"}")
		return
	}

	userID := strings.TrimSpace(r.FormValue("userid"))
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		slog.Warn("上传媒体缺少文件", "userid", userID)
		writeText(w, http.StatusBadRequest, "{\"error\":\"不支持的文件类型\"}")
		return
	}
	fileHeader := files[0]

	contentType := fileHeader.Header.Get("Content-Type")
	slog.Info(
		"上传媒体请求",
		"userid", userID,
		"fileName", fileHeader.Filename,
		"fileSize", fileHeader.Size,
		"contentType", contentType,
		"cookiePresent", strings.TrimSpace(cookieData) != "",
	)
	if !a.fileStorage.IsValidMediaType(contentType) {
		slog.Warn("不支持的文件类型", "contentType", contentType, "fileName", fileHeader.Filename)
		writeText(w, http.StatusBadRequest, "{\"error\":\"不支持的文件类型\"}")
		return
	}

	md5Value, err := a.fileStorage.CalculateMD5(fileHeader)
	if err != nil {
		slog.Error("MD5计算失败", "error", err)
		writeText(w, http.StatusInternalServerError, "{\"error\":\"MD5计算失败\"}")
		return
	}

	localPath := ""
	if existing, err := a.fileStorage.FindLocalPathByMD5(r.Context(), md5Value); err == nil && existing != "" {
		localPath = existing
	}
	if localPath == "" {
		category := a.fileStorage.CategoryFromContentType(contentType)
		saved, err := a.fileStorage.SaveFile(fileHeader, category)
		if err != nil {
			slog.Error("本地存储失败", "error", err)
			writeText(w, http.StatusInternalServerError, "{\"error\":\"本地存储失败: "+err.Error()+"\"}")
			return
		}
		localPath = saved
	}

	imgServerHost := a.imageServer.GetImgServerHost()
	uploadURL := fmt.Sprintf("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgServerHost, userID)
	slog.Info("上传请求 Headers", "host", strings.Split(imgServerHost, ":")[0], "origin", "http://v1.chat2019.cn")

	respBody, err := a.uploadToUpstream(r.Context(), uploadURL, imgServerHost, fileHeader, cookieData, referer, userAgent)
	if err != nil {
		slog.Error("上传媒体失败", "error", err, "localPath", localPath)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "上传媒体失败: " + err.Error(),
			"localPath": localPath,
		})
		return
	}

	// 尝试解析并增强返回（state==OK）
	var parsed map[string]any
	if err := json.Unmarshal([]byte(respBody), &parsed); err == nil {
		if state, _ := parsed["state"].(string); state == "OK" {
			if msg, ok := parsed["msg"].(string); ok && msg != "" {
				imgHostClean := strings.Split(imgServerHost, ":")[0]
				availablePort := ""
				if strings.HasPrefix(strings.ToLower(contentType), "video/") {
					availablePort = "8006"
				} else {
					availablePort = a.resolveImagePortByConfig(r.Context(), msg)
				}
				imageURL := fmt.Sprintf("http://%s:%s/img/Upload/%s", imgHostClean, availablePort, msg)

				localFilename := filepath.Base(strings.TrimPrefix(localPath, "/"))

				_, _ = a.mediaUpload.SaveUploadRecord(r.Context(), UploadRecord{
					UserID:           userID,
					OriginalFilename: fileHeader.Filename,
					LocalFilename:    localFilename,
					RemoteFilename:   msg,
					RemoteURL:        imageURL,
					LocalPath:        localPath,
					FileSize:         fileHeader.Size,
					FileType:         contentType,
					FileExtension:    a.fileStorage.FileExtension(fileHeader.Filename),
					FileMD5:          md5Value,
				})

				a.imageCache.AddImageToCache(userID, localPath)

				enhanced := map[string]any{
					"state":         "OK",
					"msg":           msg,
					"port":          availablePort,
					"localFilename": localFilename,
				}
				if b, err := json.Marshal(enhanced); err == nil {
					slog.Info("上传媒体成功", "userid", userID, "remoteFilename", msg, "localPath", localPath, "totalMs", time.Since(totalStart).Milliseconds())
					writeText(w, http.StatusOK, string(b))
					return
				}
			}
		}
	}

	slog.Info("上传媒体完成(未增强)", "userid", userID, "bodyLength", len(respBody), "totalMs", time.Since(totalStart).Milliseconds())
	writeText(w, http.StatusOK, respBody)
}

func (a *App) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	a.handleUploadMedia(w, r)
}

func (a *App) handleGetCachedImages(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("userid"))
	hostHeader := requestHostHeader(r)

	slog.Info("获取缓存图片", "userid", userID, "host", hostHeader)
	cached := a.imageCache.GetCachedImages(userID)
	if cached == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"port": a.resolveImagePortByConfig(r.Context(), ""),
			"data": []string{},
		})
		return
	}

	availablePort := a.resolveImagePortByConfig(r.Context(), "")
	localURLs := a.mediaUpload.ConvertPathsToLocalURLs(cached.ImageURLs, hostHeader)

	writeJSON(w, http.StatusOK, map[string]any{
		"port": availablePort,
		"data": localURLs,
	})
}

func (a *App) handleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	myUserID := r.FormValue("myUserID")
	userToID := r.FormValue("UserToID")
	vipcode := defaultString(r.FormValue("vipcode"), "")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0")

	slog.Info("收藏操作", "myUserID", myUserID, "UserToID", userToID)

	form := url.Values{}
	form.Set("myUserID", myUserID)
	form.Set("UserToID", userToID)
	form.Set("vipcode", vipcode)
	form.Set("serverPort", serverPort)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
		"Cookie":     cookieData,
	}

	upstreamStart := time.Now()
	status, body, err := a.postForm(r.Context(), "http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Do", form, headers)
	upstreamMs := time.Since(upstreamStart).Milliseconds()
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		slog.Error("收藏操作失败", "status", status, "upstreamMs", upstreamMs, "error", err)
		writeText(w, http.StatusOK, "{\"state\":\"ERROR\",\"msg\":\""+err.Error()+"\"}")
		return
	}
	slog.Info("收藏操作响应", "status", status, "bodyLength", len(body), "upstreamMs", upstreamMs)
	writeText(w, http.StatusOK, body)
}

func (a *App) handleCancelFavorite(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	myUserID := r.FormValue("myUserID")
	userToID := r.FormValue("UserToID")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0")

	slog.Info("取消收藏操作", "myUserID", myUserID, "UserToID", userToID)

	form := url.Values{}
	form.Set("myUserID", myUserID)
	form.Set("UserToID", userToID)
	form.Set("serverPort", serverPort)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
		"Cookie":     cookieData,
	}

	upstreamStart := time.Now()
	status, body, err := a.postForm(r.Context(), "http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Cancle", form, headers)
	upstreamMs := time.Since(upstreamStart).Milliseconds()
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		slog.Error("取消收藏操作失败", "status", status, "upstreamMs", upstreamMs, "error", err)
		writeText(w, http.StatusOK, "{\"state\":\"ERROR\",\"msg\":\""+err.Error()+"\"}")
		return
	}
	slog.Info("取消收藏操作响应", "status", status, "bodyLength", len(body), "upstreamMs", upstreamMs)
	writeText(w, http.StatusOK, body)
}

func (a *App) postForm(ctx context.Context, url string, form url.Values, headers map[string]string) (status int, body string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for k, v := range headers {
		if strings.EqualFold(k, "Host") {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(b), nil
}

func (a *App) uploadToUpstream(ctx context.Context, uploadURL, imgServerHost string, fileHeader *multipart.FileHeader, cookieData, referer, userAgent string) (string, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer func() { _ = pw.Close() }()
		defer func() { _ = writer.Close() }()

		part, err := writer.CreateFormFile("upload_file", fileHeader.Filename)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		src, err := fileHeader.Open()
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		defer src.Close()

		if _, err := io.Copy(part, src); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Host = strings.Split(imgServerHost, ":")[0]
	req.Header.Set("Origin", "http://v1.chat2019.cn")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)
	if cookieData != "" {
		req.Header.Set("Cookie", cookieData)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("上游响应异常: %s", resp.Status)
	}
	return string(b), nil
}

func defaultString(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func inferMessageType(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "text"
	}

	detected := ""
	idx := 0
	for idx < len(content) {
		open := strings.Index(content[idx:], "[")
		if open < 0 {
			break
		}
		open += idx
		close := strings.Index(content[open+1:], "]")
		if close < 0 {
			break
		}
		close += open + 1

		kind := inferMediaKindFromBracketBody(content[open+1 : close])
		if kind != "" {
			detected = pickHigherPriorityType(detected, kind)
			if detected == "image" {
				break
			}
		}
		idx = close + 1
	}

	if detected != "" {
		return detected
	}
	return "text"
}

func pickHigherPriorityType(current, next string) string {
	if next == "" {
		return current
	}
	if current == "" {
		return next
	}
	if typePriority(next) < typePriority(current) {
		return next
	}
	return current
}

func typePriority(tp string) int {
	switch tp {
	case "image":
		return 1
	case "video":
		return 2
	case "audio":
		return 3
	case "file":
		return 4
	default:
		return 100
	}
}
