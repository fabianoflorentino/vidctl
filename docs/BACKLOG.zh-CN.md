# Backlog

待处理或已解决的问题和改进，从最新到最旧。

## 2026-10-11 — 已解决：音频提取 + 一次一个视频的模型

- **阶段 6（音频提取）：** 新增的 `internal/audio` 包为 mp3/opus/aac/flac/wav/复制构建 ffmpeg 命令（清晰的校验、基于 seek 的裁剪、loudnorm EBU R128），通过 `Task.Kind = "audio"` 接入队列，并使用相同的进度事件（现在启用 `-progress pipe:1`）；绑定 `ExtractAudio`/`SuggestAudioOutput`/`OpenAudioDialog`，并在侧边栏新增 **“提取音频”** 部分。
- **一次一个视频：**“添加视频…”改为 **“切换视频…”** — 切换会替换当前选择并重置按文件选项（不批量入队）；“转换队列”改为 **“任务队列”**；侧边栏部分重命名：“切割”和“视频调整”。

## 2026-09-27 — 已解决：顶部栏把 ffmpeg 指示器拉长到标题

- 在主窗口中，`ffmpeg OK` 指示器的药丸形背景被拉长到中间的“vidctl” — 是网格项在 `1fr` 列中被拉伸。
- 修复：移除背景/描边（药丸的 `background` + `border-radius` + `padding`）并添加 `justify-self: start`；指示器现在只有圆点 + 文本。
- 文件：`frontend/src/style.css`（`.ffmpeg-pill`）。
