package webhook

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Message struct {
    Content   string   `json:"content,omitempty"`
    Embeds    []Embed  `json:"embeds,omitempty"`
    Username  string   `json:"username,omitempty"`
    AvatarURL string   `json:"avatar_url,omitempty"`
}

type Embed struct {
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Color       int      `json:"color"`
    Fields      []Field  `json:"fields,omitempty"`
    Timestamp   string   `json:"timestamp,omitempty"`
    Image       *Image   `json:"image,omitempty"`
}

type Field struct {
    Name   string `json:"name"`
    Value  string `json:"value"`
    Inline bool   `json:"inline,omitempty"`
}

type Image struct {
    URL string `json:"url"`
}

type Sender struct {
    httpClient *http.Client
}

func NewSender() *Sender {
    return &Sender{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (s *Sender) Send(webhookURL string, message Message) error {
    if webhookURL == "" {
        return nil
    }
    payloadBytes, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal webhook message: %w", err)
    }
    req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
    if err != nil {
        return fmt.Errorf("failed to create webhook request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := s.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send webhook message: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("webhook request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
    }
    return nil
}