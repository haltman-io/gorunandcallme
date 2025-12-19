package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
)

type WebhookClient struct {
	http    *http.Client
	url     string
	headers map[string]string
}

func NewWebhookClient(httpc *http.Client, u string, headers map[string]string) (*WebhookClient, error) {
	if u == "" {
		return nil, errors.New("webhook url is empty")
	}
	if _, err := url.Parse(u); err != nil {
		return nil, err
	}
	return &WebhookClient{http: httpc, url: u, headers: headers}, nil
}

func (w *WebhookClient) Name() string { return "webhook" }
func (w *WebhookClient) MaxTextChars() int { return 6000 }
func (w *WebhookClient) MaxAttachBytes() int { return 10000000 }

func (w *WebhookClient) SendText(text string) error {
	body := map[string]any{
		"text": text,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", w.url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

func (w *WebhookClient) SendFile(filename string, contentType string, data []byte, caption string) error {
	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	_ = mp.WriteField("caption", caption)
	fw, err := mp.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	_, _ = fw.Write(data)
	_ = mp.Close()

	req, _ := http.NewRequest("POST", w.url, &buf)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}
