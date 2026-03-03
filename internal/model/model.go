package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	BaseModel
	BiliUID     int64      `gorm:"uniqueIndex;not null" json:"bili_uid"`
	BiliName    string     `gorm:"size:100" json:"bili_name"`
	BiliFace    string     `gorm:"size:500" json:"bili_face"`
	SESSData    string     `gorm:"size:500" json:"-"`
	BiliJct     string     `gorm:"size:100" json:"-"`
	IsLoggedIn  bool       `gorm:"default:false" json:"is_logged_in"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

func (User) TableName() string { return "users" }

type Comment struct {
	BaseModel
	CommentID    int64     `gorm:"uniqueIndex;not null" json:"comment_id"`
	VideoBVID    string    `gorm:"size:20;index;not null" json:"video_bvid"`
	VideoAID     int64     `json:"video_aid"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	AuthorID     int64     `gorm:"index" json:"author_id"`
	AuthorName   string    `gorm:"size:100" json:"author_name"`
	ReplyID      int64     `gorm:"default:0" json:"reply_id"`
	ReplyStatus  int       `gorm:"default:0" json:"reply_status"`
	ReplyContent string    `gorm:"type:text" json:"reply_content"`
	IsAIReply    bool      `gorm:"default:false" json:"is_ai_reply"`
	CommentTime  time.Time `json:"comment_time"`
}

func (Comment) TableName() string { return "comments" }

type Message struct {
	BaseModel
	MessageID    int64     `gorm:"uniqueIndex;not null" json:"message_id"`
	SenderID     int64     `gorm:"index;not null" json:"sender_id"`
	SenderName   string    `gorm:"size:100" json:"sender_name"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	ReplyStatus  int       `gorm:"default:0" json:"reply_status"`
	ReplyContent string    `gorm:"type:text" json:"reply_content"`
	IsAIReply    bool      `gorm:"default:false" json:"is_ai_reply"`
	IsRead       bool      `gorm:"default:false" json:"is_read"`
	MessageTime  time.Time `json:"message_time"`
}

func (Message) TableName() string { return "messages" }

type Interaction struct {
	BaseModel
	VideoBVID    string    `gorm:"size:20;index;not null" json:"video_bvid"`
	VideoTitle   string    `gorm:"size:500" json:"video_title"`
	VideoOwnerID int64     `gorm:"index" json:"video_owner_id"`
	VideoOwner   string    `gorm:"size:100" json:"video_owner"`
	ActionType   string    `gorm:"size:20;index;not null" json:"action_type"`
	CoinCount    int       `gorm:"default:0" json:"coin_count"`
	Success      bool      `gorm:"default:true" json:"success"`
	ErrorMessage string    `gorm:"type:text" json:"error_message"`
	ActionTime   time.Time `json:"action_time"`
}

func (Interaction) TableName() string { return "interactions" }

type TagRanking struct {
	BaseModel
	TagName    string    `gorm:"size:100;uniqueIndex;not null" json:"tag_name"`
	TagID      int64     `json:"tag_id"`
	HotValue   int64     `json:"hot_value"`
	VideoCount int       `json:"video_count"`
	Rank       int       `json:"rank"`
	Category   string    `gorm:"size:50" json:"category"`
	RecordDate time.Time `gorm:"index" json:"record_date"`
}

func (TagRanking) TableName() string { return "tag_rankings" }

type LLMChatLog struct {
	BaseModel
	Provider      string `gorm:"size:50;index" json:"provider"`
	Model         string `gorm:"size:100" json:"model"`
	InputType     string `gorm:"size:20;index" json:"input_type"`
	InputID       int64  `gorm:"index" json:"input_id"`
	InputContent  string `gorm:"type:text" json:"input_content"`
	OutputContent string `gorm:"type:text" json:"output_content"`
	PromptTokens  int    `json:"prompt_tokens"`
	OutputTokens  int    `json:"output_tokens"`
	TotalTokens   int    `json:"total_tokens"`
	Success       bool   `gorm:"default:true" json:"success"`
	ErrorMessage  string `gorm:"type:text" json:"error_message"`
	Duration      int64  `json:"duration"`
}

func (LLMChatLog) TableName() string { return "llm_chat_logs" }

type Setting struct {
	BaseModel
	Key   string `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

func (Setting) TableName() string { return "settings" }

type Task struct {
	BaseModel
	TaskType   string     `gorm:"size:50;index;not null" json:"task_type"`
	TargetID   int64      `gorm:"index" json:"target_id"`
	TargetData string     `gorm:"type:text" json:"target_data"`
	Status     int        `gorm:"default:0;index" json:"status"`
	Result     string     `gorm:"type:text" json:"result"`
	RetryCount int        `gorm:"default:0" json:"retry_count"`
	MaxRetry   int        `gorm:"default:3" json:"max_retry"`
	RunAt      *time.Time `json:"run_at"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func (Task) TableName() string { return "tasks" }

const (
	TaskStatusPending = 0
	TaskStatusRunning = 1
	TaskStatusSuccess = 2
	TaskStatusFailed  = 3
)

const (
	TaskTypeReplyComment = "reply_comment"
	TaskTypeReplyMessage = "reply_message"
	TaskTypeLikeVideo    = "like_video"
	TaskTypeCoinVideo    = "coin_video"
	TaskTypeTripleVideo  = "triple_video"
)
