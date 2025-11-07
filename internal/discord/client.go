package discord

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    token      string
    httpClient *http.Client
}

func NewClient(token string, httpClient *http.Client) *Client {
    if httpClient == nil {
        httpClient = &http.Client{Timeout: 10 * time.Second}
    }
    return &Client{token: token, httpClient: httpClient}
}

func (c *Client) setCommonHeaders(req *http.Request) {
    req.Header.Set("Authorization", c.token)
    req.Header.Set("User-Agent", BROWSER_USER_AGENT)
    req.Header.Set("X-Super-Properties", X_SUPER_PROPERTIES)
    req.Header.Set("X-Discord-Locale", X_DISCORD_LOCALE)
    req.Header.Set("Referer", REFERER_URL)
    req.Header.Set("Content-Type", "application/json")
}

func (c *Client) ValidateToken() (*User, error) {
    req, err := http.NewRequest("GET", DISCORD_API_BASE+"/users/@me", nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request for token validation: %w", err)
    }
    c.setCommonHeaders(req)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request for token validation: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("token validation failed, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
    }
    var user User
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, fmt.Errorf("failed to decode user data: %w", err)
    }
    if user.Bot {
        return nil, fmt.Errorf("provided token belongs to a bot user. This script requires a user token")
    }
    return &user, nil
}

func (c *Client) CreateGuild(serverName string) (*Guild, error) {
    payload := CreateGuildPayload{
        Name:            serverName,
        Icon:            nil,
        Channels:        []interface{}{},
        SystemChannelID: nil,
    }
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal create guild payload: %w", err)
    }
    maxRetries := 3
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if attempt > 0 {
            time.Sleep(time.Duration(attempt*2) * time.Second)
        }
        req, err := http.NewRequest("POST", DISCORD_API_BASE+"/guilds", bytes.NewBuffer(payloadBytes))
        if err != nil {
            lastErr = fmt.Errorf("failed to create request for guild creation: %w", err)
            continue
        }
        c.setCommonHeaders(req)
        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = fmt.Errorf("failed to execute request for guild creation: %w", err)
            continue
        }
        bodyBytes, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
            var newGuild Guild
            if err := json.Unmarshal(bodyBytes, &newGuild); err != nil {
                lastErr = fmt.Errorf("failed to decode guild creation response: %w", err)
                continue
            }
            return &newGuild, nil
        }
        responseStr := string(bodyBytes)
        var discordError struct {
            Message string `json:"message"`
            Code    int    `json:"code"`
        }
        if json.Unmarshal(bodyBytes, &discordError) == nil {
            switch discordError.Code {
            case 10008:
                return nil, fmt.Errorf("account is likely restricted from creating servers. Try using a different account. Status code: %d, Error code: %d (%s)", resp.StatusCode, discordError.Code, discordError.Message)
            case 20028:
                if attempt < maxRetries-1 {
                    time.Sleep(5 * time.Second)
                    continue
                }
                return nil, fmt.Errorf("rate limited by Discord. Try again later or increase INTERVAL_SECONDS. Status code: %d", resp.StatusCode)
            case 40001:
                return nil, fmt.Errorf("unauthorized. Your token may be invalid or expired. Status code: %d", resp.StatusCode)
            default:
                lastErr = fmt.Errorf("failed to create guild, status code: %d. Discord error code: %d (%s). Response: %s", resp.StatusCode, discordError.Code, discordError.Message, responseStr)
            }
        } else {
            lastErr = fmt.Errorf("failed to create guild, status code: %d, response: %s", resp.StatusCode, responseStr)
        }
        if resp.StatusCode == http.StatusUnauthorized {
            return nil, lastErr
        }
        if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
            return nil, lastErr
        }
    }
    return nil, fmt.Errorf("failed to create guild after %d attempts: %v", maxRetries, lastErr)
}

func (c *Client) DeleteGuild(guildID string) error {
    req, err := http.NewRequest("DELETE", DISCORD_API_BASE+"/guilds/"+guildID, nil)
    if err != nil {
        return fmt.Errorf("failed to create request for guild deletion: %w", err)
    }
    req.Header.Set("Authorization", c.token)
    req.Header.Set("User-Agent", BROWSER_USER_AGENT)
    req.Header.Set("X-Super-Properties", X_SUPER_PROPERTIES)
    req.Header.Set("X-Discord-Locale", X_DISCORD_LOCALE)
    req.Header.Set("Referer", REFERER_URL)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request for guild deletion: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusNoContent {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to delete guild, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
    }
    return nil
}

func (c *Client) CreateInvite(guildID string) (string, error) {
    req, err := http.NewRequest("GET", DISCORD_API_BASE+"/guilds/"+guildID+"/channels", nil)
    if err != nil {
        return "", fmt.Errorf("failed to create request for guild channels: %w", err)
    }
    c.setCommonHeaders(req)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to get guild channels: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("failed to get guild channels, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
    }
    var channels []struct {
        ID   string `json:"id"`
        Type int    `json:"type"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
        return "", fmt.Errorf("failed to decode guild channels: %w", err)
    }
    var textChannelID string
    for _, channel := range channels {
        if channel.Type == 0 {
            textChannelID = channel.ID
            break
        }
    }
    if textChannelID == "" {
        return "", fmt.Errorf("no text channel found in guild")
    }
    invitePayload := map[string]interface{}{
        "max_age":   0,
        "max_uses":  0,
        "temporary": false,
    }
    invitePayloadBytes, err := json.Marshal(invitePayload)
    if err != nil {
        return "", fmt.Errorf("failed to marshal invite payload: %w", err)
    }
    req, err = http.NewRequest("POST", DISCORD_API_BASE+"/channels/"+textChannelID+"/invites", bytes.NewBuffer(invitePayloadBytes))
    if err != nil {
        return "", fmt.Errorf("failed to create request for invite creation: %w", err)
    }
    c.setCommonHeaders(req)
    resp, err = c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to create invite: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("failed to create invite, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
    }
    var invite struct {
        Code string `json:"code"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&invite); err != nil {
        return "", fmt.Errorf("failed to decode invite response: %w", err)
    }
    return fmt.Sprintf("https://discord.gg/%s", invite.Code), nil
}

func SuggestSolution(errorCode int) string {
    switch errorCode {
    case 10008:
        return "This account may be restricted from creating servers. Try using a different Discord account."
    case 20028, 429:
        return "You're being rate limited. Try increasing the INTERVAL_SECONDS value or wait a while before trying again."
    case 40001, 401:
        return "Your token appears to be invalid or expired. Try generating a new token."
    default:
        return "General Discord API error. Verify your account is in good standing and not restricted."
    }
}