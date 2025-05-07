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
	// ANSI escape codes for colors
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
	// Generic User-Agent to mimic a browser
	BROWSER_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36"
	// X-Super-Properties header (base64 encoded JSON)
	// This value can become outdated. For a real application, this should be obtained dynamically or kept updated.
	// Content (decoded): {"os":"Windows","browser":"Chrome","device":"","system_locale":"en-US","browser_user_agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36","browser_version":"105.0.0.0","os_version":"10","release_channel":"stable","client_build_number":280100,"client_event_source":null}
	X_SUPER_PROPERTIES = "eyJvcyI6IldpbmRvd3MiLCJicm93c2VyIjoiQ2hyb21lIiwiZGV2aWNlIjoiIiwic3lzdGVtX2xvY2FsZSI6ImVuLVVTIiwiYnJvd3Nlcl91c2VyX2FnZW50IjoiTW96aWxsYS81LjAgKFdpbmRvd3MgTlQgMTAuMDsgV2luNjQ7IHg2NCkgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzEwNS4wLjAuMCBTYWZhcmkvNTM3LjM2IiwiYnJvd3Nlcl92ZXJzaW9uIjoiMTA1LjAuMC4wIiwib3NfdmVyc2lvbiI6IjEwIiwicmVsZWFzZV9jaGFubmVsIjoic3RhYmxlIiwiY2xpZW50X2J1aWxkX251bWJlciI6MjgwMTAwLCJjbGllbnRfZXZlbnRfc291cmNlIjpudWxsfQ=="
	X_DISCORD_LOCALE   = "en-US" // Or "tr" for Turkish
	REFERER_URL        = "https://discord.com/channels/@me"
)

// init runs before main and is used here to enable virtual terminal processing on Windows.
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
			// Optional: Log success if needed, or just let it work silently
			// log.Println("Successfully enabled virtual terminal processing for colors on Windows.")
		}
	}
}

// murmurhash3_32_gc_go is a Go port of the provided JavaScript murmurhash3_32_gc function.
func murmurhash3_32_gc_go(key string, seed uint32) uint32 {
	data := []byte(key)
	length := len(data)
	nblocks := length / 4

	h1 := seed

	c1 := uint32(0xcc9e2d51)
	c2 := uint32(0x1b873593)

	// Body
	for i := 0; i < nblocks; i++ {
		k1 := binary.LittleEndian.Uint32(data[i*4:])

		k1 = uint32(int32(k1) * int32(c1))
		k1 = (k1 << 15) | (k1 >> 17) // ROTL32(k1, 15)
		k1 = uint32(int32(k1) * int32(c2))

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19) // ROTL32(h1, 13)
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
		k1_tail = (k1_tail << 15) | (k1_tail >> 17) // ROTL32(k1_tail, 15)
		k1_tail = uint32(int32(k1_tail) * int32(c2))
		h1 ^= k1_tail
	}

	// Finalization
	h1 ^= uint32(length)
	h1 ^= h1 >> 16
	h1 = uint32(int32(h1) * int32(-2048144789)) // Corresponds to Math.imul($, 2246822507)
	h1 ^= h1 >> 13
	h1 = uint32(int32(h1) * int32(-1030157003)) // Corresponds to Math.imul($, 3266489909)
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
	Bot           bool   `json:"bot"` // Added to check if the token is a bot token
}

type CreateGuildPayload struct {
	Name            string        `json:"name"`
	Icon            *string       `json:"icon"` // Use *string for null
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
	req.Header.Set("Content-Type", "application/json") // Usually for POST/PUT
}

func validateToken(token string) (*User, error) {
	req, err := http.NewRequest("GET", DISCORD_API_BASE+"/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for token validation: %w", err)
	}
	// Remove Content-Type for GET, set other common headers
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
	setCommonHeaders(req, token) // Set all common headers including Content-Type

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for guild creation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK { // 201 is typical, 200 can happen
		bodyBytes, _ := io.ReadAll(resp.Body)
		responseStr := string(bodyBytes)
		// Check for specific 403 with code 10008
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
	// For DELETE, Content-Type is not strictly needed for the request body, but other headers are good.
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

	// Discord API returns 204 No Content on successful deletion
	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete guild, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func main() {
	// Print ASCII Art Title and Credit
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
	fmt.Println("") // Add a blank line for spacing

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your Discord token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		log.Fatalf(ColorRed + "Token cannot be empty." + ColorReset)
		return
	}

	log.Println("Validating token...")
	user, err := validateToken(token)
	if err != nil {
		log.Fatalf(ColorRed+"Token validation failed: %v"+ColorReset, err)
		return
	}
	log.Printf(ColorGreen+"Token validated. Logged in as: %s#%s (ID: %s)"+ColorReset, user.Username, user.Discriminator, user.ID)
	if user.Bot { // Double check, though validateToken should catch this.
		log.Fatalf(ColorRed+"Error: The provided token is for a bot account. This script requires a user account token."+ColorReset)
		return
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
				continue // Try again on next tick
			}

			if newGuild == nil || newGuild.ID == "" {
				log.Println(ColorRed + "Failed to create guild (server) or guild ID is empty." + ColorReset)
				continue
			}

			log.Printf("Guild (server) created: %s (ID: %s)", newGuild.Name, newGuild.ID)

			hashKey := fmt.Sprintf("2025-02_skill_trees:%s", newGuild.ID)
			// The JS murmurhash3_32_gc is called with key and implicit seed 0.
			hashValue := murmurhash3_32_gc_go(hashKey, 0) % 10000

			if hashValue >= 10 && hashValue < 20 {
				log.Printf(ColorGreen+"🎉 FOUND GUILD (SERVER) WITH TAG: %s (ID: %s) HASH: %d 🎉"+ColorReset, newGuild.Name, newGuild.ID, hashValue)
				log.Println(ColorGreen + "Stopping script as a guild (server) with tags has been found." + ColorReset)
				return // Exit main, stopping the script
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