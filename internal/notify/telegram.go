package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type TelegramClient struct {
	http      *http.Client
	botToken  string
	chatID    string
	parseMode string
}

func NewTelegramClient(httpc *http.Client, botToken, chatID, parseMode string) (*TelegramClient, error) {
	if botToken == "" || chatID == "" {
		return nil, errors.New("telegram bot_token and chat_id are required")
	}
	return &TelegramClient{
		http:      httpc,
		botToken:  botToken,
		chatID:    chatID,
		parseMode: parseMode,
	}, nil
}

func (t *TelegramClient) Name() string { return "telegram" }
func (t *TelegramClient) MaxTextChars() int { return 3800 } // keep margin below 4096
func (t *TelegramClient) MaxAttachBytes() int { return 45000000 }

func (t *TelegramClient) SendText(text string) error {
	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", url.PathEscape(t.botToken))

	// Telegram MarkdownV2 requires escaping.
	escaped := EscapeMarkdownV2(text)

	body := map[string]any{
		"chat_id": t.chatID,
		"text":    escaped,
		"parse_mode": "MarkdownV2",
		"disable_web_page_preview": true,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", api, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

func (t *TelegramClient) SendFile(filename string, contentType string, data []byte, caption string) error {
	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", url.PathEscape(t.botToken))

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	_ = w.WriteField("chat_id", t.chatID)
	// Caption also needs escaping if using MarkdownV2.
	_ = w.WriteField("caption", EscapeMarkdownV2(caption))
	_ = w.WriteField("parse_mode", "MarkdownV2")

	fw, err := w.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	_, _ = fw.Write(data)
	_ = w.Close()

	req, _ := http.NewRequest("POST", api, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

// EscapeMarkdownV2 escapes reserved characters required by Telegram MarkdownV2.
func EscapeMarkdownV2(s string) string {
	// Telegram requires escaping: _ * [ ] ( ) ~ ` > # + - = | { } . !
	replacer := strings.NewReplacer(
		`_`, `\_`,
		`*`, `\*`,
		`[`, `\[`,
		`]`, `\]`,
		`(`, `\(`,
		`)`, `\)`,
		`~`, `\~`,
		"`", "\\`",
		`>`, `\>`,
		`#`, `\#`,
		`+`, `\+`,
		`-`, `\-`,
		`=`, `\=`,
		`|`, `\|`,
		`{`, `\{`,
		`}`, `\}`,
		`.`, `\.`,
		`!`, `\!`,
	)
	return replacer.Replace(s)
}
