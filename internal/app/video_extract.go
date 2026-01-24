package app

// 视频抽帧任务：负责从视频中提取图片并维护任务状态（创建/运行/取消/继续/删除）。
// 说明：底层通过 ffprobe 获取视频信息，通过 ffmpeg 执行抽帧；任务异步运行，结果落盘到 ./upload 下。

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"liao/internal/config"
)

var execCommandContext = exec.CommandContext

type VideoExtractSourceType string

const (
	VideoExtractSourceUpload  VideoExtractSourceType = "upload"
	VideoExtractSourceMtPhoto VideoExtractSourceType = "mtPhoto"
)

type VideoExtractMode string

const (
	VideoExtractModeKeyframe VideoExtractMode = "keyframe"
	VideoExtractModeFPS      VideoExtractMode = "fps"
	VideoExtractModeAll      VideoExtractMode = "all"
)

type VideoExtractKeyframeMode string

const (
	VideoExtractKeyframeIFrame VideoExtractKeyframeMode = "iframe"
	VideoExtractKeyframeScene  VideoExtractKeyframeMode = "scene"
)

type VideoExtractTaskStatus string

const (
	VideoExtractStatusPending     VideoExtractTaskStatus = "PENDING"
	VideoExtractStatusPreparing   VideoExtractTaskStatus = "PREPARING"
	VideoExtractStatusRunning     VideoExtractTaskStatus = "RUNNING"
	VideoExtractStatusPausedUser  VideoExtractTaskStatus = "PAUSED_USER"
	VideoExtractStatusPausedLimit VideoExtractTaskStatus = "PAUSED_LIMIT"
	VideoExtractStatusFinished    VideoExtractTaskStatus = "FINISHED"
	VideoExtractStatusFailed      VideoExtractTaskStatus = "FAILED"
)

type VideoExtractStopReason string

const (
	VideoExtractStopReasonMaxFrames VideoExtractStopReason = "MAX_FRAMES"
	VideoExtractStopReasonEndSec    VideoExtractStopReason = "END_SEC"
	VideoExtractStopReasonEOF       VideoExtractStopReason = "EOF"
	VideoExtractStopReasonUser      VideoExtractStopReason = "USER"
	VideoExtractStopReasonError     VideoExtractStopReason = "ERROR"
)

type VideoExtractOutputFormat string

const (
	VideoExtractOutputJPG VideoExtractOutputFormat = "jpg"
	VideoExtractOutputPNG VideoExtractOutputFormat = "png"
)

type VideoProbeResult struct {
	DurationSec float64 `json:"durationSec"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AvgFPS      float64 `json:"avgFps,omitempty"`
}

type VideoExtractCreateRequest struct {
	UserID string `json:"userId,omitempty"`

	SourceType VideoExtractSourceType `json:"sourceType"`
	LocalPath  string                 `json:"localPath,omitempty"` // sourceType=upload
	MD5        string                 `json:"md5,omitempty"`       // sourceType=mtPhoto

	Mode         VideoExtractMode         `json:"mode"`
	KeyframeMode VideoExtractKeyframeMode `json:"keyframeMode,omitempty"`
	SceneThresh  *float64                 `json:"sceneThreshold,omitempty"`
	FPS          *float64                 `json:"fps,omitempty"`

	StartSec  *float64 `json:"startSec,omitempty"`
	EndSec    *float64 `json:"endSec,omitempty"`
	MaxFrames int      `json:"maxFrames"`

	OutputFormat VideoExtractOutputFormat `json:"outputFormat,omitempty"`
	JPGQuality   *int                     `json:"jpgQuality,omitempty"`
}

type VideoExtractContinueRequest struct {
	TaskID string `json:"taskId"`

	// EndSec/maxFrames 为“新的上限值”（绝对值）。前端如需“追加”，可自行计算后传入。
	EndSec    *float64 `json:"endSec,omitempty"`
	MaxFrames *int     `json:"maxFrames,omitempty"`
}

type VideoExtractCancelRequest struct {
	TaskID string `json:"taskId"`
}

type VideoExtractDeleteRequest struct {
	TaskID      string `json:"taskId"`
	DeleteFiles bool   `json:"deleteFiles"`
}

type VideoExtractTask struct {
	TaskID string `json:"taskId"`
	UserID string `json:"userId,omitempty"`

	SourceType VideoExtractSourceType `json:"sourceType"`
	SourceRef  string                 `json:"sourceRef"`

	OutputDirLocalPath string                   `json:"outputDirLocalPath"`
	OutputDirURL       string                   `json:"outputDirUrl,omitempty"`
	OutputFormat       VideoExtractOutputFormat `json:"outputFormat"`
	JPGQuality         *int                     `json:"jpgQuality,omitempty"`

	Mode         VideoExtractMode         `json:"mode"`
	KeyframeMode VideoExtractKeyframeMode `json:"keyframeMode,omitempty"`
	FPS          *float64                 `json:"fps,omitempty"`
	SceneThresh  *float64                 `json:"sceneThreshold,omitempty"`

	StartSec  *float64 `json:"startSec,omitempty"`
	EndSec    *float64 `json:"endSec,omitempty"`
	MaxFrames int      `json:"maxFrames"`

	FramesExtracted int      `json:"framesExtracted"`
	VideoWidth      int      `json:"videoWidth"`
	VideoHeight     int      `json:"videoHeight"`
	DurationSec     *float64 `json:"durationSec,omitempty"`

	CursorOutTimeSec *float64 `json:"cursorOutTimeSec,omitempty"`

	Status     VideoExtractTaskStatus `json:"status"`
	StopReason VideoExtractStopReason `json:"stopReason,omitempty"`
	LastError  string                 `json:"lastError,omitempty"`

	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`

	// 运行时增强字段（仅当任务在本进程内运行/近期运行过才有值）
	Runtime *VideoExtractRuntimeView `json:"runtime,omitempty"`
}

type VideoExtractFrame struct {
	Seq int    `json:"seq"`
	URL string `json:"url"`
}

type VideoExtractFramesPage struct {
	Items      []VideoExtractFrame `json:"items"`
	NextCursor int                 `json:"nextCursor"`
	HasMore    bool                `json:"hasMore"`
}

type VideoExtractRuntimeView struct {
	Frame      int      `json:"frame"`
	OutTimeSec float64  `json:"outTimeSec"`
	Speed      string   `json:"speed,omitempty"`
	Logs       []string `json:"logs,omitempty"`
}

type videoExtractRuntime struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	cmd    *exec.Cmd

	lastFrame     int
	lastOutTimeMs int64
	lastSpeed     string

	logs []string
}

func (r *videoExtractRuntime) snapshot(maxLogs int) VideoExtractRuntimeView {
	r.mu.Lock()
	defer r.mu.Unlock()

	outTimeSec := float64(r.lastOutTimeMs) / 1_000_000.0
	logs := []string{}
	if maxLogs > 0 && len(r.logs) > 0 {
		start := 0
		if len(r.logs) > maxLogs {
			start = len(r.logs) - maxLogs
		}
		logs = append(logs, r.logs[start:]...)
	}
	return VideoExtractRuntimeView{
		Frame:      r.lastFrame,
		OutTimeSec: outTimeSec,
		Speed:      r.lastSpeed,
		Logs:       logs,
	}
}

func (r *videoExtractRuntime) appendLog(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, line)
	if len(r.logs) > 200 {
		r.logs = r.logs[len(r.logs)-200:]
	}
}

func (r *videoExtractRuntime) setProgress(frame int, outTimeMs int64, speed string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if frame >= 0 {
		r.lastFrame = frame
	}
	if outTimeMs >= 0 {
		r.lastOutTimeMs = outTimeMs
	}
	if strings.TrimSpace(speed) != "" {
		r.lastSpeed = strings.TrimSpace(speed)
	}
}

type VideoExtractService struct {
	db        *sql.DB
	cfg       config.Config
	fileStore *FileStorageService
	mtPhoto   mtPhotoFilePathResolver

	queue    chan string
	closing  chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	runtimes map[string]*videoExtractRuntime
}

type mtPhotoFilePathResolver interface {
	ResolveFilePath(ctx context.Context, md5Value string) (*MtPhotoFilePath, error)
}

func NewVideoExtractService(db *sql.DB, cfg config.Config, fileStore *FileStorageService, mtPhoto mtPhotoFilePathResolver) *VideoExtractService {
	s := &VideoExtractService{
		db:        db,
		cfg:       cfg,
		fileStore: fileStore,
		mtPhoto:   mtPhoto,
		queue:     make(chan string, cfg.VideoExtractQueueSize),
		closing:   make(chan struct{}),
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	workers := cfg.VideoExtractWorkers
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.workerLoop()
		}()
	}
	return s
}

func (s *VideoExtractService) Shutdown() {
	select {
	case <-s.closing:
		return
	default:
		close(s.closing)
	}

	s.mu.Lock()
	for _, rt := range s.runtimes {
		if rt != nil && rt.cancel != nil {
			rt.cancel()
		}
	}
	s.mu.Unlock()

	s.wg.Wait()
}

func (s *VideoExtractService) Enqueue(taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("taskId 为空")
	}

	select {
	case <-s.closing:
		return fmt.Errorf("服务已关闭")
	default:
	}

	select {
	case s.queue <- taskID:
		return nil
	default:
		return fmt.Errorf("队列已满")
	}
}

func (s *VideoExtractService) GetRuntime(taskID string) *videoExtractRuntime {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runtimes[taskID]
}

func (s *VideoExtractService) CancelTask(taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}

	s.mu.Lock()
	rt := s.runtimes[taskID]
	s.mu.Unlock()

	if rt != nil && rt.cancel != nil {
		rt.cancel()
		return true
	}
	return false
}

func (s *VideoExtractService) CreateTask(ctx context.Context, req VideoExtractCreateRequest) (taskID string, probe VideoProbeResult, err error) {
	if s == nil || s.db == nil || s.fileStore == nil {
		return "", VideoProbeResult{}, fmt.Errorf("服务未初始化")
	}

	if req.MaxFrames <= 0 {
		return "", VideoProbeResult{}, fmt.Errorf("maxFrames 非法")
	}

	sourceType := req.SourceType
	if sourceType != VideoExtractSourceUpload && sourceType != VideoExtractSourceMtPhoto {
		return "", VideoProbeResult{}, fmt.Errorf("sourceType 非法")
	}

	mode := req.Mode
	switch mode {
	case VideoExtractModeKeyframe, VideoExtractModeFPS, VideoExtractModeAll:
	default:
		return "", VideoProbeResult{}, fmt.Errorf("mode 非法")
	}

	if mode == VideoExtractModeFPS {
		if req.FPS == nil || *req.FPS <= 0 {
			return "", VideoProbeResult{}, fmt.Errorf("fps 非法")
		}
	}
	if mode == VideoExtractModeKeyframe {
		k := req.KeyframeMode
		if k != VideoExtractKeyframeIFrame && k != VideoExtractKeyframeScene {
			// 默认 I 帧
			req.KeyframeMode = VideoExtractKeyframeIFrame
		}
		if req.KeyframeMode == VideoExtractKeyframeScene {
			if req.SceneThresh == nil {
				v := 0.3
				req.SceneThresh = &v
			}
			if req.SceneThresh != nil && (*req.SceneThresh < 0 || *req.SceneThresh > 1) {
				return "", VideoProbeResult{}, fmt.Errorf("sceneThreshold 非法")
			}
		}
	}

	if req.StartSec != nil && *req.StartSec < 0 {
		return "", VideoProbeResult{}, fmt.Errorf("startSec 非法")
	}
	if req.EndSec != nil && *req.EndSec < 0 {
		return "", VideoProbeResult{}, fmt.Errorf("endSec 非法")
	}
	if req.StartSec != nil && req.EndSec != nil && *req.EndSec > 0 && *req.EndSec <= *req.StartSec {
		return "", VideoProbeResult{}, fmt.Errorf("endSec 必须大于 startSec")
	}

	if req.OutputFormat == "" {
		req.OutputFormat = VideoExtractOutputJPG
	}
	switch req.OutputFormat {
	case VideoExtractOutputJPG, VideoExtractOutputPNG:
	default:
		return "", VideoProbeResult{}, fmt.Errorf("outputFormat 非法")
	}
	if req.JPGQuality != nil {
		if *req.JPGQuality < 1 || *req.JPGQuality > 31 {
			return "", VideoProbeResult{}, fmt.Errorf("jpgQuality 非法（1-31）")
		}
		if req.OutputFormat != VideoExtractOutputJPG {
			req.JPGQuality = nil
		}
	}

	sourceRef := ""
	inputAbs := ""
	switch sourceType {
	case VideoExtractSourceUpload:
		sourceRef = normalizeUploadLocalPathInput(req.LocalPath)
		if sourceRef == "" {
			return "", VideoProbeResult{}, fmt.Errorf("localPath 为空")
		}
		inputAbs, err = s.resolveUploadAbsPath(sourceRef)
		if err != nil {
			return "", VideoProbeResult{}, err
		}
	case VideoExtractSourceMtPhoto:
		md5Value := strings.TrimSpace(req.MD5)
		if md5Value == "" {
			return "", VideoProbeResult{}, fmt.Errorf("md5 为空")
		}
		if !isHexMD5(md5Value) {
			return "", VideoProbeResult{}, fmt.Errorf("md5 非法")
		}
		sourceRef = md5Value
		if s.mtPhoto == nil {
			return "", VideoProbeResult{}, fmt.Errorf("mtPhoto 未配置")
		}
		item, err2 := s.mtPhoto.ResolveFilePath(ctx, md5Value)
		if err2 != nil || item == nil {
			if err2 != nil {
				return "", VideoProbeResult{}, fmt.Errorf("解析 mtPhoto 文件路径失败: %w", err2)
			}
			return "", VideoProbeResult{}, fmt.Errorf("解析 mtPhoto 文件路径失败")
		}
		inputAbs, err = resolveLspLocalPath(s.cfg.LspRoot, item.FilePath)
		if err != nil {
			return "", VideoProbeResult{}, fmt.Errorf("文件路径非法: %w", err)
		}
	}

	probe, err = s.ProbeVideo(ctx, inputAbs)
	if err != nil {
		return "", VideoProbeResult{}, err
	}
	if probe.Width <= 0 || probe.Height <= 0 {
		return "", VideoProbeResult{}, fmt.Errorf("视频宽高解析失败")
	}

	taskID = uuid.NewString()
	outputDirLocalPath := filepath.ToSlash(filepath.Join("/extract", taskID))
	outputFramesLocalPath := filepath.ToSlash(filepath.Join(outputDirLocalPath, "frames"))
	outputFramesAbsPath := filepath.Join(s.fileStore.baseUploadAbs, filepath.FromSlash(strings.TrimPrefix(outputFramesLocalPath, "/")))
	if err := os.MkdirAll(outputFramesAbsPath, 0o755); err != nil {
		return "", VideoProbeResult{}, fmt.Errorf("创建输出目录失败: %w", err)
	}

	now := time.Now()
	durationSec := probe.DurationSec
	if durationSec <= 0 {
		durationSec = 0
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO video_extract_task
		(task_id, user_id, source_type, source_ref, input_abs_path, output_dir_local_path, output_format, jpg_quality,
		 mode, keyframe_mode, fps, scene_threshold, start_sec, end_sec, max_frames_total, frames_extracted,
		 video_width, video_height, duration_sec, cursor_out_time_sec, status, stop_reason, last_error, last_logs, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID,
		nullStringIfEmpty(strings.TrimSpace(req.UserID)),
		string(sourceType),
		sourceRef,
		inputAbs,
		outputDirLocalPath,
		string(req.OutputFormat),
		nullIntIfNil(req.JPGQuality),
		string(mode),
		nullStringIfEmpty(string(req.KeyframeMode)),
		nullFloatIfNil(req.FPS),
		nullFloatIfNil(req.SceneThresh),
		nullFloatIfNil(req.StartSec),
		nullFloatIfNil(req.EndSec),
		req.MaxFrames,
		0,
		probe.Width,
		probe.Height,
		nullFloatIfZero(durationSec),
		nil,
		string(VideoExtractStatusPending),
		nil,
		nil,
		nil,
		now,
		now,
	)
	if err != nil {
		return "", VideoProbeResult{}, err
	}

	_ = s.Enqueue(taskID)
	return taskID, probe, nil
}

func (s *VideoExtractService) ProbeVideo(ctx context.Context, inputAbsPath string) (VideoProbeResult, error) {
	inputAbsPath = strings.TrimSpace(inputAbsPath)
	if inputAbsPath == "" {
		return VideoProbeResult{}, fmt.Errorf("输入路径为空")
	}
	if fi, err := os.Stat(inputAbsPath); err != nil || fi.IsDir() {
		return VideoProbeResult{}, fmt.Errorf("输入文件不存在")
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	out, err := execCommandContext(ctx,
		s.cfg.FFprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,avg_frame_rate",
		"-show_entries", "format=duration",
		"-of", "json",
		inputAbsPath,
	).Output()
	if err != nil {
		return VideoProbeResult{}, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	var parsed struct {
		Streams []struct {
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			AvgFrameRate string `json:"avg_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return VideoProbeResult{}, fmt.Errorf("ffprobe 输出解析失败: %w", err)
	}

	width := 0
	height := 0
	avgFPS := 0.0
	if len(parsed.Streams) > 0 {
		width = parsed.Streams[0].Width
		height = parsed.Streams[0].Height
		avgFPS = parseFFprobeFrameRate(parsed.Streams[0].AvgFrameRate)
	}
	durationSec := parseFFprobeDuration(parsed.Format.Duration)

	return VideoProbeResult{
		DurationSec: durationSec,
		Width:       width,
		Height:      height,
		AvgFPS:      avgFPS,
	}, nil
}

func parseFFprobeDuration(v string) float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	if f < 0 {
		return 0
	}
	return f
}

func parseFFprobeFrameRate(v string) float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if !strings.Contains(v, "/") {
		f, _ := strconv.ParseFloat(v, 64)
		if f > 0 && !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f
		}
		return 0
	}
	parts := strings.SplitN(v, "/", 2)
	num, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	den, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	f := num / den
	if f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

var md5Re = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

func isHexMD5(v string) bool {
	return md5Re.MatchString(strings.TrimSpace(v))
}

func (s *VideoExtractService) resolveUploadAbsPath(localPath string) (string, error) {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return "", fmt.Errorf("localPath 为空")
	}
	if !strings.HasPrefix(localPath, "/") {
		localPath = "/" + localPath
	}

	// 临时输入视频：/tmp/video_extract_inputs/...（物理路径位于 fileStore.baseTempAbs）
	tempPrefix := "/" + tempVideoExtractInputsDir + "/"
	if strings.HasPrefix(localPath, tempPrefix) {
		if s.fileStore == nil {
			return "", fmt.Errorf("文件服务未初始化")
		}

		baseTempAbs := strings.TrimSpace(s.fileStore.baseTempAbs)
		if baseTempAbs == "" {
			baseTempAbs = filepath.Join(os.TempDir(), "video_extract_inputs")
		}

		inner := strings.TrimPrefix(localPath, tempPrefix)
		cleanInner := filepath.Clean(filepath.FromSlash(inner))
		if cleanInner == "." || cleanInner == string(filepath.Separator) {
			return "", fmt.Errorf("localPath 非法")
		}

		full := filepath.Join(baseTempAbs, cleanInner)
		rel, err := filepathRelFn(baseTempAbs, full)
		if err != nil {
			return "", fmt.Errorf("路径解析失败: %w", err)
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("检测到路径越界")
		}

		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			return "", fmt.Errorf("视频文件不存在")
		}
		return full, nil
	}

	clean := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if clean == "." || clean == string(filepath.Separator) {
		return "", fmt.Errorf("localPath 非法")
	}

	full := filepath.Join(s.fileStore.baseUploadAbs, clean)
	rel, err := filepathRelFn(s.fileStore.baseUploadAbs, full)
	if err != nil {
		return "", fmt.Errorf("路径解析失败: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("检测到路径越界")
	}

	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		return "", fmt.Errorf("视频文件不存在")
	}
	return full, nil
}

func (s *VideoExtractService) workerLoop() {
	for {
		select {
		case <-s.closing:
			return
		case taskID := <-s.queue:
			taskID = strings.TrimSpace(taskID)
			if taskID == "" {
				continue
			}
			if err := s.runTask(taskID); err != nil {
				slog.Error("抽帧任务执行失败", "taskId", taskID, "error", err)
			}
		}
	}
}

type taskRow struct {
	TaskID string

	SourceType string
	SourceRef  string
	InputAbs   string

	OutputDirLocalPath string
	OutputFormat       string
	JPGQuality         sql.NullInt64

	Mode         string
	KeyframeMode sql.NullString
	FPS          sql.NullFloat64
	SceneThresh  sql.NullFloat64

	StartSec sql.NullFloat64
	EndSec   sql.NullFloat64

	MaxFramesTotal  int
	FramesExtracted int

	VideoWidth  int
	VideoHeight int

	DurationSec      sql.NullFloat64
	CursorOutTimeSec sql.NullFloat64

	Status VideoExtractTaskStatus
}

func (s *VideoExtractService) loadTaskRow(ctx context.Context, taskID string) (*taskRow, error) {
	row := &taskRow{}
	err := s.db.QueryRowContext(ctx, `SELECT task_id, source_type, source_ref, input_abs_path,
		output_dir_local_path, output_format, jpg_quality,
		mode, keyframe_mode, fps, scene_threshold,
		start_sec, end_sec, max_frames_total, frames_extracted,
		video_width, video_height, duration_sec, cursor_out_time_sec, status
		FROM video_extract_task WHERE task_id = ?`, taskID).Scan(
		&row.TaskID,
		&row.SourceType,
		&row.SourceRef,
		&row.InputAbs,
		&row.OutputDirLocalPath,
		&row.OutputFormat,
		&row.JPGQuality,
		&row.Mode,
		&row.KeyframeMode,
		&row.FPS,
		&row.SceneThresh,
		&row.StartSec,
		&row.EndSec,
		&row.MaxFramesTotal,
		&row.FramesExtracted,
		&row.VideoWidth,
		&row.VideoHeight,
		&row.DurationSec,
		&row.CursorOutTimeSec,
		&row.Status,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}
	return row, nil
}

func (s *VideoExtractService) runTask(taskID string) error {
	ctx := context.Background()

	task, err := s.loadTaskRow(ctx, taskID)
	if err != nil {
		return err
	}

	// 避免重复并发运行同一任务
	s.mu.Lock()
	if _, ok := s.runtimes[taskID]; ok {
		s.mu.Unlock()
		return nil
	}
	rt := &videoExtractRuntime{}
	rtCtx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	s.runtimes[taskID] = rt
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.runtimes, taskID)
		s.mu.Unlock()
	}()

	if err := s.setTaskStatus(ctx, taskID, VideoExtractStatusRunning, "", ""); err != nil {
		rt.appendLog("更新任务状态失败: " + err.Error())
	}

	startSecAbs := 0.0
	if task.StartSec.Valid {
		startSecAbs = task.StartSec.Float64
	}
	if task.CursorOutTimeSec.Valid && task.CursorOutTimeSec.Float64 > startSecAbs {
		// 续跑：从上次游标略微后移，避免重复输出最后一帧
		startSecAbs = task.CursorOutTimeSec.Float64 + 0.001
	}

	endSecAbs := 0.0
	hasEndLimit := task.EndSec.Valid && task.EndSec.Float64 > 0
	if hasEndLimit {
		endSecAbs = task.EndSec.Float64
	}
	if hasEndLimit && endSecAbs > 0 && endSecAbs <= startSecAbs {
		// 已经到达/超过 endSec，无需继续
		_ = s.setTaskStatus(ctx, taskID, VideoExtractStatusPausedLimit, string(VideoExtractStopReasonEndSec), "")
		return nil
	}

	framesRemaining := task.MaxFramesTotal - task.FramesExtracted
	if framesRemaining <= 0 {
		_ = s.setTaskStatus(ctx, taskID, VideoExtractStatusPausedLimit, string(VideoExtractStopReasonMaxFrames), "")
		return nil
	}

	outputFormat := strings.ToLower(strings.TrimSpace(task.OutputFormat))
	if outputFormat == "" {
		outputFormat = string(VideoExtractOutputJPG)
	}

	outputDirLocalPath := filepath.ToSlash(task.OutputDirLocalPath)
	if !strings.HasPrefix(outputDirLocalPath, "/") {
		outputDirLocalPath = "/" + outputDirLocalPath
	}
	framesLocalPath := filepath.ToSlash(filepath.Join(outputDirLocalPath, "frames"))
	framesAbsPath := filepath.Join(s.fileStore.baseUploadAbs, filepath.FromSlash(strings.TrimPrefix(framesLocalPath, "/")))
	if err := os.MkdirAll(framesAbsPath, 0o755); err != nil {
		_ = s.setTaskStatus(ctx, taskID, VideoExtractStatusFailed, string(VideoExtractStopReasonError), "创建输出目录失败: "+err.Error())
		return err
	}

	startNumber := task.FramesExtracted + 1
	outputPattern := filepath.Join(framesAbsPath, fmt.Sprintf("frame_%%06d.%s", outputFormat))

	args := make([]string, 0, 32)
	args = append(args, "-hide_banner", "-nostdin")

	// 进度输出到 stdout，便于解析 out_time_ms/frame/speed
	args = append(args, "-progress", "pipe:1", "-stats_period", "1")

	if startSecAbs > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", startSecAbs))
	}
	args = append(args, "-i", task.InputAbs)

	if hasEndLimit && endSecAbs > 0 {
		dur := endSecAbs - startSecAbs
		if dur > 0 {
			args = append(args, "-t", fmt.Sprintf("%.3f", dur))
		}
	}

	// 说明：抽帧不需要音频
	args = append(args, "-an")

	// 滤镜
	filter := ""
	mode := strings.ToLower(strings.TrimSpace(task.Mode))
	switch VideoExtractMode(mode) {
	case VideoExtractModeKeyframe:
		km := strings.ToLower(strings.TrimSpace(task.KeyframeMode.String))
		if VideoExtractKeyframeMode(km) == VideoExtractKeyframeScene {
			th := 0.3
			if task.SceneThresh.Valid {
				th = task.SceneThresh.Float64
			}
			filter = fmt.Sprintf("select='gt(scene\\,%0.3f)'", th)
			args = append(args, "-vsync", "vfr")
		} else {
			filter = "select='eq(pict_type\\,I)'"
			args = append(args, "-vsync", "vfr")
		}
	case VideoExtractModeFPS:
		fps := 1.0
		if task.FPS.Valid && task.FPS.Float64 > 0 {
			fps = task.FPS.Float64
		}
		filter = fmt.Sprintf("fps=%0.3f", fps)
		args = append(args, "-vsync", "vfr")
	default:
		// all：默认输出所有帧
		args = append(args, "-vsync", "0")
	}
	if strings.TrimSpace(filter) != "" {
		args = append(args, "-vf", filter)
	}

	// 帧数限制
	args = append(args, "-frames:v", strconv.Itoa(framesRemaining))

	// 输出质量
	if outputFormat == string(VideoExtractOutputJPG) && task.JPGQuality.Valid && task.JPGQuality.Int64 > 0 {
		args = append(args, "-q:v", strconv.FormatInt(task.JPGQuality.Int64, 10))
	}

	args = append(args, "-start_number", strconv.Itoa(startNumber))
	args = append(args, outputPattern)

	cmd := execCommandContext(rtCtx, s.cfg.FFmpegPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = s.setTaskStatus(ctx, taskID, VideoExtractStatusFailed, string(VideoExtractStopReasonError), "ffmpeg 启动失败: "+err.Error())
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = s.setTaskStatus(ctx, taskID, VideoExtractStatusFailed, string(VideoExtractStopReasonError), "ffmpeg 启动失败: "+err.Error())
		return err
	}

	rt.cmd = cmd

	if err := cmd.Start(); err != nil {
		_ = s.setTaskStatus(ctx, taskID, VideoExtractStatusFailed, string(VideoExtractStopReasonError), "ffmpeg 启动失败: "+err.Error())
		return err
	}

	// 读取 stdout progress
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		scanner := bufio.NewScanner(stdout)
		frame := -1
		outTimeMs := int64(-1)
		speed := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// 记录少量关键日志，避免刷屏
			if strings.HasPrefix(line, "progress=") || strings.HasPrefix(line, "out_time_ms=") || strings.HasPrefix(line, "frame=") || strings.HasPrefix(line, "speed=") {
				rt.appendLog(line)
			}
			if strings.HasPrefix(line, "frame=") {
				if v, err := strconv.Atoi(strings.TrimPrefix(line, "frame=")); err == nil {
					frame = v
				}
			}
			if strings.HasPrefix(line, "out_time_ms=") {
				if v, err := strconv.ParseInt(strings.TrimPrefix(line, "out_time_ms="), 10, 64); err == nil {
					outTimeMs = v
				}
			}
			if strings.HasPrefix(line, "speed=") {
				speed = strings.TrimPrefix(line, "speed=")
			}
			if line == "progress=end" || line == "progress=continue" {
				rt.setProgress(frame, outTimeMs, speed)
				_ = s.updateProgress(ctx, taskID, task, startSecAbs, outTimeMs, frame)
			}
		}
	}()

	// 读取 stderr（错误/警告）
	stderrDone := make(chan struct{})
	lastStderrLine := ""
	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			rt.appendLog(line)
			if strings.TrimSpace(line) != "" {
				lastStderrLine = line
			}
		}
	}()

	// 写入帧索引：根据输出文件是否存在顺序插入，支持“运行中预览”
	nextSeq := startNumber
	flushFrames := func() {
		for {
			filename := fmt.Sprintf("frame_%06d.%s", nextSeq, outputFormat)
			abs := filepath.Join(framesAbsPath, filename)
			if _, err := os.Stat(abs); err != nil {
				return
			}
			rel := filepath.ToSlash(filepath.Join(framesLocalPath, filename))
			_ = s.insertFrame(ctx, taskID, nextSeq, rel)
			nextSeq++
		}
	}

	ticker := time.NewTicker(600 * time.Millisecond)
	defer ticker.Stop()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	var waitErr error
loop:
	for {
		select {
		case <-rtCtx.Done():
			flushFrames()
			waitErr = <-waitDone
			break loop
		case waitErr = <-waitDone:
			flushFrames()
			break loop
		case <-ticker.C:
			flushFrames()
			// 定期把最近进度刷入 DB（节流）
			rt.mu.Lock()
			outTimeMs := rt.lastOutTimeMs
			frame := rt.lastFrame
			rt.mu.Unlock()
			_ = s.updateProgress(ctx, taskID, task, startSecAbs, outTimeMs, frame)
		}
	}

	<-progressDone
	<-stderrDone
	flushFrames()

	// 结束时再次读取任务数据（frames_extracted 已在 insertFrame 中累加）
	task2, err := s.loadTaskRow(ctx, taskID)
	if err != nil {
		return err
	}

	// 保存最后日志片段（最多 50 行）
	lastLogsJSON := ""
	{
		view := rt.snapshot(50)
		if len(view.Logs) > 0 {
			if b, err := json.Marshal(view.Logs); err == nil {
				lastLogsJSON = string(b)
			}
		}
	}

	if errors.Is(rtCtx.Err(), context.Canceled) {
		_ = s.setTaskStatusWithLogs(ctx, taskID, VideoExtractStatusPausedUser, string(VideoExtractStopReasonUser), "", lastLogsJSON)
		return nil
	}

	if waitErr != nil {
		msg := strings.TrimSpace(lastStderrLine)
		if msg == "" {
			msg = waitErr.Error()
		}
		_ = s.setTaskStatusWithLogs(ctx, taskID, VideoExtractStatusFailed, string(VideoExtractStopReasonError), msg, lastLogsJSON)
		return waitErr
	}

	// 判断停止原因：maxFrames/endSec/EOF
	reason := VideoExtractStopReasonEOF
	status := VideoExtractStatusFinished

	if task2.FramesExtracted >= task2.MaxFramesTotal {
		reason = VideoExtractStopReasonMaxFrames
		status = VideoExtractStatusPausedLimit
	}

	if task2.EndSec.Valid && task2.EndSec.Float64 > 0 {
		endSec := task2.EndSec.Float64
		cur := 0.0
		if task2.CursorOutTimeSec.Valid {
			cur = task2.CursorOutTimeSec.Float64
		}
		dur := 0.0
		if task2.DurationSec.Valid {
			dur = task2.DurationSec.Float64
		}
		// 若 endSec 明显小于视频总时长，则视为“可继续的限制”；接近总时长则视为自然结束。
		if dur <= 0 || endSec < dur-0.05 {
			if cur >= endSec-0.02 {
				reason = VideoExtractStopReasonEndSec
				status = VideoExtractStatusPausedLimit
			}
		}
	}

	_ = s.setTaskStatusWithLogs(ctx, taskID, status, string(reason), "", lastLogsJSON)
	return nil
}

func (s *VideoExtractService) updateProgress(ctx context.Context, taskID string, base *taskRow, startSecAbs float64, outTimeMs int64, frame int) error {
	if outTimeMs < 0 && frame < 0 {
		return nil
	}
	cursorAbs := startSecAbs
	if outTimeMs >= 0 {
		cursorAbs = startSecAbs + float64(outTimeMs)/1_000_000.0
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE video_extract_task SET cursor_out_time_sec = ?, updated_at = ? WHERE task_id = ?`,
		cursorAbs,
		time.Now(),
		taskID,
	)
	return nil
}

func (s *VideoExtractService) insertFrame(ctx context.Context, taskID string, seq int, relPath string) error {
	if seq <= 0 || strings.TrimSpace(relPath) == "" {
		return nil
	}
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO video_extract_frame (task_id, seq, rel_path, time_sec, created_at)
		VALUES (?, ?, ?, NULL, ?)
		ON DUPLICATE KEY UPDATE rel_path = VALUES(rel_path)`,
		taskID,
		seq,
		relPath,
		now,
	)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected > 0 {
		// frames_extracted 按“最大 seq”更新，避免重复插入导致累加错误
		_, _ = s.db.ExecContext(ctx, `UPDATE video_extract_task SET frames_extracted = GREATEST(frames_extracted, ?), updated_at = ? WHERE task_id = ?`,
			seq,
			now,
			taskID,
		)
	}
	return nil
}

func (s *VideoExtractService) setTaskStatus(ctx context.Context, taskID string, status VideoExtractTaskStatus, stopReason string, lastError string) error {
	return s.setTaskStatusWithLogs(ctx, taskID, status, stopReason, lastError, "")
}

func (s *VideoExtractService) setTaskStatusWithLogs(ctx context.Context, taskID string, status VideoExtractTaskStatus, stopReason string, lastError string, lastLogsJSON string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil
	}
	now := time.Now()
	stopReason = strings.TrimSpace(stopReason)
	lastError = strings.TrimSpace(lastError)
	lastLogsJSON = strings.TrimSpace(lastLogsJSON)

	_, err := s.db.ExecContext(ctx, `UPDATE video_extract_task
		SET status = ?, stop_reason = ?, last_error = ?, last_logs = ?, updated_at = ?
		WHERE task_id = ?`,
		string(status),
		nullStringIfEmpty(stopReason),
		nullStringIfEmpty(lastError),
		nullStringIfEmpty(lastLogsJSON),
		now,
		taskID,
	)
	return err
}

func nullIntIfNil(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullFloatIfNil(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullFloatIfZero(v float64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func (s *VideoExtractService) ContinueTask(ctx context.Context, req VideoExtractContinueRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("服务未初始化")
	}
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		return fmt.Errorf("taskId 为空")
	}

	task, err := s.loadTaskRow(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == VideoExtractStatusRunning || task.Status == VideoExtractStatusPreparing {
		return fmt.Errorf("任务运行中，无法继续")
	}
	if task.Status != VideoExtractStatusPausedUser && task.Status != VideoExtractStatusPausedLimit && task.Status != VideoExtractStatusFinished {
		return fmt.Errorf("当前状态不支持继续: %s", task.Status)
	}

	updates := make([]string, 0, 4)
	args := make([]any, 0, 6)

	if req.EndSec != nil {
		if *req.EndSec < 0 {
			return fmt.Errorf("endSec 非法")
		}
		if task.StartSec.Valid && *req.EndSec > 0 && *req.EndSec <= task.StartSec.Float64 {
			return fmt.Errorf("endSec 必须大于 startSec")
		}
		updates = append(updates, "end_sec = ?")
		args = append(args, nullFloatIfNil(req.EndSec))
	}
	if req.MaxFrames != nil {
		if *req.MaxFrames <= 0 {
			return fmt.Errorf("maxFrames 非法")
		}
		if *req.MaxFrames < task.FramesExtracted {
			return fmt.Errorf("maxFrames 不能小于已输出帧数")
		}
		updates = append(updates, "max_frames_total = ?")
		args = append(args, *req.MaxFrames)
	}

	if len(updates) == 0 {
		return fmt.Errorf("未提供任何扩展参数")
	}

	updates = append(updates, "status = ?")
	args = append(args, string(VideoExtractStatusPending))
	updates = append(updates, "stop_reason = NULL")
	updates = append(updates, "last_error = NULL")
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())

	args = append(args, taskID)
	query := "UPDATE video_extract_task SET " + strings.Join(updates, ", ") + " WHERE task_id = ?"
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return s.Enqueue(taskID)
}

func (s *VideoExtractService) DeleteTask(ctx context.Context, req VideoExtractDeleteRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("服务未初始化")
	}
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		return fmt.Errorf("taskId 为空")
	}

	_ = s.CancelTask(taskID)

	task, err := s.loadTaskRow(ctx, taskID)
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, "DELETE FROM video_extract_frame WHERE task_id = ?", taskID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM video_extract_task WHERE task_id = ?", taskID); err != nil {
		return err
	}

	if req.DeleteFiles {
		outDirLocal := filepath.ToSlash(task.OutputDirLocalPath)
		if !strings.HasPrefix(outDirLocal, "/") {
			outDirLocal = "/" + outDirLocal
		}
		clean := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(outDirLocal, "/")))
		full := filepath.Join(s.fileStore.baseUploadAbs, clean)
		rel, err := filepathRelFn(s.fileStore.baseUploadAbs, full)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			_ = os.RemoveAll(full)
		}
	}

	return nil
}

func (s *VideoExtractService) CancelAndMark(ctx context.Context, req VideoExtractCancelRequest) error {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		return fmt.Errorf("taskId 为空")
	}
	s.CancelTask(taskID)
	return s.setTaskStatus(ctx, taskID, VideoExtractStatusPausedUser, string(VideoExtractStopReasonUser), "")
}

func (s *VideoExtractService) ListTasks(ctx context.Context, page, pageSize int, hostHeader string) (items []VideoExtractTask, total int, err error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("服务未初始化")
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM video_extract_task").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT task_id, user_id, source_type, source_ref,
		output_dir_local_path, output_format, jpg_quality,
		mode, keyframe_mode, fps, scene_threshold,
		start_sec, end_sec, max_frames_total, frames_extracted,
		video_width, video_height, duration_sec, cursor_out_time_sec,
		status, stop_reason, last_error, created_at, updated_at
		FROM video_extract_task
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var t VideoExtractTask
		var userID sql.NullString
		var sourceType, sourceRef string
		var outputDirLocalPath, outputFormat string
		var jpgQuality sql.NullInt64
		var mode string
		var keyframeMode sql.NullString
		var fps sql.NullFloat64
		var sceneThresh sql.NullFloat64
		var startSec sql.NullFloat64
		var endSec sql.NullFloat64
		var maxFramesTotal int
		var framesExtracted int
		var videoWidth, videoHeight int
		var durationSec sql.NullFloat64
		var cursorOutTimeSec sql.NullFloat64
		var status VideoExtractTaskStatus
		var stopReason sql.NullString
		var lastError sql.NullString
		var createdAt, updatedAt time.Time

		if err := rows.Scan(
			&t.TaskID,
			&userID,
			&sourceType,
			&sourceRef,
			&outputDirLocalPath,
			&outputFormat,
			&jpgQuality,
			&mode,
			&keyframeMode,
			&fps,
			&sceneThresh,
			&startSec,
			&endSec,
			&maxFramesTotal,
			&framesExtracted,
			&videoWidth,
			&videoHeight,
			&durationSec,
			&cursorOutTimeSec,
			&status,
			&stopReason,
			&lastError,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, 0, err
		}

		if userID.Valid {
			t.UserID = userID.String
		}
		t.SourceType = VideoExtractSourceType(sourceType)
		t.SourceRef = sourceRef
		t.OutputDirLocalPath = outputDirLocalPath
		t.OutputDirURL = s.toUploadURL(outputDirLocalPath, hostHeader)
		t.OutputFormat = VideoExtractOutputFormat(outputFormat)
		if jpgQuality.Valid {
			v := int(jpgQuality.Int64)
			t.JPGQuality = &v
		}
		t.Mode = VideoExtractMode(mode)
		if keyframeMode.Valid {
			t.KeyframeMode = VideoExtractKeyframeMode(keyframeMode.String)
		}
		if fps.Valid {
			v := fps.Float64
			t.FPS = &v
		}
		if sceneThresh.Valid {
			v := sceneThresh.Float64
			t.SceneThresh = &v
		}
		if startSec.Valid {
			v := startSec.Float64
			t.StartSec = &v
		}
		if endSec.Valid {
			v := endSec.Float64
			t.EndSec = &v
		}
		t.MaxFrames = maxFramesTotal
		t.FramesExtracted = framesExtracted
		t.VideoWidth = videoWidth
		t.VideoHeight = videoHeight
		if durationSec.Valid {
			v := durationSec.Float64
			t.DurationSec = &v
		}
		if cursorOutTimeSec.Valid {
			v := cursorOutTimeSec.Float64
			t.CursorOutTimeSec = &v
		}
		t.Status = status
		if stopReason.Valid {
			t.StopReason = VideoExtractStopReason(stopReason.String)
		}
		if lastError.Valid {
			t.LastError = lastError.String
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		t.UpdatedAt = updatedAt.Format(time.RFC3339)

		if rt := s.GetRuntime(t.TaskID); rt != nil {
			view := rt.snapshot(20)
			t.Runtime = &view
		}

		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *VideoExtractService) GetTaskDetail(ctx context.Context, taskID string, frameCursor int, frameLimit int, hostHeader string) (task VideoExtractTask, frames VideoExtractFramesPage, err error) {
	if s == nil || s.db == nil {
		return VideoExtractTask{}, VideoExtractFramesPage{}, fmt.Errorf("服务未初始化")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return VideoExtractTask{}, VideoExtractFramesPage{}, fmt.Errorf("taskId 为空")
	}
	if frameCursor < 0 {
		frameCursor = 0
	}
	if frameLimit <= 0 {
		frameLimit = s.cfg.VideoExtractFramePageSz
	}
	if frameLimit > 300 {
		frameLimit = 300
	}

	var userID sql.NullString
	var sourceType, sourceRef string
	var outputDirLocalPath, outputFormat string
	var jpgQuality sql.NullInt64
	var mode string
	var keyframeMode sql.NullString
	var fps sql.NullFloat64
	var sceneThresh sql.NullFloat64
	var startSec sql.NullFloat64
	var endSec sql.NullFloat64
	var maxFramesTotal int
	var framesExtracted int
	var videoWidth, videoHeight int
	var durationSec sql.NullFloat64
	var cursorOutTimeSec sql.NullFloat64
	var status VideoExtractTaskStatus
	var stopReason sql.NullString
	var lastError sql.NullString
	var lastLogs sql.NullString
	var createdAt, updatedAt time.Time

	if err := s.db.QueryRowContext(ctx, `SELECT task_id, user_id, source_type, source_ref,
		output_dir_local_path, output_format, jpg_quality,
		mode, keyframe_mode, fps, scene_threshold,
		start_sec, end_sec, max_frames_total, frames_extracted,
		video_width, video_height, duration_sec, cursor_out_time_sec,
		status, stop_reason, last_error, last_logs, created_at, updated_at
		FROM video_extract_task WHERE task_id = ?`, taskID).Scan(
		&task.TaskID,
		&userID,
		&sourceType,
		&sourceRef,
		&outputDirLocalPath,
		&outputFormat,
		&jpgQuality,
		&mode,
		&keyframeMode,
		&fps,
		&sceneThresh,
		&startSec,
		&endSec,
		&maxFramesTotal,
		&framesExtracted,
		&videoWidth,
		&videoHeight,
		&durationSec,
		&cursorOutTimeSec,
		&status,
		&stopReason,
		&lastError,
		&lastLogs,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return VideoExtractTask{}, VideoExtractFramesPage{}, fmt.Errorf("任务不存在")
		}
		return VideoExtractTask{}, VideoExtractFramesPage{}, err
	}

	if userID.Valid {
		task.UserID = userID.String
	}
	task.SourceType = VideoExtractSourceType(sourceType)
	task.SourceRef = sourceRef
	task.OutputDirLocalPath = outputDirLocalPath
	task.OutputDirURL = s.toUploadURL(outputDirLocalPath, hostHeader)
	task.OutputFormat = VideoExtractOutputFormat(outputFormat)
	if jpgQuality.Valid {
		v := int(jpgQuality.Int64)
		task.JPGQuality = &v
	}
	task.Mode = VideoExtractMode(mode)
	if keyframeMode.Valid {
		task.KeyframeMode = VideoExtractKeyframeMode(keyframeMode.String)
	}
	if fps.Valid {
		v := fps.Float64
		task.FPS = &v
	}
	if sceneThresh.Valid {
		v := sceneThresh.Float64
		task.SceneThresh = &v
	}
	if startSec.Valid {
		v := startSec.Float64
		task.StartSec = &v
	}
	if endSec.Valid {
		v := endSec.Float64
		task.EndSec = &v
	}
	task.MaxFrames = maxFramesTotal
	task.FramesExtracted = framesExtracted
	task.VideoWidth = videoWidth
	task.VideoHeight = videoHeight
	if durationSec.Valid {
		v := durationSec.Float64
		task.DurationSec = &v
	}
	if cursorOutTimeSec.Valid {
		v := cursorOutTimeSec.Float64
		task.CursorOutTimeSec = &v
	}
	task.Status = status
	if stopReason.Valid {
		task.StopReason = VideoExtractStopReason(stopReason.String)
	}
	if lastError.Valid {
		task.LastError = lastError.String
	}
	task.CreatedAt = createdAt.Format(time.RFC3339)
	task.UpdatedAt = updatedAt.Format(time.RFC3339)

	if rt := s.GetRuntime(taskID); rt != nil {
		view := rt.snapshot(60)
		task.Runtime = &view
	} else if lastLogs.Valid && strings.TrimSpace(lastLogs.String) != "" {
		var logs []string
		if err := json.Unmarshal([]byte(lastLogs.String), &logs); err == nil && len(logs) > 0 {
			view := VideoExtractRuntimeView{Logs: logs}
			task.Runtime = &view
		}
	}

	rows, err := s.db.QueryContext(ctx, `SELECT seq, rel_path FROM video_extract_frame
		WHERE task_id = ? AND seq > ?
		ORDER BY seq ASC
		LIMIT ?`, taskID, frameCursor, frameLimit+1)
	if err != nil {
		return task, VideoExtractFramesPage{}, err
	}
	defer rows.Close()

	items := make([]VideoExtractFrame, 0, frameLimit)
	nextCursor := frameCursor
	hasMore := false
	for rows.Next() {
		var seq int
		var relPath string
		if err := rows.Scan(&seq, &relPath); err != nil {
			return task, VideoExtractFramesPage{}, err
		}
		if len(items) >= frameLimit {
			hasMore = true
			break
		}
		items = append(items, VideoExtractFrame{
			Seq: seq,
			URL: s.toUploadURL(relPath, hostHeader),
		})
		nextCursor = seq
	}
	if err := rows.Err(); err != nil {
		return task, VideoExtractFramesPage{}, err
	}

	frames = VideoExtractFramesPage{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
	return task, frames, nil
}

func (s *VideoExtractService) toUploadURL(localPath string, hostHeader string) string {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return ""
	}
	if !strings.HasPrefix(localPath, "/") {
		localPath = "/" + localPath
	}
	host := strings.TrimSpace(hostHeader)
	if host == "" {
		host = fmt.Sprintf("localhost:%d", s.cfg.ServerPort)
	}
	return "http://" + host + "/upload" + localPath
}
