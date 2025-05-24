package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 配置参数
const (
	defaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings"
	defaultTimeout = 30 * time.Second
)

// Client 是调用 Embedding API 的客户端
type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewClient 创建一个新的 Embedding 客户端
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Request 是发送给 API 的请求结构
type Request struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Dimension      string `json:"dimension"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

// Response 是从 API 接收的响应结构
type Response struct {
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    []Embedding `json:"data,omitempty"`
	Model   string      `json:"model,omitempty"`
	Object  string      `json:"object,omitempty"`
	Usage   Usage       `json:"usage,omitempty"`
	ID      string      `json:"id,omitempty"`
	Status  int         `json:"status,omitempty"`
}

// Embedding 表示一个文本的嵌入向量
type Embedding struct {
	Object    string    `json:"object,omitempty"`
	Index     int       `json:"index,omitempty"`
	Embedding []float64 `json:"embedding,omitempty"`
}

// Usage 包含使用的 token 信息
type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// GetEmbeddings 获取文本的嵌入向量
func (c *Client) GetEmbeddings(texts string, model string, dimension string) (*Response, error) {

	// 构建请求体
	reqBody := Request{
		Model:          model,
		Input:          texts,
		Dimension:      dimension,
		EncodingFormat: "float",
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 请求失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应 JSON
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w，响应内容: %s", err, string(body))
	}

	return &response, nil
}

// ConvertToPGVector 将 Embedding 结构体中的向量数据转换为 PostgreSQL 可接受的向量格式
func ConvertToPGVector(embedding []float64) string {
	var strs []string
	for _, val := range embedding {
		strs = append(strs, strconv.FormatFloat(val, 'f', -1, 64))
	}
	return "[" + strings.Join(strs, ",") + "]"
}
