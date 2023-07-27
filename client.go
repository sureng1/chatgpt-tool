package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	roleSystem = "system"
	roleUser   = "user"
)

var globalClient *Client

type Client struct {
	*openai.Client
}

func getClient(token string, proxy string) (*Client, error) {
	cfg := openai.DefaultConfig(token)
	if proxy != "" {
		pu, err := url.Parse(proxy)
		if err != nil {
			log.Printf("get proxy failed. no proxy. %s", err)
			return nil, err
		}

		cfg.HTTPClient.Transport = &http.Transport{Proxy: http.ProxyURL(pu)}
	}
	c := openai.NewClientWithConfig(cfg)
	return &Client{Client: c}, nil
}

func (c *Client) unary(ctx context.Context, content string) (string, error) {
	var messages = []openai.ChatCompletionMessage{
		{
			Role:    roleUser,
			Content: content,
		},
	}
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: messages,
	}
	rsp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	return rsp.Choices[0].Message.Content, err
}

type Writer interface {
	io.Writer
	Flush()
}

type Conversation struct {
	msgExpiredAt []time.Time
	messages     []openai.ChatCompletionMessage
	tempMsg      *strings.Builder
}

func (c *Conversation) Write(p []byte) (n int, err error) {
	return c.tempMsg.Write(p)
}

func (c *Conversation) Flush() {}

func (c *Conversation) Close() error {
	msg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: c.tempMsg.String(),
	}
	c.AddMsg(msg)
	c.tempMsg.Reset()
	return nil
}
func (c *Conversation) AddMsg(msg openai.ChatCompletionMessage) {
	c.messages = append(c.messages, msg)
	c.msgExpiredAt = append(c.msgExpiredAt, time.Now().Add(defaultExpire))
}

var maxToken = 4096
var defaultExpire = time.Minute * 5

func (c *Conversation) GetMessages() []openai.ChatCompletionMessage {
	var msgs []openai.ChatCompletionMessage
	restToken := maxToken
	now := time.Now()
	for i := len(c.messages) - 1; i >= 0 && restToken > 0; i-- { //token需要从最新的对话开始算起，干掉旧的还没过期的对话，如果token限制了的话
		msg := c.messages[i]
		expired := c.msgExpiredAt[i]
		if expired.Before(now) {
			break
		}
		if len(msg.Content) > restToken {
			break
		}
		restToken -= len(msg.Content)
		msgs = append(msgs, c.messages[i])
	}
	for i := 0; i < len(msgs)/2; i++ {
		j := len(msgs) - i - 1
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs
}

var defaultConv = &Conversation{
	msgExpiredAt: nil,
	messages:     nil,
	tempMsg:      &strings.Builder{},
}

// stream writer.Close() 交给外部逻辑去控制
func (c *Client) stream(ctx context.Context, content string, out Writer) error {
	var conv = defaultConv
	mulW := io.MultiWriter(out, conv)

	msg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	}
	conv.AddMsg(msg)
	msgs := conv.GetMessages()
	if len(msgs) == 0 {
		return fmt.Errorf("message 超过token或全部过期了. maxtokens: %d", maxToken)
	}
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: msgs,
		Stream:   true,
	}

	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Println("create chat failed", err)
		return err
	}
	defer stream.Close()

	defer func() {
		out.Write([]byte("\n"))
		out.Flush()
	}()
	defer func() {
		conv.Flush()
		conv.Close()
	}()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		for _, c := range response.Choices {
			mulW.Write([]byte(c.Delta.Content))
		}
		out.Flush()
	}

	return nil
}
