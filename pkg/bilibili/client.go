package bilibili

import (
	"context"
	"fmt"
	"strconv"

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

	client := newInnerClient(cfg)
	return &Client{
		inner:  client,
		config: cfg,
	}, nil
}

func newInnerClient(cfg *Config) *biligo.Client {
	client := biligo.NewClient()
	client.SetCredential(&biligo.Credential{
		SessData:   cfg.SESSData,
		BiliJct:    cfg.BiliJct,
		DedeUserID: fmt.Sprintf("%d", cfg.UserID),
	})
	return client
}

func NewClientFromCookieString(rawCookie string) (*Client, error) {
	credential := biligo.NewCredentialFromCookieString(rawCookie)
	if err := credential.EnsureSessData(); err != nil {
		return nil, err
	}
	var userID int64
	if credential.DedeUserID != "" {
		userID, _ = strconv.ParseInt(credential.DedeUserID, 10, 64)
	}
	return NewClient(&Config{
		SESSData: credential.SessData,
		BiliJct:  credential.BiliJct,
		UserID:   userID,
	})
}

func (c *Client) GetInner() *biligo.Client {
	return c.inner
}

func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
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
	if c.config == nil {
		c.config = &Config{}
	}
	c.config.SESSData = sessData
	c.config.BiliJct = biliJct
	c.config.UserID = userID
	c.inner = newInnerClient(c.config)
}

func (c *Client) GetConfig() *Config {
	return c.config
}

func (c *Client) ensureAvailable() error {
	if c == nil || c.inner == nil {
		return fmt.Errorf("bilibili client is not initialized, configure bilibili login first")
	}
	return nil
}

type QRCodeLogin struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

type QRCodeLoginState struct {
	Success bool      `json:"success"`
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Client  *Client   `json:"-"`
	User    *UserInfo `json:"user,omitempty"`
}

func GenerateLoginQRCode(ctx context.Context) (*QRCodeLogin, error) {
	client := biligo.NewClient()
	result, err := client.Login().QRCodeGenerate(ctx)
	if err != nil {
		return nil, err
	}
	return &QRCodeLogin{URL: result.URL, Key: result.QRCodeKey}, nil
}

func PollLoginQRCode(ctx context.Context, key string) (*QRCodeLoginState, error) {
	client := biligo.NewClient()
	result, err := client.Login().QRCodePoll(ctx, key)
	if err != nil {
		return nil, err
	}
	state := &QRCodeLoginState{
		Success: result.Code == 0,
		Code:    result.Code,
		Message: result.Message,
	}
	if !state.Success {
		return state, nil
	}
	credential := client.Credential()
	if credential == nil || credential.SessData == "" {
		return nil, fmt.Errorf("qrcode login succeeded but no credential found")
	}
	var userID int64
	if credential.DedeUserID != "" {
		userID, _ = strconv.ParseInt(credential.DedeUserID, 10, 64)
	}
	wrapped, err := NewClient(&Config{
		SESSData: credential.SessData,
		BiliJct:  credential.BiliJct,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}
	info, err := wrapped.GetUserInfo(ctx)
	if err == nil {
		state.User = info
	}
	state.Client = wrapped
	return state, nil
}
