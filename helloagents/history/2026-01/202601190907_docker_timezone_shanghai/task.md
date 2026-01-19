# 轻量迭代任务清单：docker_timezone_shanghai

目标：在使用应用侧 `now` 写入时间字段时，确保容器默认时区为东八区（Asia/Shanghai），避免与 `DB_URL.serverTimezone` 不一致导致时间偏移/排序异常。

## 任务

- [√] Docker：设置镜像默认 `TZ=Asia/Shanghai`，并写入 `/etc/localtime` 与 `/etc/timezone`。
- [√] 知识库：在架构文档补充 `TZ` 环境变量说明与与 DB 时区一致性要求。
- [√] 变更记录：更新 `helloagents/CHANGELOG.md` 记录本次修复。
- [√] 验证：`go test ./...`

## 复验

- [?] 容器复验：启动容器后检查 `date` 输出是否为东八区时间；并复验“全站图片库”重传后排序是否稳定。
  > 备注: 需在目标环境执行。

