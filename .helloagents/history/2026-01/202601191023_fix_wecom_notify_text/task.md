# 任务清单: GitHub Actions 企业微信通知改用文本并补充提交信息

目录: `helloagents/history/2026-01/202601191023_fix_wecom_notify_text/`

---

## 1. Docker 构建通知（Build and Push Docker Image）
- [√] 1.1 调整 `.github/workflows/docker-publish.yml`：企业微信通知从 `markdown` 改为 `text`
- [√] 1.2 按指定格式补齐字段：项目/分支/提交短SHA/提交信息/提交者/镜像标签/提交链接/详情链接
- [√] 1.3 镜像标签补充 `sha-<full sha>`，并在通知中输出所有实际推送的 tags（逗号分隔）

## 2. Release 通知（可选对齐）
- [√] 2.1 调整 `.github/workflows/release.yml`：企业微信通知从 `markdown` 改为 `text`
- [√] 2.2 增加提交信息字段（取当前 Tag 指向提交的 subject）

## 3. 文档与记录
- [√] 3.1 更新 `helloagents/CHANGELOG.md` 记录本次修复
- [√] 3.2 更新 `helloagents/history/index.md` 并迁移方案包至 `helloagents/history/`
