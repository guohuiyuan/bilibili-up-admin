package repository

import (
	"context"
	"encoding/json"
	"time"

	"bilibili-up-admin/internal/model"

	"gorm.io/gorm"
)

// CommentRepository 评论仓库
type CommentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建评论仓库
func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建评论记录
func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// Update 更新评论记录
func (r *CommentRepository) Update(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

// GetByID 根据ID获取
func (r *CommentRepository) GetByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.WithContext(ctx).First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetByCommentID 根据B站评论ID获取
func (r *CommentRepository) GetByCommentID(ctx context.Context, commentID int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.WithContext(ctx).Where("comment_id = ?", commentID).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// List 获取评论列表
func (r *CommentRepository) List(ctx context.Context, videoBVID string, replyStatus int, page, pageSize int) ([]model.Comment, int64, error) {
	var comments []model.Comment
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Comment{})
	if videoBVID != "" {
		query = query.Where("video_bvid = ?", videoBVID)
	}
	if replyStatus >= 0 {
		query = query.Where("reply_status = ?", replyStatus)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Order("comment_time DESC").Offset(offset).Limit(pageSize).Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetUnreplied 获取未回复评论
func (r *CommentRepository) GetUnreplied(ctx context.Context, limit int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.WithContext(ctx).
		Where("reply_status = ?", 0).
		Order("comment_time DESC").
		Limit(limit).
		Find(&comments).Error
	return comments, err
}

// UpdateReplyStatus 更新回复状态
func (r *CommentRepository) UpdateReplyStatus(ctx context.Context, commentID int64, status int, replyContent string) error {
	return r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("comment_id = ?", commentID).
		Updates(map[string]interface{}{
			"reply_status":  status,
			"reply_content": replyContent,
		}).Error
}

// MessageRepository 私信仓库
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 创建私信仓库
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建私信记录
func (r *MessageRepository) Create(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetByMessageID 根据B站消息ID获取
func (r *MessageRepository) GetByMessageID(ctx context.Context, messageID int64) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).Where("message_id = ?", messageID).First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// List 获取私信列表
func (r *MessageRepository) List(ctx context.Context, senderID int64, replyStatus int, page, pageSize int) ([]model.Message, int64, error) {
	var messages []model.Message
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Message{})
	if senderID > 0 {
		query = query.Where("sender_id = ?", senderID)
	}
	if replyStatus >= 0 {
		query = query.Where("reply_status = ?", replyStatus)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Order("message_time DESC").Offset(offset).Limit(pageSize).Find(&messages).Error
	if err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// GetUnreplied 获取未回复私信
func (r *MessageRepository) GetUnreplied(ctx context.Context, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).
		Where("reply_status = ?", 0).
		Order("message_time DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// InteractionRepository 互动记录仓库
type InteractionRepository struct {
	db *gorm.DB
}

// NewInteractionRepository 创建互动记录仓库
func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Create 创建互动记录
func (r *InteractionRepository) Create(ctx context.Context, interaction *model.Interaction) error {
	return r.db.WithContext(ctx).Create(interaction).Error
}

// List 获取互动记录列表
func (r *InteractionRepository) List(ctx context.Context, actionType string, page, pageSize int) ([]model.Interaction, int64, error) {
	var interactions []model.Interaction
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Interaction{})
	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Order("action_time DESC").Offset(offset).Limit(pageSize).Find(&interactions).Error
	if err != nil {
		return nil, 0, err
	}

	return interactions, total, nil
}

// GetStats 获取统计数据
func (r *InteractionRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (map[string]int64, error) {
	stats := make(map[string]int64)

	var likeCount, coinCount, favoriteCount int64

	r.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("action_type = ? AND success = ? AND action_time BETWEEN ?", "like", true, startTime, endTime).
		Count(&likeCount)

	r.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("action_type = ? AND success = ? AND action_time BETWEEN ?", "coin", true, startTime, endTime).
		Count(&coinCount)

	r.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("action_type = ? AND success = ? AND action_time BETWEEN ?", "favorite", true, startTime, endTime).
		Count(&favoriteCount)

	stats["like"] = likeCount
	stats["coin"] = coinCount
	stats["favorite"] = favoriteCount

	return stats, nil
}

// TagRankingRepository 标签热度仓库
type TagRankingRepository struct {
	db *gorm.DB
}

// NewTagRankingRepository 创建标签热度仓库
func NewTagRankingRepository(db *gorm.DB) *TagRankingRepository {
	return &TagRankingRepository{db: db}
}

// Create 创建标签热度记录
func (r *TagRankingRepository) Create(ctx context.Context, ranking *model.TagRanking) error {
	return r.db.WithContext(ctx).Create(ranking).Error
}

// BatchCreate 批量创建
func (r *TagRankingRepository) BatchCreate(ctx context.Context, rankings []model.TagRanking) error {
	return r.db.WithContext(ctx).CreateInBatches(rankings, 100).Error
}

// GetByTagName 根据标签名获取
func (r *TagRankingRepository) GetByTagName(ctx context.Context, tagName string) (*model.TagRanking, error) {
	var ranking model.TagRanking
	err := r.db.WithContext(ctx).Where("tag_name = ?", tagName).First(&ranking).Error
	if err != nil {
		return nil, err
	}
	return &ranking, nil
}

// ListByDate 获取指定日期的排行
func (r *TagRankingRepository) ListByDate(ctx context.Context, date time.Time, limit int) ([]model.TagRanking, error) {
	var rankings []model.TagRanking
	err := r.db.WithContext(ctx).
		Where("record_date = ?", date.Format("2006-01-02")).
		Order("rank ASC").
		Limit(limit).
		Find(&rankings).Error
	return rankings, err
}

// GetLatest 获取最新排行
func (r *TagRankingRepository) GetLatest(ctx context.Context, limit int) ([]model.TagRanking, error) {
	var rankings []model.TagRanking
	err := r.db.WithContext(ctx).
		Where("record_date = (SELECT MAX(record_date) FROM tag_rankings)").
		Order("rank ASC").
		Limit(limit).
		Find(&rankings).Error
	return rankings, err
}

// LLMChatLogRepository 大模型对话日志仓库
type LLMChatLogRepository struct {
	db *gorm.DB
}

// NewLLMChatLogRepository 创建大模型对话日志仓库
func NewLLMChatLogRepository(db *gorm.DB) *LLMChatLogRepository {
	return &LLMChatLogRepository{db: db}
}

// Create 创建日志
func (r *LLMChatLogRepository) Create(ctx context.Context, log *model.LLMChatLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetStats 获取统计数据
func (r *LLMChatLogRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalCalls int64
	var totalTokens int
	var successCount int64

	r.db.WithContext(ctx).Model(&model.LLMChatLog{}).
		Where("created_at BETWEEN ?", startTime, endTime).
		Count(&totalCalls)

	r.db.WithContext(ctx).Model(&model.LLMChatLog{}).
		Where("created_at BETWEEN ? AND success = ?", startTime, endTime, true).
		Select("COALESCE(SUM(total_tokens), 0)").Scan(&totalTokens)

	r.db.WithContext(ctx).Model(&model.LLMChatLog{}).
		Where("created_at BETWEEN ? AND success = ?", startTime, endTime, true).
		Count(&successCount)

	stats["total_calls"] = totalCalls
	stats["total_tokens"] = totalTokens
	stats["success_count"] = successCount
	stats["success_rate"] = float64(0)
	if totalCalls > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalCalls) * 100
	}

	return stats, nil
}

// TaskRepository 任务仓库
type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) GetByKey(ctx context.Context, key string) (*model.Setting, error) {
	var setting model.Setting
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func (r *SettingRepository) Set(ctx context.Context, key, value string) error {
	setting := model.Setting{Key: key}
	return r.db.WithContext(ctx).Where(model.Setting{Key: key}).Assign(model.Setting{
		Key:   key,
		Value: value,
	}).FirstOrCreate(&setting).Error
}

func (r *SettingRepository) GetJSON(ctx context.Context, key string, out any) error {
	setting, err := r.GetByKey(ctx, key)
	if err != nil || setting == nil || setting.Value == "" {
		return err
	}
	return json.Unmarshal([]byte(setting.Value), out)
}

func (r *SettingRepository) SetJSON(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Set(ctx, key, string(data))
}

type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓库
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create 创建任务
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// Update 更新任务
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// GetByID 根据ID获取
func (r *TaskRepository) GetByID(ctx context.Context, id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetPendingTasks 获取待执行任务
func (r *TaskRepository) GetPendingTasks(ctx context.Context, limit int) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.WithContext(ctx).
		Where("status = ? AND (run_at IS NULL OR run_at <= ?)", model.TaskStatusPending, time.Now()).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// UpdateStatus 更新任务状态
func (r *TaskRepository) UpdateStatus(ctx context.Context, id uint, status int, result string) error {
	updates := map[string]interface{}{
		"status": status,
		"result": result,
	}

	now := time.Now()
	if status == model.TaskStatusRunning {
		updates["started_at"] = now
	} else if status == model.TaskStatusSuccess || status == model.TaskStatusFailed {
		updates["finished_at"] = now
	}

	return r.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Updates(updates).Error
}
