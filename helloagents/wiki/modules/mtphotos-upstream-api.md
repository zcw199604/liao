# MT Photos 上游 API 文档快照

> 本文档由 `https://mtmt.tech/api/` 加载的 OpenAPI JSON 整理生成，用于 Liao 项目接入 mtPhoto/MT Photos 时查阅。

## 1. 快照信息

| 项 | 值 |
|----|----|
| 文档入口 | [https://mtmt.tech/api/](https://mtmt.tech/api/) |
| OpenAPI JSON | [https://demo.mtmt.tech/api-json](https://demo.mtmt.tech/api-json) |
| API 信息 | [https://demo.mtmt.tech/api-info](https://demo.mtmt.tech/api-info) |
| 本地 OpenAPI 原始快照 | [mtphotos-openapi.json](mtphotos-openapi.json) |
| 本地 API 信息原始快照 | [mtphotos-api-info.json](mtphotos-api-info.json) |
| 整理日期 | 2026-05-29 |
| OpenAPI 标题 | MT Photos API |
| OpenAPI 版本 | 1.0 |
| 服务端版本 | 1.52.0 |
| build | 85756 |
| platform/arch | linux/x64 |
| pg/redis | pg=true, redis=true |
| 路径项数量 | 375 |
| 操作数量 | 423 |
| Schema 数量 | 27 |

## 2. 使用说明

- 文档站顶部的 `API Base URL` 默认是 `https://demo.mtmt.tech`，实际调试时应改成自己的 MT Photos 服务端地址后点击“应用”。
- 普通 API 请求推荐使用 `x-api-key` 请求头；API Key 在 MT Photos 网页端右上角用户菜单的“API 密钥”中创建。
- 需要显示照片、缩略图、视频或下载文件时，先调用 `POST /auth/auth_code`，用 `api_key` 或 `refresh_token` 换取 `auth_code`。
- `auth_code` 有效期为 24 小时内；拼到文件 URL 查询参数时需要先 `encodeURIComponent`。
- 常用媒体 URL：`GET /gateway/{type}/{md5}` 渲染缩略图，`GET /gateway/file/{id}/{md5}` 渲染预览/原文件，`GET /gateway/fileMotion/{id}/{md5}` 渲染安卓动态照片视频，`GET /gateway/fileDownload/{id}/{md5}` 下载原始文件。

## 3. 认证方式

| 名称 | 类型 | 位置 | 说明 |
|------|------|------|------|
| `bearer` | http | bearer | JWT |
| `api-key` | apiKey | header:x-api-key | API Key 认证 (格式: sk_live_xxx) |

## 4. 分组概览

| 分组 | 操作数 |
|------|--------|
| [gateway - 前端API请求主要的入口](#gateway---前端api请求主要的入口) | 191 |
| [api-share 分享管理](#api-share-分享管理) | 38 |
| [api-album 相册](#api-album-相册) | 26 |
| [system-config - 系统配置](#system-config---系统配置) | 23 |
| [gallery 仅限管理员调用](#gallery-仅限管理员调用) | 21 |
| [fileTask 仅限管理员调用](#filetask-仅限管理员调用) | 17 |
| [install-初始化](#install-初始化) | 17 |
| [api-tag 标签管理](#api-tag-标签管理) | 13 |
| [files 仅限管理员调用](#files-仅限管理员调用) | 12 |
| [people-base 仅限管理员调用](#people-base-仅限管理员调用) | 12 |
| [people-descriptor 仅限管理员调用](#people-descriptor-仅限管理员调用) | 11 |
| [users 仅限管理员调用](#users-仅限管理员调用) | 10 |
| [API Key 管理](#api-key-管理) | 6 |
| [API Key管理 - 仅限管理员调用](#api-key管理---仅限管理员调用) | 6 |
| [people 仅限管理员调用](#people-仅限管理员调用) | 6 |
| [服务端信息+用户登录](#服务端信息用户登录) | 5 |
| [folder 仅限管理员调用](#folder-仅限管理员调用) | 5 |
| [file-delete-log - 文件删除日志](#file-delete-log---文件删除日志) | 4 |

## 5. 全量接口目录

| 方法 | 路径 | 分组 | 摘要 | OperationId | 认证 | Deprecated |
|------|------|------|------|-------------|------|------------|
| GET | `/api-info` | 服务端信息+用户登录 | 获取 API 信息 | `AppController_getInfo` | 未声明 | 否 |
| POST | `/auth/rsa` | 服务端信息+用户登录 | 获取RSA公钥 | `AppController_getLoginRSAKeys` | 未声明 | 否 |
| POST | `/auth/login` | 服务端信息+用户登录 | 登录 | `AppController_login` | 未声明 | 否 |
| POST | `/auth/refresh` | 服务端信息+用户登录 | 刷新token | `AppController_refreshToken` | 未声明 | 否 |
| POST | `/auth/auth_code` | 服务端信息+用户登录 | 获取auth_code，有效时间为24小时内 | `AppController_getAuthCode` | 未声明 | 否 |
| GET | `/api-keys` | API Key 管理 | 获取当前用户的 API Key 列表 | `ApiKeyController_findAll` | api-key 或 bearer | 否 |
| POST | `/api-keys` | API Key 管理 | 创建新的 API Key | `ApiKeyController_create` | api-key 或 bearer | 否 |
| GET | `/api-keys/{id}` | API Key 管理 | 获取当前用户的单个 API Key | `ApiKeyController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/api-keys/{id}` | API Key 管理 | 更新当前用户的 API Key | `ApiKeyController_update` | api-key 或 bearer | 否 |
| DELETE | `/api-keys/{id}` | API Key 管理 | 删除当前用户的 API Key | `ApiKeyController_remove` | api-key 或 bearer | 否 |
| POST | `/api-keys/{id}/regenerate` | API Key 管理 | 重新生成当前用户的 API Key | `ApiKeyController_regenerate` | api-key 或 bearer | 否 |
| GET | `/api-keys-admin` | API Key管理 - 仅限管理员调用 | 获取所有用户的 API Key 列表（管理员） | `ApiKeyAdminController_findAll` | api-key 或 bearer | 否 |
| POST | `/api-keys-admin` | API Key管理 - 仅限管理员调用 | 为指定用户创建 API Key（管理员） | `ApiKeyAdminController_create` | api-key 或 bearer | 否 |
| GET | `/api-keys-admin/{id}` | API Key管理 - 仅限管理员调用 | 获取单个 API Key（管理员） | `ApiKeyAdminController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/api-keys-admin/{id}` | API Key管理 - 仅限管理员调用 | 更新 API Key（管理员） | `ApiKeyAdminController_update` | api-key 或 bearer | 否 |
| DELETE | `/api-keys-admin/{id}` | API Key管理 - 仅限管理员调用 | 删除 API Key（管理员） | `ApiKeyAdminController_remove` | api-key 或 bearer | 否 |
| POST | `/api-keys-admin/{id}/regenerate` | API Key管理 - 仅限管理员调用 | 重新生成 API Key（管理员） | `ApiKeyAdminController_regenerate` | api-key 或 bearer | 否 |
| PATCH | `/users/resetSuperAdminPwd` | users 仅限管理员调用 | 重置管理员密码 | `UsersController_resetSuperAdminPwd` | api-key 或 bearer | 否 |
| POST | `/users` | users 仅限管理员调用 | 创建用户 | `UsersController_create` | api-key 或 bearer | 否 |
| GET | `/users` | users 仅限管理员调用 | 用户列表 | `UsersController_findAll` | api-key 或 bearer | 否 |
| PATCH | `/users/{id}` | users 仅限管理员调用 | 更新用户信息 | `UsersController_update` | api-key 或 bearer | 否 |
| DELETE | `/users/{id}` | users 仅限管理员调用 | 删除用户 | `UsersController_remove` | api-key 或 bearer | 否 |
| GET | `/users/{id}` | users 仅限管理员调用 | 用户信息 | `UsersController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/users/resetPwd/{id}` | users 仅限管理员调用 | 重置用户密码 | `UsersController_resetPwd` | api-key 或 bearer | 否 |
| GET | `/users/userIdNameList` | users 仅限管理员调用 | 获取全部用户的 id、uid、username | `UsersController_findIdMap` | api-key 或 bearer | 否 |
| POST | `/users/{id}/avatar` | users 仅限管理员调用 | 管理员上传用户头像 | `UsersController_uploadUserAvatar` | api-key 或 bearer | 否 |
| DELETE | `/users/{id}/avatar` | users 仅限管理员调用 | 删除用户头像 | `UsersController_deleteUserAvatar` | api-key 或 bearer | 否 |
| POST | `/folder` | folder 仅限管理员调用 | 创建文件夹 | `FoldersController_create` | bearer 或 api-key | 否 |
| GET | `/folder` | folder 仅限管理员调用 | 获取文件夹列表 | `FoldersController_findAll` | bearer 或 api-key | 否 |
| GET | `/folder/{id}` | folder 仅限管理员调用 | 获取单个文件夹 | `FoldersController_findOne` | bearer 或 api-key | 否 |
| PATCH | `/folder/{id}` | folder 仅限管理员调用 | 更新文件夹 | `FoldersController_update` | bearer 或 api-key | 否 |
| DELETE | `/folder/{id}` | folder 仅限管理员调用 | 删除文件夹 | `FoldersController_remove` | bearer 或 api-key | 否 |
| POST | `/files/triggerBoundaryEvolution` | files 仅限管理员调用 | 触发边界演变 | `FilesController__triggerBoundaryEvolution` | api-key 或 bearer | 否 |
| POST | `/files/resetFile/{id}` | files 仅限管理员调用 | 重置文件状态 | `FilesController_resetStatus` | api-key 或 bearer | 否 |
| GET | `/files/faceReg/{md5}` | files 仅限管理员调用 | 根据MD5获取文件人脸描述符 | `FilesController_findFilePeopleDescriptorByMd5` | api-key 或 bearer | 否 |
| GET | `/files/count/{type}/{md5}` | files 仅限管理员调用 | 按MD5统计文件数量 | `FilesController_countFileByMD5` | api-key 或 bearer | 否 |
| POST | `/files/ocr/info` | files 仅限管理员调用 | 获取OCR任务信息 | `FilesController_getOcrInfo` | api-key 或 bearer | 否 |
| POST | `/files/ocr/task` | files 仅限管理员调用 | 获取OCR任务列表 | `FilesController_getOcrTask` | api-key 或 bearer | 否 |
| POST | `/files/ocr/result` | files 仅限管理员调用 | 提交OCR识别结果 | `FilesController_saveOcrResult` | api-key 或 bearer | 否 |
| POST | `/files/ocr/resetStatus` | files 仅限管理员调用 | 重置OCR状态 | `FilesController_resetOcrStatus` | api-key 或 bearer | 否 |
| GET | `/files/{id}` | files 仅限管理员调用 | 获取单个文件信息 | `FilesController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/files/{id}` | files 仅限管理员调用 | 更新文件信息 | `FilesController_update` | api-key 或 bearer | 否 |
| POST | `/files/broTaskFileList` | files 仅限管理员调用 | 获取浏览器任务文件列表 | `FilesController_getBrowserTaskFileList` | api-key 或 bearer | 是 |
| POST | `/files/findInGpsDistrict` | files 仅限管理员调用 | 根据行政区划或坐标测试地理位置识别 | `FilesController_findInGpsDistrict` | api-key 或 bearer | 否 |
| POST | `/fileTask/addTask` | fileTask 仅限管理员调用 | 创建后台任务 | `FileTaskController_addTask` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/active` | fileTask 仅限管理员调用 | 获取正在执行的任务列表 | `FileTaskController_getActiveJobs` | api-key 或 bearer | 否 |
| GET | `/fileTask/job/subData` | fileTask 仅限管理员调用 | 获取任务进度子数据 | `FileTaskController_getJobSubData` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/completed` | fileTask 仅限管理员调用 | 获取已完成任务列表 | `FileTaskController_getCompleted` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/waiting` | fileTask 仅限管理员调用 | 获取等待中任务列表 | `FileTaskController_getWaiting` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/paused` | fileTask 仅限管理员调用 | 获取已暂停任务列表 | `FileTaskController_getPaused` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/failed` | fileTask 仅限管理员调用 | 获取失败任务列表 | `FileTaskController_getFailed` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/isPaused` | fileTask 仅限管理员调用 | 检查任务队列是否已暂停 | `FileTaskController_isPaused` | api-key 或 bearer | 否 |
| POST | `/fileTask/jobs/pause` | fileTask 仅限管理员调用 | 暂停任务队列 | `FileTaskController_pause` | api-key 或 bearer | 否 |
| POST | `/fileTask/jobs/resume` | fileTask 仅限管理员调用 | 恢复任务队列 | `FileTaskController_resume` | api-key 或 bearer | 否 |
| GET | `/fileTask/jobs/Counts` | fileTask 仅限管理员调用 | 获取各状态任务数量统计 | `FileTaskController_getJobCounts` | api-key 或 bearer | 否 |
| GET | `/fileTask/resetAllGpsInfo` | fileTask 仅限管理员调用 | 重置所有GPS信息 | `FileTaskController_resetAllGpsInfo` | api-key 或 bearer | 否 |
| GET | `/fileTask/checkLicense` | fileTask 仅限管理员调用 | 检查许可证状态 | `FileTaskController_checkCpInfo` | api-key 或 bearer | 否 |
| GET | `/fileTask/client/{name}` | fileTask 仅限管理员调用 | 获取浏览器辅助处理模型文件 | `FileTaskController_getTfTaskFiles` | api-key 或 bearer | 是 |
| GET | `/fileTask/client/dist/{name}` | fileTask 仅限管理员调用 | 获取浏览器辅助处理模型文件（dist目录） | `FileTaskController_getTfTaskFiles2` | api-key 或 bearer | 是 |
| GET | `/fileTask/client/dist/{type}/{name}` | fileTask 仅限管理员调用 | 获取浏览器辅助处理模型文件（dist子目录） | `FileTaskController_getTfTaskFiles3` | api-key 或 bearer | 是 |
| GET | `/fileTask/{id}` | fileTask 仅限管理员调用 | 根据ID获取任务详情 | `FileTaskController_findOne` | api-key 或 bearer | 否 |
| GET | `/gallery/rootDirs` | gallery 仅限管理员调用 | 获取根目录列表 | `GalleryController_findRootDirs` | bearer 或 api-key | 否 |
| GET | `/gallery/subDirs` | gallery 仅限管理员调用 | 获取子目录列表 | `GalleryController_findSubDirs` | bearer 或 api-key | 否 |
| POST | `/gallery/findDuplicateFiles` | gallery 仅限管理员调用 | 查找重复文件 | `GalleryController_findDuplicateFilesWithGalleryIds` | bearer 或 api-key | 否 |
| GET | `/gallery/findDeletedFiles` | gallery 仅限管理员调用 | 查找已删除文件 | `GalleryController_findDeletedFiles` | bearer 或 api-key | 否 |
| POST | `/gallery/exportDeletedFiles` | gallery 仅限管理员调用 | 导出已删除文件的预览图 | `GalleryController_exportDeletedFiles` | bearer 或 api-key | 否 |
| POST | `/gallery/exportDeletedFiles/stat` | gallery 仅限管理员调用 | 导出已删除文件的预览图 - 进度查询 | `GalleryController_exportDeletedFilesStat` | bearer 或 api-key | 否 |
| POST | `/gallery/deleteDuplicateFiles` | gallery 仅限管理员调用 | 删除重复文件 | `GalleryController_deleteDuplicateFiles` | bearer 或 api-key | 否 |
| POST | `/gallery/folderPathRebase` | gallery 仅限管理员调用 | 文件夹路径重置检查 | `GalleryController_folderPathRebase` | bearer 或 api-key | 否 |
| POST | `/gallery` | gallery 仅限管理员调用 | 创建图库 | `GalleryController_create` | bearer 或 api-key | 否 |
| GET | `/gallery` | gallery 仅限管理员调用 | 获取所有图库 | `GalleryController_findAll` | bearer 或 api-key | 否 |
| GET | `/gallery/all` | gallery 仅限管理员调用 | 获取所有图库（含隐藏） | `GalleryController_findAllWithHidden` | bearer 或 api-key | 否 |
| GET | `/gallery/galleryUsers` | gallery 仅限管理员调用 | 获取图库用户列表 | `GalleryController_findAllGalleryUsers` | bearer 或 api-key | 否 |
| GET | `/gallery/stat/{id}` | gallery 仅限管理员调用 | 获取图库统计信息 | `GalleryController_statOne` | bearer 或 api-key | 否 |
| GET | `/gallery/scan/{id}` | gallery 仅限管理员调用 | 扫描图库 | `GalleryController_scanGallery` | bearer 或 api-key | 否 |
| GET | `/gallery/{id}` | gallery 仅限管理员调用 | 获取单个图库信息 | `GalleryController_findOne` | bearer 或 api-key | 否 |
| PATCH | `/gallery/{id}` | gallery 仅限管理员调用 | 更新图库信息 | `GalleryController_update` | bearer 或 api-key | 否 |
| DELETE | `/gallery/{id}` | gallery 仅限管理员调用 | 删除图库 | `GalleryController_remove` | bearer 或 api-key | 否 |
| POST | `/gallery/updateWeights` | gallery 仅限管理员调用 | 更新图库权重 | `GalleryController_updateWeights` | bearer 或 api-key | 否 |
| POST | `/gallery/createFolders` | gallery 仅限管理员调用 | 批量创建文件夹 | `GalleryController_createFolders` | bearer 或 api-key | 否 |
| POST | `/gallery/func_exclude` | gallery 仅限管理员调用 | 获取功能排除的图库ID | `GalleryController_getFuncExcludeIds` | bearer 或 api-key | 否 |
| POST | `/gallery/skippedFolderLogs` | gallery 仅限管理员调用 | 获取跳过扫描的文件夹日志 | `GalleryController_getSkippedFolderLogs` | bearer 或 api-key | 否 |
| POST | `/people-descriptor` | people-descriptor 仅限管理员调用 | 创建人物特征描述 | `PeopleDescriptorController_create` | api-key 或 bearer | 否 |
| GET | `/people-descriptor/info` | people-descriptor 仅限管理员调用 | 获取人脸识别任务信息（浏览器辅助识别用） | `PeopleDescriptorController_getInfo` | api-key 或 bearer | 是 |
| POST | `/people-descriptor/resetFileStatus` | people-descriptor 仅限管理员调用 | 重置文件人脸识别状态 | `PeopleDescriptorController_resetFileStatus` | api-key 或 bearer | 否 |
| POST | `/people-descriptor/itemDistV2` | people-descriptor 仅限管理员调用 | 计算两个特征描述之间的距离（V2版本） | `PeopleDescriptorController_itemDistV2` | api-key 或 bearer | 否 |
| POST | `/people-descriptor/faceRegTask` | people-descriptor 仅限管理员调用 | 获取人脸识别任务列表（浏览器辅助识别用） | `PeopleDescriptorController_getTfTaskFiles` | api-key 或 bearer | 是 |
| POST | `/people-descriptor/faceRegResult` | people-descriptor 仅限管理员调用 | 保存人脸识别结果（浏览器辅助识别用） | `PeopleDescriptorController_saveFaceRegResult` | api-key 或 bearer | 是 |
| POST | `/people-descriptor/findDescriptorOfFileForPeople` | people-descriptor 仅限管理员调用 | 查找人物对应的特征描述 | `PeopleDescriptorController_findDescriptorOfFileForPeople` | api-key 或 bearer | 否 |
| POST | `/people-descriptor/findLikelyBase0Descriptor` | people-descriptor 仅限管理员调用 | 查找相似的未匹配人物特征描述 | `PeopleDescriptorController_adminFindLikelyNoMatchedDescriptor` | api-key 或 bearer | 否 |
| GET | `/people-descriptor/findDescriptorOfFile/{fileId}` | people-descriptor 仅限管理员调用 | 获取文件的人脸特征描述列表 | `PeopleDescriptorController_findDescriptorOfFile` | api-key 或 bearer | 否 |
| GET | `/people-descriptor/{id}` | people-descriptor 仅限管理员调用 | 根据ID获取人物特征描述 | `PeopleDescriptorController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/people-descriptor/{id}` | people-descriptor 仅限管理员调用 | 更新人物特征描述 | `PeopleDescriptorController_update` | api-key 或 bearer | 否 |
| POST | `/people` | people 仅限管理员调用 | 创建人物 | `PeopleController_create` | bearer 或 api-key | 否 |
| GET | `/people` | people 仅限管理员调用 | 获取所有人物列表 | `PeopleController_findAll` | bearer 或 api-key | 否 |
| GET | `/people/base/{id}` | people 仅限管理员调用 | 根据人物基础ID获取人物列表 | `PeopleController_findById` | bearer 或 api-key | 否 |
| GET | `/people/{id}` | people 仅限管理员调用 | 根据ID获取人物详情 | `PeopleController_findOne` | bearer 或 api-key | 否 |
| PATCH | `/people/{id}` | people 仅限管理员调用 | 更新人物信息 | `PeopleController_update` | bearer 或 api-key | 否 |
| DELETE | `/people/{id}` | people 仅限管理员调用 | 删除人物 | `PeopleController_remove` | bearer 或 api-key | 否 |
| GET | `/system-config` | system-config - 系统配置 | 获取所有系统配置 - adminOnly | `SystemConfigController_findAll` | api-key 或 bearer | 否 |
| PATCH | `/system-config` | system-config - 系统配置 | 更新系统配置 - adminOnly | `SystemConfigController_updateByValue` | api-key 或 bearer | 否 |
| GET | `/system-config/{key}` | system-config - 系统配置 | 根据key获取系统配置 - adminOnly | `SystemConfigController_findByKey` | api-key 或 bearer | 否 |
| POST | `/system-config/patchMulti` | system-config - 系统配置 | 批量修改图库设置配置值 - adminOnly | `SystemConfigController_patchMultiForFront` | api-key 或 bearer | 否 |
| POST | `/system-config/getFFmpegHWList` | system-config - 系统配置 | 获取FFmpeg硬件加速列表 - adminOnly | `SystemConfigController_getFFmpeg_HWList` | api-key 或 bearer | 否 |
| POST | `/system-config/pgDump` | system-config - 系统配置 | 数据库备份 - adminOnly | `SystemConfigController_pgDump` | api-key 或 bearer | 否 |
| POST | `/system-config/systemStatus` | system-config - 系统配置 | 获取系统状态 | `SystemConfigController_systemStatus` | api-key 或 bearer | 否 |
| POST | `/system-config/changeTableVecLength` | system-config - 系统配置 | 修改数据库向量的长度 - adminOnly | `SystemConfigController_changeTableVecLength` | api-key 或 bearer | 否 |
| POST | `/system-config/getTableVecLength` | system-config - 系统配置 | 获取数据库向量长度 - adminOnly | `SystemConfigController_getTableVecLength` | api-key 或 bearer | 否 |
| POST | `/system-config/test/ocrApi` | system-config - 系统配置 | 测试OCR API配置 - adminOnly | `SystemConfigController_testOcrApiConfig` | api-key 或 bearer | 否 |
| POST | `/system-config/db/prepareCLIP` | system-config - 系统配置 | 准备CLIP表 - adminOnly | `SystemConfigController_prepareForClip` | api-key 或 bearer | 否 |
| POST | `/system-config/db/prepareFaceRegV2` | system-config - 系统配置 | 准备人脸识别V2表 - adminOnly | `SystemConfigController_prepareFaceRegV2` | api-key 或 bearer | 否 |
| POST | `/system-config/switchUseFaceRegV2` | system-config - 系统配置 | 切换人脸识别版本 - adminOnly | `SystemConfigController_switchUseFaceRegV2` | api-key 或 bearer | 否 |
| POST | `/system-config/configInfo` | system-config - 系统配置 | 获取配置信息 - adminOnly | `SystemConfigController_configInfo` | api-key 或 bearer | 否 |
| POST | `/system-config/dbReIndex` | system-config - 系统配置 | 重建数据库索引 - adminOnly | `SystemConfigController_dbReIndex` | api-key 或 bearer | 否 |
| POST | `/system-config/dbReIndexInfo` | system-config - 系统配置 | 获取数据库重建索引进度 - adminOnly | `SystemConfigController_dbReIndexInfo` | api-key 或 bearer | 否 |
| POST | `/system-config/dbReIndexForTZ` | system-config - 系统配置 | 重新生成时区相关的index索引 - adminOnly | `SystemConfigController_dbReIndexForTZ` | api-key 或 bearer | 否 |
| POST | `/system-config/getLibheifVersion` | system-config - 系统配置 | 获取libheif版本 - adminOnly | `SystemConfigController__getLibheifVersion` | api-key 或 bearer | 否 |
| POST | `/system-config/libheifVersion` | system-config - 系统配置 | 切换libheif版本 - adminOnly | `SystemConfigController__switchLibheifVersion` | api-key 或 bearer | 否 |
| POST | `/system-config/offlineID` | system-config - 系统配置 | 获取离线ID - adminOnly | `SystemConfigController_postOfflineID` | api-key 或 bearer | 否 |
| POST | `/system-config/verifyAuthOnlineInBrowser` | system-config - 系统配置 | 在线验证授权 - adminOnly | `SystemConfigController_verifyAuthOnlineInBrowser` | api-key 或 bearer | 否 |
| POST | `/system-config/getLogs` | system-config - 系统配置 | 获取日志 - adminOnly | `SystemConfigController_getLogsInMem` | api-key 或 bearer | 否 |
| POST | `/system-config/clearLogs` | system-config - 系统配置 | 清空日志 - adminOnly | `SystemConfigController_clearLogsInMem` | api-key 或 bearer | 否 |
| POST | `/file-delete-log` | file-delete-log - 文件删除日志 | 创建文件删除日志 | `FileDeleteLogController_create` | api-key 或 bearer | 否 |
| GET | `/file-delete-log` | file-delete-log - 文件删除日志 | 分页查询文件删除日志 | `FileDeleteLogController_findAll` | api-key 或 bearer | 否 |
| GET | `/file-delete-log/{id}` | file-delete-log - 文件删除日志 | 根据ID查询删除日志 | `FileDeleteLogController_findOne` | api-key 或 bearer | 否 |
| POST | `/file-delete-log/clearData` | file-delete-log - 文件删除日志 | 清空所有删除日志 | `FileDeleteLogController_clearAllData` | api-key 或 bearer | 否 |
| POST | `/api-album` | api-album 相册 | 新建相册 | `AlbumController_create` | api-key 或 bearer | 否 |
| GET | `/api-album` | api-album 相册 | 我的相册列表 | `AlbumController_findAll` | api-key 或 bearer | 否 |
| GET | `/api-album/{id}` | api-album 相册 | 相册详情 | `AlbumController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/api-album/{id}` | api-album 相册 | 修改相册 | `AlbumController_update` | api-key 或 bearer | 否 |
| PUT | `/api-album/{id}` | api-album 相册 | 修改相册 - patch兼容 | `AlbumController_update_put` | api-key 或 bearer | 否 |
| DELETE | `/api-album/{id}` | api-album 相册 | 删除相册 | `AlbumController_remove` | api-key 或 bearer | 否 |
| GET | `/api-album/files/{id}` | api-album 相册 | 相册文件列表 | `AlbumController_findAlbumFiles` | api-key 或 bearer | 是 |
| GET | `/api-album/filesV2/{id}` | api-album 相册 | 相册文件列表 - 时间线 | `AlbumController_findAlbumFilesV2` | api-key 或 bearer | 否 |
| GET | `/api-album/ignoreFiles/{id}` | api-album 相册 | 相册排除的文件列表 - 时间线 - 曾经在相册内手动移出的照片 | `AlbumController_findAlbumIgnoreFilesV2` | api-key 或 bearer | 否 |
| GET | `/api-album/filesFlat/{id}` | api-album 相册 | 相册文件列表 - 给PhotosFlatList用的精简数据版 | `AlbumController_findAlbumFilesFlat` | api-key 或 bearer | 否 |
| GET | `/api-album/fileInAlbums/{id}` | api-album 相册 | 文件在哪些相册中 - 返回相册id | `AlbumController_fileInAlbums` | api-key 或 bearer | 否 |
| GET | `/api-album/fileInAlbumsList/{id}` | api-album 相册 | 文件在哪些相册中 - 返回相册信息 | `AlbumController_fileInAlbumsList` | api-key 或 bearer | 否 |
| POST | `/api-album/checkForFavorites` | api-album 相册 | 检查【收藏夹】 相册是否已经创建过 | `AlbumController_checkAlbumForFav` | api-key 或 bearer | 否 |
| POST | `/api-album/addFileToAlbum` | api-album 相册 | 添加文件至相册中 | `AlbumController_addFileToAlbum` | api-key 或 bearer | 否 |
| POST | `/api-album/removeFileFromAlbum` | api-album 相册 | 将文件从相册中删除 | `AlbumController_removeFileFromAlbum` | api-key 或 bearer | 否 |
| GET | `/api-album/link/{id}` | api-album 相册 | 相册的自动更新配置 | `AlbumController_findAutoLinkList` | api-key 或 bearer | 否 |
| POST | `/api-album/link/{id}` | api-album 相册 | 添加 相册 自动配置 | `AlbumController_addAutoLink` | api-key 或 bearer | 否 |
| DELETE | `/api-album/link/{id}` | api-album 相册 | 删除 相册 自动配置 | `AlbumController_delAutoLink` | api-key 或 bearer | 否 |
| POST | `/api-album/linkSyncFiles/{id}` | api-album 相册 | 相册 自动关联 更新文件 | `AlbumController_syncAutoLink` | api-key 或 bearer | 否 |
| POST | `/api-album/hlinkAlbum` | api-album 相册 | 相册硬链接 - 触发同步 | `AlbumController_hlinkAlbum` | api-key 或 bearer | 否 |
| POST | `/api-album/addAlbumHLink` | api-album 相册 | 相册 硬链接 创建- admin only | `AlbumController_addAlbumHLink` | api-key 或 bearer | 否 |
| POST | `/api-album/updateAlbumHLink` | api-album 相册 | 相册 硬链接 更新- admin only | `AlbumController_updateAlbumHLink` | api-key 或 bearer | 否 |
| POST | `/api-album/delAlbumHLink` | api-album 相册 | 相册 硬链接 - admin only | `AlbumController_delAlbumHLink` | api-key 或 bearer | 否 |
| GET | `/api-album/getAlbumHardLinkByAlbumId/{id}` | api-album 相册 | 相册 硬链接 | `AlbumController_getAlbumHardLinkByAlbumId` | api-key 或 bearer | 否 |
| GET | `/api-album/getAlbumHardLinkById/{id}` | api-album 相册 | 相册 硬链接 | `AlbumController_getAlbumHardLinkById` | api-key 或 bearer | 否 |
| POST | `/api-album/findAllForHardLink/list` | api-album 相册 | 硬链接 显示的全部相册列表 - admin only | `AlbumController_findAllForHardLink` | api-key 或 bearer | 否 |
| GET | `/people-base/count` | people-base 仅限管理员调用 | 获取人物基础总数 | `PeopleBaseController_count` | bearer 或 api-key | 否 |
| GET | `/people-base/findForGenPeople` | people-base 仅限管理员调用 | 获取待生成人物的PeopleBase列表 | `PeopleBaseController_findForGenPeople` | bearer 或 api-key | 否 |
| GET | `/people-base/distance` | people-base 仅限管理员调用 | 计算两个人物基础之间的距离 | `PeopleBaseController_baseIdDistance` | bearer 或 api-key | 否 |
| GET | `/people-base/findAllPeopleBaseForMerge` | people-base 仅限管理员调用 | 分页获取所有人物基础列表（用于合并） | `PeopleBaseController_findAllPeopleBase` | bearer 或 api-key | 否 |
| GET | `/people-base/findAllMergerPeopleBase` | people-base 仅限管理员调用 | 获取所有已合并的人物基础列表 | `PeopleBaseController_findAllMergerPeopleBase` | bearer 或 api-key | 否 |
| GET | `/people-base/findPeopleBaseFiles` | people-base 仅限管理员调用 | 根据人物基础ID获取关联的文件列表 | `PeopleBaseController_findPeopleBaseFiles` | bearer 或 api-key | 否 |
| POST | `/people-base/findFileMD5ByFileIds` | people-base 仅限管理员调用 | 根据文件ID列表获取MD5值（用于显示封面） | `PeopleBaseController_findMD5ByIds` | bearer 或 api-key | 否 |
| POST | `/people-base/findBaseInfoByIds` | people-base 仅限管理员调用 | 根据人物基础ID列表获取基础信息 - adminOnly | `PeopleBaseController_findBaseInfoByIds` | bearer 或 api-key | 否 |
| POST | `/people-base/adminMergeBaseIds` | people-base 仅限管理员调用 | 合并人物基础 - adminOnly | `PeopleBaseController_adminMergeBaseIds` | bearer 或 api-key | 否 |
| POST | `/people-base/adminSetBaseId` | people-base 仅限管理员调用 | 设置人物基础（合并或更新名称）- adminOnly | `PeopleBaseController_adminSetBaseId` | bearer 或 api-key | 否 |
| GET | `/people-base/baseInFileInfo` | people-base 仅限管理员调用 | 获取人物基础对应照片识别的人脸信息 - adminOnly | `PeopleBaseController_peopleInFileInfo` | bearer 或 api-key | 否 |
| POST | `/people-base/getNameFromPeople` | people-base 仅限管理员调用 | 根据人物基础ID获取人物名称 - adminOnly | `PeopleBaseController_getNameFromPeople` | bearer 或 api-key | 否 |
| POST | `/api-tag` | api-tag 标签管理 | 创建标签 | `TagController_create` | api-key 或 bearer | 否 |
| GET | `/api-tag` | api-tag 标签管理 | 获取标签列表 | `TagController_findAll` | api-key 或 bearer | 否 |
| GET | `/api-tag/tag/{id}` | api-tag 标签管理 | 获取标签详情 | `TagController_findTagDetail` | api-key 或 bearer | 否 |
| PATCH | `/api-tag/tag/{id}` | api-tag 标签管理 | 更新标签（PATCH） | `TagController_updateTag` | api-key 或 bearer | 否 |
| PUT | `/api-tag/tag/{id}` | api-tag 标签管理 | 更新标签（PUT） | `TagController_updateTag_put` | api-key 或 bearer | 否 |
| GET | `/api-tag/files/{id}` | api-tag 标签管理 | 获取标签关联的文件列表 | `TagController_findTagFiles` | api-key 或 bearer | 否 |
| POST | `/api-tag/editFileTag` | api-tag 标签管理 | 编辑文件标签 | `TagController_editFileTag` | api-key 或 bearer | 否 |
| POST | `/api-tag/fileAddTags` | api-tag 标签管理 | 批量为文件添加标签 | `TagController_fileAddTags` | api-key 或 bearer | 否 |
| POST | `/api-tag/fileDelTagsInDb` | api-tag 标签管理 | 批量删除文件标签（仅数据库） | `TagController_fileDelTagsInDb` | api-key 或 bearer | 否 |
| POST | `/api-tag/saveToExif` | api-tag 标签管理 | 批量保存标签到 EXIF | `TagController_saveToExif` | api-key 或 bearer | 否 |
| POST | `/api-tag/hideTag` | api-tag 标签管理 | 隐藏空标签 | `TagController_hideTag` | api-key 或 bearer | 否 |
| POST | `/api-tag/hideEmptyTags` | api-tag 标签管理 | 隐藏所有空标签 | `TagController_hideEmptyTags` | api-key 或 bearer | 否 |
| POST | `/api-tag/tagNames` | api-tag 标签管理 | 根据 ID 获取标签名称 | `TagController_getTagNames` | api-key 或 bearer | 否 |
| GET | `/gateway/test` | gateway - 前端API请求主要的入口 | 测试接口 | `GatewayController_test` | api-key 或 bearer | 否 |
| GET | `/gateway/userInfo` | gateway - 前端API请求主要的入口 | 用户信息-当前登录用户 | `GatewayController_getUserInfo` | api-key 或 bearer | 否 |
| GET | `/gateway/filesInTimeline` | gateway - 前端API请求主要的入口 | 所有文件 | `GatewayController_findAllFiles` | api-key 或 bearer | 是 |
| GET | `/gateway/filesInTimelineV2` | gateway - 前端API请求主要的入口 | 所有文件-时间线 | `GatewayController_findAllFilesV2` | api-key 或 bearer | 否 |
| GET | `/gateway/timeline` | gateway - 前端API请求主要的入口 | 照片-时间线按月分组统计数 | `GatewayController_getTimelineData` | api-key 或 bearer | 否 |
| POST | `/gateway/timelineMonth` | gateway - 前端API请求主要的入口 | 照片-时间线 月数据 | `GatewayController_getTimelineMonthData` | api-key 或 bearer | 否 |
| GET | `/gateway/myGalleryList` | gateway - 前端API请求主要的入口 | 用户的图库列表 | `GatewayController_userGalleryList` | api-key 或 bearer | 否 |
| POST | `/gateway/galleryNames` | gateway - 前端API请求主要的入口 | 获取图库名称 | `GatewayController_getGalleryNames` | api-key 或 bearer | 否 |
| POST | `/gateway/dayFileMore` | gateway - 前端API请求主要的入口 | 单天剩余文件 | `GatewayController_findDayFileMore` | api-key 或 bearer | 否 |
| POST | `/gateway/dayFiles` | gateway - 前端API请求主要的入口 | 某一天的所有文件 | `GatewayController_dayAllFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/filesInfo` | gateway - 前端API请求主要的入口 | 下载前查询文件信息 | `GatewayController_findFilesInfo` | api-key 或 bearer | 否 |
| GET | `/gateway/filesInTimelineCount` | gateway - 前端API请求主要的入口 | 时间线中所有文件的数量 | `GatewayController_findAllFilesNum` | api-key 或 bearer | 否 |
| POST | `/gateway/user/profile` | gateway - 前端API请求主要的入口 | 更新个人资料 | `GatewayController_updateProfile` | api-key 或 bearer | 否 |
| GET | `/gateway/avatar/{fileName}` | gateway - 前端API请求主要的入口 | 显示用户头像 | `GatewayController_renderAvatar` | api-key 或 bearer | 否 |
| POST | `/gateway/avatar` | gateway - 前端API请求主要的入口 | 上传头像 | `GatewayController_uploadAvatar` | api-key 或 bearer | 否 |
| POST | `/gateway/folderFilesInDisk` | gateway - 前端API请求主要的入口 | 查看文件夹文件 - 实时读取硬盘文件列表 | `GatewayControllerPart1_folderFilesInDisk` | api-key 或 bearer | 否 |
| POST | `/gateway/annualData` | gateway - 前端API请求主要的入口 | 获取年度统计数据 | `GatewayControllerPart1_annualData` | api-key 或 bearer | 否 |
| POST | `/gateway/refreshFileDescriptorBatch` | gateway - 前端API请求主要的入口 | 刷新照片人脸 | `GatewayControllerPart1_refreshFileDescriptor` | api-key 或 bearer | 否 |
| POST | `/gateway/getTranscodeError` | gateway - 前端API请求主要的入口 | 查询转码错误信息 | `GatewayControllerPart1_getTranscodeError` | api-key 或 bearer | 否 |
| POST | `/gateway/addFaceRect` | gateway - 前端API请求主要的入口 | 手动添加人脸识别框 | `GatewayControllerPart1_addFaceRect` | api-key 或 bearer | 否 |
| GET | `/gateway/fileInfo/{id}/{md5}` | gateway - 前端API请求主要的入口 | 显示文件的详细信息 | `GatewayControllerPart2_getFileDetail` | api-key 或 bearer | 否 |
| GET | `/gateway/fileInfoById/{id}` | gateway - 前端API请求主要的入口 | 显示文件的详细信息 | `GatewayControllerPart2_getFileServerPath` | api-key 或 bearer | 否 |
| GET | `/gateway/exifInfo/{id}` | gateway - 前端API请求主要的入口 | 显示文件的exif信息 | `GatewayControllerPart2_fileExifInfo` | api-key 或 bearer | 否 |
| GET | `/gateway/fileTags/{id}` | gateway - 前端API请求主要的入口 | 文件的标签列表 | `GatewayControllerPart2_findFileTags` | api-key 或 bearer | 否 |
| POST | `/gateway/extra/make` | gateway - 前端API请求主要的入口 | 获取照片包含的相机品牌列表 | `GatewayControllerPart2_fileExtraMake` | api-key 或 bearer | 否 |
| POST | `/gateway/extra/models` | gateway - 前端API请求主要的入口 | 获取照片包含的设备列表 | `GatewayControllerPart2_fileExtraModelsWithMake` | api-key 或 bearer | 否 |
| GET | `/gateway/extra/models` | gateway - 前端API请求主要的入口 | 获取照片包含的设备列表 | `GatewayControllerPart2_fileExtraModels` | api-key 或 bearer | 否 |
| POST | `/gateway/extra/lens` | gateway - 前端API请求主要的入口 | 获取照片包含的镜头列表 | `GatewayControllerPart2_fileExtraLensWithModel` | api-key 或 bearer | 否 |
| GET | `/gateway/extra/lens` | gateway - 前端API请求主要的入口 | 获取照片包含的镜头列表 | `GatewayControllerPart2_fileExtraLens` | api-key 或 bearer | 否 |
| POST | `/gateway/extra/placeL1` | gateway - 前端API请求主要的入口 | 获取地点列表 - 省 | `GatewayControllerPart2_filePlaceL1` | api-key 或 bearer | 否 |
| POST | `/gateway/extra/placeL2` | gateway - 前端API请求主要的入口 | 获取地点列表 - 市 | `GatewayControllerPart2_filePlaceL2` | api-key 或 bearer | 否 |
| POST | `/gateway/extra/placeL3` | gateway - 前端API请求主要的入口 | 获取地点列表 - 区 | `GatewayControllerPart2_filePlaceL3` | api-key 或 bearer | 否 |
| GET | `/gateway/ocrInfo/{id}` | gateway - 前端API请求主要的入口 | 显示文件的OCR结果 | `GatewayControllerPart2_fileOcrInfo` | api-key 或 bearer | 否 |
| POST | `/gateway/filesPath` | gateway - 前端API请求主要的入口 | 获取指定ids文件的地址 | `GatewayControllerPart2_filesPath` | api-key 或 bearer | 否 |
| POST | `/gateway/filesInMD5` | gateway - 前端API请求主要的入口 | 根据MD5查询文件列表 | `GatewayControllerPart2_filesInMD5` | api-key 或 bearer | 否 |
| GET | `/gateway/refreshFileThumbs/{id}` | gateway - 前端API请求主要的入口 | 刷新文件的缩略图 | `GatewayControllerPart2_refreshFileThumbs` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadFileThumbs/{id}` | gateway - 前端API请求主要的入口 | 上传文件缩略图 | `GatewayControllerPart2_uploadFileThumbs` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadFileThumbsForApp/{id}` | gateway - 前端API请求主要的入口 | App上传文件缩略图 | `GatewayControllerPart2_uploadFileThumbsForApp` | api-key 或 bearer | 否 |
| POST | `/gateway/HDThumbsConfig` | gateway - 前端API请求主要的入口 | 获取高清缩略图配置 | `GatewayControllerPart2_getHDThumbsConfig` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadFileHDThumbs/{id}` | gateway - 前端API请求主要的入口 | 上传高清缩略图 | `GatewayControllerPart2_uploadFileHdThumbs` | api-key 或 bearer | 否 |
| POST | `/gateway/transcode` | gateway - 前端API请求主要的入口 | 触发视频转码 | `GatewayControllerPart2_transcodeFile` | api-key 或 bearer | 否 |
| GET | `/gateway/fileInfoRT/{id}` | gateway - 前端API请求主要的入口 | 获取文件最新EXIF信息 | `GatewayControllerPart2_getFileInfoRealTime` | api-key 或 bearer | 否 |
| POST | `/gateway/refreshFileDescriptor` | gateway - 前端API请求主要的入口 | 刷新照片人脸 | `GatewayControllerPart2_refreshFileDescriptor` | api-key 或 bearer | 否 |
| POST | `/gateway/fileStat/{id}/{md5}` | gateway - 前端API请求主要的入口 | 检查文件是否存在 | `GatewayControllerPart2_statOneFile` | api-key 或 bearer | 否 |
| GET | `/gateway/fileStreamLink/{id}` | gateway - 前端API请求主要的入口 | 获取串流地址 | `GatewayControllerPart2_fileStreamLink` | api-key 或 bearer | 否 |
| GET | `/gateway/stream/{auth_code}/{name}` | gateway - 前端API请求主要的入口 | 下载文件原图 | `GatewayControllerPart2_fileStreamPlay` | api-key 或 bearer | 否 |
| GET | `/gateway/streamV2/{name}` | gateway - 前端API请求主要的入口 | 下载文件原图V2 | `GatewayControllerPart2_fileStreamPlayV2` | api-key 或 bearer | 否 |
| GET | `/gateway/file/{id}/{md5}` | gateway - 前端API请求主要的入口 | 显示文件原图 | `GatewayControllerPart2_renderFile` | api-key 或 bearer | 否 |
| GET | `/gateway/fileForApi/{id}/{md5}` | gateway - 前端API请求主要的入口 | 显示文件的大图 - 已废弃 | `GatewayControllerPart2_renderFileForOpen` | api-key 或 bearer | 是 |
| GET | `/gateway/fileMotion/{id}/{md5}` | gateway - 前端API请求主要的入口 | 显示动态照片的视频部分 | `GatewayControllerPart2_renderMotionPhoto` | api-key 或 bearer | 否 |
| GET | `/gateway/flv/{id}/{md5}` | gateway - 前端API请求主要的入口 | 视频实时转码为flv | `GatewayControllerPart2_renderFileFlv` | api-key 或 bearer | 否 |
| GET | `/gateway/jpeg/{md5}` | gateway - 前端API请求主要的入口 | 显示heic图片的详情 | `GatewayControllerPart2_renderImgWebp` | api-key 或 bearer | 否 |
| GET | `/gateway/fileDownload/{id}/{md5}` | gateway - 前端API请求主要的入口 | 下载文件的原图 | `GatewayControllerPart2_downloadFile` | api-key 或 bearer | 否 |
| POST | `/gateway/fileDownloadStat/{id}/{md5}` | gateway - 前端API请求主要的入口 | 获取下载文件的大小 | `GatewayControllerPart2_downloadStatFile` | api-key 或 bearer | 否 |
| GET | `/gateway/fileZIP/{downloadKey}` | gateway - 前端API请求主要的入口 | 打包下载文件 | `GatewayControllerPart2_downloadZIP` | api-key 或 bearer | 否 |
| GET | `/gateway/addressCountByCity` | gateway - 前端API请求主要的入口 | 以市为单位的照片数量 | `GatewayControllerPart2_addressCountByCity` | api-key 或 bearer | 否 |
| GET | `/gateway/addressCountByDistrict/{city}` | gateway - 前端API请求主要的入口 | 以区、县为单位的照片数量 | `GatewayControllerPart2_addressCountByDistrict` | api-key 或 bearer | 否 |
| GET | `/gateway/addressCountByTownship/{city}/{district}` | gateway - 前端API请求主要的入口 | 以村、街道为单位的照片数量 | `GatewayControllerPart2_addressCountByTownship` | api-key 或 bearer | 否 |
| GET | `/gateway/filesInAddress` | gateway - 前端API请求主要的入口 | 对应地区下的所有照片 | `GatewayControllerPart2_filesInAddress` | api-key 或 bearer | 是 |
| GET | `/gateway/filesInAddressV2` | gateway - 前端API请求主要的入口 | 对应地区下的所有照片 | `GatewayControllerPart2_filesInAddressV2` | api-key 或 bearer | 否 |
| GET | `/gateway/classifyTopList` | gateway - 前端API请求主要的入口 | 按事物场景分类 | `GatewayControllerPart2_classifyTopList` | api-key 或 bearer | 否 |
| GET | `/gateway/classifyFileList` | gateway - 前端API请求主要的入口 | 按事物场景分类-文件列表 | `GatewayControllerPart2_classifyFileList` | api-key 或 bearer | 否 |
| POST | `/gateway/editFileClassify` | gateway - 前端API请求主要的入口 | 修改文件智能分类属性 | `GatewayControllerPart2_editFileClassify` | api-key 或 bearer | 否 |
| GET | `/gateway/filesInCategoriesV2` | gateway - 前端API请求主要的入口 | 按类型分类的文件列表 | `GatewayControllerPart2_filesInCategoriesV2` | api-key 或 bearer | 否 |
| GET | `/gateway/filesInTrash` | gateway - 前端API请求主要的入口 | 回收站中的文件 - 已废弃 | `GatewayControllerPart2_filesInTrash` | api-key 或 bearer | 是 |
| GET | `/gateway/filesInTrashV2` | gateway - 前端API请求主要的入口 | 回收站中的文件 - 已废弃 | `GatewayControllerPart2_filesInTrashV2` | api-key 或 bearer | 是 |
| GET | `/gateway/filesInTrashFlat` | gateway - 前端API请求主要的入口 | 回收站中的文件 | `GatewayControllerPart2_filesInTrashFlat` | api-key 或 bearer | 否 |
| POST | `/gateway/findSimilarFiles` | gateway - 前端API请求主要的入口 | 查找相似文件 | `GatewayControllerPart2_findDuplicateFilesWithGalleryIds` | api-key 或 bearer | 否 |
| GET | `/gateway/filesInTrashAdmin` | gateway - 前端API请求主要的入口 | 管理员-查看全部用户在回收站中的文件 | `GatewayControllerPart2_filesInTrashAdmin` | api-key 或 bearer | 否 |
| POST | `/gateway/findFilesWithInvalidGps` | gateway - 前端API请求主要的入口 | 管理员-查看无法识别的GPS坐标 | `GatewayControllerPart2_findFilesWithInvalidGps` | api-key 或 bearer | 否 |
| POST | `/gateway/hideFiles` | gateway - 前端API请求主要的入口 | 添加照片到隐私相册中 | `GatewayControllerPart3_addHideFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/cancelHideFiles` | gateway - 前端API请求主要的入口 | 从隐私相册内移出 | `GatewayControllerPart3_cancelHideFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/passwordCode` | gateway - 前端API请求主要的入口 | 验证用户密码，验证通过后返回passwordCode 用于访问 /gateway/filesInHide | `GatewayControllerPart3_pwdCode` | api-key 或 bearer | 否 |
| POST | `/gateway/filesInHide` | gateway - 前端API请求主要的入口 | 隐私相册中的照片 | `GatewayControllerPart3_filesInHide` | api-key 或 bearer | 否 |
| GET | `/gateway/recentFiles` | gateway - 前端API请求主要的入口 | 最近添加的文件 | `GatewayControllerPart3_filesRecent` | api-key 或 bearer | 否 |
| GET | `/gateway/peopleList` | gateway - 前端API请求主要的入口 | 人物列表 | `GatewayControllerPart3_peopleList` | api-key 或 bearer | 否 |
| GET | `/gateway/people/{id}` | gateway - 前端API请求主要的入口 | 人物详情 | `GatewayControllerPart3_peopleInfo` | api-key 或 bearer | 否 |
| PATCH | `/gateway/people/{id}` | gateway - 前端API请求主要的入口 | 修改人物详情 | `GatewayControllerPart3_updatePeopleInfo` | api-key 或 bearer | 否 |
| PUT | `/gateway/people/{id}` | gateway - 前端API请求主要的入口 | 修改人物详情 - patch兼容 | `GatewayControllerPart3_updatePeopleInfo_put` | api-key 或 bearer | 否 |
| POST | `/gateway/multiHidePeople` | gateway - 前端API请求主要的入口 | 一键显示或隐藏人物 | `GatewayControllerPart3_multiHidePeople` | api-key 或 bearer | 否 |
| POST | `/gateway/peopleNames` | gateway - 前端API请求主要的入口 | 获取人物名称 | `GatewayControllerPart3_getPeopleNames` | api-key 或 bearer | 否 |
| PATCH | `/gateway/reassignPeopleFile/{id}` | gateway - 前端API请求主要的入口 | 修改人物详情 | `GatewayControllerPart3_reassignPeopleFile` | api-key 或 bearer | 否 |
| PUT | `/gateway/reassignPeopleFile/{id}` | gateway - 前端API请求主要的入口 | 修改人物详情 - patch兼容 | `GatewayControllerPart3_reassignPeopleFile_put` | api-key 或 bearer | 否 |
| POST | `/gateway/editFileDescriptor` | gateway - 前端API请求主要的入口 | 修改人物详情 | `GatewayControllerPart3_editFileDescriptor` | api-key 或 bearer | 否 |
| GET | `/gateway/peopleFileList` | gateway - 前端API请求主要的入口 | 人物关联的文件列表 | `GatewayControllerPart3_peopleFileList` | api-key 或 bearer | 否 |
| GET | `/gateway/peopleFileListV2` | gateway - 前端API请求主要的入口 | 人物关联的文件列表 | `GatewayControllerPart3_peopleFileListV2` | api-key 或 bearer | 否 |
| GET | `/gateway/peopleDescriptorList` | gateway - 前端API请求主要的入口 | 人脸特征列表 - 管理员可调用 | `GatewayControllerPart3_peopleDescriptorList` | api-key 或 bearer | 否 |
| GET | `/gateway/descriptorDistanceList` | gateway - 前端API请求主要的入口 | 特征相似度列表 - 管理员可调用 | `GatewayControllerPart3_descriptorDistanceList` | api-key 或 bearer | 否 |
| GET | `/gateway/cache` | gateway - 前端API请求主要的入口 | cache value - 管理员可调用 | `GatewayControllerPart3_getCacheValue` | api-key 或 bearer | 否 |
| GET | `/gateway/peopleInFileInfo` | gateway - 前端API请求主要的入口 | 照片识别的人脸信息 | `GatewayControllerPart3_peopleInFileInfo` | api-key 或 bearer | 否 |
| POST | `/gateway/people/merge` | gateway - 前端API请求主要的入口 | 合并人物 | `GatewayControllerPart3_mergePeople` | api-key 或 bearer | 否 |
| POST | `/gateway/people/split/{id}` | gateway - 前端API请求主要的入口 | 拆分人物 | `GatewayControllerPart3_resetUserPeople` | api-key 或 bearer | 否 |
| POST | `/gateway/people/distance` | gateway - 前端API请求主要的入口 | 计算people下descriptor的distance - 管理员可调用 | `GatewayControllerPart3_calcPeopleDistance` | api-key 或 bearer | 否 |
| DELETE | `/gateway/files` | gateway - 前端API请求主要的入口 | 删除 | `GatewayControllerPart3_deleteFiles` | api-key 或 bearer | 否 |
| PATCH | `/gateway/files` | gateway - 前端API请求主要的入口 | 从回收站恢复 | `GatewayControllerPart3_restoreFiles` | api-key 或 bearer | 否 |
| PUT | `/gateway/files` | gateway - 前端API请求主要的入口 | 从回收站恢复 - patch兼容 | `GatewayControllerPart3_restoreFiles_put` | api-key 或 bearer | 否 |
| POST | `/gateway/deleteFilesPermanently` | gateway - 前端API请求主要的入口 | 从回收站删除 | `GatewayControllerPart3_deleteFromTrash` | api-key 或 bearer | 否 |
| GET | `/gateway/deleteFilesPermanentlyStatus` | gateway - 前端API请求主要的入口 | 获取永久删除文件状态 | `GatewayControllerPart3_deleteFilesPermanentlyStatus` | api-key 或 bearer | 否 |
| POST | `/gateway/deleteSimilarFiles` | gateway - 前端API请求主要的入口 | 删除相似文件 | `GatewayControllerPart3_deleteSimilarFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/hideSimilarFiles` | gateway - 前端API请求主要的入口 | 忽略相似照片 | `GatewayControllerPart3_hideSimilarFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/cancelHideSimilarFiles` | gateway - 前端API请求主要的入口 | 取消忽略相似照片 | `GatewayControllerPart3_cancelHideSimilarFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/similarFilesInHide` | gateway - 前端API请求主要的入口 | 忽略相似照片列表 | `GatewayControllerPart3_similarFilesInHide` | api-key 或 bearer | 否 |
| POST | `/gateway/user/pwd` | gateway - 前端API请求主要的入口 | 修改自己的密码 | `GatewayControllerPart3_userUpdatePwd` | api-key 或 bearer | 否 |
| POST | `/gateway/user/delete` | gateway - 前端API请求主要的入口 | 用户申请注销账号 | `GatewayControllerPart3_userUpdateDelete` | api-key 或 bearer | 否 |
| POST | `/gateway/user/cover` | gateway - 前端API请求主要的入口 | 自定义 自动相册的封面 | `GatewayControllerPart3_userUpdateCover` | api-key 或 bearer | 否 |
| POST | `/gateway/otp/generate` | gateway - 前端API请求主要的入口 | 生成双因素认证 | `GatewayControllerPart3_otpGen` | api-key 或 bearer | 否 |
| POST | `/gateway/otp/verify` | gateway - 前端API请求主要的入口 | 验证双因素认证 | `GatewayControllerPart3_otpVerify` | api-key 或 bearer | 否 |
| POST | `/gateway/otp/disable` | gateway - 前端API请求主要的入口 | 禁用双因素认证 | `GatewayControllerPart3_otpDisable` | api-key 或 bearer | 否 |
| GET | `/gateway/lang` | gateway - 前端API请求主要的入口 | 获取系统语言 | `GatewayControllerPart4_getSysLang` | api-key 或 bearer | 否 |
| GET | `/gateway/mapCenter` | gateway - 前端API请求主要的入口 | 获取mapbox 的 accessToken | `GatewayControllerPart4_getMapCenter` | api-key 或 bearer | 否 |
| GET | `/gateway/mapboxToken` | gateway - 前端API请求主要的入口 | 获取mapbox 的 accessToken | `GatewayControllerPart4_getMapboxToken` | api-key 或 bearer | 否 |
| GET | `/gateway/maptilerToken` | gateway - 前端API请求主要的入口 | 获取maptiler 的 accessToken | `GatewayControllerPart4_getMaptilerToken` | api-key 或 bearer | 否 |
| GET | `/gateway/mapType` | gateway - 前端API请求主要的入口 | 获取地图的类型 | `GatewayControllerPart4_getMapType` | api-key 或 bearer | 否 |
| GET | `/gateway/staticmap/amap/{location}` | gateway - 前端API请求主要的入口 | 获取高德静态地图url | `GatewayControllerPart4_staticMapAmap` | api-key 或 bearer | 否 |
| GET | `/gateway/amap/test/{key}/{secret}` | gateway - 前端API请求主要的入口 | 测试高德开放平台api key 私钥是否有效 | `GatewayControllerPart4_testAmapApiKey` | api-key 或 bearer | 否 |
| GET | `/gateway/qqmap/test/{key}/{secret}` | gateway - 前端API请求主要的入口 | 测试腾讯地图api key 私钥是否有效 | `GatewayControllerPart4_testQQmapApiKey` | api-key 或 bearer | 否 |
| GET | `/gateway/tianmap/test/{key}` | gateway - 前端API请求主要的入口 | 测试天地图api key是否有效 | `GatewayControllerPart4_testTianDiTuApiKey` | api-key 或 bearer | 否 |
| GET | `/gateway/mapbox/test/{token}` | gateway - 前端API请求主要的入口 | 测试 mapbox api key 是否有效 | `GatewayControllerPart4_testMapboxApiToken` | api-key 或 bearer | 否 |
| GET | `/gateway/maptiler/test/{token}` | gateway - 前端API请求主要的入口 | 测试 maptilerapi key 是否有效 | `GatewayControllerPart4_testMaptilerApiToken` | api-key 或 bearer | 否 |
| GET | `/gateway/allFilesForMap` | gateway - 前端API请求主要的入口 | 地图上的照片 | `GatewayControllerPart4_getAllFilesForMap` | api-key 或 bearer | 否 |
| GET | `/gateway/allFilesForMapDirect` | gateway - 前端API请求主要的入口 | 地图上的照片-原始坐标 | `GatewayControllerPart4_getFilesForMapDirect` | api-key 或 bearer | 否 |
| POST | `/gateway/areaFilesMD5` | gateway - 前端API请求主要的入口 | 根据文件ID列表获取文件信息 | `GatewayControllerPart4_getFileMD5List` | api-key 或 bearer | 否 |
| POST | `/gateway/fileInIds` | gateway - 前端API请求主要的入口 | 根据文件ID列表获取文件详情 | `GatewayControllerPart4_getFileInIds` | api-key 或 bearer | 否 |
| POST | `/gateway/enableFileBackup` | gateway - 前端API请求主要的入口 | 启用文件备份功能 | `GatewayControllerPart4_enableFileBackup` | api-key 或 bearer | 否 |
| POST | `/gateway/changeAppUploadStatus` | gateway - 前端API请求主要的入口 | 通知服务器是否在备份文件 | `GatewayControllerPart4_changeAppUploadStatus` | api-key 或 bearer | 否 |
| POST | `/gateway/checkFileId` | gateway - 前端API请求主要的入口 | 判断文件是否存在 | `GatewayControllerPart4_checkFileId` | api-key 或 bearer | 否 |
| POST | `/gateway/resetFileStatus` | gateway - 前端API请求主要的入口 | 请求重置异常状态文件 | `GatewayControllerPart4_fixFileStatus` | api-key 或 bearer | 否 |
| GET | `/gateway/backupDist/root` | gateway - 前端API请求主要的入口 | 备份目的地-根目录 | `GatewayControllerPart4_backupDistRoot` | api-key 或 bearer | 否 |
| GET | `/gateway/backupDist/sub` | gateway - 前端API请求主要的入口 | 备份目的地-子目录 | `GatewayControllerPart4_backupDistSubDir` | api-key 或 bearer | 否 |
| GET | `/gateway/backupDist/refresh` | gateway - 前端API请求主要的入口 | 备份目的地-刷新 | `GatewayControllerPart4_backupDistRefreshDir` | api-key 或 bearer | 否 |
| POST | `/gateway/backupDist/verify` | gateway - 前端API请求主要的入口 | 备份目的地-验证 | `GatewayControllerPart4_backupDistVerify` | api-key 或 bearer | 否 |
| POST | `/gateway/checkPathForUpload` | gateway - 前端API请求主要的入口 | 上传文件前，检查文件在服务端是否存在 | `GatewayControllerPart4_checkPathForUpload` | api-key 或 bearer | 否 |
| POST | `/gateway/upload` | gateway - 前端API请求主要的入口 | 上传文件 - multipart方式 | `GatewayControllerPart4_uploadFile` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadForShare` | gateway - 前端API请求主要的入口 | 上传文件 - multipart方式 - 网页分享链接 | `GatewayControllerPart4_uploadFileForShare` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadV2` | gateway - 前端API请求主要的入口 | 上传文件 - Binary方式 | `GatewayControllerPart4_uploadFileV2` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/check` | gateway - 前端API请求主要的入口 | 上传文件 - 分块上传前检查 | `GatewayControllerPart4_uploadChunkCheck` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/checkInShare` | gateway - 前端API请求主要的入口 | 分块上传-检查(分享链接) | `GatewayControllerPart4_uploadChunkCheckInShare` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/upload` | gateway - 前端API请求主要的入口 | 分块上传 - multipart | `GatewayControllerPart4_uploadChunkUpload` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/merge` | gateway - 前端API请求主要的入口 | 分块上传 - 完成后触发合并文件 | `GatewayControllerPart4_uploadChunkMerge` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/mergeStatus` | gateway - 前端API请求主要的入口 | 分块上传 - 获取合并进度状态 | `GatewayControllerPart4_uploadChunkMergeStatus` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/mergeInShare` | gateway - 前端API请求主要的入口 | 分块上传 - 完成后触发合并文件 - 分享链接中 | `GatewayControllerPart4_uploadChunkMergeInShare` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/mergeStatusForShare` | gateway - 前端API请求主要的入口 | 分块上传 - 获取合并进度状态 - 分享链接中使用 | `GatewayControllerPart4_uploadChunkMergeStatusForShare` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/uploadBin` | gateway - 前端API请求主要的入口 | 分块上传 - 上传文件-binary content 上传方式 | `GatewayControllerPart4_uploadChunkBin` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/uploadWeb` | gateway - 前端API请求主要的入口 | 分块上传-网页端 | `GatewayControllerPart4_uploadChunkWeb` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadChunk/uploadWebInShare` | gateway - 前端API请求主要的入口 | 分块上传-网页端(分享链接) | `GatewayControllerPart4_uploadChunkWebInShare` | api-key 或 bearer | 否 |
| POST | `/gateway/echo` | gateway - 前端API请求主要的入口 | 测试回显 | `GatewayControllerPart4_echo` | api-key 或 bearer | 否 |
| GET | `/gateway/licenseInfo` | gateway - 前端API请求主要的入口 | 订阅信息 - 管理员可调用 | `GatewayControllerPart4_licenseInfo` | api-key 或 bearer | 否 |
| GET | `/gateway/trail` | gateway - 前端API请求主要的入口 | 开始试用 - 管理员可调用 | `GatewayControllerPart4_startTrail` | api-key 或 bearer | 否 |
| POST | `/gateway/bindLicense` | gateway - 前端API请求主要的入口 | 使用激活码-添加订阅 - 管理员可调用 | `GatewayControllerPart4_bindLicense` | api-key 或 bearer | 否 |
| POST | `/gateway/verifyAuthOnline` | gateway - 前端API请求主要的入口 | 触发联网验证 - 管理员可调用 | `GatewayControllerPart4_forceVerifyCpStatusLive` | api-key 或 bearer | 否 |
| POST | `/gateway/coordinate/convert` | gateway - 前端API请求主要的入口 | gps坐标转为autonavi | `GatewayControllerPart4_coordinateConvert` | api-key 或 bearer | 否 |
| POST | `/gateway/coordinate/parse` | gateway - 前端API请求主要的入口 | 自动处理从 腾讯、高德地图坐标拾取器中粘贴的值 | `GatewayControllerPart4_coordinateAutoParse` | api-key 或 bearer | 否 |
| GET | `/gateway/folders/root` | gateway - 前端API请求主要的入口 | 文件夹视图-顶级 | `GatewayControllerPart5_folderTopList` | api-key 或 bearer | 否 |
| GET | `/gateway/folderInfo/{id}` | gateway - 前端API请求主要的入口 | 文件夹-信息 | `GatewayControllerPart5_folderInfo` | api-key 或 bearer | 否 |
| GET | `/gateway/folderSubFile/{id}` | gateway - 前端API请求主要的入口 | 文件夹-获取当前及下级文件夹文件 id、MD5 | `GatewayControllerPart5_folderSubFile` | api-key 或 bearer | 否 |
| POST | `/gateway/folderAutoCover/{id}` | gateway - 前端API请求主要的入口 | 文件夹-自动设置空封面的文件夹，显示下级文件夹的文件 | `GatewayControllerPart5_folderAutoCover` | api-key 或 bearer | 否 |
| GET | `/gateway/folders/{id}` | gateway - 前端API请求主要的入口 | 文件夹视图-文件夹详情 | `GatewayControllerPart5_folderViewDetail` | api-key 或 bearer | 是 |
| GET | `/gateway/foldersV2/{id}` | gateway - 前端API请求主要的入口 | 文件夹视图-文件夹详情 | `GatewayControllerPart5_folderViewDetailV2` | api-key 或 bearer | 否 |
| GET | `/gateway/folderFiles/{id}` | gateway - 前端API请求主要的入口 | 文件夹视图-文件夹详情-文件列表 | `GatewayControllerPart5_folderFileInTimeline` | api-key 或 bearer | 否 |
| GET | `/gateway/folderBreadcrumbs/{id}` | gateway - 前端API请求主要的入口 | 文件夹地址的面包屑 | `GatewayControllerPart5_folderBreadcrumbs` | api-key 或 bearer | 否 |
| POST | `/gateway/folders/create` | gateway - 前端API请求主要的入口 | 文件夹视图-新建文件夹 | `GatewayControllerPart5_folderCreate` | api-key 或 bearer | 否 |
| POST | `/gateway/folderPathEdit` | gateway - 前端API请求主要的入口 | 文件夹视图-重命名、移动、删除 | `GatewayControllerPart5_folderEdit` | api-key 或 bearer | 否 |
| POST | `/gateway/filePathEdit` | gateway - 前端API请求主要的入口 | 文件路径编辑 | `GatewayControllerPart5_filePathEdit` | api-key 或 bearer | 否 |
| POST | `/gateway/folder_files_move/preview` | gateway - 前端API请求主要的入口 | 整理文件夹下的文件 - 预览移动路径 | `GatewayControllerPart5_folder_files_move_preview` | api-key 或 bearer | 否 |
| POST | `/gateway/folder_files_move/move` | gateway - 前端API请求主要的入口 | 整理文件夹下的文件 - 移动文件 | `GatewayControllerPart5_folder_files_move_run` | api-key 或 bearer | 否 |
| POST | `/gateway/folders/delete_empty` | gateway - 前端API请求主要的入口 | 删除文件夹下面的 空文件夹 | `GatewayControllerPart5_folder_delete_empty` | api-key 或 bearer | 否 |
| POST | `/gateway/folder_files_move/status` | gateway - 前端API请求主要的入口 | 整理文件夹 获取处理进度 | `GatewayControllerPart5_folder_files_move_status` | api-key 或 bearer | 否 |
| PATCH | `/gateway/setFolderCover/{id}` | gateway - 前端API请求主要的入口 | 修改文件夹封面 | `GatewayControllerPart5_setFolderCover` | api-key 或 bearer | 否 |
| PUT | `/gateway/setFolderCover/{id}` | gateway - 前端API请求主要的入口 | 修改文件夹封面 - 兼容PATCH | `GatewayControllerPart5_setFolderCover_put` | api-key 或 bearer | 否 |
| POST | `/gateway/scanAfterUpload` | gateway - 前端API请求主要的入口 | 更新刚上传的文件的状态 | `GatewayControllerPart5_scanAfterUpload` | api-key 或 bearer | 否 |
| POST | `/gateway/scanAfterUploadInShare` | gateway - 前端API请求主要的入口 | 更新刚上传的文件的状态 - 分享的链接中 | `GatewayControllerPart5_scanAfterUploadInShare` | api-key 或 bearer | 否 |
| POST | `/gateway/folderDebugInfo` | gateway - 前端API请求主要的入口 | 获取文件夹的调试信息 - 管理员可调用 | `GatewayControllerPart5_folderDebugInfo` | api-key 或 bearer | 否 |
| POST | `/gateway/updateFileDate` | gateway - 前端API请求主要的入口 | 更新文件的拍摄日期 | `GatewayControllerPart5_updateFileDate` | api-key 或 bearer | 否 |
| POST | `/gateway/updateFileName` | gateway - 前端API请求主要的入口 | 修改文件的名称 | `GatewayControllerPart5_updateFileName` | api-key 或 bearer | 否 |
| POST | `/gateway/editFileExtra` | gateway - 前端API请求主要的入口 | 编辑文件额外信息 | `GatewayControllerPart5_editFileDesc` | api-key 或 bearer | 否 |
| POST | `/gateway/editFileGps` | gateway - 前端API请求主要的入口 | 编辑文件GPS信息 | `GatewayControllerPart5_editFileGPS` | api-key 或 bearer | 否 |
| POST | `/gateway/resetFileGps` | gateway - 前端API请求主要的入口 | 重置文件GPS信息 | `GatewayControllerPart5_resetFileGps` | api-key 或 bearer | 否 |
| POST | `/gateway/editFileRotate` | gateway - 前端API请求主要的入口 | 旋转文件 | `GatewayControllerPart5_editFileRotate` | api-key 或 bearer | 否 |
| POST | `/gateway/searchTips` | gateway - 前端API请求主要的入口 | 搜索提示 | `GatewayControllerPart5_searchTips` | api-key 或 bearer | 否 |
| POST | `/gateway/search` | gateway - 前端API请求主要的入口 | 搜索 | `GatewayControllerPart5_searchFiles` | api-key 或 bearer | 否 |
| POST | `/gateway/searchCLIP` | gateway - 前端API请求主要的入口 | 搜索-CLIP | `GatewayControllerPart5_getCLIPTextMatchedId` | api-key 或 bearer | 否 |
| POST | `/gateway/searchV2` | gateway - 前端API请求主要的入口 | 搜索-v2 | `GatewayControllerPart5_searchFilesV2` | api-key 或 bearer | 否 |
| POST | `/gateway/searchResultTipsBox` | gateway - 前端API请求主要的入口 | 搜索结果提示框 | `GatewayControllerPart5_searchResultTipsBox` | api-key 或 bearer | 否 |
| POST | `/gateway/searchCLIPV2` | gateway - 前端API请求主要的入口 | 搜索-CLIP | `GatewayControllerPart5_searchCLIPV2` | api-key 或 bearer | 否 |
| POST | `/gateway/memory` | gateway - 前端API请求主要的入口 | 那年今日 | `GatewayControllerPart5_getMemoryList` | api-key 或 bearer | 否 |
| POST | `/gateway/memoryWeekFileList` | gateway - 前端API请求主要的入口 | 往年照片 - 一周 - 文件列表 | `GatewayControllerPart5_memoryWeekFileList` | api-key 或 bearer | 否 |
| POST | `/gateway/CLIP_status` | gateway - 前端API请求主要的入口 | 是否可以用使用CLIP搜索 | `GatewayControllerPart5_searchCLIPStatus` | api-key 或 bearer | 否 |
| POST | `/gateway/nongLi` | gateway - 前端API请求主要的入口 | 获取阳历日期的农历日期 | `GatewayControllerPart5_getNongLiInfo` | api-key 或 bearer | 否 |
| POST | `/gateway/livePhotoMovCheck` | gateway - 前端API请求主要的入口 | 检查livePhoto视频部分是否正确 | `GatewayControllerPart5_livePhotoMovCheck` | api-key 或 bearer | 否 |
| POST | `/gateway/uploadForLivePhotoMov/{photoMD5}/{videoMD5}` | gateway - 前端API请求主要的入口 | 上传动态照片视频部分 | `GatewayControllerPart5_uploadForLivePhotoMov` | api-key 或 bearer | 否 |
| GET | `/gateway/{type}/{md5}` | gateway - 前端API请求主要的入口 | 显示文件的缩略图 | `GatewayControllerPartEnd_renderThumb` | api-key 或 bearer | 否 |
| POST | `/api-share` | api-share 分享管理 | 创建分享 | `ShareController_create` | api-key 或 bearer | 否 |
| GET | `/api-share` | api-share 分享管理 | 我的分享列表 | `ShareController_findAll` | api-key 或 bearer | 否 |
| GET | `/api-share/shareToMe` | api-share 分享管理 | 分享给我的列表 | `ShareController_findAllShareToMe` | api-key 或 bearer | 否 |
| GET | `/api-share/users` | api-share 分享管理 | 查询可分享的用户列表 | `ShareController_findUsers` | api-key 或 bearer | 否 |
| GET | `/api-share/link/{id}` | api-share 分享管理 | 查询 相册的分享链接 key | `ShareController_createShare` | api-key 或 bearer | 否 |
| GET | `/api-share/visit/album/{key}` | api-share 分享管理 | 根据链接分享的key获取相册的信息 | `ShareController_getShareInfo` | api-key 或 bearer | 否 |
| GET | `/api-share/album/{id}` | api-share 分享管理 | 开启分享相册时，查询这个相册是否有分享信息 | `ShareController_findOneByAlbumId` | api-key 或 bearer | 否 |
| GET | `/api-share/albumInfo/{albumId}` | api-share 分享管理 | 打开他人分享的相册时，根据albumId，获取相册的信息 | `ShareController_findAlbumInfoByAlbumId` | api-key 或 bearer | 否 |
| GET | `/api-share/albumFiles/{albumId}` | api-share 分享管理 | 打开他人分享的相册时，根据albumId，获取相册的文件列表 | `ShareController_findAlbumFilesByAlbumId` | api-key 或 bearer | 否 |
| GET | `/api-share/albumFilesFlat/{albumId}` | api-share 分享管理 | 打开他人分享的相册时，根据albumId，获取相册的文件列表 - 平铺列表 | `ShareController_findAlbumFilesFlatByAlbumId` | api-key 或 bearer | 否 |
| POST | `/api-share/dayFileMoreForUser` | api-share 分享管理 | 单天剩余文件 - 已登录用户 | `ShareController_dayFileMoreForUser` | api-key 或 bearer | 否 |
| POST | `/api-share/dayFileMore` | api-share 分享管理 | 单天剩余文件 - 链接 | `ShareController_findDayFileMore` | api-key 或 bearer | 否 |
| GET | `/api-share/album/link/{id}` | api-share 分享管理 | 获取相册的自动更新配置 | `ShareController_findAutoLinkList` | api-key 或 bearer | 否 |
| POST | `/api-share/album/link/{id}` | api-share 分享管理 | 添加 相册 自动配置 | `ShareController_addAutoLink` | api-key 或 bearer | 否 |
| DELETE | `/api-share/album/link/{id}` | api-share 分享管理 | 删除 分享的相册 自动配置 | `ShareController_delAutoLink` | api-key 或 bearer | 否 |
| GET | `/api-share/visit/albumFiles/{key}` | api-share 分享管理 | 查询相册分享链接的文件列表 - 网页使用 | `ShareController_findAlbumFilesByKey` | api-key 或 bearer | 否 |
| GET | `/api-share/visit/albumFilesFlat/{key}` | api-share 分享管理 | 查询相册分享链接的文件列表 | `ShareController_findAlbumFilesByKeyFlat` | api-key 或 bearer | 否 |
| GET | `/api-share/fileInfo/{albumId}/{fileId}` | api-share 分享管理 | 显示文件的详细信息 - 检查共享权限 | `ShareController_getFileDetail` | api-key 或 bearer | 否 |
| GET | `/api-share/fileInfoByKey/{key}/{fileId}` | api-share 分享管理 | 查询相册分享链接的文件详情 | `ShareController_getFileDetailByKey` | api-key 或 bearer | 否 |
| GET | `/api-share/amap/{key}/{location}` | api-share 分享管理 | 获取高德静态地图url | `ShareController_staticMapAmap` | api-key 或 bearer | 否 |
| POST | `/api-share/filesInfo` | api-share 分享管理 | 下载前查询文件信息 - 分享的链接 | `ShareController_getFilesInfo` | api-key 或 bearer | 否 |
| POST | `/api-share/addFileToAlbum` | api-share 分享管理 | 添加文件到分享相册 | `ShareController_addFileToAlbum` | api-key 或 bearer | 否 |
| POST | `/api-share/removeFileFromAlbum` | api-share 分享管理 | 从分享相册移除文件 | `ShareController_removeFileFromAlbum` | api-key 或 bearer | 否 |
| GET | `/api-share/{id}` | api-share 分享管理 | 查询分享信息 | `ShareController_findOne` | api-key 或 bearer | 否 |
| PATCH | `/api-share/{id}` | api-share 分享管理 | 更新分享信息 | `ShareController_update` | api-key 或 bearer | 否 |
| PUT | `/api-share/{id}` | api-share 分享管理 | 更新分享信息(PUT) | `ShareController_update_put` | api-key 或 bearer | 否 |
| DELETE | `/api-share/{id}` | api-share 分享管理 | 删除分享 | `ShareController_remove` | api-key 或 bearer | 否 |
| POST | `/api-share/createFilesLink` | api-share 分享管理 | 创建分享 - 文件链接分享 | `ShareController_createFileLink` | api-key 或 bearer | 否 |
| POST | `/api-share/getFilesLink/{id}` | api-share 分享管理 | 查询分享 - 文件链接分享 | `ShareController_getShareFileLinkInfo` | api-key 或 bearer | 否 |
| POST | `/api-share/updateFilesLink/{id}` | api-share 分享管理 | 修改分享 - 文件链接分享 | `ShareController_updateFileLink` | api-key 或 bearer | 否 |
| POST | `/api-share/delFilesLink/{id}` | api-share 分享管理 | 删除分享 - 文件链接分享 | `ShareController_delFileLink` | api-key 或 bearer | 否 |
| POST | `/api-share/filesLink/count` | api-share 分享管理 | 我的分享列表 - 链接分享的文件 - 数量 | `ShareController_countAllSingleFiles` | api-key 或 bearer | 否 |
| POST | `/api-share/filesLink/list` | api-share 分享管理 | 我的分享列表 - 链接分享的文件 - 列表 | `ShareController_findAllSingleFiles` | api-key 或 bearer | 否 |
| POST | `/api-share/filesLink/list/{id}` | api-share 分享管理 | 我的分享列表 - 链接分享的文件 - 文件列表 | `ShareController_getFileLinkFiles` | api-key 或 bearer | 否 |
| POST | `/api-share/visit/filesLink/{key}` | api-share 分享管理 | 根据链接分享的key获取file的信息 | `ShareController_getFileShareInfo` | api-key 或 bearer | 否 |
| POST | `/api-share/visit/filesLinkFiles/{key}` | api-share 分享管理 | 查询链接分享链接的文件列表 | `ShareController_findShareFileListByKey` | api-key 或 bearer | 否 |
| POST | `/api-share/linkFileInfoByKey/{key}/{fileId}` | api-share 分享管理 | 查询文件分享链接的文件详情 | `ShareController_getLinkFileDetailByKey` | api-key 或 bearer | 否 |
| POST | `/api-share/linkFileInfoAmap/{key}/{location}` | api-share 分享管理 | 获取高德静态地图url - 文件分享链接 | `ShareController_linkFileInfoAmap` | api-key 或 bearer | 否 |
| GET | `/install/status` | install-初始化 | 获取安装状态 | `InstallController_findStatus` | 未声明 | 否 |
| POST | `/install/createAdminAccount` | install-初始化 | 创建管理员用户 | `InstallController_create` | 未声明 | 否 |
| GET | `/install/rootDirs` | install-初始化 | 获取根目录列表 | `InstallController_findRootDirs` | 未声明 | 否 |
| GET | `/install/subDirs` | install-初始化 | 获取子目录列表 | `InstallController_findSubDirs` | 未声明 | 否 |
| POST | `/install/createFolders` | install-初始化 | 批量创建文件夹 | `InstallController_createFolders` | 未声明 | 否 |
| POST | `/install/gallery` | install-初始化 | 创建图库 | `InstallController_createGallery` | 未声明 | 否 |
| GET | `/install/gallery` | install-初始化 | 获取图库列表 | `InstallController_getGalleryList` | 未声明 | 否 |
| DELETE | `/install/gallery/{id}` | install-初始化 | 删除图库 | `InstallController_deleteGallery` | 未声明 | 否 |
| PATCH | `/install/gallery/{id}` | install-初始化 | 更新图库 | `InstallController_updateGallery` | 未声明 | 否 |
| GET | `/install/gallery/scan/{id}` | install-初始化 | 扫描图库 | `InstallController_scanGallery` | 未声明 | 否 |
| PATCH | `/install/system-config` | install-初始化 | 更新系统配置 | `InstallController_updateByKey` | 未声明 | 否 |
| GET | `/install/system-config/{key}` | install-初始化 | 获取系统配置 | `InstallController_findByKey` | 未声明 | 否 |
| POST | `/install/upgrade` | install-初始化 | 手动升级 | `InstallController_update` | 未声明 | 否 |
| POST | `/install/autoUpgrade` | install-初始化 | 自动升级 | `InstallController_autoUpgrade` | 未声明 | 否 |
| GET | `/install/memory` | install-初始化 | 获取内存使用情况 | `InstallController_memoryUsage` | bearer | 否 |
| POST | `/install/reload` | install-初始化 | 重载服务 | `InstallController_reloadServer` | bearer | 否 |
| GET | `/install/trail` | install-初始化 | 开始试用 | `InstallController_startTrail` | 未声明 | 否 |

## 6. 接口详情

## 服务端信息+用户登录

### GET /api-info
- **摘要:** 获取 API 信息
- **OperationId:** `AppController_getInfo`
- **认证:** 未声明

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `type` | any | 否 | 可选值: all 或留空 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回服务端信息 | application/json: object { version: string, build: string, activated: boolean, arch: string, platform: string, tzOffset: number, faceRegVer: string, dbStatus: string } |

### POST /auth/rsa
- **摘要:** 获取RSA公钥
- **OperationId:** `AppController_getLoginRSAKeys`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回RSA公钥用于登录密码加密 | application/json: object { publicKey: string, ver: number } |

### POST /auth/login
- **摘要:** 登录
- **OperationId:** `AppController_login`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { username: string, password: string, otp: string, dev: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 登录成功 | application/json: object { username: string, id: number, uid: string, isAdmin: boolean, access_token: string, refresh_token: string, expires_in: number, auth_code: string } |

### POST /auth/refresh
- **摘要:** 刷新token
- **OperationId:** `AppController_refreshToken`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { token: string, ver: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 刷新成功 | application/json: object { access_token: string, refresh_token: string, expires_in: number, auth_code: string } |
| 401 | 未授权，需要重新登录 |  |

### POST /auth/auth_code
- **摘要:** 获取auth_code，有效时间为24小时内
- **说明:** 用处：在各种显示缩略图或者视频播放时需要带上auth_code参数，比如: /gateway/{type}/{md5} , /gateway/file/{id}/{md5} 等 
- **OperationId:** `AppController_getAuthCode`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { refresh_token: string, api_key: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 获取成功 | application/json: object { auth_code: string } |

## API Key 管理

### GET /api-keys
- **摘要:** 获取当前用户的 API Key 列表
- **OperationId:** `ApiKeyController_findAll`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回当前用户的所有 API Key 列表 | application/json: array<object> |
| 401 | 未授权 |  |

### POST /api-keys
- **摘要:** 创建新的 API Key
- **OperationId:** `ApiKeyController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { name: string, expiresAt: string:date-time, remark: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | API Key 创建成功，plainKey 仅返回一次，请妥善保存 | application/json: object { id: string, name: string, plainKey: string, expiresAt: string:date-time, createdAt: string:date-time } |
| 401 | 未授权 |  |

### GET /api-keys/{id}
- **摘要:** 获取当前用户的单个 API Key
- **OperationId:** `ApiKeyController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回 API Key 详情 | application/json: object { id: string, userId: number, name: string, isActive: boolean, expiresAt: string:date-time, remark: string, createdAt: string:date-time, updatedAt: string:date-time, ... } |
| 401 | 未授权 |  |
| 404 | API Key 不存在 |  |

### PATCH /api-keys/{id}
- **摘要:** 更新当前用户的 API Key
- **OperationId:** `ApiKeyController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { name: string, isActive: boolean, expiresAt: string:date-time, remark: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | API Key 更新成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 404 | API Key 不存在或更新失败 |  |

### DELETE /api-keys/{id}
- **摘要:** 删除当前用户的 API Key
- **OperationId:** `ApiKeyController_remove`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | API Key 删除成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 404 | API Key 不存在 |  |

### POST /api-keys/{id}/regenerate
- **摘要:** 重新生成当前用户的 API Key
- **OperationId:** `ApiKeyController_regenerate`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | API Key 重新生成成功，旧的 Key 已失效 | application/json: object { id: string, plainKey: string } |
| 401 | 未授权 |  |
| 404 | API Key 不存在 |  |

## API Key管理 - 仅限管理员调用

### GET /api-keys-admin
- **摘要:** 获取所有用户的 API Key 列表（管理员）
- **OperationId:** `ApiKeyAdminController_findAll`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `userId` | number | 否 | 按用户ID筛选 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回所有用户的 API Key 列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### POST /api-keys-admin
- **摘要:** 为指定用户创建 API Key（管理员）
- **OperationId:** `ApiKeyAdminController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { userId: number, name: string, expiresAt: string:date-time, remark: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | API Key 创建成功，plainKey 仅返回一次，请妥善保存 | application/json: object { id: string, userId: number, name: string, plainKey: string, expiresAt: string:date-time, createdAt: string:date-time, message: string } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /api-keys-admin/{id}
- **摘要:** 获取单个 API Key（管理员）
- **OperationId:** `ApiKeyAdminController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回 API Key 详情 | application/json: object { id: string, userId: number, name: string, isActive: boolean, expiresAt: string:date-time, remark: string, createdAt: string:date-time, updatedAt: string:date-time, ... } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | API Key 不存在 |  |

### PATCH /api-keys-admin/{id}
- **摘要:** 更新 API Key（管理员）
- **OperationId:** `ApiKeyAdminController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { name: string, isActive: boolean, expiresAt: string:date-time, remark: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | API Key 更新成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | API Key 不存在或更新失败 |  |

### DELETE /api-keys-admin/{id}
- **摘要:** 删除 API Key（管理员）
- **OperationId:** `ApiKeyAdminController_remove`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | API Key 删除成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | API Key 不存在 |  |

### POST /api-keys-admin/{id}/regenerate
- **摘要:** 重新生成 API Key（管理员）
- **OperationId:** `ApiKeyAdminController_regenerate`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | API Key ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | API Key 重新生成成功，旧的 Key 已失效 | application/json: object { id: string, plainKey: string, message: string } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | API Key 不存在 |  |

## users 仅限管理员调用

### PATCH /users/resetSuperAdminPwd
- **摘要:** 重置管理员密码
- **说明:** 仅提供给官网教程内的 重置管理员密码 调用，其他场景调用都会失败
- **OperationId:** `UsersController_resetSuperAdminPwd`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { check: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 重置结果 | application/json: object { n: number } |

### POST /users
- **摘要:** 创建用户
- **OperationId:** `UsersController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateUserDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 用户创建成功 | application/json: object { success: boolean } |
| 400 | 请求参数错误 |  |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /users
- **摘要:** 用户列表
- **OperationId:** `UsersController_findAll`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回用户列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### PATCH /users/{id}
- **摘要:** 更新用户信息
- **OperationId:** `UsersController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 用户ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateUserDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 用户信息更新成功 | application/json: User |
| 400 | 请求参数错误或用户名已被使用 |  |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### DELETE /users/{id}
- **摘要:** 删除用户
- **OperationId:** `UsersController_remove`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 用户ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 用户删除成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | 用户不存在 |  |

### GET /users/{id}
- **摘要:** 用户信息
- **OperationId:** `UsersController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 用户ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回用户详情 | application/json: User |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | 用户不存在 |  |

### PATCH /users/resetPwd/{id}
- **摘要:** 重置用户密码
- **OperationId:** `UsersController_resetPwd`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 用户ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 密码重置成功，返回新密码 | application/json: object { password: string } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /users/userIdNameList
- **摘要:** 获取全部用户的 id、uid、username
- **OperationId:** `UsersController_findIdMap`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回用户简要信息列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### POST /users/{id}/avatar
- **摘要:** 管理员上传用户头像
- **说明:** 管理员为指定用户上传头像，非 JPEG 图片会自动转换为 JPEG
- **OperationId:** `UsersController_uploadUserAvatar`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 用户ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| multipart/form-data | 是 | object { file: string:binary } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回上传结果 | application/json: object { success: boolean, avatar: string, msg: string } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | 用户不存在 |  |

### DELETE /users/{id}/avatar
- **摘要:** 删除用户头像
- **说明:** 管理员删除指定用户的头像文件，并清空用户 avatar 字段
- **OperationId:** `UsersController_deleteUserAvatar`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 用户ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 删除结果 | application/json: object { success: boolean, avatar: string, msg: string } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | 用户不存在 |  |

## folder 仅限管理员调用

### POST /folder
- **摘要:** 创建文件夹
- **说明:** 创建新的文件夹记录
- **OperationId:** `FoldersController_create`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateFolderDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: Folder |
| 400 | 请求参数错误 |  |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /folder
- **摘要:** 获取文件夹列表
- **说明:** 分页获取所有文件夹列表
- **OperationId:** `FoldersController_findAll`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pageNo` | number | 否 | 页码，默认1 |
| query | `pageSize` | number | 否 | 每页数量，默认10 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 查询成功 | application/json: array<Folder> |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /folder/{id}
- **摘要:** 获取单个文件夹
- **说明:** 根据ID获取文件夹详细信息
- **OperationId:** `FoldersController_findOne`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 文件夹ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 查询成功 | application/json: Folder |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### PATCH /folder/{id}
- **摘要:** 更新文件夹
- **说明:** 根据ID更新文件夹信息
- **OperationId:** `FoldersController_update`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 文件夹ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateFolderDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 更新成功 | application/json: Folder |
| 400 | 请求参数错误 |  |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### DELETE /folder/{id}
- **摘要:** 删除文件夹
- **说明:** 根据ID删除文件夹记录
- **OperationId:** `FoldersController_remove`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 文件夹ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 删除成功 | application/json: object { raw: object, affected: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

## files 仅限管理员调用

### POST /files/triggerBoundaryEvolution
- **摘要:** 触发边界演变
- **说明:** 根据行政区划代码和乡镇名称触发边界演变处理
- **OperationId:** `FilesController__triggerBoundaryEvolution`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { adcode: string, township: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 边界演变处理完成或跳过（点数不足/正在处理中） | application/json: object { n: number, msg: string } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /files/resetFile/{id}
- **摘要:** 重置文件状态
- **说明:** 重置文件的status、proxyStatus、transcodeStatus、peopleDescriptorStatus为初始值
- **OperationId:** `FilesController_resetStatus`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 重置成功 | application/json: object { affected: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /files/faceReg/{md5}
- **摘要:** 根据MD5获取文件人脸描述符
- **说明:** 通过文件MD5值查询文件的人脸描述符信息
- **OperationId:** `FilesController_findFilePeopleDescriptorByMd5`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `md5` | string | 是 | 文件MD5值 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人脸描述符信息 | application/json: object { hasReg: boolean, list: array<object> } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /files/count/{type}/{md5}
- **摘要:** 按MD5统计文件数量
- **说明:** 统计指定MD5值的文件数量，type必须为COUNT_CHECK
- **OperationId:** `FilesController_countFileByMD5`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `type` | string | 是 | 类型标识，需为COUNT_CHECK |
| path | `md5` | string | 是 | 文件MD5值 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件数量 | application/json: #/components/schemas/ |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /files/ocr/info
- **摘要:** 获取OCR任务信息
- **说明:** 获取正在处理的OCR数量和待处理的OCR总数
- **OperationId:** `FilesController_getOcrInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回OCR任务统计 | application/json: object { inProcess: number, total: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /files/ocr/task
- **摘要:** 获取OCR任务列表
- **说明:** 获取待处理的OCR任务文件列表，每次最多返回100条。调用 /gateway/file/{id}/{md5} 可以获取预览图或者原文件(加参数 ?type=ori)
- **OperationId:** `FilesController_getOcrTask`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回OCR任务列表 | application/json: object { items: array<object>, MD5HasAdded: object, ignoreNum: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /files/ocr/result
- **摘要:** 提交OCR识别结果
- **说明:** 提交文件的OCR识别数据，更新文件OCR状态
- **OperationId:** `FilesController_saveOcrResult`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { MD5: string, files: array<number>, results: array<object> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { raw: object, affected: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /files/ocr/resetStatus
- **摘要:** 重置OCR状态
- **说明:** 重置文件的OCR状态，解决意外退出时未处理的图片； 使用场景：当调用 /files/ocr/task 时，接口会将返回数据列表中的files ocrStatus改为1，避免相同的文件在接口多次返回，导致重复识别；如果上次/files/ocr/task获取的文件没有全部识别完成，则需要调用这个接口来重置file的ocrStatus，让后面再请求/files/ocr/task接口时可以获取到这些未处理的文件
- **OperationId:** `FilesController_resetOcrStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { raw: object, affected: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /files/{id}
- **摘要:** 获取单个文件信息
- **说明:** 根据文件ID查询文件详细信息
- **OperationId:** `FilesController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详细信息 | application/json: File |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### PATCH /files/{id}
- **摘要:** 更新文件信息
- **说明:** 根据文件ID更新文件信息
- **OperationId:** `FilesController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateFileDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新后的文件信息 | application/json: File |
| 400 | 请求参数错误 |  |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /files/broTaskFileList
- **摘要:** 获取浏览器任务文件列表
- **说明:** 获取浏览器任务文件列表，用于人脸识别、场景识别等任务，仅允许127.0.0.1访问
- **OperationId:** `FilesController_getBrowserTaskFileList`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { passCode: string, type: string, pageNo: number, pageSize: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件列表 | application/json: array<object> |

### POST /files/findInGpsDistrict
- **摘要:** 根据行政区划或坐标测试地理位置识别
- **说明:** 根据adcode查询行政区划边界，或根据坐标测试GPS识别（支持maptiler和findInGpsScatterPolygon两种模式）
- **OperationId:** `FilesController_findInGpsDistrict`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { lat: number, lng: number, adcode: string, type: string, token: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回地理位置信息 | application/json: object |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

## fileTask 仅限管理员调用

### POST /fileTask/addTask
- **摘要:** 创建后台任务
- **OperationId:** `FileTaskController_addTask`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("scanFiles", "scanFilesForUpload", "updateAllFilesGalleryIds", "bindRawFile", "generatePhotoThumb", "generateVideoPreview", "scanLivePhotos", "fillGpsInfo", "clearAlbumInvalidFileId", "detectFileFaces", "genPeopleBase", "genPeople", "genPeopleCover", "detectFileCategories", "cleanGarbageData", "cleanNoGalleryData", "upgradeCacheFolder", "refreshTimelineCache", "fixThumb", "refreshThumb", "refreshExif", "checkDescriptorPass", "resetAllGpsInfo", "resetAllFaceDescriptor", "resetFailedFaceDescriptor", "reGenPeopleBase", "reGenPeople", "resetFileCategories", "resetFailedCategories", "testLimit", "ocrTask", "resetFailedOCR", "resetFileOcr", "clipTask", "resetFailedClip", "resetFileClip", "fillGpsInfoFix", "fixHDRVideoThumbs", "fixNotSRGBPhotoThumbs", "detectFileFacesV2", "fileSimilarTask", "resetFailedSimilar", "resetFileSimilar", "videoTranscode", "imageHdThumb", "resetVideoTranscode", "resetImageHdThumb", "resetFailedImageHdThumb", "resetFailedVideoTranscode", "clearDataAfterDeleteFilesPermanently", "syncAllAlbumAutoLink", "clearAllJobs", "adminDeleteUserSetCover_address", "adminDeleteUserSetCover_classify"), info: object, force: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 清空所有任务成功 | application/json: object { success: boolean } |
| 201 | 任务创建成功 | application/json: object { id: string, name: string, data: object, opts: object } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/active
- **摘要:** 获取正在执行的任务列表
- **OperationId:** `FileTaskController_getActiveJobs`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回正在执行的任务列表 | application/json: object { jobs: array<object>, subData: object, THUMB_TASK_MAX_NUM: number } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/job/subData
- **摘要:** 获取任务进度子数据
- **OperationId:** `FileTaskController_getJobSubData`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回任务进度子数据 | application/json: object { stage: object, data: object } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/completed
- **摘要:** 获取已完成任务列表
- **OperationId:** `FileTaskController_getCompleted`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `end` | number | 否 | 结束索引 |
| query | `start` | number | 否 | 起始索引 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回已完成的任务列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/waiting
- **摘要:** 获取等待中任务列表
- **OperationId:** `FileTaskController_getWaiting`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `end` | number | 否 | 结束索引 |
| query | `start` | number | 否 | 起始索引 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回等待中的任务列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/paused
- **摘要:** 获取已暂停任务列表
- **OperationId:** `FileTaskController_getPaused`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `end` | number | 否 | 结束索引 |
| query | `start` | number | 否 | 起始索引 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回已暂停的任务列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/failed
- **摘要:** 获取失败任务列表
- **OperationId:** `FileTaskController_getFailed`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `end` | number | 否 | 结束索引 |
| query | `start` | number | 否 | 起始索引 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回失败的任务列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/isPaused
- **摘要:** 检查任务队列是否已暂停
- **OperationId:** `FileTaskController_isPaused`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回任务队列是否已暂停 | application/json: boolean |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### POST /fileTask/jobs/pause
- **摘要:** 暂停任务队列
- **OperationId:** `FileTaskController_pause`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 暂停成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### POST /fileTask/jobs/resume
- **摘要:** 恢复任务队列
- **OperationId:** `FileTaskController_resume`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 恢复成功 | application/json: object { success: boolean } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/jobs/Counts
- **摘要:** 获取各状态任务数量统计
- **OperationId:** `FileTaskController_getJobCounts`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回各状态任务数量统计 | application/json: object { active: number, completed: number, failed: number, delayed: number, waiting: number, paused: number } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/resetAllGpsInfo
- **摘要:** 重置所有GPS信息
- **OperationId:** `FileTaskController_resetAllGpsInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | GPS重置任务已创建 | application/json: object { id: string, name: string, data: object } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/checkLicense
- **摘要:** 检查许可证状态
- **OperationId:** `FileTaskController_checkCpInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 许可证检查完成 | application/json: object { n: number } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /fileTask/client/{name}
- **摘要:** 获取浏览器辅助处理模型文件
- **OperationId:** `FileTaskController_getTfTaskFiles`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `name` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回模型文件内容 |  |
| 302 | 未授权时重定向到登录页 |  |
| 404 | 文件不存在或路径越权 |  |

### GET /fileTask/client/dist/{name}
- **摘要:** 获取浏览器辅助处理模型文件（dist目录）
- **OperationId:** `FileTaskController_getTfTaskFiles2`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `name` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回dist目录下的模型文件内容 |  |
| 404 | 文件不存在或路径越权 |  |

### GET /fileTask/client/dist/{type}/{name}
- **摘要:** 获取浏览器辅助处理模型文件（dist子目录）
- **OperationId:** `FileTaskController_getTfTaskFiles3`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `type` | string | 是 |  |
| path | `name` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回dist子目录下的模型文件内容 |  |
| 404 | 文件不存在或路径越权 |  |

### GET /fileTask/{id}
- **摘要:** 根据ID获取任务详情
- **OperationId:** `FileTaskController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回任务详情 | application/json: object { id: string, name: string, data: object, timestamp: number, processedOn: number, finishedOn: number, progress: number, returnvalue: object, ... } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |
| 404 | 任务不存在 |  |

## gallery 仅限管理员调用

### GET /gallery/rootDirs
- **摘要:** 获取根目录列表
- **说明:** 获取系统可用的根目录路径列表
- **OperationId:** `GalleryController_findRootDirs`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回根目录列表 | application/json: array<string> |

### GET /gallery/subDirs
- **摘要:** 获取子目录列表
- **说明:** 根据指定路径获取子目录列表
- **OperationId:** `GalleryController_findSubDirs`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `path` | string | 是 | 父目录路径 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回子目录列表 | application/json: array<string> |

### POST /gallery/findDuplicateFiles
- **摘要:** 查找重复文件
- **说明:** 根据图库ID列表查找重复文件
- **OperationId:** `GalleryController_findDuplicateFilesWithGalleryIds`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { galleryIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回重复文件列表 | application/json: array<object> |

### GET /gallery/findDeletedFiles
- **摘要:** 查找已删除文件
- **说明:** 查找系统中已被删除但数据库中仍有记录的文件，按日期分组返回
- **OperationId:** `GalleryController_findDeletedFiles`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的已删除文件列表 | application/json: object { result: array<object>, duplicateFiles: object, totalCount: number, ver: number } |

### POST /gallery/exportDeletedFiles
- **摘要:** 导出已删除文件的预览图
- **说明:** 查找系统中已被删除但数据库中仍有记录的文件，然后导出这个文件的预览图
- **OperationId:** `GalleryController_exportDeletedFiles`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gallery/exportDeletedFiles/stat
- **摘要:** 导出已删除文件的预览图 - 进度查询
- **OperationId:** `GalleryController_exportDeletedFilesStat`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gallery/deleteDuplicateFiles
- **摘要:** 删除重复文件
- **说明:** 删除指定的重复文件记录
- **OperationId:** `GalleryController_deleteDuplicateFiles`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number, MD5: string, galleryIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object \| object |

### POST /gallery/folderPathRebase
- **摘要:** 文件夹路径重置检查
- **说明:** 检查并更新图库文件夹路径
- **OperationId:** `GalleryController_folderPathRebase`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number, newPath: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回路径重置检查结果 | application/json: object \| object |

### POST /gallery
- **摘要:** 创建图库
- **说明:** 创建新的图库
- **OperationId:** `GalleryController_create`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateGalleryDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 返回创建的图库信息 | application/json: object { id: number, name: string, cover: number, weights: number, fileDeleteOnlyAdmin: boolean, hide: boolean, func_exclude: array<string>, folders: array<object> } |

### GET /gallery
- **摘要:** 获取所有图库
- **说明:** 查出所有的图库，并包含文件夹信息 - web admin 图库管理
- **OperationId:** `GalleryController_findAll`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库列表（不含隐藏图库） | application/json: array<object> |

### GET /gallery/all
- **摘要:** 获取所有图库（含隐藏）
- **说明:** 查出所有的图库，并包含文件夹信息 - 含隐藏图库 - web admin 图库管理
- **OperationId:** `GalleryController_findAllWithHidden`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库列表（含隐藏图库） | application/json: array<object> |

### GET /gallery/galleryUsers
- **摘要:** 获取图库用户列表
- **说明:** 获取所有图库关联的用户信息
- **OperationId:** `GalleryController_findAllGalleryUsers`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库用户列表 | application/json: array<object> |

### GET /gallery/stat/{id}
- **摘要:** 获取图库统计信息
- **说明:** 获取单个图库或所有图库的统计信息。id为"list"返回图库列表用于统计，id为"all"返回所有图库汇总统计
- **OperationId:** `GalleryController_statOne`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 图库ID，特殊值：list-返回图库列表，all-返回所有图库统计 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回统计信息 | application/json: array<object> \| object |

### GET /gallery/scan/{id}
- **摘要:** 扫描图库
- **说明:** 触发指定图库的文件扫描
- **OperationId:** `GalleryController_scanGallery`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 图库ID |
| query | `type` | any | 否 | 扫描类型 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回扫描结果 | application/json: object { scanResult: string, msg: string } |

### GET /gallery/{id}
- **摘要:** 获取单个图库信息
- **说明:** 根据ID获取图库详细信息
- **OperationId:** `GalleryController_findOne`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 图库ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库详细信息 | application/json: object { id: number, name: string, cover: number, weights: number, fileDeleteOnlyAdmin: boolean, hide: boolean, func_exclude: array<string>, folders: array<object> } |

### PATCH /gallery/{id}
- **摘要:** 更新图库信息
- **说明:** 根据ID更新图库信息
- **OperationId:** `GalleryController_update`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 图库ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateGalleryDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { id: number, name: string, cover: number, weights: number, fileDeleteOnlyAdmin: boolean, hide: boolean, func_exclude: array<string> } |

### DELETE /gallery/{id}
- **摘要:** 删除图库
- **说明:** 删除指定图库，同时更新关联用户的图库信息
- **OperationId:** `GalleryController_remove`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 图库ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object { raw: array<object>, affected: number } |

### POST /gallery/updateWeights
- **摘要:** 更新图库权重
- **说明:** 更新指定图库的排序权重
- **OperationId:** `GalleryController_updateWeights`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number, weights: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { generatedMaps: array<object>, raw: array<object>, affected: number } |

### POST /gallery/createFolders
- **摘要:** 批量创建文件夹
- **说明:** 根据路径列表批量创建文件夹记录
- **OperationId:** `GalleryController_createFolders`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folders: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回创建结果 | application/json: array<object> |

### POST /gallery/func_exclude
- **摘要:** 获取功能排除的图库ID
- **说明:** 获取指定类型功能排除的图库ID列表，type为all时返回详细的排除配置
- **OperationId:** `GalleryController_getFuncExcludeIds`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回排除的图库ID列表或详细配置 | application/json: array<number> \| array<object> |

### POST /gallery/skippedFolderLogs
- **摘要:** 获取跳过扫描的文件夹日志
- **说明:** 获取是否有图库的文件夹跳过了扫描
- **OperationId:** `GalleryController_getSkippedFolderLogs`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回跳过扫描的文件夹日志列表 | application/json: array<object> |

## people-descriptor 仅限管理员调用

### POST /people-descriptor
- **摘要:** 创建人物特征描述
- **OperationId:** `PeopleDescriptorController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreatePeopleDescriptorDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: object { id: number, files: array<number>, pass: boolean, peopleBaseId: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /people-descriptor/info
- **摘要:** 获取人脸识别任务信息（浏览器辅助识别用）
- **OperationId:** `PeopleDescriptorController_getInfo`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 获取成功 | application/json: object { inProcess: number, total: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /people-descriptor/resetFileStatus
- **摘要:** 重置文件人脸识别状态
- **OperationId:** `PeopleDescriptorController_resetFileStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 重置成功 | application/json: object { raw: object, affected: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /people-descriptor/itemDistV2
- **摘要:** 计算两个特征描述之间的距离（V2版本）
- **OperationId:** `PeopleDescriptorController_itemDistV2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id1: number, id2: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 计算成功 | application/json: object { distance: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /people-descriptor/faceRegTask
- **摘要:** 获取人脸识别任务列表（浏览器辅助识别用）
- **OperationId:** `PeopleDescriptorController_getTfTaskFiles`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 获取成功 | application/json: object { items: array<object>, MD5HasAdded: object, ignoreNum: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /people-descriptor/faceRegResult
- **摘要:** 保存人脸识别结果（浏览器辅助识别用）
- **OperationId:** `PeopleDescriptorController_saveFaceRegResult`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { MD5: string, scale: number, width: number, height: number, files: array<number>, result: array<object> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 保存成功 | application/json: object { pass: boolean, ids: array<number> } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /people-descriptor/findDescriptorOfFileForPeople
- **摘要:** 查找人物对应的特征描述
- **OperationId:** `PeopleDescriptorController_findDescriptorOfFileForPeople`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { peopleBaseIds: array<number>, fileId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 查询成功 | application/json: array<object> |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### POST /people-descriptor/findLikelyBase0Descriptor
- **摘要:** 查找相似的未匹配人物特征描述
- **OperationId:** `PeopleDescriptorController_adminFindLikelyNoMatchedDescriptor`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { descriptorId: number, vec: string, distance: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 查询成功 | application/json: array<object> |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### GET /people-descriptor/findDescriptorOfFile/{fileId}
- **摘要:** 获取文件的人脸特征描述列表
- **OperationId:** `PeopleDescriptorController_findDescriptorOfFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `fileId` | number | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 查询成功 | application/json: array<object> |
| 401 | 未授权访问 |  |

### GET /people-descriptor/{id}
- **摘要:** 根据ID获取人物特征描述
- **OperationId:** `PeopleDescriptorController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 特征描述ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 查询成功 | application/json: object { id: number, files: array<number>, pass: boolean, peopleBaseId: number, box: object } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

### PATCH /people-descriptor/{id}
- **摘要:** 更新人物特征描述
- **OperationId:** `PeopleDescriptorController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 特征描述ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdatePeopleDescriptorDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 更新成功 | application/json: object { id: number, files: array<number>, pass: boolean, peopleBaseId: number } |
| 401 | 未授权访问 |  |
| 403 | 无权限访问 |  |

## people 仅限管理员调用

### POST /people
- **摘要:** 创建人物
- **OperationId:** `PeopleController_create`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreatePeopleDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 人物创建成功 | application/json: object { id: number, name: string, userId: number, cover: number, isHide: boolean, baseIds: array<number>, count: number, ver: number } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /people
- **摘要:** 获取所有人物列表
- **OperationId:** `PeopleController_findAll`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回所有人物列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /people/base/{id}
- **摘要:** 根据人物基础ID获取人物列表
- **OperationId:** `PeopleController_findById`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 人物基础ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物列表 | application/json: array<object> |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### GET /people/{id}
- **摘要:** 根据ID获取人物详情
- **OperationId:** `PeopleController_findOne`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 人物ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物详情 | application/json: object { id: number, name: string, userId: number, cover: number, isHide: boolean, baseIds: array<number>, count: number, ver: number, ... } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### PATCH /people/{id}
- **摘要:** 更新人物信息
- **OperationId:** `PeopleController_update`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 人物ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdatePeopleDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 人物信息更新成功 | application/json: object { id: number, name: string, userId: number, cover: number, isHide: boolean, baseIds: array<number>, count: number, ver: number } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

### DELETE /people/{id}
- **摘要:** 删除人物
- **OperationId:** `PeopleController_remove`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 人物ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 人物删除成功 | application/json: object { raw: object, affected: number } |
| 401 | 未授权 |  |
| 403 | 无权限（需要管理员权限） |  |

## system-config - 系统配置

### GET /system-config
- **摘要:** 获取所有系统配置 - adminOnly
- **OperationId:** `SystemConfigController_findAll`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回所有系统配置列表 | application/json: array<object> |

### PATCH /system-config
- **摘要:** 更新系统配置 - adminOnly
- **OperationId:** `SystemConfigController_updateByValue`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateSystemConfigDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 更新配置成功 | application/json: object { success: boolean, message: string } |

### GET /system-config/{key}
- **摘要:** 根据key获取系统配置 - adminOnly
- **OperationId:** `SystemConfigController_findByKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 配置键名 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回指定key的配置信息 | application/json: object { id: number, key: string, value: string, type: string, description: string, created_at: string, updated_at: string } |

### POST /system-config/patchMulti
- **摘要:** 批量修改图库设置配置值 - adminOnly
- **OperationId:** `SystemConfigController_patchMultiForFront`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | array<any> |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 批量修改配置成功 | application/json: object { n: number } |

### POST /system-config/getFFmpegHWList
- **摘要:** 获取FFmpeg硬件加速列表 - adminOnly
- **OperationId:** `SystemConfigController_getFFmpeg_HWList`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回FFmpeg硬件加速列表 | application/json: array<object> |

### POST /system-config/pgDump
- **摘要:** 数据库备份 - adminOnly
- **OperationId:** `SystemConfigController_pgDump`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 数据库备份成功 | application/json: object { n: number, distPath: string } |

### POST /system-config/systemStatus
- **摘要:** 获取系统状态
- **OperationId:** `SystemConfigController_systemStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回系统状态信息 | application/json: object { pgEnable: boolean, redisEnable: boolean } |

### POST /system-config/changeTableVecLength
- **摘要:** 修改数据库向量的长度 - adminOnly
- **OperationId:** `SystemConfigController_changeTableVecLength`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("face_v2", "CLIP"), len: enum(128, 512, 1024) } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 修改向量长度成功 | application/json: object { success: boolean, message: string } |

### POST /system-config/getTableVecLength
- **摘要:** 获取数据库向量长度 - adminOnly
- **OperationId:** `SystemConfigController_getTableVecLength`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("face_v2", "CLIP") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回数据库向量长度 | application/json: object { len: number } |

### POST /system-config/test/ocrApi
- **摘要:** 测试OCR API配置 - adminOnly
- **OperationId:** `SystemConfigController_testOcrApiConfig`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { uri: string, api_key: string, type: enum("facial", "ocr") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 测试OCR API配置成功 | application/json: object { pass: boolean, err: string, response: object } |

### POST /system-config/db/prepareCLIP
- **摘要:** 准备CLIP表 - adminOnly
- **OperationId:** `SystemConfigController_prepareForClip`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | CLIP表准备完成 | application/json: object { success: boolean, message: string } |

### POST /system-config/db/prepareFaceRegV2
- **摘要:** 准备人脸识别V2表 - adminOnly
- **OperationId:** `SystemConfigController_prepareFaceRegV2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 人脸识别V2表准备完成 | application/json: object { success: boolean, message: string } |

### POST /system-config/switchUseFaceRegV2
- **摘要:** 切换人脸识别版本 - adminOnly
- **OperationId:** `SystemConfigController_switchUseFaceRegV2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { open: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 切换人脸识别版本成功 | application/json: object { key: string, value: string } |

### POST /system-config/configInfo
- **摘要:** 获取配置信息 - adminOnly
- **OperationId:** `SystemConfigController_configInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回系统配置信息 | application/json: object { cacheVer: number, faceRegVer: string, categoryOneId: number, cpuThreadNum: number, taskMaxThreadNum: number, faceApiConfig: object, dbTZ: string } |

### POST /system-config/dbReIndex
- **摘要:** 重建数据库索引 - adminOnly
- **OperationId:** `SystemConfigController_dbReIndex`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回重建索引的数量 | application/json: object { n: number } |

### POST /system-config/dbReIndexInfo
- **摘要:** 获取数据库重建索引进度 - adminOnly
- **OperationId:** `SystemConfigController_dbReIndexInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回数据库重建索引进度 | application/json: object { progress: number, status: string, startTime: number } |

### POST /system-config/dbReIndexForTZ
- **摘要:** 重新生成时区相关的index索引 - adminOnly
- **OperationId:** `SystemConfigController_dbReIndexForTZ`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回重新生成时区索引的结果 | application/json: object { success: boolean, message: string } |

### POST /system-config/getLibheifVersion
- **摘要:** 获取libheif版本 - adminOnly
- **OperationId:** `SystemConfigController__getLibheifVersion`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回libheif版本信息 | application/json: object { version: string, list: array<string> } |

### POST /system-config/libheifVersion
- **摘要:** 切换libheif版本 - adminOnly
- **OperationId:** `SystemConfigController__switchLibheifVersion`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileName: enum("libheif.so.1.17.6", "libheif.so.1.18.0") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 切换libheif版本成功 | application/json: object { n: number } |

### POST /system-config/offlineID
- **摘要:** 获取离线ID - adminOnly
- **OperationId:** `SystemConfigController_postOfflineID`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回离线ID信息 | application/json: object { offlineId: string } |

### POST /system-config/verifyAuthOnlineInBrowser
- **摘要:** 在线验证授权 - adminOnly
- **OperationId:** `SystemConfigController_verifyAuthOnlineInBrowser`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("getData", "postData"), data: object } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 获取验证数据 | application/json: object { params: object, response: object } |

### POST /system-config/getLogs
- **摘要:** 获取日志 - adminOnly
- **OperationId:** `SystemConfigController_getLogsInMem`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回内存中的日志 | application/json: object { data: array<object> } |

### POST /system-config/clearLogs
- **摘要:** 清空日志 - adminOnly
- **OperationId:** `SystemConfigController_clearLogsInMem`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 清空日志成功 | application/json: object { n: number } |

## file-delete-log - 文件删除日志

### POST /file-delete-log
- **摘要:** 创建文件删除日志
- **说明:** 管理员权限：创建一条文件删除日志记录
- **OperationId:** `FileDeleteLogController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateFileDeleteLogDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功，返回创建的日志记录 | application/json: object { id: number, filePath: string, fileName: string, fileSize: number, deleteTime: string:date-time, userId: number } |
| 401 | 未授权访问 |  |
| 403 | 无管理员权限 |  |

### GET /file-delete-log
- **摘要:** 分页查询文件删除日志
- **说明:** 管理员权限：分页获取文件删除日志列表，按删除时间倒序排列
- **OperationId:** `FileDeleteLogController_findAll`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pageSize` | number | 否 | 每页数量，默认为20 |
| query | `pageNo` | number | 否 | 页码，默认为1 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回日志列表和总数 | application/json: object { count: number, list: array<object> } |
| 401 | 未授权访问 |  |
| 403 | 无管理员权限 |  |

### GET /file-delete-log/{id}
- **摘要:** 根据ID查询删除日志
- **说明:** 管理员权限：根据日志ID获取单条文件删除日志详情
- **OperationId:** `FileDeleteLogController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 日志ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回日志详情 | application/json: object { id: number, filePath: string, fileName: string, fileSize: number, deleteTime: string:date-time, userId: number } |
| 401 | 未授权访问 |  |
| 403 | 无管理员权限 |  |
| 404 | 日志不存在 |  |

### POST /file-delete-log/clearData
- **摘要:** 清空所有删除日志
- **说明:** 管理员权限：清空文件删除日志表中的所有数据（危险操作，不可恢复）
- **OperationId:** `FileDeleteLogController_clearAllData`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 清空成功 | application/json: object { raw: object, affected: number } |
| 401 | 未授权访问 |  |
| 403 | 无管理员权限 |  |

## api-album 相册

### POST /api-album
- **摘要:** 新建相册
- **OperationId:** `AlbumController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateAlbumDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: object { id: number, name: string, desc: string, count: number } |

### GET /api-album
- **摘要:** 我的相册列表
- **OperationId:** `AlbumController_findAll`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册列表 | application/json: array<object> |

### GET /api-album/{id}
- **摘要:** 相册详情
- **OperationId:** `AlbumController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |
| query | `tzOffset` | number | 否 | 客户端时区，比如 UTC+8 为 -480 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册详情 | application/json: object { id: number, name: string, desc: string, cover: string, count: number, weights: number, mtime: string:date-time, create_time: string:date-time, ... } |

### PATCH /api-album/{id}
- **摘要:** 修改相册
- **OperationId:** `AlbumController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateAlbumDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 修改成功 | application/json: object { affected: number } |

### PUT /api-album/{id}
- **摘要:** 修改相册 - patch兼容
- **OperationId:** `AlbumController_update_put`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateAlbumDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 修改成功 | application/json: object { affected: number } |

### DELETE /api-album/{id}
- **摘要:** 删除相册
- **OperationId:** `AlbumController_remove`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 删除成功 | application/json: object { affected: number } |

### GET /api-album/files/{id}
- **摘要:** 相册文件列表
- **OperationId:** `AlbumController_findAlbumFiles`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件列表（已废弃） | application/json: array<object> |

### GET /api-album/filesV2/{id}
- **摘要:** 相册文件列表 - 时间线
- **OperationId:** `AlbumController_findAlbumFilesV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |
| query | `listVer` | string | 否 | 列表版本，v2时返回更多文件 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表 | application/json: object { result: array<object>, duplicateFiles: object, totalCount: number } |

### GET /api-album/ignoreFiles/{id}
- **摘要:** 相册排除的文件列表 - 时间线 - 曾经在相册内手动移出的照片
- **OperationId:** `AlbumController_findAlbumIgnoreFilesV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的排除文件列表 | application/json: object { result: array<object>, duplicateFiles: object, totalCount: number } |

### GET /api-album/filesFlat/{id}
- **摘要:** 相册文件列表 - 给PhotosFlatList用的精简数据版
- **OperationId:** `AlbumController_findAlbumFilesFlat`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回精简的文件列表 | application/json: array<object> |

### GET /api-album/fileInAlbums/{id}
- **摘要:** 文件在哪些相册中 - 返回相册id
- **OperationId:** `AlbumController_fileInAlbums`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回包含该文件的相册ID数组 | application/json: array<number> |

### GET /api-album/fileInAlbumsList/{id}
- **摘要:** 文件在哪些相册中 - 返回相册信息
- **OperationId:** `AlbumController_fileInAlbumsList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回包含该文件的相册信息列表 | application/json: array<object> |

### POST /api-album/checkForFavorites
- **摘要:** 检查【收藏夹】 相册是否已经创建过
- **OperationId:** `AlbumController_checkAlbumForFav`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 返回收藏夹相册信息 | application/json: object { id: number, name: string, desc: string, cover: string, count: number } |

### POST /api-album/addFileToAlbum
- **摘要:** 添加文件至相册中
- **OperationId:** `AlbumController_addFileToAlbum`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: string, files: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 添加成功 | application/json: object { id: number, name: string, desc: string, count: number } |

### POST /api-album/removeFileFromAlbum
- **摘要:** 将文件从相册中删除
- **OperationId:** `AlbumController_removeFileFromAlbum`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: string, files: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 删除成功 | application/json: object { id: number, name: string, desc: string, count: number } |

### GET /api-album/link/{id}
- **摘要:** 相册的自动更新配置
- **OperationId:** `AlbumController_findAutoLinkList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回自动更新配置列表 | application/json: array<object> |

### POST /api-album/link/{id}
- **摘要:** 添加 相册 自动配置
- **OperationId:** `AlbumController_addAutoLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("tag", "folder", "people"), value: string, exclude: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 添加成功 | application/json: object { n: number } |

### DELETE /api-album/link/{id}
- **摘要:** 删除 相册 自动配置
- **OperationId:** `AlbumController_delAutoLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { linkId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 删除成功 | application/json: object { n: number } |

### POST /api-album/linkSyncFiles/{id}
- **摘要:** 相册 自动关联 更新文件
- **OperationId:** `AlbumController_syncAutoLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 同步成功 | application/json: object { n: number } |

### POST /api-album/hlinkAlbum
- **摘要:** 相册硬链接 - 触发同步
- **OperationId:** `AlbumController_hlinkAlbum`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 同步成功 | application/json: object { success: boolean, msg: string } |

### POST /api-album/addAlbumHLink
- **摘要:** 相册 硬链接 创建- admin only
- **OperationId:** `AlbumController_addAlbumHLink`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: number, dest: string, folder_name_type: enum("", "Y", "YM", "YMD"), file_name_type: enum("", "time") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: object { id: number, albumId: number, dest: string, folder_name_type: string, file_name_type: string } |

### POST /api-album/updateAlbumHLink
- **摘要:** 相册 硬链接 更新- admin only
- **OperationId:** `AlbumController_updateAlbumHLink`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number, dest: string, folder_name_type: enum("", "Y", "YM", "YMD"), file_name_type: enum("", "time") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 更新成功 | application/json: object { affected: number } |

### POST /api-album/delAlbumHLink
- **摘要:** 相册 硬链接 - admin only
- **OperationId:** `AlbumController_delAlbumHLink`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: number, id: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 删除成功 | application/json: object { affected: number } |

### GET /api-album/getAlbumHardLinkByAlbumId/{id}
- **摘要:** 相册 硬链接
- **OperationId:** `AlbumController_getAlbumHardLinkByAlbumId`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 相册ID或"all" |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册的硬链接配置列表 | application/json: array<object> |

### GET /api-album/getAlbumHardLinkById/{id}
- **摘要:** 相册 硬链接
- **OperationId:** `AlbumController_getAlbumHardLinkById`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 硬链接ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回硬链接详情 | application/json: object { id: number, dest: string, folder_name_type: string, file_name_type: string, msg: string, success_count: number, total_count: number, run_time: string:date-time, ... } |

### POST /api-album/findAllForHardLink/list
- **摘要:** 硬链接 显示的全部相册列表 - admin only
- **OperationId:** `AlbumController_findAllForHardLink`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { userIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 返回相册列表 | application/json: array<object> |

## people-base 仅限管理员调用

### GET /people-base/count
- **摘要:** 获取人物基础总数
- **OperationId:** `PeopleBaseController_count`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物基础总数 | application/json: number |

### GET /people-base/findForGenPeople
- **摘要:** 获取待生成人物的PeopleBase列表
- **OperationId:** `PeopleBaseController_findForGenPeople`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回待生成人物的PeopleBase列表 | application/json: array<object> |

### GET /people-base/distance
- **摘要:** 计算两个人物基础之间的距离
- **OperationId:** `PeopleBaseController_baseIdDistance`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `id2` | number | 否 | 人物基础ID2 |
| query | `id1` | number | 否 | 人物基础ID1 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回两个人物基础之间的平均距离值（0-1之间，越小越相似） | application/json: number |

### GET /people-base/findAllPeopleBaseForMerge
- **摘要:** 分页获取所有人物基础列表（用于合并）
- **OperationId:** `PeopleBaseController_findAllPeopleBase`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pageSize` | number | 否 | 每页数量 |
| query | `pageNo` | number | 否 | 页码 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物基础列表 | application/json: array<object> |

### GET /people-base/findAllMergerPeopleBase
- **摘要:** 获取所有已合并的人物基础列表
- **OperationId:** `PeopleBaseController_findAllMergerPeopleBase`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回已合并的人物基础列表 | application/json: array<object> |

### GET /people-base/findPeopleBaseFiles
- **摘要:** 根据人物基础ID获取关联的文件列表
- **OperationId:** `PeopleBaseController_findPeopleBaseFiles`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `id` | number | 是 | 人物基础ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表 | application/json: array<object> |

### POST /people-base/findFileMD5ByFileIds
- **摘要:** 根据文件ID列表获取MD5值（用于显示封面）
- **OperationId:** `PeopleBaseController_findMD5ByIds`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件MD5信息列表 | application/json: array<object> |

### POST /people-base/findBaseInfoByIds
- **摘要:** 根据人物基础ID列表获取基础信息 - adminOnly
- **OperationId:** `PeopleBaseController_findBaseInfoByIds`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物基础信息列表 | application/json: array<object> |

### POST /people-base/adminMergeBaseIds
- **摘要:** 合并人物基础 - adminOnly
- **OperationId:** `PeopleBaseController_adminMergeBaseIds`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { targetId: number, ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 合并成功 | application/json: object { n: number } |

### POST /people-base/adminSetBaseId
- **摘要:** 设置人物基础（合并或更新名称）- adminOnly
- **OperationId:** `PeopleBaseController_adminSetBaseId`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { baseId: number, ids: array<number>, type: enum("merge", "setName"), name: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 操作成功 | application/json: object { n: number } |

### GET /people-base/baseInFileInfo
- **摘要:** 获取人物基础对应照片识别的人脸信息 - adminOnly
- **OperationId:** `PeopleBaseController_peopleInFileInfo`
- **认证:** bearer 或 api-key

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `fileId` | string | 否 | 文件ID |
| query | `baseId` | string | 否 | 人物基础ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人脸信息列表 | application/json: array<object> |

### POST /people-base/getNameFromPeople
- **摘要:** 根据人物基础ID获取人物名称 - adminOnly
- **OperationId:** `PeopleBaseController_getNameFromPeople`
- **认证:** bearer 或 api-key

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { peopleBaseId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物名称 | application/json: object { name: string } |

## api-tag 标签管理

### POST /api-tag
- **摘要:** 创建标签
- **说明:** 创建新的标签（注意：创建的标签不会直接关联文件）
- **OperationId:** `TagController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateTagDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功，返回创建的标签信息 | application/json: object { id: number, name: string, is_hide: boolean } |
| 401 | 未授权访问 |  |

### GET /api-tag
- **摘要:** 获取标签列表
- **说明:** 获取用户可访问图库中的所有标签，支持分页限制
- **OperationId:** `TagController_findAll`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 否 | 图库 ID 列表（逗号分隔），不传则使用用户所有图库 |
| query | `type` | string | 否 | 类型：all 或不传，为 all 时返回最多 10000 条 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回标签列表 | application/json: array<object> |
| 401 | 未授权访问 |  |

### GET /api-tag/tag/{id}
- **摘要:** 获取标签详情
- **说明:** 根据标签 ID 获取标签的详细信息
- **OperationId:** `TagController_findTagDetail`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 标签 ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回标签详情 | application/json: object { id: number, name: string, is_hide: boolean } |
| 401 | 未授权访问 |  |
| 403 | 无权访问该标签 |  |

### PATCH /api-tag/tag/{id}
- **摘要:** 更新标签（PATCH）
- **说明:** 更新标签的隐藏状态
- **OperationId:** `TagController_updateTag`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 标签 ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { is_hide: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { n: number } |
| 401 | 未授权访问 |  |

### PUT /api-tag/tag/{id}
- **摘要:** 更新标签（PUT）
- **说明:** 更新标签的隐藏状态（与 PATCH 功能相同）
- **OperationId:** `TagController_updateTag_put`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 标签 ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { is_hide: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { n: number } |
| 401 | 未授权访问 |  |

### GET /api-tag/files/{id}
- **摘要:** 获取标签关联的文件列表
- **说明:** 获取指定标签下的所有文件，按日期分组，排除 RAW 文件
- **OperationId:** `TagController_findTagFiles`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 标签 ID |
| query | `galleryIds` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表 | application/json: array<object> |
| 401 | 未授权访问 |  |

### POST /api-tag/editFileTag
- **摘要:** 编辑文件标签
- **说明:** 为文件添加或删除标签，支持添加新标签或已存在的标签
- **OperationId:** `TagController_editFileTag`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, fileId: number, tagId: number, tagName: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 操作成功，返回结果 | application/json: object \| object \| object |
| 401 | 未授权访问或无权修改该文件 |  |

### POST /api-tag/fileAddTags
- **摘要:** 批量为文件添加标签
- **说明:** 为指定文件批量添加多个已存在的标签，并同步到 EXIF 信息
- **OperationId:** `TagController_fileAddTags`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: number, tagIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 添加成功 | application/json: object \| object |
| 401 | 未授权访问或无权修改该文件 |  |

### POST /api-tag/fileDelTagsInDb
- **摘要:** 批量删除文件标签（仅数据库）
- **说明:** 批量删除文件的标签，仅修改数据库，不同步到文件 EXIF 信息（用于辅助 fileAddTags、editFileTag 使用）
- **OperationId:** `TagController_fileDelTagsInDb`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number>, tagIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 删除成功 | application/json: object { n: number } |
| 401 | 未授权访问 |  |

### POST /api-tag/saveToExif
- **摘要:** 批量保存标签到 EXIF
- **说明:** 将文件的标签信息批量保存到 EXIF 元数据中
- **OperationId:** `TagController_saveToExif`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 保存成功 | application/json: object { n: number } |
| 401 | 未授权访问 |  |

### POST /api-tag/hideTag
- **摘要:** 隐藏空标签
- **说明:** 用户隐藏指定的空标签（标签下没有文件）
- **OperationId:** `TagController_hideTag`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { tagId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 隐藏成功 | application/json: object { n: number } |
| 401 | 未授权访问 |  |

### POST /api-tag/hideEmptyTags
- **摘要:** 隐藏所有空标签
- **说明:** 隐藏用户所有的空标签（标签下没有文件）
- **OperationId:** `TagController_hideEmptyTags`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 操作成功 | application/json: object { n: number } |
| 401 | 未授权访问 |  |

### POST /api-tag/tagNames
- **摘要:** 根据 ID 获取标签名称
- **说明:** 根据标签 ID 列表批量获取标签名称
- **OperationId:** `TagController_getTagNames`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { tagIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 返回标签名称列表 | application/json: array<object> |
| 401 | 未授权访问 |  |

## gateway - 前端API请求主要的入口

### GET /gateway/test
- **摘要:** 测试接口
- **说明:** 用于测试的接口
- **OperationId:** `GatewayController_test`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回的固定测试结果 | application/json: object { n: number } |

### GET /gateway/userInfo
- **摘要:** 用户信息-当前登录用户
- **说明:** 获取当前登录用户的详细信息
- **OperationId:** `GatewayController_getUserInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回用户信息 | application/json: object { id: number, username: string, nickname: string, avatar: string, uid: string, isAdmin: boolean, otpEnable: boolean } |

### GET /gateway/filesInTimeline
- **摘要:** 所有文件
- **说明:** 获取所有文件列表 - 已废弃
- **OperationId:** `GatewayController_findAllFiles`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `_t` | number | 否 | 时间戳 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件列表 | application/json: array<object> |

### GET /gateway/filesInTimelineV2
- **摘要:** 所有文件-时间线
- **说明:** 获取时间线上的所有文件，支持按图库过滤
- **OperationId:** `GatewayController_findAllFilesV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 否 | 多个图库ID，用下划线分隔 |
| query | `galleryId` | number | 否 | 单个图库ID |
| query | `_t` | number | 否 | 时间戳 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件列表 | application/json: array<object> |

### GET /gateway/timeline
- **摘要:** 照片-时间线按月分组统计数
- **说明:** 获取照片按月分组的统计数据
- **OperationId:** `GatewayController_getTimelineData`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `platform` | string | 否 | 平台类型 |
| query | `galleryIds` | string | 否 | 多个图库ID，用下划线分隔 |
| query | `galleryId` | number | 否 | 单个图库ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按月统计数据 | application/json: array<object> |

### POST /gateway/timelineMonth
- **摘要:** 照片-时间线 月数据
- **说明:** 获取指定月份的照片数据
- **OperationId:** `GatewayController_getTimelineMonthData`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { galleryId: number, galleryIds: string, platform: string, month: string, monthList: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回月份数据 | application/json: array<object> |

### GET /gateway/myGalleryList
- **摘要:** 用户的图库列表
- **说明:** 获取当前用户可访问的图库列表
- **OperationId:** `GatewayController_userGalleryList`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库列表 | application/json: array<object> |

### POST /gateway/galleryNames
- **摘要:** 获取图库名称
- **OperationId:** `GatewayController_getGalleryNames`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { galleryIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库名称列表 | application/json: array<object> |

### POST /gateway/dayFileMore
- **摘要:** 单天剩余文件
- **说明:** 获取单天未显示完的剩余文件
- **OperationId:** `GatewayController_findDayFileMore`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number>, inTrash: boolean, order: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回剩余文件列表 | application/json: array<object> |

### POST /gateway/dayFiles
- **摘要:** 某一天的所有文件
- **说明:** 获取指定日期范围内的所有文件
- **OperationId:** `GatewayController_dayAllFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { tokenAtStart: number, tokenAtEnd: number, galleryIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件列表 | application/json: array<object> |

### POST /gateway/filesInfo
- **摘要:** 下载前查询文件信息
- **说明:** 下载前查询文件详细信息，支持批量下载
- **OperationId:** `GatewayController_findFilesInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number>, albumId: number, type: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件信息列表 | application/json: array<object> |

### GET /gateway/filesInTimelineCount
- **摘要:** 时间线中所有文件的数量
- **说明:** 获取时间线中所有文件的总数量
- **OperationId:** `GatewayController_findAllFilesNum`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `_t` | number | 否 | 时间戳 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件数量 | application/json: object { count: number } |

### POST /gateway/user/profile
- **摘要:** 更新个人资料
- **说明:** 用户更新自己的昵称
- **OperationId:** `GatewayController_updateProfile`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { nickname: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { success: boolean, nickname: string, msg: string } |

### GET /gateway/avatar/{fileName}
- **摘要:** 显示用户头像
- **说明:** 返回用户头像图片文件流
- **OperationId:** `GatewayController_renderAvatar`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `fileName` | string | 是 | 头像文件名，格式为 {uid}.jpg |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回头像图片文件流 |  |
| 404 | 头像不存在 |  |

### POST /gateway/avatar
- **摘要:** 上传头像
- **说明:** 用户上传自己的头像，支持JPG/PNG/WEBP格式，最大2MB
- **OperationId:** `GatewayController_uploadAvatar`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| multipart/form-data | 是 | object { file: string:binary } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回上传结果 | application/json: object { success: boolean, avatar: string, msg: string } |

### POST /gateway/folderFilesInDisk
- **摘要:** 查看文件夹文件 - 实时读取硬盘文件列表
- **OperationId:** `GatewayControllerPart1_folderFilesInDisk`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folderId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gateway/annualData
- **摘要:** 获取年度统计数据
- **OperationId:** `GatewayControllerPart1_annualData`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { year: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gateway/refreshFileDescriptorBatch
- **摘要:** 刷新照片人脸
- **OperationId:** `GatewayControllerPart1_refreshFileDescriptor`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gateway/getTranscodeError
- **摘要:** 查询转码错误信息
- **OperationId:** `GatewayControllerPart1_getTranscodeError`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { inputFilePath: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gateway/addFaceRect
- **摘要:** 手动添加人脸识别框
- **OperationId:** `GatewayControllerPart1_addFaceRect`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: string, x: number, y: number, w: number, h: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### GET /gateway/fileInfo/{id}/{md5}
- **摘要:** 显示文件的详细信息
- **说明:** 根据ID和MD5获取文件详细信息 - 不检查图库权限
- **OperationId:** `GatewayControllerPart2_getFileDetail`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| path | `md5` | string | 是 | 文件MD5值 |
| query | `albumId` | string | 否 | 相册ID（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详细信息 | application/json: object { id: number, MD5: string, fileName: string, filePath: string, fileSize: number, fileType: string, tokenAt: string, width: number, ... } |
| 404 | 文件未找到 |  |

### GET /gateway/fileInfoById/{id}
- **摘要:** 显示文件的详细信息
- **说明:** 根据ID获取文件详细信息 - 实时检查图库权限
- **OperationId:** `GatewayControllerPart2_getFileServerPath`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| query | `albumId` | number | 否 | 相册ID（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详细信息 | application/json: object { id: number, MD5: string, fileName: string, filePath: string, fileSize: number, fileType: string, tokenAt: string } |

### GET /gateway/exifInfo/{id}
- **摘要:** 显示文件的exif信息
- **说明:** 获取文件的EXIF信息 - 实时检查图库权限
- **OperationId:** `GatewayControllerPart2_fileExifInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回EXIF信息 | application/json: object { Make: string, Model: string, DateTimeOriginal: string, ExposureTime: string, FNumber: string, ISOSpeedRatings: number, FocalLength: string, GPSLatitude: number, ... } |

### GET /gateway/fileTags/{id}
- **摘要:** 文件的标签列表
- **说明:** 获取文件关联的标签列表
- **OperationId:** `GatewayControllerPart2_findFileTags`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回标签列表 | application/json: array<object> |

### POST /gateway/extra/make
- **摘要:** 获取照片包含的相机品牌列表
- **说明:** 获取用户可访问照片中的所有相机品牌
- **OperationId:** `GatewayControllerPart2_fileExtraMake`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相机品牌列表 | application/json: array<object> |

### POST /gateway/extra/models
- **摘要:** 获取照片包含的设备列表
- **说明:** 根据相机品牌获取设备型号列表
- **OperationId:** `GatewayControllerPart2_fileExtraModelsWithMake`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { make: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回设备型号列表 | application/json: array<object> |

### GET /gateway/extra/models
- **摘要:** 获取照片包含的设备列表
- **说明:** 获取用户可访问照片中的所有设备型号
- **OperationId:** `GatewayControllerPart2_fileExtraModels`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回设备型号列表 | application/json: array<object> |

### POST /gateway/extra/lens
- **摘要:** 获取照片包含的镜头列表
- **说明:** 根据相机品牌和型号获取镜头列表
- **OperationId:** `GatewayControllerPart2_fileExtraLensWithModel`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { make: string, model: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回镜头列表 | application/json: array<object> |

### GET /gateway/extra/lens
- **摘要:** 获取照片包含的镜头列表
- **说明:** 获取用户可访问照片中的所有镜头
- **OperationId:** `GatewayControllerPart2_fileExtraLens`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回镜头列表 | application/json: array<object> |

### POST /gateway/extra/placeL1
- **摘要:** 获取地点列表 - 省
- **说明:** 获取照片中的省份列表
- **OperationId:** `GatewayControllerPart2_filePlaceL1`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回省份列表 | application/json: array<object> |

### POST /gateway/extra/placeL2
- **摘要:** 获取地点列表 - 市
- **说明:** 根据省份获取城市列表
- **OperationId:** `GatewayControllerPart2_filePlaceL2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { province: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回城市列表 | application/json: array<object> |

### POST /gateway/extra/placeL3
- **摘要:** 获取地点列表 - 区
- **说明:** 根据省份和城市获取区县列表
- **OperationId:** `GatewayControllerPart2_filePlaceL3`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { province: string, city: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回区县列表 | application/json: array<object> |

### GET /gateway/ocrInfo/{id}
- **摘要:** 显示文件的OCR结果
- **说明:** 获取文件的OCR识别结果 - 实时检查图库权限
- **OperationId:** `GatewayControllerPart2_fileOcrInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| query | `albumId` | number | 否 | 相册ID（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回OCR识别结果列表 | application/json: array<object> |

### POST /gateway/filesPath
- **摘要:** 获取指定ids文件的地址
- **说明:** 根据文件ID列表获取文件路径
- **OperationId:** `GatewayControllerPart2_filesPath`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件路径列表 | application/json: array<object> |

### POST /gateway/filesInMD5
- **摘要:** 根据MD5查询文件列表
- **说明:** 根据MD5值获取文件路径列表
- **OperationId:** `GatewayControllerPart2_filesInMD5`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { MD5: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件路径列表 | application/json: array<object> |

### GET /gateway/refreshFileThumbs/{id}
- **摘要:** 刷新文件的缩略图
- **说明:** 重新生成指定文件的缩略图
- **OperationId:** `GatewayControllerPart2_refreshFileThumbs`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| query | `videoSec` | number | 否 | 视频截图秒数（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### POST /gateway/uploadFileThumbs/{id}
- **摘要:** 上传文件缩略图
- **说明:** 为指定文件上传自定义缩略图
- **OperationId:** `GatewayControllerPart2_uploadFileThumbs`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| multipart/form-data | 是 | object { file: string:binary } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### POST /gateway/uploadFileThumbsForApp/{id}
- **摘要:** App上传文件缩略图
- **说明:** App端为指定文件上传自定义缩略图
- **OperationId:** `GatewayControllerPart2_uploadFileThumbsForApp`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| multipart/form-data | 是 | object { file: string:binary } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number, msg: string } |

### POST /gateway/HDThumbsConfig
- **摘要:** 获取高清缩略图配置
- **说明:** 获取高清缩略图的配置信息
- **OperationId:** `GatewayControllerPart2_getHDThumbsConfig`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回高清缩略图配置 | application/json: object { enabled: boolean, quality: number, maxWidth: number, maxHeight: number, configTargetFolder: boolean } |

### POST /gateway/uploadFileHDThumbs/{id}
- **摘要:** 上传高清缩略图
- **说明:** 为指定文件上传高清缩略图
- **OperationId:** `GatewayControllerPart2_uploadFileHdThumbs`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| multipart/form-data | 是 | object { file: string:binary } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number, msg: string } |

### POST /gateway/transcode
- **摘要:** 触发视频转码
- **说明:** 触发视频文件转码任务
- **OperationId:** `GatewayControllerPart2_transcodeFile`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, fileIds: array<number>, albumId: number, galleryId: number, force: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回转码任务结果 | application/json: object { n: number, code: string, transcodeStatus: number } |

### GET /gateway/fileInfoRT/{id}
- **摘要:** 获取文件最新EXIF信息
- **说明:** 获取文件最新的EXIF信息，并返回与数据库中的差异数据
- **OperationId:** `GatewayControllerPart2_getFileInfoRealTime`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回EXIF差异数据 | application/json: object { id: number, MD5: string, fileName: string, width: number, height: number, exifInfo: object } |

### POST /gateway/refreshFileDescriptor
- **摘要:** 刷新照片人脸
- **说明:** 重新识别指定文件的人脸信息
- **OperationId:** `GatewayControllerPart2_refreshFileDescriptor`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回刷新结果 | application/json: object { n: number } |

### POST /gateway/fileStat/{id}/{md5}
- **摘要:** 检查文件是否存在
- **说明:** 检查指定文件在磁盘上是否存在
- **OperationId:** `GatewayControllerPart2_statOneFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| path | `md5` | string | 是 | 文件MD5值 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件状态信息 | application/json: object { size: number, ino: number, mtime: string, ctime: string, message: string } |

### GET /gateway/fileStreamLink/{id}
- **摘要:** 获取串流地址
- **说明:** 获取文件串流地址，30分钟内有效
- **OperationId:** `GatewayControllerPart2_fileStreamLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| query | `shareAlbumId` | number | 否 | 分享相册ID（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回串流地址 | application/json: object { link: string, ttl: number } |

### GET /gateway/stream/{auth_code}/{name}
- **摘要:** 下载文件原图
- **说明:** 通过授权码下载文件原图
- **OperationId:** `GatewayControllerPart2_fileStreamPlay`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `auth_code` | string | 是 | 授权码 |
| path | `name` | string | 是 | 文件名 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件流 |  |
| 404 | 文件未找到 |  |

### GET /gateway/streamV2/{name}
- **摘要:** 下载文件原图V2
- **说明:** 需要调用 /auth/auth_code 获取auth_code之后，带上auth_code参数才能访问
- **OperationId:** `GatewayControllerPart2_fileStreamPlayV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `auth_code` | string | 是 | 授权码 |
| query | `type` | string | 否 | 类型：transcode-转码文件 |
| path | `name` | string | 是 | 文件名 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件流 |  |
| 404 | 文件未找到 |  |

### GET /gateway/file/{id}/{md5}
- **摘要:** 显示文件原图
- **说明:** 需要调用 /auth/auth_code 获取auth_code之后，带上auth_code参数才能访问
- **OperationId:** `GatewayControllerPart2_renderFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| path | `md5` | string | 是 | 文件MD5值 |
| query | `albumId` | number | 否 | 相册ID（可选） |
| query | `auth_code` | string | 是 | 授权码 |
| query | `type` | string | 否 | 类型：proxy-预览图、hd-高清预览图、ori-原图、transcode-视频的转码文件、motion-动态照片视频 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件流 |  |
| 404 | 文件未找到 |  |

### GET /gateway/fileForApi/{id}/{md5}
- **摘要:** 显示文件的大图 - 已废弃
- **OperationId:** `GatewayControllerPart2_renderFileForOpen`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |
| path | `md5` | string | 是 |  |
| query | `api_key` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件内容 |  |
| 404 | 文件未找到 |  |

### GET /gateway/fileMotion/{id}/{md5}
- **摘要:** 显示动态照片的视频部分
- **说明:** 获取动态照片（Motion Photo）的视频部分, 需要调用 /auth/auth_code 获取auth_code之后，带上auth_code参数才能访问
- **OperationId:** `GatewayControllerPart2_renderMotionPhoto`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| path | `md5` | string | 是 | 文件MD5值 |
| query | `albumId` | number | 否 | 相册ID（可选） |
| query | `auth_code` | string | 是 | 授权码 |
| query | `app` | string | 否 | 应用类型：ios \| android \| oh \| web |
| query | `type` | string | 否 | 类型：photo-照片部分、video-视频部分 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回动态照片的视频部分 |  |
| 404 | 文件未找到或不是动态照片 |  |

### GET /gateway/flv/{id}/{md5}
- **摘要:** 视频实时转码为flv
- **OperationId:** `GatewayControllerPart2_renderFileFlv`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |
| path | `md5` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回FLV视频流 |  |
| 404 | 文件未找到 |  |

### GET /gateway/jpeg/{md5}
- **摘要:** 显示heic图片的详情
- **OperationId:** `GatewayControllerPart2_renderImgWebp`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `md5` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回HEIC转换后的JPEG图片 |  |
| 404 | 文件不存在 |  |

### GET /gateway/fileDownload/{id}/{md5}
- **摘要:** 下载文件的原图
- **说明:** 需要调用 /auth/auth_code 获取auth_code之后，带上auth_code参数才能访问
- **OperationId:** `GatewayControllerPart2_downloadFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 文件ID |
| path | `md5` | string | 是 | 文件MD5值 |
| query | `albumId` | number | 否 | 相册ID（可选） |
| query | `auth_code` | string | 是 | 认证码 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件内容 |  |
| 404 | 文件未找到 |  |

### POST /gateway/fileDownloadStat/{id}/{md5}
- **摘要:** 获取下载文件的大小
- **OperationId:** `GatewayControllerPart2_downloadStatFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |
| path | `md5` | string | 是 |  |
| query | `type` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件大小 | application/json: object { fileSize: number } |

### GET /gateway/fileZIP/{downloadKey}
- **摘要:** 打包下载文件
- **OperationId:** `GatewayControllerPart2_downloadZIP`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `downloadKey` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回ZIP文件流 |  |
| 404 | 下载密钥无效或已过期 |  |

### GET /gateway/addressCountByCity
- **摘要:** 以市为单位的照片数量
- **OperationId:** `GatewayControllerPart2_addressCountByCity`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 |  |
| query | `type` | type | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按市统计的照片数量 | application/json: array<object> |

### GET /gateway/addressCountByDistrict/{city}
- **摘要:** 以区、县为单位的照片数量
- **OperationId:** `GatewayControllerPart2_addressCountByDistrict`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `city` | string | 是 |  |
| query | `galleryIds` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按区县统计的照片数量 | application/json: array<object> |

### GET /gateway/addressCountByTownship/{city}/{district}
- **摘要:** 以村、街道为单位的照片数量
- **OperationId:** `GatewayControllerPart2_addressCountByTownship`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `city` | string | 是 |  |
| path | `district` | string | 是 |  |
| query | `galleryIds` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按村街道统计的照片数量 | application/json: array<object> |

### GET /gateway/filesInAddress
- **摘要:** 对应地区下的所有照片
- **OperationId:** `GatewayControllerPart2_filesInAddress`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `type` | string | 是 |  |
| query | `city` | string | 是 |  |
| query | `district` | string | 否 |  |
| query | `township` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回指定地区的照片列表 | application/json: array<object> |

### GET /gateway/filesInAddressV2
- **摘要:** 对应地区下的所有照片
- **OperationId:** `GatewayControllerPart2_filesInAddressV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 |  |
| query | `type` | string | 是 |  |
| query | `city` | string | 是 |  |
| query | `district` | string | 否 |  |
| query | `township` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回指定地区的照片列表（按日期分组） | application/json: array<object> |

### GET /gateway/classifyTopList
- **摘要:** 按事物场景分类
- **OperationId:** `GatewayControllerPart2_classifyTopList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 |  |
| query | `type` | type | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按场景分类的照片列表 | application/json: array<object> |

### GET /gateway/classifyFileList
- **摘要:** 按事物场景分类-文件列表
- **OperationId:** `GatewayControllerPart2_classifyFileList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 |  |
| query | `id` | string | 否 |  |
| query | `cid` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分类下的照片列表（按日期分组） | application/json: array<object> |

### POST /gateway/editFileClassify
- **摘要:** 修改文件智能分类属性
- **OperationId:** `GatewayControllerPart2_editFileClassify`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, cid: number, fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### GET /gateway/filesInCategoriesV2
- **摘要:** 按类型分类的文件列表
- **说明:** 获取截图、自拍、视频、动态照片、全景、大文件等分类的文件列表
- **OperationId:** `GatewayControllerPart2_filesInCategoriesV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 |  |
| query | `type` | enum("screenshots", "selfies", "videos", "livePhotos", "pano", "largeFile") | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按类型分类的文件列表（按日期分组） | application/json: array<object> |

### GET /gateway/filesInTrash
- **摘要:** 回收站中的文件 - 已废弃
- **OperationId:** `GatewayControllerPart2_filesInTrash`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回回收站文件列表 | application/json: array<object> |

### GET /gateway/filesInTrashV2
- **摘要:** 回收站中的文件 - 已废弃
- **OperationId:** `GatewayControllerPart2_filesInTrashV2`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回回收站文件列表（按日期分组） | application/json: array<object> |

### GET /gateway/filesInTrashFlat
- **摘要:** 回收站中的文件
- **OperationId:** `GatewayControllerPart2_filesInTrashFlat`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `showName` | boolean | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回回收站文件列表 | application/json: array<object> |

### POST /gateway/findSimilarFiles
- **摘要:** 查找相似文件
- **说明:** 在指定图库中查找相似的照片文件
- **OperationId:** `GatewayControllerPart2_findDuplicateFilesWithGalleryIds`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { galleryIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相似文件列表 | application/json: array<object> |

### GET /gateway/filesInTrashAdmin
- **摘要:** 管理员-查看全部用户在回收站中的文件
- **OperationId:** `GatewayControllerPart2_filesInTrashAdmin`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `showName` | boolean | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回全部用户回收站文件列表 | application/json: array<object> |

### POST /gateway/findFilesWithInvalidGps
- **摘要:** 管理员-查看无法识别的GPS坐标
- **OperationId:** `GatewayControllerPart2_findFilesWithInvalidGps`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { pageNo: number, pageSize: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回无法识别GPS的文件列表 | application/json: object { count: number, list: array<object> } |

### POST /gateway/hideFiles
- **摘要:** 添加照片到隐私相册中
- **OperationId:** `GatewayControllerPart3_addHideFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { identifiers: array<object> } |

### POST /gateway/cancelHideFiles
- **摘要:** 从隐私相册内移出
- **OperationId:** `GatewayControllerPart3_cancelHideFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { affected: number } |

### POST /gateway/passwordCode
- **摘要:** 验证用户密码，验证通过后返回passwordCode 用于访问 /gateway/filesInHide
- **OperationId:** `GatewayControllerPart3_pwdCode`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { password: string, passwordEnc: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回密码验证结果 | application/json: object { passwordCode: string, msg: string } |

### POST /gateway/filesInHide
- **摘要:** 隐私相册中的照片
- **OperationId:** `GatewayControllerPart3_filesInHide`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { passwordCode: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回隐私相册照片列表（按日期分组） | application/json: array<object> |

### GET /gateway/recentFiles
- **摘要:** 最近添加的文件
- **OperationId:** `GatewayControllerPart3_filesRecent`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回最近添加的文件列表 | application/json: array<object> |

### GET /gateway/peopleList
- **摘要:** 人物列表
- **OperationId:** `GatewayControllerPart3_peopleList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 是 | 图库ID筛选 |
| query | `type` | string | 是 | 类型筛选 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物列表 | application/json: array<object> |

### GET /gateway/people/{id}
- **摘要:** 人物详情
- **OperationId:** `GatewayControllerPart3_peopleInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物详情 | application/json: object { id: number, name: string, cover: string, fileNum: number, baseIds: array<number>, isHide: boolean } |

### PATCH /gateway/people/{id}
- **摘要:** 修改人物详情
- **OperationId:** `GatewayControllerPart3_updatePeopleInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdatePeopleDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### PUT /gateway/people/{id}
- **摘要:** 修改人物详情 - patch兼容
- **OperationId:** `GatewayControllerPart3_updatePeopleInfo_put`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdatePeopleDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### POST /gateway/multiHidePeople
- **摘要:** 一键显示或隐藏人物
- **OperationId:** `GatewayControllerPart3_multiHidePeople`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { peopleFileNum: number, hideLTE: boolean, showGT: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number, msg: string } |

### POST /gateway/peopleNames
- **摘要:** 获取人物名称
- **OperationId:** `GatewayControllerPart3_getPeopleNames`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { peopleIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物名称列表 | application/json: array<object> |

### PATCH /gateway/reassignPeopleFile/{id}
- **摘要:** 修改人物详情
- **OperationId:** `GatewayControllerPart3_reassignPeopleFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### PUT /gateway/reassignPeopleFile/{id}
- **摘要:** 修改人物详情 - patch兼容
- **OperationId:** `GatewayControllerPart3_reassignPeopleFile_put`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### POST /gateway/editFileDescriptor
- **摘要:** 修改人物详情
- **OperationId:** `GatewayControllerPart3_editFileDescriptor`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, baseId: number, descriptorId: number, boxId: number, targetPeopleId: number, fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number, code: string, msg: string } |

### GET /gateway/peopleFileList
- **摘要:** 人物关联的文件列表
- **OperationId:** `GatewayControllerPart3_peopleFileList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `peopleId` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物关联的文件列表 | application/json: array<object> |

### GET /gateway/peopleFileListV2
- **摘要:** 人物关联的文件列表
- **OperationId:** `GatewayControllerPart3_peopleFileListV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `peopleId` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人物关联的文件列表（按日期分组） | application/json: array<object> |

### GET /gateway/peopleDescriptorList
- **摘要:** 人脸特征列表 - 管理员可调用
- **OperationId:** `GatewayControllerPart3_peopleDescriptorList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pageSize` | number | 否 |  |
| query | `pageNo` | number | 否 |  |
| query | `descriptorId` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回人脸特征列表 | application/json: object { count: number, list: array<object> } |

### GET /gateway/descriptorDistanceList
- **摘要:** 特征相似度列表 - 管理员可调用
- **OperationId:** `GatewayControllerPart3_descriptorDistanceList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pageSize` | number | 否 |  |
| query | `pageNo` | number | 否 |  |
| query | `threshold` | number | 否 |  |
| query | `descriptorId` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回特征相似度列表 | application/json: object { count: number, n: number, list: array<object> } |

### GET /gateway/cache
- **摘要:** cache value - 管理员可调用
- **OperationId:** `GatewayControllerPart3_getCacheValue`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `key` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回缓存值 | application/json: object { key: string, value: object } |

### GET /gateway/peopleInFileInfo
- **摘要:** 照片识别的人脸信息
- **OperationId:** `GatewayControllerPart3_peopleInFileInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `fileId` | string | 否 |  |
| query | `peopleId` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回照片识别的人脸信息 | application/json: array<object> |

### POST /gateway/people/merge
- **摘要:** 合并人物
- **OperationId:** `GatewayControllerPart3_mergePeople`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { peopleIds: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回合并结果 | application/json: object { msg: string } |

### POST /gateway/people/split/{id}
- **摘要:** 拆分人物
- **OperationId:** `GatewayControllerPart3_resetUserPeople`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回拆分结果 | application/json: object { msg: string } |

### POST /gateway/people/distance
- **摘要:** 计算people下descriptor的distance - 管理员可调用
- **OperationId:** `GatewayControllerPart3_calcPeopleDistance`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { peopleId: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回特征距离计算结果 | application/json: object { descriptorIds: array<number>, result: array<object>, code: string, msg: string } |

### DELETE /gateway/files
- **摘要:** 删除
- **OperationId:** `GatewayControllerPart3_deleteFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object { deleteIds: array<number>, identifiers: array<object>, code: string } |

### PATCH /gateway/files
- **摘要:** 从回收站恢复
- **OperationId:** `GatewayControllerPart3_restoreFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回恢复结果 | application/json: object { affected: number } |

### PUT /gateway/files
- **摘要:** 从回收站恢复 - patch兼容
- **OperationId:** `GatewayControllerPart3_restoreFiles_put`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回恢复结果 | application/json: object { affected: number } |

### POST /gateway/deleteFilesPermanently
- **摘要:** 从回收站删除
- **OperationId:** `GatewayControllerPart3_deleteFromTrash`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回永久删除结果 | application/json: object { ids: array<number>, code: string, msg: string } |

### GET /gateway/deleteFilesPermanentlyStatus
- **摘要:** 获取永久删除文件状态
- **说明:** 获取当前用户永久删除文件的进度状态
- **OperationId:** `GatewayControllerPart3_deleteFilesPermanentlyStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除进度信息 | application/json: object { i: number, len: number } |

### POST /gateway/deleteSimilarFiles
- **摘要:** 删除相似文件
- **OperationId:** `GatewayControllerPart3_deleteSimilarFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number>, checkItems: array<object> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object { deleteIds: array<number>, identifiers: array<object>, code: string, path: string } |

### POST /gateway/hideSimilarFiles
- **摘要:** 忽略相似照片
- **OperationId:** `GatewayControllerPart3_hideSimilarFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回忽略结果 | application/json: object { fileIds: array<number>, identifiers: array<object> } |

### POST /gateway/cancelHideSimilarFiles
- **摘要:** 取消忽略相似照片
- **OperationId:** `GatewayControllerPart3_cancelHideSimilarFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回取消忽略结果 | application/json: object { affected: number } |

### POST /gateway/similarFilesInHide
- **摘要:** 忽略相似照片列表
- **OperationId:** `GatewayControllerPart3_similarFilesInHide`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回忽略的相似照片列表 | application/json: array<object> |

### POST /gateway/user/pwd
- **摘要:** 修改自己的密码
- **OperationId:** `GatewayControllerPart3_userUpdatePwd`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { oldPwd: string, newPwd: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回修改密码结果 | application/json: object { affected: number, code: string } |

### POST /gateway/user/delete
- **摘要:** 用户申请注销账号
- **OperationId:** `GatewayControllerPart3_userUpdateDelete`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("get", "delete", "cancel"), pwd: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回注销账号状态 | application/json: object { status: boolean, code: string } |

### POST /gateway/user/cover
- **摘要:** 自定义 自动相册的封面
- **OperationId:** `GatewayControllerPart3_userUpdateCover`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, type2: string, md5: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回设置封面结果 | application/json: object { affected: number, code: string, msg: string } |

### POST /gateway/otp/generate
- **摘要:** 生成双因素认证
- **说明:** 生成双因素认证(2FA)的密钥和二维码
- **OperationId:** `GatewayControllerPart3_otpGen`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { domain: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回双因素认证密钥和二维码 | application/json: object { secret: string, uri: string, userName: string } |

### POST /gateway/otp/verify
- **摘要:** 验证双因素认证
- **说明:** 验证双因素认证令牌，验证成功后启用2FA
- **OperationId:** `GatewayControllerPart3_otpVerify`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { secret: string, token: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回验证结果 | application/json: object { n: number, code: string, msg: string } |

### POST /gateway/otp/disable
- **摘要:** 禁用双因素认证
- **说明:** 通过验证6位验证码来禁用双因素认证
- **OperationId:** `GatewayControllerPart3_otpDisable`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { token: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回禁用结果 | application/json: object { n: number, code: string, msg: string } |

### GET /gateway/lang
- **摘要:** 获取系统语言
- **OperationId:** `GatewayControllerPart4_getSysLang`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回系统语言信息 | application/json: object { arch: string, platform: string, IS_ELECTRON: boolean, value: string } |

### GET /gateway/mapCenter
- **摘要:** 获取mapbox 的 accessToken
- **OperationId:** `GatewayControllerPart4_getMapCenter`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回地图中心坐标 | application/json: object { center: array<number> } |

### GET /gateway/mapboxToken
- **摘要:** 获取mapbox 的 accessToken
- **OperationId:** `GatewayControllerPart4_getMapboxToken`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回地图配置信息 | application/json: object { amapWebKey: string, amapWebCode: string, lang: string, type: string, token: string, maptilerToken: string, center: array<number> } |

### GET /gateway/maptilerToken
- **摘要:** 获取maptiler 的 accessToken
- **OperationId:** `GatewayControllerPart4_getMaptilerToken`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回Maptiler配置信息 | application/json: object { amapWebKey: string, amapWebCode: string, lang: string, type: string, token: string, center: array<number> } |

### GET /gateway/mapType
- **摘要:** 获取地图的类型
- **OperationId:** `GatewayControllerPart4_getMapType`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回地图类型 | application/json: object { type: string } |

### GET /gateway/staticmap/amap/{location}
- **摘要:** 获取高德静态地图url
- **OperationId:** `GatewayControllerPart4_staticMapAmap`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `location` | string | 是 |  |
| query | `type` | string | 否 | type 传 app， 或者留空 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回高德静态地图URL | application/json: object { url: string } |

### GET /gateway/amap/test/{key}/{secret}
- **摘要:** 测试高德开放平台api key 私钥是否有效
- **OperationId:** `GatewayControllerPart4_testAmapApiKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 |  |
| path | `secret` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回测试结果 | application/json: object { success: boolean, msg: string } |

### GET /gateway/qqmap/test/{key}/{secret}
- **摘要:** 测试腾讯地图api key 私钥是否有效
- **OperationId:** `GatewayControllerPart4_testQQmapApiKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 |  |
| path | `secret` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回测试结果 | application/json: object { success: boolean, msg: string } |

### GET /gateway/tianmap/test/{key}
- **摘要:** 测试天地图api key是否有效
- **OperationId:** `GatewayControllerPart4_testTianDiTuApiKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回测试结果 | application/json: object { success: boolean, msg: string } |

### GET /gateway/mapbox/test/{token}
- **摘要:** 测试 mapbox api key 是否有效
- **OperationId:** `GatewayControllerPart4_testMapboxApiToken`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `token` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回测试结果 | application/json: object { success: boolean, msg: string } |

### GET /gateway/maptiler/test/{token}
- **摘要:** 测试 maptilerapi key 是否有效
- **OperationId:** `GatewayControllerPart4_testMaptilerApiToken`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `token` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回测试结果 | application/json: object { success: boolean, msg: string } |

### GET /gateway/allFilesForMap
- **摘要:** 地图上的照片
- **说明:** 获取用于地图显示的照片坐标列表（转换为高德坐标系）
- **OperationId:** `GatewayControllerPart4_getAllFilesForMap`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 否 | 图库IDs（可选） |
| query | `albumId` | number | 否 | 相册ID（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回地图上的照片坐标列表 | application/json: array<object> |

### GET /gateway/allFilesForMapDirect
- **摘要:** 地图上的照片-原始坐标
- **说明:** 获取用于地图显示的照片原始GPS坐标列表（不转换坐标系）
- **OperationId:** `GatewayControllerPart4_getFilesForMapDirect`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `galleryIds` | string | 否 | 图库IDs（可选） |
| query | `albumId` | number | 否 | 相册ID（可选） |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回地图上的照片原始坐标列表 | application/json: array<object> |

### POST /gateway/areaFilesMD5
- **摘要:** 根据文件ID列表获取文件信息
- **说明:** 用于地图模式查看图片列表和照片备份检查
- **OperationId:** `GatewayControllerPart4_getFileMD5List`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件MD5列表 | application/json: array<object> |

### POST /gateway/fileInIds
- **摘要:** 根据文件ID列表获取文件详情
- **说明:** 地图模式中查看地点的图片详情
- **OperationId:** `GatewayControllerPart4_getFileInIds`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详情列表（按日期分组） | application/json: array<object> |

### POST /gateway/enableFileBackup
- **摘要:** 启用文件备份功能
- **OperationId:** `GatewayControllerPart4_enableFileBackup`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回备份文件夹信息 | application/json: object { id: number, path: string } |

### POST /gateway/changeAppUploadStatus
- **摘要:** 通知服务器是否在备份文件
- **OperationId:** `GatewayControllerPart4_changeAppUploadStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { status: boolean, distId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { success: boolean } |

### POST /gateway/checkFileId
- **摘要:** 判断文件是否存在
- **OperationId:** `GatewayControllerPart4_checkFileId`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件是否存在 | application/json: object { exist: boolean } |

### POST /gateway/resetFileStatus
- **摘要:** 请求重置异常状态文件
- **OperationId:** `GatewayControllerPart4_fixFileStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回重置结果 | application/json: object { success: boolean, code: string, msg: string } |

### GET /gateway/backupDist/root
- **摘要:** 备份目的地-根目录
- **说明:** 获取用户可用的备份目的地根目录列表
- **OperationId:** `GatewayControllerPart4_backupDistRoot`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回备份目的地根目录列表 | application/json: array<object> |

### GET /gateway/backupDist/sub
- **摘要:** 备份目的地-子目录
- **说明:** 获取指定文件夹下的子目录列表
- **OperationId:** `GatewayControllerPart4_backupDistSubDir`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pid` | number | 是 | 父文件夹ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回子目录列表 | application/json: array<object> |

### GET /gateway/backupDist/refresh
- **摘要:** 备份目的地-刷新
- **说明:** 刷新指定文件夹，扫描新增的子文件夹
- **OperationId:** `GatewayControllerPart4_backupDistRefreshDir`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `pid` | number | 是 | 文件夹ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回刷新结果 | application/json: object { n: number, msg: string } |

### POST /gateway/backupDist/verify
- **摘要:** 备份目的地-验证
- **说明:** 验证文件夹ID及路径是否正确
- **OperationId:** `GatewayControllerPart4_backupDistVerify`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { pathList: object } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回验证结果 | application/json: object { n: number } |

### POST /gateway/checkPathForUpload
- **摘要:** 上传文件前，检查文件在服务端是否存在
- **OperationId:** `GatewayControllerPart4_checkPathForUpload`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ctime: number, deviceName: string, dist_id: number, duplicate: enum(0, 1), fileName: string, name_type: enum("", "time"), md5: string, size: number, ... } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件检查结果 | application/json: object { id: number, msg: string, abort: boolean, code: string } |

### POST /gateway/upload
- **摘要:** 上传文件 - multipart方式
- **OperationId:** `GatewayControllerPart4_uploadFile`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回上传结果 | application/json: object { id: number, msg: string, abort: boolean } |

### POST /gateway/uploadForShare
- **摘要:** 上传文件 - multipart方式 - 网页分享链接
- **OperationId:** `GatewayControllerPart4_uploadFileForShare`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回上传结果 | application/json: object { id: number, msg: string, abort: boolean } |

### POST /gateway/uploadV2
- **摘要:** 上传文件 - Binary方式
- **OperationId:** `GatewayControllerPart4_uploadFileV2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回上传结果 | application/json: object { id: number, msg: string } |

### POST /gateway/uploadChunk/check
- **摘要:** 上传文件 - 分块上传前检查
- **OperationId:** `GatewayControllerPart4_uploadChunkCheck`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileName: string, deviceName: string, ctime: number, dist_id: number, source_folder_path: enum("__NO_SUB_FOLDER__", "", "/Pictures/WeiXin"), duplicate: enum(0, 1), name_type: enum("", "time"), MD5: string, ... } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分块上传检查结果 | application/json: object { id: number, chunkIndex: number, msg: string, abort: boolean, code: string } |

### POST /gateway/uploadChunk/checkInShare
- **摘要:** 分块上传-检查(分享链接)
- **说明:** 在分享链接中分块上传前检查文件是否已存在
- **OperationId:** `GatewayControllerPart4_uploadChunkCheckInShare`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分块上传检查结果 | application/json: object { id: number, fileNameForSave: string, fileExist: boolean, existParts: array<string>, galleryId: number, msg: string, abort: boolean } |

### POST /gateway/uploadChunk/upload
- **摘要:** 分块上传 - multipart
- **OperationId:** `GatewayControllerPart4_uploadChunkUpload`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分块上传结果 | application/json: object { n: number, msg: string } |

### POST /gateway/uploadChunk/merge
- **摘要:** 分块上传 - 完成后触发合并文件
- **OperationId:** `GatewayControllerPart4_uploadChunkMerge`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回合并结果 | application/json: object { id: number, msg: string } |

### POST /gateway/uploadChunk/mergeStatus
- **摘要:** 分块上传 - 获取合并进度状态
- **说明:** 获取当前文件分块上传合并的进度状态
- **OperationId:** `GatewayControllerPart4_uploadChunkMergeStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回合并进度信息 | application/json: object { type: string, fileName: string, step: string, fileSize: number } |

### POST /gateway/uploadChunk/mergeInShare
- **摘要:** 分块上传 - 完成后触发合并文件 - 分享链接中
- **OperationId:** `GatewayControllerPart4_uploadChunkMergeInShare`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gateway/uploadChunk/mergeStatusForShare
- **摘要:** 分块上传 - 获取合并进度状态 - 分享链接中使用
- **说明:** 获取当前文件分块上传合并的进度状态
- **OperationId:** `GatewayControllerPart4_uploadChunkMergeStatusForShare`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回合并进度信息 | application/json: object { type: string, fileName: string, step: string, fileSize: number } |

### POST /gateway/uploadChunk/uploadBin
- **摘要:** 分块上传 - 上传文件-binary content 上传方式
- **OperationId:** `GatewayControllerPart4_uploadChunkBin`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 |  |  |

### POST /gateway/uploadChunk/uploadWeb
- **摘要:** 分块上传-网页端
- **说明:** 网页端分块上传文件
- **OperationId:** `GatewayControllerPart4_uploadChunkWeb`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分块上传结果 | application/json: object { n: number, msg: string } |

### POST /gateway/uploadChunk/uploadWebInShare
- **摘要:** 分块上传-网页端(分享链接)
- **说明:** 在分享链接中网页端分块上传文件
- **OperationId:** `GatewayControllerPart4_uploadChunkWebInShare`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分块上传结果 | application/json: object { n: number, msg: string, abort: boolean } |

### POST /gateway/echo
- **摘要:** 测试回显
- **说明:** 用于测试的回显接口
- **OperationId:** `GatewayControllerPart4_echo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { key: string, value: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回测试结果 | application/json: object { n: number } |

### GET /gateway/licenseInfo
- **摘要:** 订阅信息 - 管理员可调用
- **OperationId:** `GatewayControllerPart4_licenseInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回订阅信息 | application/json: object { startTime: string, endTime: string, clientId: string, orderId: string, offlineMode: boolean, liveAuthMsg: string } |

### GET /gateway/trail
- **摘要:** 开始试用 - 管理员可调用
- **OperationId:** `GatewayControllerPart4_startTrail`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回试用结果 | application/json: object { msg: string, n: number } |

### POST /gateway/bindLicense
- **摘要:** 使用激活码-添加订阅 - 管理员可调用
- **OperationId:** `GatewayControllerPart4_bindLicense`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { license: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回激活结果 | application/json: object { msg: string, n: number } |

### POST /gateway/verifyAuthOnline
- **摘要:** 触发联网验证 - 管理员可调用
- **OperationId:** `GatewayControllerPart4_forceVerifyCpStatusLive`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回验证结果 | application/json: object { n: number, msg: string } |

### POST /gateway/coordinate/convert
- **摘要:** gps坐标转为autonavi
- **OperationId:** `GatewayControllerPart4_coordinateConvert`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { latitude: number, longitude: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回转换后的坐标 | application/json: object { type: string, latitude: number, longitude: number } |

### POST /gateway/coordinate/parse
- **摘要:** 自动处理从 腾讯、高德地图坐标拾取器中粘贴的值
- **OperationId:** `GatewayControllerPart4_coordinateAutoParse`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { latitude: number, longitude: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理后的坐标 | application/json: object { latitude: number, longitude: number } |

### GET /gateway/folders/root
- **摘要:** 文件夹视图-顶级
- **OperationId:** `GatewayControllerPart5_folderTopList`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回顶级文件夹列表 | application/json: object { path: string, folderList: array<object>, fileList: array<any> } |

### GET /gateway/folderInfo/{id}
- **摘要:** 文件夹-信息
- **OperationId:** `GatewayControllerPart5_folderInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件夹信息 | application/json: object { id: number, name: string, path: string, subFolders: array<number>, cover: string, s_cover: string, subFileNum: number, isTop: boolean } |

### GET /gateway/folderSubFile/{id}
- **摘要:** 文件夹-获取当前及下级文件夹文件 id、MD5
- **OperationId:** `GatewayControllerPart5_folderSubFile`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |
| query | `count` | number | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件列表 | application/json: array<object> |

### POST /gateway/folderAutoCover/{id}
- **摘要:** 文件夹-自动设置空封面的文件夹，显示下级文件夹的文件
- **OperationId:** `GatewayControllerPart5_folderAutoCover`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理结果 | application/json: object { n: number } |

### GET /gateway/folders/{id}
- **摘要:** 文件夹视图-文件夹详情
- **OperationId:** `GatewayControllerPart5_folderViewDetail`
- **认证:** api-key 或 bearer
- **Deprecated:** 是

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件夹详情 | application/json: object { path: string, folderList: array<object>, fileList: array<any> } |

### GET /gateway/foldersV2/{id}
- **摘要:** 文件夹视图-文件夹详情
- **OperationId:** `GatewayControllerPart5_folderViewDetailV2`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件夹详情 | application/json: object { path: string, folderList: array<object>, fileList: array<any>, trashNum: number } |

### GET /gateway/folderFiles/{id}
- **摘要:** 文件夹视图-文件夹详情-文件列表
- **OperationId:** `GatewayControllerPart5_folderFileInTimeline`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 |  |
| query | `withSub` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表 | application/json: object { result: array<object>, totalCount: number, duplicateFiles: object } |

### GET /gateway/folderBreadcrumbs/{id}
- **摘要:** 文件夹地址的面包屑
- **OperationId:** `GatewayControllerPart5_folderBreadcrumbs`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回面包屑导航列表 | application/json: array<object> |

### POST /gateway/folders/create
- **摘要:** 文件夹视图-新建文件夹
- **OperationId:** `GatewayControllerPart5_folderCreate`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回创建结果 | application/json: object { n: number, msg: string, id: number } |

### POST /gateway/folderPathEdit
- **摘要:** 文件夹视图-重命名、移动、删除
- **OperationId:** `GatewayControllerPart5_folderEdit`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, folderId: number, distId: number, name: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回操作结果 | application/json: object { n: number, code: string, path: string } |

### POST /gateway/filePathEdit
- **摘要:** 文件路径编辑
- **说明:** 移动或复制文件到指定目录
- **OperationId:** `GatewayControllerPart5_filePathEdit`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: enum("move", "copy"), fileIds: array<number>, distId: number, overwrite: enum(0, 1, 2) } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回操作结果 | application/json: object { n: number, code: string, duplicateFiles: array<string> } |

### POST /gateway/folder_files_move/preview
- **摘要:** 整理文件夹下的文件 - 预览移动路径
- **OperationId:** `GatewayControllerPart5_folder_files_move_preview`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folderId: number, folderNameType: string, fileNameType: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回预览结果 | application/json: object { msg: string, needMoveFiles: array<any>, moveToPath: object } |

### POST /gateway/folder_files_move/move
- **摘要:** 整理文件夹下的文件 - 移动文件
- **OperationId:** `GatewayControllerPart5_folder_files_move_run`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folderId: number, folderNameType: string, fileNameType: string, deleteEmptyFolder: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回移动任务结果 | application/json: object { n: number, taskId: string, msg: string } |

### POST /gateway/folders/delete_empty
- **摘要:** 删除文件夹下面的 空文件夹
- **OperationId:** `GatewayControllerPart5_folder_delete_empty`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folderId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object { deletedFolders: array<string>, code: string } |

### POST /gateway/folder_files_move/status
- **摘要:** 整理文件夹 获取处理进度
- **OperationId:** `GatewayControllerPart5_folder_files_move_status`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folderId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回处理进度 | application/json: object { type: string, progress: number, total: number, done: boolean } |

### PATCH /gateway/setFolderCover/{id}
- **摘要:** 修改文件夹封面
- **OperationId:** `GatewayControllerPart5_setFolderCover`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { s_cover: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回修改结果 | application/json: object { n: number } |

### PUT /gateway/setFolderCover/{id}
- **摘要:** 修改文件夹封面 - 兼容PATCH
- **OperationId:** `GatewayControllerPart5_setFolderCover_put`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回修改结果 | application/json: object { n: number } |

### POST /gateway/scanAfterUpload
- **摘要:** 更新刚上传的文件的状态
- **OperationId:** `GatewayControllerPart5_scanAfterUpload`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { n: number, msg: string } |

### POST /gateway/scanAfterUploadInShare
- **摘要:** 更新刚上传的文件的状态 - 分享的链接中
- **OperationId:** `GatewayControllerPart5_scanAfterUploadInShare`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { n: number, msg: string } |

### POST /gateway/folderDebugInfo
- **摘要:** 获取文件夹的调试信息 - 管理员可调用
- **OperationId:** `GatewayControllerPart5_folderDebugInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folderId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件夹调试信息 | application/json: object { folder: object, items: array<any>, diskData: object, msg: string } |

### POST /gateway/updateFileDate
- **摘要:** 更新文件的拍摄日期
- **OperationId:** `GatewayControllerPart5_updateFileDate`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { n: number, code: string } |

### POST /gateway/updateFileName
- **摘要:** 修改文件的名称
- **OperationId:** `GatewayControllerPart5_updateFileName`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: number, fileName: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回修改结果 | application/json: object { n: number, code: string, path: string } |

### POST /gateway/editFileExtra
- **摘要:** 编辑文件额外信息
- **说明:** 编辑文件的描述或评分等额外信息
- **OperationId:** `GatewayControllerPart5_editFileDesc`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: number, type: enum("desc", "rating"), value: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回编辑结果 | application/json: object { n: number } |

### POST /gateway/editFileGps
- **摘要:** 编辑文件GPS信息
- **说明:** 修改文件的GPS坐标信息
- **OperationId:** `GatewayControllerPart5_editFileGPS`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: number, latitude: number, longitude: number, type: enum("GCJ-02", "WGS-84", "BD-09") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回编辑结果 | application/json: object { n: number } |

### POST /gateway/resetFileGps
- **摘要:** 重置文件GPS信息
- **说明:** 清除文件的GPS坐标，并重新从EXIF中读取
- **OperationId:** `GatewayControllerPart5_resetFileGps`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回重置结果 | application/json: object { n: number } |

### POST /gateway/editFileRotate
- **摘要:** 旋转文件
- **说明:** 旋转照片文件，支持指定旋转角度
- **OperationId:** `GatewayControllerPart5_editFileRotate`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileId: number, deg: enum(90, 180, 270) } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回旋转结果 | application/json: object { n: number, MD5: string, code: string } |

### POST /gateway/searchTips
- **摘要:** 搜索提示
- **说明:** 搜索时提供人物和标签的提示建议
- **OperationId:** `GatewayControllerPart5_searchTips`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { key: string, type: enum("people", "tag") } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回搜索提示列表 | application/json: array<object> |

### POST /gateway/search
- **摘要:** 搜索
- **OperationId:** `GatewayControllerPart5_searchFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { key: string, model: string, lens: string, rating: number, tokenAtStart: number, tokenAtEnd: number, mtimeStart: number, mtimeEnd: number, ... } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回搜索结果 | application/json: object { result: array<object>, totalCount: number, list: array<any> } |

### POST /gateway/searchCLIP
- **摘要:** 搜索-CLIP
- **OperationId:** `GatewayControllerPart5_getCLIPTextMatchedId`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { key: string, imgId: number, count: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回CLIP搜索结果 | application/json: object { list: array<object>, totalCount: number } |

### POST /gateway/searchV2
- **摘要:** 搜索-v2
- **OperationId:** `GatewayControllerPart5_searchFilesV2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { galleryIds: array<number>, searchType: string, searchKey: string, placeL1: string, placeL2: string, placeL3: string, tagUseOr: boolean, tags: array<number>, ... } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回搜索结果 | application/json: object { result: array<object> } |

### POST /gateway/searchResultTipsBox
- **摘要:** 搜索结果提示框
- **说明:** 搜索结果中显示路径、备注、文件名、OCR等提示信息
- **OperationId:** `GatewayControllerPart5_searchResultTipsBox`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { id: number, type: string, searchKey: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回提示信息 | application/json: object { text: string, highlights: array<any> } |

### POST /gateway/searchCLIPV2
- **摘要:** 搜索-CLIP
- **OperationId:** `GatewayControllerPart5_searchCLIPV2`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { galleryIds: array<number>, searchKey: string, count: number, imgId: number, placeL1: string, placeL2: string, placeL3: string, tags: array<number>, ... } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回CLIP搜索结果 | application/json: object { list: array<object>, totalCount: number } |

### POST /gateway/memory
- **摘要:** 那年今日
- **OperationId:** `GatewayControllerPart5_getMemoryList`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { month: number, date: number, galleryId: string, galleryIds: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回那年今日的照片列表 | application/json: array<object> |

### POST /gateway/memoryWeekFileList
- **摘要:** 往年照片 - 一周 - 文件列表
- **OperationId:** `GatewayControllerPart5_memoryWeekFileList`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { year: number, month: number, date: number, range: number, galleryIds: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回往年同期照片 | application/json: object { result: array<object> } |

### POST /gateway/CLIP_status
- **摘要:** 是否可以用使用CLIP搜索
- **OperationId:** `GatewayControllerPart5_searchCLIPStatus`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回CLIP状态 | application/json: object { active: boolean } |

### POST /gateway/nongLi
- **摘要:** 获取阳历日期的农历日期
- **OperationId:** `GatewayControllerPart5_getNongLiInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { time: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回农历日期信息 | application/json: object { year: number, month: number, day: number, monthCn: string, dayCn: string } |

### POST /gateway/livePhotoMovCheck
- **摘要:** 检查livePhoto视频部分是否正确
- **OperationId:** `GatewayControllerPart5_livePhotoMovCheck`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { fileMd5List: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回动态照片视频检查结果 | application/json: object { videoIdx: object, videoList: array<object> } |

### POST /gateway/uploadForLivePhotoMov/{photoMD5}/{videoMD5}
- **摘要:** 上传动态照片视频部分
- **说明:** 用于修复动态照片的视频部分，覆盖错误的MOV文件
- **OperationId:** `GatewayControllerPart5_uploadForLivePhotoMov`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `photoMD5` | string | 是 |  |
| path | `videoMD5` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回上传结果 | application/json: object { n: number, msg: string, abort: boolean } |

### GET /gateway/{type}/{md5}
- **摘要:** 显示文件的缩略图
- **说明:** 需要调用 /auth/auth_code 获取auth_code之后，带上auth_code参数才能访问
- **OperationId:** `GatewayControllerPartEnd_renderThumb`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `type` | string | 是 | 缩略图类型：h220-PC缩略图, s260-app缩略图, preview-视频前5s的动图, poster-视频封面, proxy-预览图, portrait-人物封面, live_preview-动态照片预览 |
| path | `md5` | string | 是 | 文件MD5值 |
| query | `albumId` | string | 否 | 相册ID，如果在相册内需要 |
| query | `id` | number | 否 | 文件ID ，可选 |
| query | `auth_code` | string | 是 | 授权码 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回缩略图文件流 |  |
| 404 | 文件未找到 |  |

## api-share 分享管理

### POST /api-share
- **摘要:** 创建分享
- **OperationId:** `ShareController_create`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateShareDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建分享成功 | application/json: object { id: number } |

### GET /api-share
- **摘要:** 我的分享列表
- **OperationId:** `ShareController_findAll`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回我分享的相册列表 | application/json: array<object> |

### GET /api-share/shareToMe
- **摘要:** 分享给我的列表
- **OperationId:** `ShareController_findAllShareToMe`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分享给我的相册列表 | application/json: array<object> |

### GET /api-share/users
- **摘要:** 查询可分享的用户列表
- **OperationId:** `ShareController_findUsers`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回可分享的用户列表 | application/json: array<object> |

### GET /api-share/link/{id}
- **摘要:** 查询 相册的分享链接 key
- **OperationId:** `ShareController_createShare`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册的分享链接key | application/json: object { key: string, id: number } |

### GET /api-share/visit/album/{key}
- **摘要:** 根据链接分享的key获取相册的信息
- **OperationId:** `ShareController_getShareInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册信息 | application/json: object { id: number, name: string, desc: string, cover: string, count: number, expires_in: number, auth_code: string, showUpload: boolean, ... } |

### GET /api-share/album/{id}
- **摘要:** 开启分享相册时，查询这个相册是否有分享信息
- **OperationId:** `ShareController_findOneByAlbumId`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册的分享信息 | application/json: object { id: number, albumId: number, userId: number, link: boolean, key: string, linkPwd: string, vUserIds: array<number>, cUserIds: array<number> } |

### GET /api-share/albumInfo/{albumId}
- **摘要:** 打开他人分享的相册时，根据albumId，获取相册的信息
- **OperationId:** `ShareController_findAlbumInfoByAlbumId`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `albumId` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册详细信息 | application/json: object { id: number, name: string, desc: string, cover: string, count: number, mtime: string:date-time, create_time: string:date-time, userId: number, ... } |

### GET /api-share/albumFiles/{albumId}
- **摘要:** 打开他人分享的相册时，根据albumId，获取相册的文件列表
- **OperationId:** `ShareController_findAlbumFilesByAlbumId`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `albumId` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表 | application/json: array<object> |

### GET /api-share/albumFilesFlat/{albumId}
- **摘要:** 打开他人分享的相册时，根据albumId，获取相册的文件列表 - 平铺列表
- **OperationId:** `ShareController_findAlbumFilesFlatByAlbumId`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `albumId` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回平铺的文件列表 | application/json: array<object> |

### POST /api-share/dayFileMoreForUser
- **摘要:** 单天剩余文件 - 已登录用户
- **OperationId:** `ShareController_dayFileMoreForUser`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: number, ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回指定ID的文件列表 | application/json: array<object> |

### POST /api-share/dayFileMore
- **摘要:** 单天剩余文件 - 链接
- **OperationId:** `ShareController_findDayFileMore`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { key: string, pwd: string, ids: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回指定ID的文件列表 | application/json: array<object> |

### GET /api-share/album/link/{id}
- **摘要:** 获取相册的自动更新配置
- **OperationId:** `ShareController_findAutoLinkList`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回相册的自动更新配置列表 | application/json: array<object> |

### POST /api-share/album/link/{id}
- **摘要:** 添加 相册 自动配置
- **OperationId:** `ShareController_addAutoLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { type: string, value: string, exclude: boolean } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 添加结果 | application/json: object { n: number, message: string, statusCode: number } |

### DELETE /api-share/album/link/{id}
- **摘要:** 删除 分享的相册 自动配置
- **OperationId:** `ShareController_delAutoLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 相册ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { linkId: number } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 删除结果 | application/json: object { n: number } |

### GET /api-share/visit/albumFiles/{key}
- **摘要:** 查询相册分享链接的文件列表 - 网页使用
- **OperationId:** `ShareController_findAlbumFilesByKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表或错误信息 | application/json: array<object> \| object |

### GET /api-share/visit/albumFilesFlat/{key}
- **摘要:** 查询相册分享链接的文件列表
- **OperationId:** `ShareController_findAlbumFilesByKeyFlat`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回平铺的文件列表或错误信息 | application/json: array<object> \| object |

### GET /api-share/fileInfo/{albumId}/{fileId}
- **摘要:** 显示文件的详细信息 - 检查共享权限
- **OperationId:** `ShareController_getFileDetail`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `albumId` | number | 是 | 相册ID |
| path | `fileId` | number | 是 | 文件ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详细信息 | application/json: object { id: number, fileName: string, fileType: string, fileSize: number, width: number, height: number, tokenAt: string, gps: string, ... } |

### GET /api-share/fileInfoByKey/{key}/{fileId}
- **摘要:** 查询相册分享链接的文件详情
- **OperationId:** `ShareController_getFileDetailByKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| path | `fileId` | number | 是 | 文件ID |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详细信息或错误信息 | application/json: object \| object |

### GET /api-share/amap/{key}/{location}
- **摘要:** 获取高德静态地图url
- **OperationId:** `ShareController_staticMapAmap`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| path | `location` | string | 是 | GPS坐标(经度,纬度) |
| path | `type` | string | 是 | app\|留空 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回高德静态地图URL | application/json: object { url: string } |

### POST /api-share/filesInfo
- **摘要:** 下载前查询文件信息 - 分享的链接
- **OperationId:** `ShareController_getFilesInfo`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { ids: array<number>, key: string, pwd: string, type: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件信息列表 | application/json: object { list: array<object>, msg: string } |

### POST /api-share/addFileToAlbum
- **摘要:** 添加文件到分享相册
- **OperationId:** `ShareController_addFileToAlbum`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: number, files: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回添加结果 | application/json: object { n: number } |

### POST /api-share/removeFileFromAlbum
- **摘要:** 从分享相册移除文件
- **OperationId:** `ShareController_removeFileFromAlbum`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { albumId: number, files: array<number> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回移除结果 | application/json: object { n: number } |

### GET /api-share/{id}
- **摘要:** 查询分享信息
- **OperationId:** `ShareController_findOne`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分享信息 | application/json: object { id: number, albumId: number, userId: number, link: boolean, key: string, vUserIds: array<number>, cUserIds: array<number> } |

### PATCH /api-share/{id}
- **摘要:** 更新分享信息
- **OperationId:** `ShareController_update`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateShareDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { id: number } |

### PUT /api-share/{id}
- **摘要:** 更新分享信息(PUT)
- **OperationId:** `ShareController_update_put`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateShareDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { id: number } |

### DELETE /api-share/{id}
- **摘要:** 删除分享
- **OperationId:** `ShareController_remove`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object { n: number } |

### POST /api-share/createFilesLink
- **摘要:** 创建分享 - 文件链接分享
- **OperationId:** `ShareController_createFileLink`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateShareFilesDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 返回创建的文件分享信息 | application/json: object { id: number, key: string, cover: string } |

### POST /api-share/getFilesLink/{id}
- **摘要:** 查询分享 - 文件链接分享
- **OperationId:** `ShareController_getShareFileLinkInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件分享信息 | application/json: object { id: number, key: string, cover: string, count: number, files: array<number>, showExif: boolean, showDownload: boolean } |

### POST /api-share/updateFilesLink/{id}
- **摘要:** 修改分享 - 文件链接分享
- **OperationId:** `ShareController_updateFileLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateShareFilesDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回更新结果 | application/json: object { id: number } |

### POST /api-share/delFilesLink/{id}
- **摘要:** 删除分享 - 文件链接分享
- **OperationId:** `ShareController_delFileLink`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回删除结果 | application/json: object { n: number } |

### POST /api-share/filesLink/count
- **摘要:** 我的分享列表 - 链接分享的文件 - 数量
- **OperationId:** `ShareController_countAllSingleFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分享文件数量和封面列表 | application/json: object { count: number, cover: array<string> } |

### POST /api-share/filesLink/list
- **摘要:** 我的分享列表 - 链接分享的文件 - 列表
- **OperationId:** `ShareController_findAllSingleFiles`
- **认证:** api-key 或 bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分享文件列表 | application/json: array<object> |

### POST /api-share/filesLink/list/{id}
- **摘要:** 我的分享列表 - 链接分享的文件 - 文件列表
- **OperationId:** `ShareController_getFileLinkFiles`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 分享ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回分享文件列表 | application/json: array<object> |

### POST /api-share/visit/filesLink/{key}
- **摘要:** 根据链接分享的key获取file的信息
- **OperationId:** `ShareController_getFileShareInfo`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件分享信息或错误信息 | application/json: object { expires_in: number, auth_code: string, showExif: boolean, showDownload: boolean, file: object, msg: string } |

### POST /api-share/visit/filesLinkFiles/{key}
- **摘要:** 查询链接分享链接的文件列表
- **OperationId:** `ShareController_findShareFileListByKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回按日期分组的文件列表或错误信息 | application/json: array<object> \| object |

### POST /api-share/linkFileInfoByKey/{key}/{fileId}
- **摘要:** 查询文件分享链接的文件详情
- **OperationId:** `ShareController_getLinkFileDetailByKey`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| path | `fileId` | number | 是 | 文件ID |
| query | `pwd` | string | 否 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回文件详细信息或错误信息 | application/json: object \| object |

### POST /api-share/linkFileInfoAmap/{key}/{location}
- **摘要:** 获取高德静态地图url - 文件分享链接
- **OperationId:** `ShareController_linkFileInfoAmap`
- **认证:** api-key 或 bearer

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 分享链接key |
| path | `location` | string | 是 | GPS坐标(经度,纬度) |
| path | `type` | string | 是 | app \| 留空 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回高德静态地图URL | application/json: object { url: string } |

## install-初始化

### GET /install/status
- **摘要:** 获取安装状态
- **OperationId:** `InstallController_findStatus`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回安装状态 | application/json: object { isInstalled: boolean } |

### POST /install/createAdminAccount
- **摘要:** 创建管理员用户
- **OperationId:** `InstallController_create`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateUserDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: object { success: boolean } |

### GET /install/rootDirs
- **摘要:** 获取根目录列表
- **OperationId:** `InstallController_findRootDirs`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回根目录列表 | application/json: array<object> |

### GET /install/subDirs
- **摘要:** 获取子目录列表
- **OperationId:** `InstallController_findSubDirs`
- **认证:** 未声明

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `path` | string | 是 |  |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回子目录列表 | application/json: array<object> |

### POST /install/createFolders
- **摘要:** 批量创建文件夹
- **OperationId:** `InstallController_createFolders`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { folders: array<string> } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: array<object> |

### POST /install/gallery
- **摘要:** 创建图库
- **OperationId:** `InstallController_createGallery`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateGalleryDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 创建成功 | application/json: object { id: number, name: string } |

### GET /install/gallery
- **摘要:** 获取图库列表
- **OperationId:** `InstallController_getGalleryList`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回图库列表 | application/json: array<object> |

### DELETE /install/gallery/{id}
- **摘要:** 删除图库
- **OperationId:** `InstallController_deleteGallery`
- **认证:** 未声明

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 图库ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 删除成功 | application/json: object { raw: object, affected: number } |

### PATCH /install/gallery/{id}
- **摘要:** 更新图库
- **OperationId:** `InstallController_updateGallery`
- **认证:** 未声明

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | number | 是 | 图库ID |

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | UpdateGalleryDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 更新成功 | application/json: object { id: number, name: string } |

### GET /install/gallery/scan/{id}
- **摘要:** 扫描图库
- **OperationId:** `InstallController_scanGallery`
- **认证:** 未声明

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 图库ID |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 扫描成功 | application/json: object { n: number } |

### PATCH /install/system-config
- **摘要:** 更新系统配置
- **OperationId:** `InstallController_updateByKey`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | CreateSystemConfigDto |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 更新成功 | application/json: object { key: string, value: string } |

### GET /install/system-config/{key}
- **摘要:** 获取系统配置
- **OperationId:** `InstallController_findByKey`
- **认证:** 未声明

**参数:**
| 位置 | 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `key` | string | 是 | 配置键名 |

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回系统配置 | application/json: object { key: string, value: string } |

### POST /install/upgrade
- **摘要:** 手动升级
- **OperationId:** `InstallController_update`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 升级成功 | application/json: object \| object |

### POST /install/autoUpgrade
- **摘要:** 自动升级
- **OperationId:** `InstallController_autoUpgrade`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
| Content-Type | 必填 | Schema | 说明 |
|--------------|------|--------|------|
| application/json | 是 | object { version: string, build: string, SHA1: string } |  |

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 升级成功 | application/json: object { success: boolean } |
| 400 | 错误的更新链接或SHA1值验证失败 |  |

### GET /install/memory
- **摘要:** 获取内存使用情况
- **OperationId:** `InstallController_memoryUsage`
- **认证:** bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回内存使用情况 | application/json: object { rss: number, heapTotal: number, heapUsed: number, external: number, arrayBuffers: number } |

### POST /install/reload
- **摘要:** 重载服务
- **OperationId:** `InstallController_reloadServer`
- **认证:** bearer

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 201 | 重载成功 | application/json: object { result: string } |

### GET /install/trail
- **摘要:** 开始试用
- **OperationId:** `InstallController_startTrail`
- **认证:** 未声明

**参数:**
- 无

**请求体:**
- 无

**响应:**
| 状态码 | 说明 | Content/Schema |
|--------|------|----------------|
| 200 | 返回试用信息 | application/json: object { trial: boolean } |

## 7. Component Schemas

### CreateUserDto
- **类型:** object
- **必填字段:** `username`, `password`, `otp_secret`, `isAdmin`, `isEnabled`

| 字段 | 类型 | 说明 |
|------|------|------|
| `username` | string |  |
| `nickname` | string | 用户昵称 |
| `avatar` | string | 头像路径 |
| `email` | string |  |
| `password` | string |  |
| `otp_secret` | string |  |
| `isAdmin` | boolean |  |
| `isEnabled` | boolean |  |
| `isSuperAdmin` | boolean |  |
| `galleries` | array<string> |  |

### UpdateUserDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `username` | string |  |
| `nickname` | string | 用户昵称 |
| `avatar` | string | 头像路径 |
| `email` | string |  |
| `password` | string |  |
| `otp_secret` | string |  |
| `isAdmin` | boolean |  |
| `isEnabled` | boolean |  |
| `isSuperAdmin` | boolean |  |
| `galleries` | array<string> |  |

### User
- **类型:** object
- **必填字段:** `id`, `uid`, `username`, `nickname`, `avatar`, `email`, `password`, `otp_secret`, `isAdmin`, `isSuperAdmin`, `isEnabled`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | number | 用户ID |
| `uid` | string | 用户uuid |
| `username` | string | 用户名 |
| `nickname` | string | 用户昵称 |
| `avatar` | string | 头像路径 |
| `email` | string | 电子邮箱地址 |
| `password` | string | 登录密码 |
| `otp_secret` | string | 2FA密钥 |
| `isAdmin` | boolean | 是否为管理员 |
| `isSuperAdmin` | boolean | 是否为超级管理员 |
| `isEnabled` | boolean | 是否可用 |

### CreateFolderDto
- **类型:** object
- **必填字段:** `name`, `path`

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string |  |
| `path` | string |  |
| `cover` | string |  |
| `s_cover` | string |  |
| `ino` | string |  |
| `mtime` | string:date-time |  |
| `birthtime` | string:date-time |  |
| `subFolders` | array<string> |  |
| `subFileNum` | number |  |
| `files` | array<string> |  |
| `isDeleted` | boolean |  |

### Folder
- **类型:** object
- **必填字段:** `id`, `name`, `path`, `ino`, `mtime`, `birthtime`, `cover`, `s_cover`, `subFileNum`, `isDeleted`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | number | 文件夹ID |
| `name` | string | 文件夹的名称 |
| `path` | string | 文件夹的路径 |
| `ino` | string | 文件夹的inode值 |
| `mtime` | string:date-time | 文件夹的修改时间 |
| `birthtime` | string:date-time | 文件夹的创建时间 |
| `cover` | string | 文件夹的封面 |
| `s_cover` | string | 指定的文件夹的封面 |
| `subFileNum` | number | 子文件数量 |
| `isDeleted` | boolean | 是否为被删除 |

### UpdateFolderDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string |  |
| `path` | string |  |
| `cover` | string |  |
| `s_cover` | string |  |
| `ino` | string |  |
| `mtime` | string:date-time |  |
| `birthtime` | string:date-time |  |
| `subFolders` | array<string> |  |
| `subFileNum` | number |  |
| `files` | array<string> |  |
| `isDeleted` | boolean |  |

### FileExtra
- **类型:** object

### FileGPS
- **类型:** object

### FileGPSInfo
- **类型:** object

### File
- **类型:** object
- **必填字段:** `id`, `fileName`, `fileType`, `filePath`, `fileSize`, `galleryIds`, `tokenAt`, `mtime`, `addAt`, `MD5`, `duration`, `width`, `height`, `orientation`, `rotation`, `m_rotate`, `status`, `proxyStatus`, `previewStatus`, `peopleDescriptorStatus`, `categoryStatus`, `ocrStatus`, `clipStatus`, `transcodeStatus`, `similarStatus`, `similar_value`, `livePhotosVideoId`, `isLivePhotosVideo`, `live_photo_UUID`, `isScreenshot`, `isScreenRecord`, `isSelfie`, `extra`, `gps`, `gpsInfo`, `folderId`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | number | 文件ID |
| `fileName` | string | 文件名称 |
| `fileType` | string | 文件类型 |
| `filePath` | string | 文件路径 |
| `fileSize` | number | 文件大小（字节） |
| `galleryIds` | array<string> | 所属图库ID列表 |
| `tokenAt` | string:date-time | 拍摄日期 |
| `mtime` | string:date-time | 最后修改日期 |
| `addAt` | string:date-time | 文件入库时间 |
| `MD5` | string | 文件MD5值 |
| `duration` | number | 视频时长（秒） |
| `width` | number | 图片/视频宽度 |
| `height` | number | 图片/视频高度 |
| `orientation` | number | EXIF方向信息 |
| `rotation` | number | 旋转角度 |
| `m_rotate` | number | 手动旋转角度 |
| `status` | number | 文件处理状态：0未处理，1处理中，2处理成功，-1出错，-10已删除 |
| `proxyStatus` | number | 代理文件状态：0未处理，1处理中，2已处理，12忽略，-1出错 |
| `previewStatus` | number | 预览状态：0未处理，1处理中，2已处理，-1出错 |
| `peopleDescriptorStatus` | number | 人脸描述符状态：0未处理，1处理中，2已处理，-1出错 |
| `categoryStatus` | number | 场景分类状态：0未处理，1处理中，2已处理，-1出错 |
| `ocrStatus` | number | OCR状态：0未处理，1处理中，2已处理，-1出错 |
| `clipStatus` | number | CLIP特征状态：0未处理，1处理中，2已处理，-1出错 |
| `transcodeStatus` | number | 转码状态：0未处理，1处理中，2已处理，12忽略，-1出错 |
| `similarStatus` | number | 相似图状态：0未处理，1处理中，2已处理，-1出错 |
| `similar_value` | string | 相似图标识值 |
| `livePhotosVideoId` | number | 关联的livePhotos 文件id |
| `isLivePhotosVideo` | boolean | 是否为Live Photo的视频文件 |
| `live_photo_UUID` | string | Live Photo的UUID |
| `isScreenshot` | boolean | 是否为截图 |
| `isScreenRecord` | boolean | 是否为屏幕录制 |
| `isSelfie` | boolean | 是否为自拍 |
| `extra` | FileExtra | 部分exif信息 |
| `gps` | FileGPS | gps信息 |
| `gpsInfo` | FileGPSInfo | gps逆地理位置信息 |
| `folderId` | number | 所属文件夹ID |

### UpdateFileDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `fileName` | string |  |
| `fileType` | string |  |
| `filePath` | string |  |
| `fileSize` | number |  |
| `tokenAt` | string:date-time |  |
| `mtime` | string:date-time |  |
| `addAt` | string:date-time |  |
| `MD5` | string |  |
| `galleryIds` | array<string> |  |
| `duration` | number |  |
| `width` | number |  |
| `height` | number |  |
| `orientation` | number |  |
| `rotation` | number |  |
| `m_rotate` | number |  |
| `isLivePhotosVideo` | boolean |  |
| `livePhotosVideoId` | number |  |
| `folderId` | number |  |
| `status` | enum(-1, 0, 1, 2) |  |
| `proxyStatus` | enum(-1, 0, 1, 2) |  |
| `previewStatus` | enum(-1, 0, 1, 2) |  |
| `peopleDescriptorStatus` | enum(-1, 0, 1, 2) |  |
| `categoryStatus` | enum(-1, 0, 1, 2) |  |
| `ocrStatus` | enum(-1, 0, 1, 2) |  |
| `clipStatus` | enum(-1, 0, 1, 2) |  |
| `transcodeStatus` | enum(-1, 0, 1, 2, 12) |  |
| `similarStatus` | enum(-1, 0, 1, 2) |  |
| `similar_value` | string |  |
| `live_photo_UUID` | string |  |
| `extra` | object |  |
| `gps` | object |  |
| `gpsInfo` | object |  |

### CreateGalleryDto
- **类型:** object
- **必填字段:** `name`, `hide`, `folders`

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string |  |
| `cover` | number |  |
| `weights` | number |  |
| `hide` | boolean |  |
| `folders` | array<string> |  |
| `userIds` | number |  |
| `func_exclude` | array<string> |  |

### UpdateGalleryDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string |  |
| `cover` | number |  |
| `weights` | number |  |
| `hide` | boolean |  |
| `folders` | array<string> |  |
| `userIds` | number |  |
| `func_exclude` | array<string> |  |

### Box
- **类型:** object
- **必填字段:** `x`, `y`, `width`, `height`

| 字段 | 类型 | 说明 |
|------|------|------|
| `x` | number |  |
| `y` | number |  |
| `width` | number |  |
| `height` | number |  |

### CreatePeopleDescriptorDto
- **类型:** object
- **必填字段:** `box`, `files`, `vec_low`, `vec_high`, `pass`

| 字段 | 类型 | 说明 |
|------|------|------|
| `box` | Box |  |
| `files` | array<string> |  |
| `vec_low` | array<string> |  |
| `vec_high` | array<string> |  |
| `pass` | boolean |  |

### UpdatePeopleDescriptorDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `box` | Box |  |
| `files` | array<string> |  |
| `vec_low` | array<string> |  |
| `vec_high` | array<string> |  |
| `pass` | boolean |  |

### CreatePeopleDto
- **类型:** object
- **必填字段:** `id`, `name`, `cover`, `count`, `isHide`, `userId`, `ver`, `baseIds`, `files`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | number |  |
| `name` | string |  |
| `cover` | number |  |
| `count` | number |  |
| `isHide` | boolean |  |
| `userId` | number |  |
| `ver` | number |  |
| `baseIds` | array<string> |  |
| `files` | object |  |

### UpdatePeopleDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | number |  |
| `name` | string |  |
| `cover` | number |  |
| `count` | number |  |
| `isHide` | boolean |  |
| `userId` | number |  |
| `ver` | number |  |
| `baseIds` | array<string> |  |
| `files` | object |  |

### CreateSystemConfigDto
- **类型:** object
- **必填字段:** `key`, `value`

| 字段 | 类型 | 说明 |
|------|------|------|
| `key` | string |  |
| `value` | string |  |
| `hide` | boolean |  |

### CreateFileDeleteLogDto
- **类型:** object
- **必填字段:** `path`, `userId`

| 字段 | 类型 | 说明 |
|------|------|------|
| `path` | string | 文件原始路径 |
| `userId` | number | 删除文件的用户ID |
| `type` | number | 删除类型：1-用户删除, 2-重复文件清理, 3-删除文件夹 |

### CreateAlbumDto
- **类型:** object
- **必填字段:** `name`

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string |  |
| `desc` | string |  |
| `weights` | number |  |
| `count` | number |  |
| `mtime` | string:date-time |  |
| `create_time` | string:date-time |  |
| `cover` | string |  |
| `startTime` | string:date-time |  |
| `endTime` | string:date-time |  |
| `files` | array<string> |  |
| `ignore_files` | array<string> |  |
| `auto_files` | array<string> |  |
| `sort_type` | string |  |
| `deleted` | boolean |  |
| `hide` | boolean |  |
| `theme` | string |  |
| `extra_time1` | string:date-time |  |

### UpdateAlbumDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string |  |
| `desc` | string |  |
| `weights` | number |  |
| `count` | number |  |
| `mtime` | string:date-time |  |
| `create_time` | string:date-time |  |
| `cover` | string |  |
| `startTime` | string:date-time |  |
| `endTime` | string:date-time |  |
| `files` | array<string> |  |
| `ignore_files` | array<string> |  |
| `auto_files` | array<string> |  |
| `sort_type` | string |  |
| `deleted` | boolean |  |
| `hide` | boolean |  |
| `theme` | string |  |
| `extra_time1` | string:date-time |  |

### CreateTagDto
- **类型:** object
- **必填字段:** `name`

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 标签名称 |

### CreateShareDto
- **类型:** object
- **必填字段:** `userId`, `albumId`, `link`, `linkPwd`, `key`, `isSingleFile`

| 字段 | 类型 | 说明 |
|------|------|------|
| `userId` | number |  |
| `albumId` | number |  |
| `link` | boolean |  |
| `linkPwd` | string |  |
| `key` | string |  |
| `isSingleFile` | boolean |  |
| `linkEndTime` | string:date-time |  |
| `vUserIds` | array<string> |  |
| `cUserIds` | array<string> |  |

### UpdateShareDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `userId` | number |  |
| `albumId` | number |  |
| `link` | boolean |  |
| `linkPwd` | string |  |
| `key` | string |  |
| `isSingleFile` | boolean |  |
| `linkEndTime` | string:date-time |  |
| `vUserIds` | array<string> |  |
| `cUserIds` | array<string> |  |

### CreateShareFilesDto
- **类型:** object
- **必填字段:** `userId`, `files`, `count`, `albumId`, `cover`, `link`, `linkPwd`, `key`, `desc`, `showExif`, `showDownload`

| 字段 | 类型 | 说明 |
|------|------|------|
| `userId` | number |  |
| `files` | array<string> |  |
| `count` | number |  |
| `albumId` | number |  |
| `cover` | string |  |
| `link` | boolean |  |
| `linkPwd` | string |  |
| `key` | string |  |
| `desc` | string |  |
| `linkEndTime` | string:date-time |  |
| `showExif` | boolean |  |
| `showDownload` | boolean |  |

### UpdateShareFilesDto
- **类型:** object

| 字段 | 类型 | 说明 |
|------|------|------|
| `userId` | number |  |
| `files` | array<string> |  |
| `count` | number |  |
| `albumId` | number |  |
| `cover` | string |  |
| `link` | boolean |  |
| `linkPwd` | string |  |
| `key` | string |  |
| `desc` | string |  |
| `linkEndTime` | string:date-time |  |
| `showExif` | boolean |  |
| `showDownload` | boolean |  |
