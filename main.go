//@efetutorial

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"golang.org/x/sys/windows"
)

const (
	INTERVAL_SECONDS     = 67
	DELETE_DELAY_SECONDS = 2
	SERVER_NAME_CONST    = "Tag server"
	DISCORD_API_BASE     = "https://discord.com/api/v9" // Common API version
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
	BROWSER_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36"
	X_SUPER_PROPERTIES = "eyJvcyI6IldpbmRvd3MiLCJicm93c2VyIjoiQ2hyb21lIiwiZGV2aWNlIjoiIiwic3lzdGVtX2xvY2FsZSI6ImVuLVVTIiwiYnJvd3Nlcl91c2VyX2FnZW50IjoiTW96aWxsYS81LjAgKFdpbmRvd3MgTlQgMTAuMDsgV2luNjQ7IHg2NCkgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzEwNS4wLjAuMCBTYWZhcmkvNTM3LjM2IiwiYnJvd3Nlcl92ZXJzaW9uIjoiMTA1LjAuMC4wIiwib3NfdmVyc2lvbiI6IjEwIiwicmVsZWFzZV9jaGFubmVsIjoic3RhYmxlIiwiY2xpZW50X2J1aWxkX251bWJlciI6MjgwMTAwLCJjbGllbnRfZXZlbnRfc291cmNlIjpudWxsfQ=="
	X_DISCORD_LOCALE   = "en-US"
	REFERER_URL        = "https://discord.com/channels/@me"
	WEBHOOK_SUCCESS_COLOR = 5763719
	WEBHOOK_ERROR_COLOR   = 15548997
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	proxyList  []string
	proxyLock  sync.Mutex
	proxyIndex int
)

func init() {
	if runtime.GOOS == "windows" {
		var outMode uint32
		out := windows.Handle(os.Stdout.Fd())
		if err := windows.GetConsoleMode(out, &outMode); err != nil {
			log.Printf("Warning: Failed to get console mode: %v. Colors might not work.", err)
			return
		}

		outMode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

		if err := windows.SetConsoleMode(out, outMode); err != nil {
			log.Printf("Warning: Failed to set console mode: %v. Colors might not work.", err)
		}
	}
}

func murmurhash3_32_gc_go(key string, seed uint32) uint32 {
	data := []byte(key)
	length := len(data)
	nblocks := length / 4

	h1 := seed

	const c1 uint32 = 0xcc9e2d51
	const c2 uint32 = 0x1b873593

	// Body
	for i := 0; i < nblocks; i++ {
		k1 := binary.LittleEndian.Uint32(data[i*4:])

		k1 *= c1
		k1 = (k1 << 15) | (k1 >> 17) // rotl32(k1, 15)
		k1 *= c2

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19) // rotl32(h1, 13)
		h1 = h1*5 + 0xe6546b64
	}

	// Tail
	tail := data[nblocks*4:]
	var k1 uint32 = 0
	switch length & 3 {
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0])
		k1 *= c1
		k1 = (k1 << 15) | (k1 >> 17) // rotl32(k1, 15)
		k1 *= c2
		h1 ^= k1
	}

	// Finalization
	h1 ^= uint32(length)
	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
	h1 ^= h1 >> 16

	return h1
}

type Guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Bot           bool   `json:"bot"`
}

type CreateGuildPayload struct {
	Name            string        `json:"name"`
	Icon            *string       `json:"icon"`
	Channels        []interface{} `json:"channels"`
	SystemChannelID *string       `json:"system_channel_id"`
}

// Proxy desteği: proxy.txt'den proxy'leri oku
func readProxiesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open proxy file: %w", err)
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := strings.TrimSpace(scanner.Text())
		if proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading proxy file: %w", err)
	}
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no valid proxies found in the file")
	}
	return proxies, nil
}

// Proxy desteği: Sıradaki proxy'yi döndür
func getNextProxy() string {
	proxyLock.Lock()
	defer proxyLock.Unlock()
	if len(proxyList) == 0 {
		return ""
	}
	proxy := proxyList[proxyIndex%len(proxyList)]
	proxyIndex++
	return proxy
}

// Proxy desteği: HTTP client'ı proxy ile oluştur
func newHttpClientWithProxy(proxyAddr string) *http.Client {
	if proxyAddr == "" {
		return &http.Client{Timeout: 10 * time.Second}
	}
	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return &http.Client{Timeout: 10 * time.Second}
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
}

func setCommonHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", BROWSER_USER_AGENT)
	req.Header.Set("X-Super-Properties", X_SUPER_PROPERTIES)
	req.Header.Set("X-Discord-Locale", X_DISCORD_LOCALE)
	req.Header.Set("Referer", REFERER_URL)
	req.Header.Set("Content-Type", "application/json")
}

func validateToken(token string) (*User, error) {
	req, err := http.NewRequest("GET", DISCORD_API_BASE+"/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for token validation: %w", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", BROWSER_USER_AGENT)
	req.Header.Set("X-Super-Properties", X_SUPER_PROPERTIES)
	req.Header.Set("X-Discord-Locale", X_DISCORD_LOCALE)
	req.Header.Set("Referer", REFERER_URL)

	resp, err := httpClient.Do(req)
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

func createGuild(token string) (*Guild, error) {
	payload := CreateGuildPayload{
		Name:            SERVER_NAME_CONST,
		Icon:            nil,
		Channels:        []interface{}{},
		SystemChannelID: nil,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create guild payload: %w", err)
	}

	// Add retry logic for temporary failures
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying guild creation (attempt %d/%d)...", attempt+1, maxRetries)
			// Exponential backoff between retries
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
		
		req, err := http.NewRequest("POST", DISCORD_API_BASE+"/guilds", bytes.NewBuffer(payloadBytes))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request for guild creation: %w", err)
			continue
		}
		setCommonHeaders(req, token)

		resp, err := httpClient.Do(req)
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
		
		// Parse specific errors
		responseStr := string(bodyBytes)
		var discordError struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		
		if json.Unmarshal(bodyBytes, &discordError) == nil {
			// Check for specific error codes
			switch discordError.Code {
			case 10008: // Unknown Message
				return nil, fmt.Errorf("account is likely restricted from creating servers. Try using a different account. Status code: %d, Error code: %d (%s)", resp.StatusCode, discordError.Code, discordError.Message)
			case 20028: // Rate limited
				if attempt < maxRetries-1 {
					log.Printf("Rate limited. Waiting before retry...")
					time.Sleep(5 * time.Second)
					continue
				}
				return nil, fmt.Errorf("rate limited by Discord. Try again later or increase INTERVAL_SECONDS. Status code: %d", resp.StatusCode)
			case 40001: // Unauthorized
				return nil, fmt.Errorf("unauthorized. Your token may be invalid or expired. Status code: %d", resp.StatusCode)
			default:
				lastErr = fmt.Errorf("failed to create guild, status code: %d. Discord error code: %d (%s). Response: %s", resp.StatusCode, discordError.Code, discordError.Message, responseStr)
			}
		} else {
			lastErr = fmt.Errorf("failed to create guild, status code: %d, response: %s", resp.StatusCode, responseStr)
		}
		
		// Don't retry on permanent errors like 401 Unauthorized
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, lastErr
		}
		
		// Check if we should retry based on status code
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return nil, lastErr
		}
	}
	
	return nil, fmt.Errorf("failed to create guild after %d attempts: %v", maxRetries, lastErr)
}

func suggestSolution(errorCode int) string {
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

func deleteGuild(token string, guildID string) error {
	req, err := http.NewRequest("DELETE", DISCORD_API_BASE+"/guilds/"+guildID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for guild deletion: %w", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", BROWSER_USER_AGENT)
	req.Header.Set("X-Super-Properties", X_SUPER_PROPERTIES)
	req.Header.Set("X-Discord-Locale", X_DISCORD_LOCALE)
	req.Header.Set("Referer", REFERER_URL)

	resp, err := httpClient.Do(req)
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

func createInvite(token string, guildID string) (string, error) {
	// First, we need to get the channels in the guild to create an invite
	req, err := http.NewRequest("GET", DISCORD_API_BASE+"/guilds/"+guildID+"/channels", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for guild channels: %w", err)
	}
	setCommonHeaders(req, token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get guild channels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get guild channels, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response to get the channels
	var channels []struct {
		ID   string `json:"id"`
		Type int    `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return "", fmt.Errorf("failed to decode guild channels: %w", err)
	}

	// Find the first text channel
	var textChannelID string
	for _, channel := range channels {
		// Type 0 is a text channel
		if channel.Type == 0 {
			textChannelID = channel.ID
			break
		}
	}

	if textChannelID == "" {
		return "", fmt.Errorf("no text channel found in guild")
	}

	// Create the invite for the text channel
	invitePayload := map[string]interface{}{
		"max_age":   0, // Never expire
		"max_uses":  0, // Unlimited uses
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
	setCommonHeaders(req, token)

	resp, err = httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create invite: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create invite, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response to get the invite code
	var invite struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invite); err != nil {
		return "", fmt.Errorf("failed to decode invite response: %w", err)
	}

	return fmt.Sprintf("https://discord.gg/%s", invite.Code), nil
}

type WebhookMessage struct {
	Content   string         `json:"content,omitempty"`
	Embeds    []WebhookEmbed `json:"embeds,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
}

type WebhookEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []WebhookEmbedField `json:"fields,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Image       *WebhookEmbedImage  `json:"image,omitempty"`
}

type WebhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type WebhookEmbedImage struct {
	URL string `json:"url"`
}

func sendWebhookMessage(webhookURL string, message WebhookMessage) error {
	if webhookURL == "" {
		return nil
	}

	payloadBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook message: %w", err)
	}

	// Proxy desteği: webhook için proxy kullanılmaz, doğrudan gönder
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
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

func readTokensFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open token file: %w", err)
	}
	defer file.Close()

	var tokens []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		if token != "" {
			tokens = append(tokens, token)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading token file: %w", err)
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("no valid tokens found in the file")
	}

	return tokens, nil
}

func runForToken(token string, webhookURL string, tokenIndex int) {
	prefix := ""
	if tokenIndex >= 0 {
		prefix = fmt.Sprintf("[Token %d] ", tokenIndex+1)
	}

	logPrefix := func(message string) string {
		return prefix + message
	}

	log.Println(logPrefix("Validating token..."))
	user, err := validateToken(token)
	if err != nil {
		log.Printf(ColorRed+"%s"+ColorReset, logPrefix(fmt.Sprintf("Token validation failed: %v", err)))
		return
	}
	
	log.Printf(ColorGreen+"%s"+ColorReset, logPrefix(fmt.Sprintf("Token validated. Logged in as: %s#%s (ID: %s)", user.Username, user.Discriminator, user.ID)))
	
	if user.Bot { 
		log.Printf(ColorRed+"%s"+ColorReset, logPrefix("Error: The provided token is for a bot account. This script requires a user account token."))
		return
	}

	if webhookURL != "" {
		startMessage := WebhookMessage{
			Embeds: []WebhookEmbed{
				{
					Title:       fmt.Sprintf("Discord Tag Tool Started - Token %d", tokenIndex+1),
					Description: fmt.Sprintf("Tool started successfully and logged in as %s#%s", user.Username, user.Discriminator),
					Color:       WEBHOOK_SUCCESS_COLOR,
					Fields: []WebhookEmbedField{
						{Name: "User ID", Value: user.ID, Inline: true},
						{Name: "Status", Value: "Starting guild creation process...", Inline: true},
						{Name: "Token Index", Value: fmt.Sprintf("%d", tokenIndex+1), Inline: true},
					},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
			Username: "Tag Finder Tool",
		}
		
		if err := sendWebhookMessage(webhookURL, startMessage); err != nil {
			log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Failed to send webhook notification: %v", err)))
		} else {
			log.Println(logPrefix("Initial webhook notification sent successfully."))
		}
	}

	log.Println(logPrefix("Starting guild (server) creation script..."))

	randomizedInterval := INTERVAL_SECONDS
	if tokenIndex > 0 {
		randomizedInterval += tokenIndex * 10
	}
	
	ticker := time.NewTicker(time.Duration(randomizedInterval) * time.Second)
	defer ticker.Stop()

	foundGuilds := make(map[string]uint32)

	// Proxy desteği: Her token için yeni bir httpClient oluştur
	var client *http.Client
	proxyAddr := getNextProxy()
	client = newHttpClientWithProxy(proxyAddr)
	// httpClient değişkenini fonksiyonun başında ayarla
	oldClient := httpClient
	httpClient = client
	defer func() { httpClient = oldClient }()

	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf(ColorRed+"%s"+ColorReset, logPrefix(fmt.Sprintf("Recovered from panic: %v", r)))
					}
				}()
				
				log.Println(logPrefix("Attempting to create a new guild (server)..."))
				newGuild, err := createGuild(token)
				if err != nil {
					errorMessage := err.Error()
					
					errorCode := 0
					if strings.Contains(errorMessage, "Error code:") || strings.Contains(errorMessage, "error code:") {
						codeParts := strings.Split(strings.ToLower(errorMessage), "error code:")
						if len(codeParts) > 1 {
							fmt.Sscanf(codeParts[1], "%d", &errorCode)
						}
					}
					
					solution := suggestSolution(errorCode)
					log.Printf(ColorRed+"%s"+ColorReset, logPrefix(fmt.Sprintf("Error creating guild (server): %v", err)))
					log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Suggestion: %s", solution)))
					
					if errorCode == 10008 {
						log.Printf(ColorYellow+"%s"+ColorReset, logPrefix("Waiting for 5 minutes before attempting again..."))
						time.Sleep(5 * time.Minute)
					}
					
					return
				}

				if newGuild == nil || newGuild.ID == "" {
					log.Println(ColorRed + logPrefix("Failed to create guild (server) or guild ID is empty.") + ColorReset)
					return
				}

				log.Printf(logPrefix(fmt.Sprintf("Guild (server) created: %s (ID: %s)", newGuild.Name, newGuild.ID)))

				hashKey := fmt.Sprintf("2025-02_skill_trees:%s", newGuild.ID)
				hashValue := murmurhash3_32_gc_go(hashKey, 0) % 10000

				if hashValue >= 10 && hashValue < 20 {
					successMessage := fmt.Sprintf("🎉 FOUND GUILD (SERVER) WITH TAG: %s (ID: %s) HASH: %d 🎉", newGuild.Name, newGuild.ID, hashValue)
					log.Print(ColorGreen + logPrefix(successMessage) + ColorReset) 
					
					foundGuilds[newGuild.ID] = hashValue
					
					inviteLink, err := createInvite(token, newGuild.ID)
					if err != nil {
						log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Failed to create invite: %v. Using direct server link instead.", err)))
						inviteLink = fmt.Sprintf("https://discord.com/channels/%s", newGuild.ID)
					} else {
						log.Printf(ColorGreen+"%s"+ColorReset, logPrefix(fmt.Sprintf("Created invite link: %s", inviteLink)))
					}
					
					if webhookURL != "" {
						tagFoundMessage := WebhookMessage{
							Content: "@everyone ADAMI SİKERİM BULDU LAN",
							Embeds: []WebhookEmbed{
								{
									Title:       "🎉 BEYLER BULDUK 🎉",
									Description: "cetrefilli adamlar klübü siker",
									Color:       WEBHOOK_SUCCESS_COLOR,
									Fields: []WebhookEmbedField{
										{Name: "Guild ID", Value: newGuild.ID, Inline: true},
										{Name: "Guild Name", Value: newGuild.Name, Inline: true},
										{Name: "Hash Value", Value: fmt.Sprintf("%d", hashValue), Inline: true},
										{Name: "Düşen hesap", Value: fmt.Sprintf("%s#%s (ID: %s)", user.Username, user.Discriminator, user.ID), Inline: false},
										{Name: "Discord sunucusu", Value: inviteLink, Inline: false},
										{Name: "Token Index", Value: fmt.Sprintf("%d", tokenIndex+1), Inline: false},
									},
									Timestamp: time.Now().Format(time.RFC3339),
									Image: &WebhookEmbedImage{
										URL: "https://media.discordapp.net/attachments/1192133027455844502/1279723107019653231/972B72F8-6C05-423C-96B3-58ADEA38A8AA.gif?ex=681c6e84&is=681b1d04&hm=0915bbe5b10603582395ad75fba6be8a02d5aa233d9acb0dbc5d4c768185fe86&=&width=318&height=300",
									},
								},
							},
							Username: "Finder Tool", 
							AvatarURL: "https://media.discordapp.net/attachments/1192133027455844502/1279723107019653231/972B72F8-6C05-423C-96B3-58ADEA38A8AA.gif?ex=681c6e84&is=681b1d04&hm=0915bbe5b10603582395ad75fba6be8a02d5aa233d9acb0dbc5d4c768185fe86&=&width=318&height=300",
						}
						
						if err := sendWebhookMessage(webhookURL, tagFoundMessage); err != nil {
							log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Failed to send webhook notification: %v", err)))
						} else {
							log.Println(logPrefix("Success webhook notification sent."))
						}
					}
					
					log.Println(ColorGreen + logPrefix("Found a guild (server) with tags, but continuing to search for more...") + ColorReset)
					printFoundGuildsSummaryForToken(foundGuilds, tokenIndex)
				} else {
					log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Guild (server) (ID: %s, Hash: %d) does not have the tag experiment. Scheduling deletion...", newGuild.ID, hashValue)))
					go func(guildID, guildName string) {
						defer func() {
							if r := recover(); r != nil {
								log.Printf(ColorRed+"%s"+ColorReset, logPrefix(fmt.Sprintf("Recovered from panic during guild deletion: %v", r)))
							}
						}()
						
						time.Sleep(DELETE_DELAY_SECONDS * time.Second)
						log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Deleting guild (server): %s (ID: %s)", guildName, guildID)))
						err := deleteGuild(token, guildID)
						if err != nil {
							log.Printf(ColorRed+"%s"+ColorReset, logPrefix(fmt.Sprintf("Error deleting guild (server) (ID: %s): %v", guildID, err)))
						} else {
							log.Printf(ColorYellow+"%s"+ColorReset, logPrefix(fmt.Sprintf("Guild (server) (ID: %s) deleted.", guildID)))
						}
					}(newGuild.ID, newGuild.Name)
				}
			}()
		}
	}
}

func printFoundGuildsSummaryForToken(foundGuilds map[string]uint32, tokenIndex int) {
	if len(foundGuilds) == 0 {
		return
	}
	
	prefix := ""
	if tokenIndex >= 0 {
		prefix = fmt.Sprintf("[Token %d] ", tokenIndex+1)
	}
	
	log.Println(ColorGreen + prefix + "=== FOUND GUILDS SUMMARY ===" + ColorReset)
	for guildID, hashValue := range foundGuilds {
		log.Printf(ColorGreen+prefix+"Guild ID: %s, Hash: %d"+ColorReset, guildID, hashValue)
	}
	log.Println(ColorGreen + prefix + "==========================" + ColorReset)
}

func main() {
	fmt.Println(ColorRed + `
██████╗ ██╗███████╗ ██████╗ ██████╗ ██████╗ ██████╗    ████████╗  █████╗  ██████╗ 
██╔══██╗██║██╔════╝██╔════╝██╔═══██╗██╔══██╗██╔══██╗   ╚══██╔══╝ ██╔══██╗██╔════╝ 
██║  ██║██║███████╗██║     ██║   ██║██████╔╝██║  ██║      ██║    ███████║███████╗ 
██║  ██║██║╚════██║██║     ██║   ██║██╔══██╗██║  ██║      ██║    ██╔══██║██╔═══██╗
██████╔╝██║███████║╚██████╗╚██████╔╝██║  ██║██████╔╝      ██║    ██║  ██║╚██████╔╝
╚═════╝ ╚═╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚═════╝       ╚═╝    ╚═╝  ╚═╝ ╚═════╝ 
                                  Tool
` + ColorReset)
	fmt.Println("                          @efetutorial")
	fmt.Println("") 

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Choose token input method:")
	fmt.Println("1. Single token")
	fmt.Println("2. Token file (one token per line)")
	fmt.Print("Enter your choice (1 or 2): ")
	choiceInput, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(choiceInput)

	var tokens []string
	var webhookURL string

	fmt.Print("Enter Discord webhook URL (optional, press Enter to skip): ")
	webhookURLInput, _ := reader.ReadString('\n')
	webhookURL = strings.TrimSpace(webhookURLInput)

	fmt.Print("Do you want to use a proxy? (Enter 1 for Yes, Enter for No): ")
	proxyChoiceInput, _ := reader.ReadString('\n')
	proxyChoice := strings.TrimSpace(proxyChoiceInput)
	if proxyChoice == "1" {
		fmt.Print("Proxy file path (default: proxy.txt): ")
		proxyFileInput, _ := reader.ReadString('\n')
		proxyFile := strings.TrimSpace(proxyFileInput)
		if proxyFile == "" {
			proxyFile = "proxy.txt"
		}
		var err error
		proxyList, err = readProxiesFromFile(proxyFile)
		if err != nil {
			log.Fatalf(ColorRed+"Failed to read proxies from file: %v"+ColorReset, err)
			return
		}
		log.Printf(ColorGreen+"%d proxy uploaded."+ColorReset, len(proxyList))
	}

	switch choice {
	case "1":
		fmt.Print("Enter your Discord token: ")
		tokenInput, _ := reader.ReadString('\n')
		token := strings.TrimSpace(tokenInput)
		
		if token == "" {
			log.Fatalln(ColorRed + "Token cannot be empty." + ColorReset)
			return
		}
		
		tokens = append(tokens, token)
	
	case "2":
		fmt.Print("Enter the path to your token file (default: token.txt): ")
		filePathInput, _ := reader.ReadString('\n')
		filePath := strings.TrimSpace(filePathInput)
		
		if filePath == "" {
			filePath = "token.txt"
		}
		
		var err error
		tokens, err = readTokensFromFile(filePath)
		if err != nil {
			log.Fatalf(ColorRed+"Failed to read tokens from file: %v"+ColorReset, err)
			return
		}
		
		log.Printf(ColorGreen+"Successfully loaded %d tokens from %s"+ColorReset, len(tokens), filePath)
	
	default:
		log.Fatalln(ColorRed + "Invalid choice. Please restart the program and enter either 1 or 2." + ColorReset)
		return
	}

	var wg sync.WaitGroup
	
	for i, token := range tokens {
		wg.Add(1)
		go func(tok string, index int) {
			defer wg.Done()
			runForToken(tok, webhookURL, index)
		}(token, i)
		
		time.Sleep(2 * time.Second)
	}
	
	wg.Wait()
}

//@efetutorial tag tool
