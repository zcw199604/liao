package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	_ = r.ParseForm()

	myUserID := defaultString(r.FormValue("myUserID"), "5be810d731d340f090b098392f9f0a31")
	vipcode := defaultString(r.FormValue("vipcode"), "")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	form := url.Values{}
	form.Set("myUserID", myUserID)
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

	status, body, err := a.postForm(r.Context(), upstreamHistoryURL, form, headers)
	if err != nil {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: "+err.Error()+"\"}")
		return
	}
	if status != http.StatusOK {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: upstream status "+fmt.Sprint(status)+"\"}")
		return
	}

	if a.userInfoCache != nil && strings.TrimSpace(body) != "" {
		var list []map[string]any
		if err := json.Unmarshal([]byte(body), &list); err == nil {
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

			list = a.userInfoCache.BatchEnrichUserInfo(list, idKey)
			list = a.userInfoCache.BatchEnrichWithLastMessage(list, myUserID)

			if enhanced, err := json.Marshal(list); err == nil {
				writeText(w, http.StatusOK, string(enhanced))
				return
			}
		}
	}

	writeText(w, http.StatusOK, body)
}

func (a *App) handleGetFavoriteUserList(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	myUserID := defaultString(r.FormValue("myUserID"), "5be810d731d340f090b098392f9f0a31")
	vipcode := defaultString(r.FormValue("vipcode"), "")
	serverPort := defaultString(r.FormValue("serverPort"), "1001")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	form := url.Values{}
	form.Set("myUserID", myUserID)
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

	status, body, err := a.postForm(r.Context(), upstreamFavoriteURL, form, headers)
	if err != nil {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: "+err.Error()+"\"}")
		return
	}
	if status != http.StatusOK {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"调用上游接口失败: upstream status "+fmt.Sprint(status)+"\"}")
		return
	}

	if a.userInfoCache != nil && strings.TrimSpace(body) != "" {
		var list []map[string]any
		if err := json.Unmarshal([]byte(body), &list); err == nil {
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

			list = a.userInfoCache.BatchEnrichUserInfo(list, idKey)
			list = a.userInfoCache.BatchEnrichWithLastMessage(list, myUserID)

			if enhanced, err := json.Marshal(list); err == nil {
				writeText(w, http.StatusOK, string(enhanced))
				return
			}
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

	form := url.Values{}
	form.Set("referrer_url", referrerURL)
	form.Set("curr_url", currURL)
	form.Set("userid", userID)

	headers := map[string]string{
		"Host":       "v1.chat2019.cn",
		"Origin":     "http://v1.chat2019.cn",
		"Referer":    referer,
		"User-Agent": userAgent,
		"Cookie":     cookieData,
	}

	status, body, err := a.postForm(r.Context(), upstreamReportURL, form, headers)
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		writeText(w, http.StatusInternalServerError, "{\"error\":\"上报失败: "+err.Error()+"\"}")
		return
	}

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

	status, body, err := a.postForm(r.Context(), upstreamMsgURL, form, headers)
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		writeText(w, http.StatusInternalServerError, "{\"error\":\"获取消息历史失败: "+err.Error()+"\"}")
		return
	}

	if a.userInfoCache != nil && strings.TrimSpace(body) != "" {
		var root any
		if err := json.Unmarshal([]byte(body), &root); err == nil {
			// 新格式：包含 contents_list（不改写 body，只写入最后消息缓存）
			if obj, ok := root.(map[string]any); ok {
				if contents, ok := obj["contents_list"]; ok {
					if arr, ok := contents.([]any); ok && len(arr) > 0 {
						if first, ok := arr[0].(map[string]any); ok {
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
					}

					writeText(w, http.StatusOK, body)
					return
				}
			}

			// 旧格式：数组，增强用户信息后返回增强数组
			if arr, ok := root.([]any); ok {
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
		}
	}

	writeText(w, http.StatusOK, body)
}

func (a *App) handleGetImgServer(w http.ResponseWriter, r *http.Request) {
	urlWithTS := upstreamImgServer + "?_=" + fmt.Sprint(time.Now().UnixMilli())
	resp, err := a.httpClient.Get(urlWithTS)
	if err != nil {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"获取图片服务器失败: "+err.Error()+"\"}")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"获取图片服务器失败: upstream status "+resp.Status+"\"}")
		return
	}
	writeText(w, http.StatusOK, string(body))
}

func (a *App) handleUpdateImgServer(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	server := strings.TrimSpace(r.FormValue("server"))
	a.imageServer.SetImgServerHost(server)
	writeText(w, http.StatusOK, "{\"success\":true}")
}

func (a *App) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(110 << 20); err != nil {
		writeText(w, http.StatusInternalServerError, "{\"error\":\"本地存储失败: "+err.Error()+"\"}")
		return
	}

	userID := strings.TrimSpace(r.FormValue("userid"))
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeText(w, http.StatusBadRequest, "{\"error\":\"不支持的文件类型\"}")
		return
	}
	fileHeader := files[0]

	contentType := fileHeader.Header.Get("Content-Type")
	if !a.fileStorage.IsValidMediaType(contentType) {
		writeText(w, http.StatusBadRequest, "{\"error\":\"不支持的文件类型\"}")
		return
	}

	md5Value, err := a.fileStorage.CalculateMD5(fileHeader)
	if err != nil {
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
			writeText(w, http.StatusInternalServerError, "{\"error\":\"本地存储失败: "+err.Error()+"\"}")
			return
		}
		localPath = saved
	}

	imgServerHost := a.imageServer.GetImgServerHost()
	uploadURL := fmt.Sprintf("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgServerHost, userID)

	respBody, err := a.uploadToUpstream(r.Context(), uploadURL, imgServerHost, fileHeader, cookieData, referer, userAgent)
	if err != nil {
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
				availablePort := detectAvailablePort(imgHostClean)
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
					writeText(w, http.StatusOK, string(b))
					return
				}
			}
		}
	}

	writeText(w, http.StatusOK, respBody)
}

func (a *App) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	a.handleUploadMedia(w, r)
}

func (a *App) handleGetCachedImages(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("userid"))
	hostHeader := r.Header.Get("Host")

	cached := a.imageCache.GetCachedImages(userID)
	if cached == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"port": "9006",
			"data": []string{},
		})
		return
	}

	imgServerHost := strings.Split(a.imageServer.GetImgServerHost(), ":")[0]
	availablePort := detectAvailablePort(imgServerHost)
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

	status, body, err := a.postForm(r.Context(), "http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Do", form, headers)
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		writeText(w, http.StatusOK, "{\"state\":\"ERROR\",\"msg\":\""+err.Error()+"\"}")
		return
	}
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

	status, body, err := a.postForm(r.Context(), "http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Cancle", form, headers)
	if err != nil || status != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("upstream status %d", status)
		}
		writeText(w, http.StatusOK, "{\"state\":\"ERROR\",\"msg\":\""+err.Error()+"\"}")
		return
	}
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
	if content == "" {
		return "text"
	}
	if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		path := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(content, "["), "]"))
		switch {
		case strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".bmp"):
			return "image"
		case strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".avi") || strings.HasSuffix(path, ".mov") || strings.HasSuffix(path, ".wmv") || strings.HasSuffix(path, ".flv"):
			return "video"
		case strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".wav") || strings.HasSuffix(path, ".aac") || strings.HasSuffix(path, ".flac"):
			return "audio"
		default:
			return "file"
		}
	}
	return "text"
}
