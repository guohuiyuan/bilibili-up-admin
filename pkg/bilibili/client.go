package bilibili

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	Cookie   string
}

type UserInfo struct {
	Mid     int64
	Name    string
	Face    string
	IsLogin bool
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg.Cookie == "" && cfg.SESSData == "" {
		return nil, fmt.Errorf("cookie or sess_data is required")
	}

	client := newInnerClient(cfg)
	return &Client{
		inner:  client,
		config: cfg,
	}, nil
}

func newInnerClient(cfg *Config) *biligo.Client {
	client := biligo.NewClient()
	if cfg.Cookie != "" {
		credential := biligo.NewCredentialFromCookieString(cfg.Cookie)
		client.SetCredential(credential)
		return client
	}
	dedeUserID := ""
	if cfg.UserID > 0 {
		dedeUserID = fmt.Sprintf("%d", cfg.UserID)
	}
	client.SetCredential(&biligo.Credential{
		SessData:   cfg.SESSData,
		BiliJct:    cfg.BiliJct,
		DedeUserID: dedeUserID,
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
		Cookie:   rawCookie,
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

func (c *Client) User() *biligo.UserService {
	return c.inner.User()
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
		Cookie:   buildCookieString(credential),
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

func buildCookieString(c *biligo.Credential) string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, 6)
	if c.SessData != "" {
		parts = append(parts, "SESSDATA="+c.SessData)
	}
	if c.BiliJct != "" {
		parts = append(parts, "bili_jct="+c.BiliJct)
	}
	if c.DedeUserID != "" {
		parts = append(parts, "DedeUserID="+c.DedeUserID)
	}
	if c.Buvid3 != "" {
		parts = append(parts, "buvid3="+c.Buvid3)
	}
	if c.Buvid4 != "" {
		parts = append(parts, "buvid4="+c.Buvid4)
	}
	if c.AcTimeValue != "" {
		parts = append(parts, "ac_time_value="+c.AcTimeValue)
	}
	return strings.Join(parts, "; ")
}

type UserVideoSearchResult struct {
	List struct {
		VList []VideoInfo `json:"vlist"`
	} `json:"list"`
}
