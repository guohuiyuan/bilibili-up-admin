package bilibili

import (
	"context"
	"fmt"

	biligo "github.com/guohuiyuan/biligo"
)

type Client struct {
	inner    *biligo.Client
	config   *Config
	userInfo *UserInfo
}

type Config struct {
	SESSData string
	BiliJct  string
	UserID   int64
}

type UserInfo struct {
	Mid     int64
	Name    string
	Face    string
	IsLogin bool
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg.SESSData == "" {
		return nil, fmt.Errorf("sess_data is required")
	}

	client := biligo.NewClient()
	client.SetCredential(&biligo.Credential{
		SessData:   cfg.SESSData,
		BiliJct:    cfg.BiliJct,
		DedeUserID: fmt.Sprintf("%d", cfg.UserID),
	})

	return &Client{
		inner:  client,
		config: cfg,
	}, nil
}

func (c *Client) GetInner() *biligo.Client {
	return c.inner
}

func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	info, err := c.inner.Login().Nav(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user info failed: %w", err)
	}

	userInfo := &UserInfo{
		Mid:     info.Mid,
		Name:    info.Uname,
		IsLogin: info.IsLogin,
	}
	c.userInfo = userInfo
	return userInfo, nil
}

func (c *Client) IsLoggedIn() bool {
	return c.userInfo != nil && c.userInfo.IsLogin
}

func (c *Client) SetCredential(sessData, biliJct string, userID int64) {
	c.config.SESSData = sessData
	c.config.BiliJct = biliJct
	c.config.UserID = userID
}

func (c *Client) GetConfig() *Config {
	return c.config
}
