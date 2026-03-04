# 任务清单: 图片端口解析并发竞速与兜底降级

目录: `helloagents/history/2026-01/202601110511_image_port_race/`

---

## 1. 后端（Go）
- [√] 1.1 将“真实图片请求”端口探测改为并发竞速：同时请求所有候选端口，首个返回有效（HTTP 200/206 + 内容校验）即返回，并取消其它请求
- [√] 1.2 增加兜底降级：真实探测全部失败（如 404/超时/HTML 占位）时，降级到 probe（仅 TCP 通断）或最终回退 fixed
- [√] 1.3 追加/更新测试用例，覆盖竞速胜出与降级逻辑

## 2. 知识库更新
- [√] 2.1 更新 `helloagents/wiki/modules/media.md` 对 probe/real 策略的描述（并发竞速 + 降级）

## 3. 测试
- [√] 3.1 运行后端测试：`go test ./...`（使用 Docker golang 镜像）

## 4. 收尾
- [√] 4.1 迁移方案包至 `helloagents/history/2026-01/202601110511_image_port_race/` 并更新 `helloagents/history/index.md`
- [√] 4.2 生成 commit
