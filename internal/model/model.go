package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint           `gorm:"primarykey;column:id" json:"id"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`
}

type User struct {
	BaseModel
	BiliUID     int64      `gorm:"column:bili_uid;uniqueIndex;not null" json:"bili_uid"`
	BiliName    string     `gorm:"column:bili_name;size:100" json:"bili_name"`
	BiliFace    string     `gorm:"column:bili_face;size:500" json:"bili_face"`
	SESSData    string     `gorm:"column:sess_data;size:500" json:"-"`
	BiliJct     string     `gorm:"column:bili_jct;size:100" json:"-"`
	IsLoggedIn  bool       `gorm:"column:is_logged_in;default:false" json:"is_logged_in"`
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
}

func (User) TableName() string { return "users" }

type Comment struct {
	BaseModel
	CommentID    int64      `gorm:"column:comment_id;uniqueIndex;not null" json:"comment_id"`
	VideoBVID    string     `gorm:"column:video_bvid;size:20;index;not null" json:"video_bvid"`
	VideoAID     int64      `gorm:"column:video_aid" json:"video_aid"`
	Content      string     `gorm:"column:content;type:text;not null" json:"content"`
	AuthorName   string     `gorm:"column:author_name;size:100" json:"author_name"`
	AuthorFace   string     `gorm:"column:author_face;size:500" json:"author_face"`
	AuthorMid    int64      `gorm:"column:author_mid" json:"author_mid"`
	LikeCount    int        `gorm:"column:like_count;default:0" json:"like_count"`
	ReplyStatus  int        `gorm:"column:reply_status;default:0;index" json:"reply_status"` // 0=未回复, 1=已回复, 2=忽略
	IsAIReply    bool       `gorm:"column:is_ai_reply;default:false" json:"is_ai_reply"`
	ReplyContent string     `gorm:"column:reply_content;type:text" json:"reply_content"`
	CommentTime  *time.Time `gorm:"column:comment_time" json:"comment_time"`
}

func (Comment) TableName() string { return "comments" }

type Message struct {
	BaseModel
	MessageID        int64      `gorm:"column:message_id;uniqueIndex;not null" json:"message_id"`
	SenderID         int64      `gorm:"column:sender_uid;index;not null" json:"sender_uid"`
	SenderName       string     `gorm:"column:sender_name;size:100" json:"sender_name"`
	SenderFace       string     `gorm:"column:sender_face;size:500" json:"sender_face"`
	ConversationUID  int64      `gorm:"column:conversation_uid;index" json:"conversation_uid"`
	ConversationName string     `gorm:"column:conversation_name;size:100" json:"conversation_name"`
	ConversationFace string     `gorm:"column:conversation_face;size:500" json:"conversation_face"`
	IsFromSelf       bool       `gorm:"column:is_from_self;default:false;index" json:"is_from_self"`
	Content          string     `gorm:"column:content;type:text;not null" json:"content"`
	MsgType          int        `gorm:"column:msg_type;default:1" json:"msg_type"`
	ReplyStatus      int        `gorm:"column:reply_status;default:0;index" json:"reply_status"` // 0=未回复, 1=已回复, 2=忽略
	IsAIReply        bool       `gorm:"column:is_ai_reply;default:false" json:"is_ai_reply"`
	ReplyContent     string     `gorm:"column:reply_content;type:text" json:"reply_content"`
	MessageTime      *time.Time `gorm:"column:message_time" json:"message_time"`
}

func (Message) TableName() string { return "messages" }

type Interaction struct {
	BaseModel
	VideoBVID    string     `gorm:"column:video_bvid;size:20;index" json:"video_bvid"`
	VideoAID     int64      `gorm:"column:video_aid" json:"video_aid"`
	VideoTitle   string     `gorm:"column:video_title;size:500" json:"video_title"`
	VideoOwnerID int64      `gorm:"column:video_owner_id" json:"video_owner_id"`
	VideoOwner   string     `gorm:"column:video_owner;size:100" json:"video_owner"`
	ActionType   string     `gorm:"column:action_type;size:20;index" json:"action_type"` // like, coin, favorite, triple
	CoinCount    int        `gorm:"column:coin_count;default:0" json:"coin_count"`
	Success      bool       `gorm:"column:success;default:true" json:"success"`
	ErrorMessage string     `gorm:"column:error_message;type:text" json:"error_message"`
	ActionTime   *time.Time `gorm:"column:action_time" json:"action_time"`
}

func (Interaction) TableName() string { return "interactions" }

type TagRanking struct {
	BaseModel
	TagName     string    `gorm:"column:tag_name;size:100;index" json:"tag_name"`
	TagID       int64     `gorm:"column:tag_id" json:"tag_id"`
	HotValue    int64     `gorm:"column:hot_value;default:0" json:"hot_value"`
	UseCount    int64     `gorm:"column:use_count;default:0" json:"use_count"`
	FollowCount int64     `gorm:"column:follow_count;default:0" json:"follow_count"`
	VideoCount  int       `gorm:"column:video_count;default:0" json:"video_count"`
	Rank        int       `gorm:"column:rank;default:0" json:"rank"`
	Category    string    `gorm:"column:category;size:50" json:"category"`
	RecordDate  time.Time `gorm:"column:record_date" json:"record_date"`
}

func (TagRanking) TableName() string { return "tag_rankings" }

type LLMChatLog struct {
	BaseModel
	Provider          string `gorm:"column:provider;size:50;index" json:"provider"`
	Model             string `gorm:"column:model;size:100" json:"model"`
	LogType           string `gorm:"column:log_type;size:30;index" json:"log_type"`
	InputType         string `gorm:"column:input_type;size:20;index" json:"input_type"` // comment/message
	InputID           int64  `gorm:"column:input_id;index" json:"input_id"`
	ConversationKey   string `gorm:"column:conversation_key;size:160;index" json:"conversation_key"`
	ConversationTitle string `gorm:"column:conversation_title;size:160" json:"conversation_title"`
	InputContent      string `gorm:"column:input_content;type:text" json:"input_content"`
	SystemPrompt      string `gorm:"column:system_prompt;type:text" json:"system_prompt"`
	RequestMessages   string `gorm:"column:request_messages;type:text" json:"request_messages"`
	OutputContent     string `gorm:"column:output_content;type:text" json:"output_content"`
	PromptTokens      int    `gorm:"column:prompt_tokens" json:"prompt_tokens"`
	OutputTokens      int    `gorm:"column:output_tokens" json:"output_tokens"`
	TotalTokens       int    `gorm:"column:total_tokens" json:"total_tokens"`
	Success           bool   `gorm:"column:success;default:true" json:"success"`
	ErrorMessage      string `gorm:"column:error_message;type:text" json:"error_message"`
	Duration          int64  `gorm:"column:duration" json:"duration"`
}

func (LLMChatLog) TableName() string { return "llm_chat_logs" }

type Setting struct {
	BaseModel
	Key   string `gorm:"column:key;uniqueIndex;size:100;not null" json:"key"`
	Value string `gorm:"column:value;type:text" json:"value"`
}

func (Setting) TableName() string { return "settings" }

type Task struct {
	BaseModel
	TaskType   string     `gorm:"column:task_type;size:50;index;not null" json:"task_type"`
	TargetID   int64      `gorm:"column:target_id;index" json:"target_id"`
	TargetData string     `gorm:"column:target_data;type:text" json:"target_data"`
	Status     int        `gorm:"column:status;default:0;index" json:"status"`
	Result     string     `gorm:"column:result;type:text" json:"result"`
	RetryCount int        `gorm:"column:retry_count;default:0" json:"retry_count"`
	MaxRetry   int        `gorm:"column:max_retry;default:3" json:"max_retry"`
	RunAt      *time.Time `gorm:"column:run_at" json:"run_at"`
	StartedAt  *time.Time `gorm:"column:started_at" json:"started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at" json:"finished_at"`
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

type LLMProvider struct {
	BaseModel
	Name        string  `gorm:"column:name;uniqueIndex;size:100;not null" json:"name"`
	Provider    string  `gorm:"column:provider;size:50;not null" json:"provider"`
	APIKey      string  `gorm:"column:api_key;size:255" json:"api_key"`
	BaseURL     string  `gorm:"column:base_url;size:255" json:"base_url"`
	Model       string  `gorm:"column:model;size:100" json:"model"`
	MaxTokens   int     `gorm:"column:max_tokens;default:1000" json:"max_tokens"`
	Temperature float64 `gorm:"column:temperature;type:decimal(5,2);default:0.7" json:"temperature"`
	Enabled     bool    `gorm:"column:enabled;default:true" json:"enabled"`
}

func (LLMProvider) TableName() string { return "llm_providers" }

type AdminUser struct {
	BaseModel
	Username           string     `gorm:"column:username;uniqueIndex;size:64;not null" json:"username"`
	PasswordHash       string     `gorm:"column:password_hash;size:255;not null" json:"-"`
	MustChangePassword bool       `gorm:"column:must_change_password;default:true" json:"must_change_password"`
	LastLoginAt        *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
}

func (AdminUser) TableName() string { return "admin_users" }

type AdminSession struct {
	BaseModel
	AdminUserID uint       `gorm:"column:admin_user_id;index;not null" json:"admin_user_id"`
	TokenHash   string     `gorm:"column:token_hash;uniqueIndex;size:128;not null" json:"-"`
	ExpiresAt   time.Time  `gorm:"column:expires_at;index;not null" json:"expires_at"`
	LastSeenAt  *time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
}

func (AdminSession) TableName() string { return "admin_sessions" }

type FanAutoReplyRecord struct {
	BaseModel
	FanUID      int64      `gorm:"column:fan_uid;uniqueIndex;not null" json:"fan_uid"`
	FanName     string     `gorm:"column:fan_name;size:100" json:"fan_name"`
	LastSeenAt  *time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
	Replied     bool       `gorm:"column:replied;default:false;index" json:"replied"`
	RepliedAt   *time.Time `gorm:"column:replied_at" json:"replied_at"`
	LastError   string     `gorm:"column:last_error;type:text" json:"last_error"`
	ReplyDigest string     `gorm:"column:reply_digest;size:64" json:"reply_digest"`
}

func (FanAutoReplyRecord) TableName() string { return "fan_auto_reply_records" }

type ReplyTemplate struct {
	BaseModel
	Channel    string     `gorm:"column:channel;size:20;index;not null" json:"channel"`
	Title      string     `gorm:"column:title;size:120;not null" json:"title"`
	Content    string     `gorm:"column:content;type:text;not null" json:"content"`
	Scene      string     `gorm:"column:scene;size:120" json:"scene"`
	UsageCount int        `gorm:"column:usage_count;default:0" json:"usage_count"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
}

func (ReplyTemplate) TableName() string { return "reply_templates" }

type ReplyExample struct {
	BaseModel
	Channel      string     `gorm:"column:channel;size:20;index;not null" json:"channel"`
	Title        string     `gorm:"column:title;size:120;not null" json:"title"`
	UserInput    string     `gorm:"column:user_input;type:text;not null" json:"user_input"`
	ReplyContent string     `gorm:"column:reply_content;type:text;not null" json:"reply_content"`
	Notes        string     `gorm:"column:notes;type:text" json:"notes"`
	SourceType   string     `gorm:"column:source_type;size:20" json:"source_type"`
	SourceID     int64      `gorm:"column:source_id;index" json:"source_id"`
	QualityScore int        `gorm:"column:quality_score;default:80" json:"quality_score"`
	UsageCount   int        `gorm:"column:usage_count;default:0" json:"usage_count"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
}

func (ReplyExample) TableName() string { return "reply_examples" }

type ReplyDraft struct {
	BaseModel
	Channel          string     `gorm:"column:channel;size:20;uniqueIndex:idx_reply_target;priority:1;not null" json:"channel"`
	TargetID         int64      `gorm:"column:target_id;uniqueIndex:idx_reply_target;priority:2;not null" json:"target_id"`
	Content          string     `gorm:"column:content;type:text" json:"content"`
	Status           string     `gorm:"column:status;size:20;default:draft" json:"status"`
	SourceType       string     `gorm:"column:source_type;size:20" json:"source_type"`
	TemplateSnapshot string     `gorm:"column:template_snapshot;type:text" json:"template_snapshot"`
	ExtraInstruction string     `gorm:"column:extra_instruction;type:text" json:"extra_instruction"`
	ModelProvider    string     `gorm:"column:model_provider;size:100" json:"model_provider"`
	ModelName        string     `gorm:"column:model_name;size:120" json:"model_name"`
	GeneratedAt      *time.Time `gorm:"column:generated_at" json:"generated_at"`
	SentAt           *time.Time `gorm:"column:sent_at" json:"sent_at"`
}

func (ReplyDraft) TableName() string { return "reply_drafts" }

// MessageAutoRule 私信自动回复规则
// match_type: "exact" 固定匹配, "regex" 正则匹配, "contains" 包含匹配
type MessageAutoRule struct {
	BaseModel
	Name       string     `gorm:"column:name;size:120;not null" json:"name"`
	MatchType  string     `gorm:"column:match_type;size:20;not null;default:exact" json:"match_type"`
	Pattern    string     `gorm:"column:pattern;type:text;not null" json:"pattern"`
	Reply      string     `gorm:"column:reply;type:text;not null" json:"reply"`
	Enabled    bool       `gorm:"column:enabled;default:true;index" json:"enabled"`
	Priority   int        `gorm:"column:priority;default:0;index" json:"priority"`
	MatchCount int        `gorm:"column:match_count;default:0" json:"match_count"`
	LastMatchAt *time.Time `gorm:"column:last_match_at" json:"last_match_at"`
}

func (MessageAutoRule) TableName() string { return "message_auto_rules" }
