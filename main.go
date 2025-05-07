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
	"os"
	"runtime" 
	"strings"
	"time"
	"golang.org/x/sys/windows"
)

const (
	INTERVAL_SECONDS     = 5
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
		} else {
		}
	}
}

func murmurhash3_32_gc_go(key string, seed uint32) uint32 {
	data := []byte(key)
	length := len(data)
	nblocks := length / 4

	h1 := seed

	c1 := uint32(0xcc9e2d51)
	c2 := uint32(0x1b873593)

	for i := 0; i < nblocks; i++ {
		k1 := binary.LittleEndian.Uint32(data[i*4:])

		k1 = uint32(int32(k1) * int32(c1))
		k1 = (k1 << 15) | (k1 >> 17) 
		k1 = uint32(int32(k1) * int32(c2))

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19) 
		h1 = uint32(int32(h1)*int32(5)) + 0xe6546b64
	}

	// Tail
	tail := data[nblocks*4:]
	var k1_tail uint32 = 0
	switch length & 3 {
	case 3:
		k1_tail ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1_tail ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1_tail ^= uint32(tail[0])
		k1_tail = uint32(int32(k1_tail) * int32(c1))
		k1_tail = (k1_tail << 15) | (k1_tail >> 17) 
		k1_tail = uint32(int32(k1_tail) * int32(c2))
		h1 ^= k1_tail
	}

	// Finalization
	h1 ^= uint32(length)
	h1 ^= h1 >> 16
	h1 = uint32(int32(h1) * int32(-2048144789)) 
	h1 ^= h1 >> 13
	h1 = uint32(int32(h1) * int32(-1030157003))
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

var httpClient = &http.Client{Timeout: 10 * time.Second}

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

	req, err := http.NewRequest("POST", DISCORD_API_BASE+"/guilds", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for guild creation: %w", err)
	}
	setCommonHeaders(req, token) 

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for guild creation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK { 
		bodyBytes, _ := io.ReadAll(resp.Body)
		responseStr := string(bodyBytes)
		if resp.StatusCode == http.StatusForbidden {
			var discordError struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			}
			if json.Unmarshal(bodyBytes, &discordError) == nil && discordError.Code == 10008 {
				return nil, fmt.Errorf("failed to create guild, status code: %d. Discord error code: %d (%s). This often indicates account restrictions or that the account is flagged. Response: %s", resp.StatusCode, discordError.Code, discordError.Message, responseStr)
			}
		}
		return nil, fmt.Errorf("failed to create guild, status code: %d, response: %s", resp.StatusCode, responseStr)
	}

	var newGuild Guild
	if err := json.NewDecoder(resp.Body).Decode(&newGuild); err != nil {
		return nil, fmt.Errorf("failed to decode guild creation response: %w", err)
	}
	return &newGuild, nil
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
}

type WebhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

func sendWebhookMessage(webhookURL string, message WebhookMessage) error {
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

func main() {
	fmt.Println(ColorRed + `
██████╗ ██╗███████╗ ██████╗ ██████╗ ██████╗     ████████╗  █████╗  ██████╗ 
██╔══██╗██║██╔════╝██╔════╝██╔═══██╗██╔══██╗    ╚══██╔══╝ ██╔══██╗██╔════╝ 
██║  ██║██║███████╗██║     ██║   ██║██████╔╝       ██║    ███████║███████╗ 
██║  ██║██║╚════██║██║     ██║   ██║██╔══██╗       ██║    ██╔══██║██╔═══██╗
██████╔╝██║███████║╚██████╗╚██████╔╝██║  ██║       ██║    ██║  ██║╚██████╔╝
╚═════╝ ╚═╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═╝       ╚═╝    ╚═╝  ╚═╝ ╚═════╝ 
                                  Tool
` + ColorReset)
	fmt.Println("                          @efetutorial")
	fmt.Println("") 

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your Discord token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		log.Fatalf(ColorRed + "Token cannot be empty." + ColorReset)
		return
	}
	
	fmt.Print("Enter Discord webhook URL (optional, press Enter to skip): ")
	webhookURL, _ := reader.ReadString('\n')
	webhookURL = strings.TrimSpace(webhookURL)

	log.Println("Validating token...")
	user, err := validateToken(token)
	if err != nil {
		log.Fatalf(ColorRed+"Token validation failed: %v"+ColorReset, err)
		return
	}
	log.Printf(ColorGreen+"Token validated. Logged in as: %s#%s (ID: %s)"+ColorReset, user.Username, user.Discriminator, user.ID)
	if user.Bot { 
		log.Fatalf(ColorRed+"Error: The provided token is for a bot account. This script requires a user account token."+ColorReset)
		return
	}

	if webhookURL != "" {
		startMessage := WebhookMessage{
			Embeds: []WebhookEmbed{
				{
					Title:       "Discord Tag Tool Started",
					Description: fmt.Sprintf("Tool started successfully and logged in as %s#%s", user.Username, user.Discriminator),
					Color:       WEBHOOK_SUCCESS_COLOR,
					Fields: []WebhookEmbedField{
						{Name: "User ID", Value: user.ID, Inline: true},
						{Name: "Status", Value: "Starting guild creation process...", Inline: true},
					},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
			Username: "Tag Finder Tool", // Changed from "Discord Tag Tool" to avoid the Discord API restriction
		}
		
		if err := sendWebhookMessage(webhookURL, startMessage); err != nil {
			log.Printf(ColorYellow+"Failed to send webhook notification: %v"+ColorReset, err)
		} else {
			log.Println("Initial webhook notification sent successfully.")
		}
	}

	log.Println("Starting guild (server) creation script...")

	ticker := time.NewTicker(INTERVAL_SECONDS * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Attempting to create a new guild (server)...")
			newGuild, err := createGuild(token)
			if err != nil {
				log.Printf(ColorRed+"Error creating guild (server): %v"+ColorReset, err)
				continue 
			}

			if newGuild == nil || newGuild.ID == "" {
				log.Println(ColorRed + "Failed to create guild (server) or guild ID is empty." + ColorReset)
				continue
			}

			log.Printf("Guild (server) created: %s (ID: %s)", newGuild.Name, newGuild.ID)

			hashKey := fmt.Sprintf("2025-02_skill_trees:%s", newGuild.ID)
			hashValue := murmurhash3_32_gc_go(hashKey, 0) % 10000

			if hashValue >= 10 && hashValue < 20 {
				successMessage := fmt.Sprintf("🎉 FOUND GUILD (SERVER) WITH TAG: %s (ID: %s) HASH: %d 🎉", newGuild.Name, newGuild.ID, hashValue)
				log.Print(ColorGreen + successMessage + ColorReset) 
				
				// Send success notification via webhook
				if webhookURL != "" {
					tagFoundMessage := WebhookMessage{
						Embeds: []WebhookEmbed{
							{
								Title:       "🎉 TAG FOUND! 🎉",
								Description: "A guild with the tag experiment has been found!",
								Color:       WEBHOOK_SUCCESS_COLOR,
								Fields: []WebhookEmbedField{
									{Name: "Guild ID", Value: newGuild.ID, Inline: true},
									{Name: "Guild Name", Value: newGuild.Name, Inline: true},
									{Name: "Hash Value", Value: fmt.Sprintf("%d", hashValue), Inline: true},
								},
								Timestamp: time.Now().Format(time.RFC3339),
							},
						},
						Username: "Tag Finder Tool", 
					}
					
					if err := sendWebhookMessage(webhookURL, tagFoundMessage); err != nil {
						log.Printf(ColorYellow+"Failed to send webhook notification: %v"+ColorReset, err)
					} else {
						log.Println("Success webhook notification sent.")
					}
				}
				
				log.Println(ColorGreen + "Stopping script as a guild (server) with tags has been found." + ColorReset)
				return 
			} else {
				log.Printf(ColorYellow+"Guild (server) (ID: %s, Hash: %d) does not have the tag experiment. Scheduling deletion..."+ColorReset, newGuild.ID, hashValue)
				go func(guildID, guildName string) {
					time.Sleep(DELETE_DELAY_SECONDS * time.Second)
					log.Printf(ColorYellow+"Deleting guild (server): %s (ID: %s)"+ColorReset, guildName, guildID)
					err := deleteGuild(token, guildID)
					if err != nil {
						log.Printf(ColorRed+"Error deleting guild (server) (ID: %s): %v"+ColorReset, guildID, err)
					} else {
						log.Printf(ColorYellow+"Guild (server) (ID: %s) deleted."+ColorReset, guildID)
					}
				}(newGuild.ID, newGuild.Name)
			}
		}
	}
}
//@efetutorial tag tool
