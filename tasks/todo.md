# rc24 同步待办

- [x] 完成官方 rc24 各提交与当前代码的逐项差异审查
- [x] 适配 HTTP/2 请求体 `GetBody` 透明重试并添加测试
- [x] 评估并适配用户级关键限流
- [x] 适配 `web/default` 渠道模型分类
- [x] 适配兑换码额度精度修复
- [x] 补齐 Claude/Gemini 原生渠道测试请求构造和 Gemini 流式路径
- [x] 审查用户/路由 merge 与 CI 变更，按当前定制记录取舍
- [x] 运行定向测试、前端检查和差异检查

## 验证结果

- `go test ./common ./relay/common ./relay/channel`：通过。
- `go test ./model -run 'TestUpdateUserAccessToken|TestUpdateOmits' -count=1`：通过。
- `bun x tsc -b --pretty false`：通过。
- `bun run build`：通过。
- `go test ./common/... ./relay/...`：仅现有 `relay/channel/claude` MIME 内容转换测试失败，本轮未修改该逻辑。
- `git diff --check`：通过。

---

# 客户端断开后的流式 Token 结算

- [x] 调整客户端断开后的文本用量结算
- [x] 删除客户可见的内部提示
- [x] 增加回归测试并完成验证
