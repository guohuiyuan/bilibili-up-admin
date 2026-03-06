package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// Update 更新私信记录
func (r *MessageRepository) Update(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Save(message).Error
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
		query = query.Where("sender_uid = ?", senderID)
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

// UpdateReplyStatus 更新私信回复状态
func (r *MessageRepository) UpdateReplyStatus(ctx context.Context, messageID int64, status int, replyContent string, isAIReply bool) error {
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("message_id = ?", messageID).
		Updates(map[string]interface{}{
			"reply_status":  status,
			"reply_content": replyContent,
			"is_ai_reply":   isAIReply,
		}).Error
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
		Where("action_type = ? AND success = ? AND action_time BETWEEN ? AND ?", "like", true, startTime, endTime).
		Count(&likeCount)

	r.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("action_type = ? AND success = ? AND action_time BETWEEN ? AND ?", "coin", true, startTime, endTime).
		Count(&coinCount)

	r.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("action_type = ? AND success = ? AND action_time BETWEEN ? AND ?", "favorite", true, startTime, endTime).
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

// GetLatestByCategory 获取分类下最新一批排行
func (r *TagRankingRepository) GetLatestByCategory(ctx context.Context, category string, limit int) ([]model.TagRanking, error) {
	var rankings []model.TagRanking
	query := r.db.WithContext(ctx).Model(&model.TagRanking{})
	subQuery := r.db.WithContext(ctx).Model(&model.TagRanking{})
	if category != "" {
		query = query.Where("category = ?", category)
		subQuery = subQuery.Where("category = ?", category)
	}

	query = query.Where("record_date = (?)", subQuery.Select("MAX(record_date)")).Order("rank ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&rankings).Error
	return rankings, err
}

// LatestRecordAt 获取分类最新记录时间
func (r *TagRankingRepository) LatestRecordAt(ctx context.Context, category string) (*time.Time, error) {
	query := r.db.WithContext(ctx).Model(&model.TagRanking{})
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var latest sql.NullTime
	if err := query.Select("MAX(record_date)").Scan(&latest).Error; err != nil {
		return nil, err
	}
	if !latest.Valid {
		return nil, nil
	}
	return &latest.Time, nil
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

func (r *LLMChatLogRepository) ListByConversation(ctx context.Context, conversationKey string, limit int) ([]model.LLMChatLog, error) {
	var logs []model.LLMChatLog
	query := r.db.WithContext(ctx).Model(&model.LLMChatLog{})
	if conversationKey != "" {
		query = query.Where("conversation_key = ?", conversationKey)
	}
	query = query.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	return logs, query.Find(&logs).Error
}

func (r *LLMChatLogRepository) ListByConversationOldestFirst(ctx context.Context, conversationKey string, limit int) ([]model.LLMChatLog, error) {
	var logs []model.LLMChatLog
	query := r.db.WithContext(ctx).Model(&model.LLMChatLog{})
	if conversationKey != "" {
		query = query.Where("conversation_key = ?", conversationKey)
	}
	query = query.Order("created_at ASC").Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	return logs, query.Find(&logs).Error
}

func (r *LLMChatLogRepository) List(ctx context.Context, inputType, conversationKey, logType string, page, pageSize int) ([]model.LLMChatLog, int64, error) {
	var logs []model.LLMChatLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LLMChatLog{})
	if inputType != "" {
		query = query.Where("input_type = ?", inputType)
	}
	if conversationKey != "" {
		query = query.Where("conversation_key LIKE ?", "%"+conversationKey+"%")
	}
	if logType != "" {
		query = query.Where("log_type = ?", logType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// GetStats 获取统计数据
func (r *LLMChatLogRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalCalls int64
	var totalTokens int
	var successCount int64

	r.db.WithContext(ctx).Model(&model.LLMChatLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Count(&totalCalls)

	r.db.WithContext(ctx).Model(&model.LLMChatLog{}).
		Where("created_at BETWEEN ? AND ? AND success = ?", startTime, endTime, true).
		Select("COALESCE(SUM(total_tokens), 0)").Scan(&totalTokens)

	r.db.WithContext(ctx).Model(&model.LLMChatLog{}).
		Where("created_at BETWEEN ? AND ? AND success = ?", startTime, endTime, true).
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

// LLMProviderRepository 大模型提供商仓库
type LLMProviderRepository struct {
	db *gorm.DB
}

func NewLLMProviderRepository(db *gorm.DB) *LLMProviderRepository {
	return &LLMProviderRepository{db: db}
}

type AdminUserRepository struct {
	db *gorm.DB
}

func NewAdminUserRepository(db *gorm.DB) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

func (r *AdminUserRepository) Create(ctx context.Context, user *model.AdminUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *AdminUserRepository) First(ctx context.Context) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.WithContext(ctx).Order("id ASC").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AdminUserRepository) GetByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AdminUserRepository) GetByID(ctx context.Context, id uint) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AdminUserRepository) Update(ctx context.Context, user *model.AdminUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

type AdminSessionRepository struct {
	db *gorm.DB
}

func NewAdminSessionRepository(db *gorm.DB) *AdminSessionRepository {
	return &AdminSessionRepository{db: db}
}

func (r *AdminSessionRepository) Create(ctx context.Context, session *model.AdminSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *AdminSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.AdminSession, error) {
	var session model.AdminSession
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *AdminSessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&model.AdminSession{}).Error
}

func (r *AdminSessionRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at <= ?", time.Now()).Delete(&model.AdminSession{}).Error
}

func (r *AdminSessionRepository) Update(ctx context.Context, session *model.AdminSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

type FanAutoReplyRecordRepository struct {
	db *gorm.DB
}

func NewFanAutoReplyRecordRepository(db *gorm.DB) *FanAutoReplyRecordRepository {
	return &FanAutoReplyRecordRepository{db: db}
}

func (r *FanAutoReplyRecordRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.FanAutoReplyRecord{}).Count(&count).Error
	return count, err
}

func (r *FanAutoReplyRecordRepository) GetByFanUID(ctx context.Context, fanUID int64) (*model.FanAutoReplyRecord, error) {
	var record model.FanAutoReplyRecord
	err := r.db.WithContext(ctx).Where("fan_uid = ?", fanUID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *FanAutoReplyRecordRepository) Create(ctx context.Context, record *model.FanAutoReplyRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *FanAutoReplyRecordRepository) Update(ctx context.Context, record *model.FanAutoReplyRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

type ReplyTemplateRepository struct {
	db *gorm.DB
}

func NewReplyTemplateRepository(db *gorm.DB) *ReplyTemplateRepository {
	return &ReplyTemplateRepository{db: db}
}

func (r *ReplyTemplateRepository) List(ctx context.Context, channel string, limit int) ([]model.ReplyTemplate, error) {
	var items []model.ReplyTemplate
	query := r.db.WithContext(ctx).Model(&model.ReplyTemplate{})
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}
	query = query.Order("usage_count DESC").Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	return items, query.Find(&items).Error
}

func (r *ReplyTemplateRepository) GetByID(ctx context.Context, id uint) (*model.ReplyTemplate, error) {
	var item model.ReplyTemplate
	err := r.db.WithContext(ctx).First(&item, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ReplyTemplateRepository) Create(ctx context.Context, item *model.ReplyTemplate) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *ReplyTemplateRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ReplyTemplate{}, id).Error
}

func (r *ReplyTemplateRepository) TouchUsage(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.ReplyTemplate{}).Where("id = ?", id).Updates(map[string]any{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": &now,
	}).Error
}

type ReplyExampleRepository struct {
	db *gorm.DB
}

func NewReplyExampleRepository(db *gorm.DB) *ReplyExampleRepository {
	return &ReplyExampleRepository{db: db}
}

func (r *ReplyExampleRepository) List(ctx context.Context, channel string, limit int) ([]model.ReplyExample, error) {
	var items []model.ReplyExample
	query := r.db.WithContext(ctx).Model(&model.ReplyExample{})
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}
	query = query.Order("quality_score DESC").Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	return items, query.Find(&items).Error
}

func (r *ReplyExampleRepository) Create(ctx context.Context, item *model.ReplyExample) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *ReplyExampleRepository) TouchUsage(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.ReplyExample{}).Where("id = ?", id).Updates(map[string]any{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": &now,
	}).Error
}

type ReplyDraftRepository struct {
	db *gorm.DB
}

func NewReplyDraftRepository(db *gorm.DB) *ReplyDraftRepository {
	return &ReplyDraftRepository{db: db}
}

func (r *ReplyDraftRepository) GetByTarget(ctx context.Context, channel string, targetID int64) (*model.ReplyDraft, error) {
	var item model.ReplyDraft
	err := r.db.WithContext(ctx).Where("channel = ? AND target_id = ?", channel, targetID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ReplyDraftRepository) Save(ctx context.Context, draft *model.ReplyDraft) error {
	var existing model.ReplyDraft
	err := r.db.WithContext(ctx).Where("channel = ? AND target_id = ?", draft.Channel, draft.TargetID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(draft).Error
	}
	if err != nil {
		return err
	}
	draft.ID = existing.ID
	draft.CreatedAt = existing.CreatedAt
	return r.db.WithContext(ctx).Save(draft).Error
}

func (r *LLMProviderRepository) List(ctx context.Context) ([]model.LLMProvider, error) {
	var list []model.LLMProvider
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (r *LLMProviderRepository) Save(ctx context.Context, provider *model.LLMProvider) error {
	var existing model.LLMProvider
	// 按照 Name 查找，如果存在则更新，不存在则创建
	if err := r.db.WithContext(ctx).Where("name = ?", provider.Name).First(&existing).Error; err == nil {
		provider.ID = existing.ID
		provider.CreatedAt = existing.CreatedAt
		return r.db.WithContext(ctx).Save(provider).Error
	}
	return r.db.WithContext(ctx).Create(provider).Error
}

func (r *LLMProviderRepository) Delete(ctx context.Context, name string) error {
	return r.db.WithContext(ctx).Where("name = ?", name).Delete(&model.LLMProvider{}).Error
}
