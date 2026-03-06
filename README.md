# B站 UP 主运营管理平台

一个基于 Go 的后台管理系统，用来把 B 站日常运营动作统一到一个管理台里：评论与私信回复、粉丝互动、热度观察、LLM 调用记录与运行配置都在同一套 `/admin` 后台完成。

这份 README 按当前代码结构说明项目，而不是按历史功能列表描述。

## 项目定位

这个项目不是一个公开站点，而是一个带登录鉴权的运营后台。

核心目标有三类：

- 聚合 B 站运营动作：评论、私信、互动、热度数据
- 把 B 站客户端和 LLM 能力封装成稳定服务层
- 通过轮询任务把同步、互动、自动回复这些动作持续化运行

默认访问入口：`/admin/login`

## 当前架构

整体分层是标准的单体后台结构：

```text
浏览器
  -> /admin 页面与 /admin/api 接口
Gin Router
  -> handler
  -> service
  -> repository
  -> SQLite / 运行时配置
          \
           -> bilibili client
           -> llm providers
```

运行时启动链路：

1. 读取 `config/config.yaml`，如果文件不存在则回退到内嵌默认配置。
2. 初始化数据库，默认使用 SQLite，并自动迁移业务表。
3. 从设置表加载应用配置，动态构造 B 站客户端和 LLM 管理器。
4. 初始化仓储、服务、处理器和 Gin 路由。
5. 启动轮询任务：评论同步、私信同步、粉丝互动、关注私信自动回复等。
6. 提供 `/admin` 页面和 `/admin/api` 接口。

## 模块划分

### 1. 入口层

- `cmd/server/main.go`

职责：

- 配置加载
- 数据库初始化与自动迁移
- 运行时依赖装配
- 轮询任务注册与启动
- 路由注册

### 2. 配置层

- `config/config.go`
- `config/config.yaml`

特点：

- 默认配置内嵌进二进制
- 支持环境变量覆盖，前缀为 `BILI_`
- 默认数据目录是 `data`
- 默认数据库驱动是 SQLite

### 3. HTTP 层

- `internal/handler/`

主要处理器：

- `auth.go`: 登录、登出、当前用户、改密
- `page.go`: 后台页面渲染入口
- `dashboard.go`: 首页概览数据
- `comment.go`: 评论列表、同步、回复、忽略
- `message.go`: 私信列表、同步、回复、忽略、未读数
- `interaction.go`: 视频互动、粉丝互动、互动统计
- `trend.go`: 热门标签、标签详情、视频排行、趋势统计
- `llm.go`: LLM 对话、提供者列表、日志、统计、测试
- `reply_workspace.go`: 评论/私信共享回复编辑器接口
- `settings.go`: 应用设置、B 站配置、LLM 提供者配置
- `observability.go`: 轮询状态与运行观测接口

### 4. 服务层

- `internal/service/`

当前主要服务：

- `dashboard_service.go`: 首页概览统计
- `comment_service.go`: 评论同步、状态管理、回复
- `message_service.go`: 私信同步、手动/AI 回复、关注私信自动回复
- `interaction_service.go`: 点赞、投币、收藏、三连和粉丝视频互动
- `trend_service.go`: 热门标签、排行与历史趋势
- `reply_workspace_service.go`: 评论/私信共享 AI 回复工作区
- `llm_service.go`: LLM 统一调用与日志能力
- `app_settings_service.go`: 应用运行配置持久化与加载
- `auth_service.go`: 管理员认证与会话

服务层是项目的核心。handler 只做 HTTP 绑定和返回，业务判断基本都在这里。

### 5. 数据访问层

- `internal/repository/repository.go`

目前仓储集中在一个文件里，覆盖以下模型：

- 评论
- 私信
- 互动记录
- 热门标签排行
- LLM 日志
- 应用设置
- LLM 提供者
- 管理员用户与会话
- 粉丝自动回复记录
- 回复模板 / 示例 / 草稿

### 6. 模型层

- `internal/model/model.go`

数据库表在启动时通过 `AutoMigrate` 自动维护，当前会迁移这些核心模型：

- `User`
- `AdminUser`
- `AdminSession`
- `FanAutoReplyRecord`
- `Comment`
- `Message`
- `Interaction`
- `TagRanking`
- `LLMChatLog`
- `Setting`
- `Task`
- `LLMProvider`
- `ReplyTemplate`
- `ReplyExample`
- `ReplyDraft`

### 7. 运行时与轮询

- `internal/runtime/`
- `internal/polling/`

运行时 `Store` 负责持有当前有效的：

- B 站客户端
- LLM 管理器

轮询任务由 `main.go` 启动时注册，当前包括：

- `trend-taginfo-sync`: 热门标签热度同步
- `video-comments-sync`: 最近视频评论同步
- `private-messages-sync`: 私信同步
- `fans-weekly-interact`: 粉丝近期视频自动互动
- `fans-follow-auto-reply`: 新关注用户私信自动回复

### 8. 前端层

- `web/templates/`
- `web/static/`
- `web/embed.go`

特点：

- 前端页面使用 Go template 渲染
- 静态资源与模板通过 `embed.FS` 打包进二进制
- 统一后台前缀是 `/admin`
- 页面与 API 同域部署，不依赖独立前端工程

模板目录按页面域拆分：

- `auth/`: 登录、改密
- `comment/`: 评论工作台
- `message/`: 私信工作台
- `like/`: 互动管理
- `trend/`: 热度分析
- `llm/`: LLM 日志与能力页
- `settings/`: 系统设置与 B 站登录配置
- `layout/`: 基础布局

### 9. 外部能力封装

- `pkg/bilibili/`: 对 biligo 的项目内封装
- `pkg/llm/`: 多模型统一接口与 provider 实现

当前 Go 模块依赖里已经接入的 LLM 方向包括：

- OpenAI
- Anthropic
- Gemini
- Ollama
- 兼容 OpenAI 协议的其它模型服务

## 路由结构

所有页面和接口都放在 `/admin` 下：

- 页面入口：`/admin/*`
- 接口入口：`/admin/api/*`
- 静态资源：`/admin/static/*`

根路径 `/` 会直接重定向到 `/admin/login`。

鉴权策略：

- `/admin/login` 和 `/admin/api/auth/login` 是匿名可访问的
- 其它 `/admin` 页面和接口都需要管理员登录
- 首次启动会自动创建默认管理员，并强制首次改密

## 当前核心业务能力

### 评论工作台

- 拉取最近视频评论并入库
- 评论列表、状态筛选、忽略处理
- AI 生成回复与手动回复
- 共享回复编辑栏与 LLM 调用日志联动

### 私信工作台

- 拉取会话与聊天记录并入库
- 按会话组织消息流
- AI 回复、手动回复、忽略
- 未读消息统计
- 关注后自动私信回复

### 互动管理

- 点赞、投币、收藏、三连
- 粉丝列表与粉丝投稿选择
- 批量互动和自动互动任务

### 热度分析

- 热门标签列表
- 标签详情视频列表
- 视频排行
- 历史与最新榜单接口

### LLM 能力与日志

- 统一 provider 管理
- 默认模型切换
- 对话测试
- 调用日志和统计

### 设置与认证

- 管理员登录、登出、改密
- B 站 Cookie 保存与二维码登录
- LLM 提供者的增删改查
- 应用运行设置保存

## 目录结构

```text
bilibili-up-admin/
├── cmd/server/                 # 应用入口与依赖装配
├── config/                     # 内嵌默认配置与加载逻辑
├── internal/
│   ├── handler/                # HTTP 处理层
│   ├── model/                  # GORM 模型
│   ├── polling/                # 后台轮询任务框架
│   ├── repository/             # 数据访问层
│   ├── runtime/                # 运行时依赖容器
│   └── service/                # 业务服务层
├── web/
│   ├── templates/              # Go 模板页面
│   ├── static/                 # JS/CSS/图片等资源
│   └── embed.go                # 嵌入模板与静态资源
├── data/                       # 默认运行数据目录
├── go.mod
└── README.md
```

## 本地运行

### 环境要求

- Go 1.24+

项目默认使用 SQLite，不需要额外安装 MySQL 或 Redis 才能启动。

### 安装依赖

```bash
go mod tidy
```

### 启动

```bash
go run cmd/server/main.go
```

启动后访问：

- `http://localhost:8080/admin/login`

首次登录默认账号：

- 用户名：`admin`
- 密码：`admin123456`

首次登录后会被强制要求修改密码。

### 构建

```bash
go build -o bili-admin cmd/server/main.go
```

## 默认配置行为

默认配置来自 `config/config.yaml`，当前默认值很简单：

```yaml
data_dir: data

server:
  port: 8080
  mode: debug

database:
  driver: sqlite
  path: bilibili-up-admin.db
```

实际生效后的 SQLite 文件路径会被归一到数据目录下，也就是默认：

- `data/bilibili-up-admin.db`

如果运行目录下存在外部 `config/config.yaml`，程序会优先读外部文件；否则读内嵌配置。

## 需要补充的运行配置

程序能启动，不代表业务就能工作。要真正使用评论、私信、互动和 AI 回复，需要在后台设置页补齐：

- B 站登录 Cookie / 二维码登录信息
- LLM provider 与默认模型
- 应用级交互策略配置

这些配置加载后会在运行时构造：

- B 站客户端
- LLM 管理器

## 关键接口分组

这里只列当前最重要的 API 分组，不展开每个参数。

### 认证

- `POST /admin/api/auth/login`
- `POST /admin/api/auth/logout`
- `GET /admin/api/auth/me`
- `POST /admin/api/auth/change-password`

### 评论

- `GET /admin/api/comments`
- `POST /admin/api/comments/sync`
- `GET /admin/api/comments/my-videos`
- `POST /admin/api/comments/:id/ai-reply`
- `POST /admin/api/comments/:id/reply`
- `POST /admin/api/comments/:id/ignore`

### 私信

- `GET /admin/api/messages`
- `POST /admin/api/messages/sync`
- `GET /admin/api/messages/unread`
- `POST /admin/api/messages/:id/ai-reply`
- `POST /admin/api/messages/:id/reply`
- `POST /admin/api/messages/:id/ignore`

### 共享回复工作区

- `GET /admin/api/reply-workspace`
- `POST /admin/api/reply-workspace/draft/generate`
- `POST /admin/api/reply-workspace/draft/send`
- `GET /admin/api/reply-workspace/templates`

### 互动

- `GET /admin/api/interactions`
- `GET /admin/api/interactions/stats`
- `GET /admin/api/fans/list`
- `GET /admin/api/fans/:id/videos`
- `POST /admin/api/videos/:id/like`
- `POST /admin/api/videos/:id/coin`
- `POST /admin/api/videos/:id/favorite`
- `POST /admin/api/videos/:id/triple`
- `POST /admin/api/videos/batch-interact`
- `POST /admin/api/fans/interact`

### 热度

- `GET /admin/api/trends/tags`
- `GET /admin/api/trends/tags/:name`
- `GET /admin/api/trends/videos`
- `GET /admin/api/trends/historical`
- `GET /admin/api/trends/latest`
- `GET /admin/api/trends/search`
- `POST /admin/api/trends/sync`
- `GET /admin/api/trends/stats`

### LLM

- `POST /admin/api/llm/chat`
- `GET /admin/api/llm/providers`
- `POST /admin/api/llm/default`
- `GET /admin/api/llm/test/:provider`
- `GET /admin/api/llm/stats`
- `GET /admin/api/llm/logs`

### 设置

- `GET /admin/api/settings/app`
- `PUT /admin/api/settings/app`
- `GET /admin/api/settings/bilibili`
- `PUT /admin/api/settings/bilibili/cookie`
- `POST /admin/api/settings/bilibili/qrcode`
- `GET /admin/api/settings/bilibili/qrcode/poll`
- `GET /admin/api/settings/llm/providers`
- `POST /admin/api/settings/llm/providers`
- `PUT /admin/api/settings/llm/providers/:name`
- `DELETE /admin/api/settings/llm/providers/:name`

## 开发说明

### 为什么前端没有单独工程

因为当前架构就是一个后端主导的管理台：

- 模板由 Go 渲染
- 页面和接口一体部署
- 静态资源跟随二进制发布

这让部署非常简单，代价是前端工程化边界没有独立 SPA 那么强。

### 为什么运行配置不直接写死在文件里

因为这个项目在启动后会把设置持久化到数据库，并基于设置动态重建：

- B 站客户端
- LLM provider 管理器

所以 `config/config.yaml` 更偏向基础启动配置，而不是完整业务配置中心。

### 当前架构上的一个明显特点

`repository.go` 目前是集中式仓储文件，适合快速迭代；如果后续业务继续扩展，按领域拆分仓储文件会更容易维护。

## 适合的后续演进方向

1. 把 repository 从单文件拆成按领域分文件。
2. 为 polling、settings、reply workspace 增加更完整的集成测试。
3. 继续收缩 reply template / example / draft 的历史兼容逻辑，和现有浮动编辑栏保持一致。

### 通义千问

```yaml
llm:
  provider: "qwen"
  api_key: "sk-xxx"
  model: "qwen-turbo"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
```

### 本地部署 (Ollama)

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434/v1"
  model: "qwen2"
```

## 🔧 扩展开发

### 添加新的大模型提供者

1. 在 `pkg/llm/` 下创建新文件，如 `new_provider.go`
2. 实现 `Provider` 接口：

```go
type NewProvider struct {
    // ...
}

func (p *NewProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
    // 实现
}

func (p *NewProvider) ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error) {
    // 实现
}

func (p *NewProvider) Stream(ctx context.Context, messages []Message, callback StreamCallback) error {
    // 实现
}

func (p *NewProvider) StreamWithSystem(ctx context.Context, systemPrompt string, messages []Message, callback StreamCallback) error {
    // 实现
}

func (p *NewProvider) Name() string {
    return "new_provider"
}

func (p *NewProvider) Models() []string {
    return []string{"model-1", "model-2"}
}
```

3. 在 `factory.go` 中注册：

```go
case ProviderNewProvider:
    return NewNewProvider(cfg)
```

## ⚠️ 注意事项

1. **Cookie安全**: SESSDATA等凭证请妥善保管，不要泄露
2. **请求频率**: B站API有频率限制，请合理控制请求频率
3. **AI回复**: AI生成的内容请人工审核后再发送
4. **法律合规**: 请遵守B站用户协议和相关法律法规

## 📄 License

GNU Affero General Public License v3.0

## 🙏 致谢

- [biligo](https://github.com/guohuiyuan/biligo) - B站API库
- [Gin](https://github.com/gin-gonic/gin) - Web框架
- [GORM](https://gorm.io/) - ORM库
