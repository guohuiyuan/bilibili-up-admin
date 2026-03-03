# B站UP主运营管理平台

一个功能完善的B站UP主运营辅助工具，支持AI智能回复评论私信、粉丝互动管理、热度分析等功能。

## 🌟 功能特性

### 1. 评论管理
- 同步视频评论
- AI智能回复（支持多种大模型）
- 手动回复
- 批量AI回复
- 评论状态管理

### 2. 私信管理
- 同步私信消息
- AI智能回复
- 手动回复
- 未读消息统计

### 3. 互动管理
- 视频点赞
- 视频投币
- 三连操作
- 批量互动
- 粉丝视频互动

### 4. 热度分析
- 热门标签排行
- 视频排行榜
- 标签搜索
- 分区排行

### 5. 大模型支持
- OpenAI (GPT-4, GPT-3.5)
- Claude
- DeepSeek
- 通义千问 (Qwen)
- Moonshot (Kimi)
- 智谱AI (GLM)
- Ollama (本地部署)

## 📦 技术栈

- **后端**: Go 1.22+, Gin, GORM
- **前端**: Go Template, Tailwind CSS
- **数据库**: MySQL
- **缓存**: Redis (可选)
- **B站API**: [biligo](https://github.com/guohuiyuan/biligo)

## 🚀 快速开始

### 1. 环境要求

- Go 1.22+
- MySQL 5.7+
- Redis (可选)

### 2. 安装依赖

```bash
cd bilibili-up-admin
go mod tidy
```

### 3. 配置文件

复制配置文件并修改：

```bash
cp config/config.yaml config/config.local.yaml
```

编辑 `config/config.yaml`，配置以下关键信息：

```yaml
# 数据库配置
database:
  host: localhost
  port: 3306
  username: root
  password: your_password
  dbname: bili_admin

# B站账号配置
bilibili:
  sess_data: "your_sess_data"
  bili_jct: "your_bili_jct"
  user_id: your_user_id

# 大模型配置
llm:
  provider: "deepseek"
  api_key: "your_api_key"
  model: "deepseek-chat"
```

### 4. 获取B站登录凭证

1. 登录B站网页版
2. 打开浏览器开发者工具 (F12)
3. 进入 Application/存储 -> Cookies -> https://www.bilibili.com
4. 复制以下值：
   - `SESSDATA` → `sess_data`
   - `bili_jct` → `bili_jct`
   - `DedeUserID` → `user_id`

### 5. 初始化数据库

```bash
# 创建数据库
mysql -u root -p -e "CREATE DATABASE bili_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 运行迁移（首次运行会自动创建表）
go run cmd/server/main.go
```

### 6. 启动服务

```bash
# 开发模式
go run cmd/server/main.go

# 或编译后运行
go build -o bili-admin cmd/server/main.go
./bili-admin
```

访问 http://localhost:8080 即可使用。

## 📁 项目结构

```
bilibili-up-admin/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── handler/                 # HTTP处理器
│   │   ├── comment.go
│   │   ├── message.go
│   │   ├── interaction.go
│   │   ├── trend.go
│   │   ├── llm.go
│   │   └── page.go
│   ├── service/                 # 业务逻辑层
│   │   ├── comment_service.go
│   │   ├── message_service.go
│   │   ├── interaction_service.go
│   │   ├── trend_service.go
│   │   └── llm_service.go
│   ├── repository/              # 数据访问层
│   │   └── repository.go
│   ├── model/                   # 数据模型
│   │   └── model.go
│   └── middleware/              # 中间件
├── pkg/
│   ├── bilibili/                # Bilibili SDK封装
│   │   ├── client.go
│   │   ├── comment.go
│   │   ├── message.go
│   │   ├── video.go
│   │   └── trend.go
│   ├── llm/                     # 大模型适配器
│   │   ├── provider.go          # 接口定义
│   │   ├── factory.go           # 工厂模式
│   │   ├── openai.go            # OpenAI实现
│   │   └── providers.go         # 其他实现
│   └── queue/                   # 任务队列
├── web/
│   ├── templates/               # HTML模板
│   │   ├── layout/
│   │   ├── comment/
│   │   ├── message/
│   │   ├── like/
│   │   └── trend/
│   └── static/                  # 静态资源
│       ├── css/
│       └── js/
├── config/
│   ├── config.yaml              # 配置文件
│   └── config.go                # 配置加载
├── migrations/                  # 数据库迁移
├── go.mod
└── go.sum
```

## 🔌 API 接口

### 评论相关

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/comments | 获取评论列表 |
| POST | /api/comments/sync | 同步评论 |
| POST | /api/comments/:id/ai-reply | AI回复 |
| POST | /api/comments/:id/reply | 手动回复 |
| POST | /api/comments/:id/ignore | 忽略评论 |
| POST | /api/comments/batch-ai-reply | 批量AI回复 |

### 私信相关

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/messages | 获取私信列表 |
| POST | /api/messages/sync | 同步私信 |
| GET | /api/messages/unread | 未读数量 |
| POST | /api/messages/:id/ai-reply | AI回复 |
| POST | /api/messages/:id/reply | 手动回复 |

### 互动相关

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/videos/:id/like | 点赞视频 |
| POST | /api/videos/:id/coin | 投币视频 |
| POST | /api/videos/:id/triple | 三连 |
| POST | /api/videos/batch-interact | 批量互动 |
| POST | /api/fans/interact | 粉丝互动 |

### 热度相关

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/trends/tags | 热门标签 |
| GET | /api/trends/tags/:name | 标签详情 |
| GET | /api/trends/videos | 视频排行 |
| GET | /api/trends/search | 标签搜索 |
| POST | /api/trends/sync | 同步数据 |

### 大模型相关

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/llm/chat | 对话 |
| GET | /api/llm/providers | 提供者列表 |
| POST | /api/llm/default | 设置默认 |
| GET | /api/llm/test/:provider | 测试连接 |

## 🤖 大模型配置

### DeepSeek (推荐)

```yaml
llm:
  provider: "deepseek"
  api_key: "sk-xxx"
  model: "deepseek-chat"
```

### OpenAI

```yaml
llm:
  provider: "openai"
  api_key: "sk-xxx"
  model: "gpt-4o-mini"
```

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

MIT License

## 🙏 致谢

- [biligo](https://github.com/guohuiyuan/biligo) - B站API库
- [Gin](https://github.com/gin-gonic/gin) - Web框架
- [GORM](https://gorm.io/) - ORM库
