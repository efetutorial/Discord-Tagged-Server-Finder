package app

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"
    "projem/internal/console"
    "projem/internal/discord"
    "projem/internal/hash"
    "projem/internal/webhook"
)

const (
    INTERVAL_SECONDS      = 67
    DELETE_DELAY_SECONDS  = 2
    SERVER_NAME_CONST     = "Tag server"
    WEBHOOK_SUCCESS_COLOR = 5763719
)

func RunForToken(token string, webhookURL string, tokenIndex int, httpClient *http.Client, sender *webhook.Sender) {
    prefix := ""
    if tokenIndex >= 0 {
        prefix = fmt.Sprintf("[Token %d] ", tokenIndex+1)
    }
    logPrefix := func(message string) string { return prefix + message }

    client := discord.NewClient(token, httpClient)
    log.Println(logPrefix("Validating token..."))
    user, err := client.ValidateToken()
    if err != nil {
        log.Printf(console.ColorRed+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Token validation failed: %v", err)))
        return
    }
    log.Printf(console.ColorGreen+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Token validated. Logged in as: %s#%s (ID: %s)", user.Username, user.Discriminator, user.ID)))
    if user.Bot {
        log.Printf(console.ColorRed+"%s"+console.ColorReset, logPrefix("Error: The provided token is for a bot account. This script requires a user account token."))
        return
    }

    if webhookURL != "" && sender != nil {
        startMsg := webhook.Message{
            Embeds: []webhook.Embed{
                {
                    Title:       fmt.Sprintf("Discord Tag Tool Started - Token %d", tokenIndex+1),
                    Description: fmt.Sprintf("Tool started successfully and logged in as %s#%s", user.Username, user.Discriminator),
                    Color:       WEBHOOK_SUCCESS_COLOR,
                    Fields: []webhook.Field{
                        {Name: "User ID", Value: user.ID, Inline: true},
                        {Name: "Status", Value: "Starting guild creation process...", Inline: true},
                        {Name: "Token Index", Value: fmt.Sprintf("%d", tokenIndex+1), Inline: true},
                    },
                    Timestamp: time.Now().Format(time.RFC3339),
                },
            },
            Username: "Tag Finder Tool",
        }
        if err := sender.Send(webhookURL, startMsg); err != nil {
            log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Failed to send webhook notification: %v", err)))
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

    for {
        select {
        case <-ticker.C:
            func() {
                defer func() {
                    if r := recover(); r != nil {
                        log.Printf(console.ColorRed+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Recovered from panic: %v", r)))
                    }
                }()

                log.Println(logPrefix("Attempting to create a new guild (server)..."))
                newGuild, err := client.CreateGuild(SERVER_NAME_CONST)
                if err != nil {
                    errorMessage := err.Error()
                    errorCode := 0
                    if strings.Contains(errorMessage, "Error code:") || strings.Contains(errorMessage, "error code:") {
                        codeParts := strings.Split(strings.ToLower(errorMessage), "error code:")
                        if len(codeParts) > 1 {
                            fmt.Sscanf(codeParts[1], "%d", &errorCode)
                        }
                    }
                    solution := discord.SuggestSolution(errorCode)
                    log.Printf(console.ColorRed+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Error creating guild (server): %v", err)))
                    log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Suggestion: %s", solution)))
                    if errorCode == 10008 {
                        log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix("Waiting for 5 minutes before attempting again..."))
                        time.Sleep(5 * time.Minute)
                    }
                    return
                }
                if newGuild == nil || newGuild.ID == "" {
                    log.Println(console.ColorRed + logPrefix("Failed to create guild (server) or guild ID is empty.") + console.ColorReset)
                    return
                }
                log.Printf(logPrefix(fmt.Sprintf("Guild (server) created: %s (ID: %s)", newGuild.Name, newGuild.ID)))

                hashKey := fmt.Sprintf("2025-02_skill_trees:%s", newGuild.ID)
                hashValue := hash.Murmur3_32(hashKey, 0) % 10000

                if hashValue >= 10 && hashValue < 20 {
                    successMessage := fmt.Sprintf("🎉 FOUND GUILD (SERVER) WITH TAG: %s (ID: %s) HASH: %d 🎉", newGuild.Name, newGuild.ID, hashValue)
                    log.Print(console.ColorGreen + logPrefix(successMessage) + console.ColorReset)

                    foundGuilds[newGuild.ID] = hashValue
                    inviteLink, err := client.CreateInvite(newGuild.ID)
                    if err != nil {
                        log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Failed to create invite: %v. Using direct server link instead.", err)))
                        inviteLink = fmt.Sprintf("https://discord.com/channels/%s", newGuild.ID)
                    } else {
                        log.Printf(console.ColorGreen+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Created invite link: %s", inviteLink)))
                    }

                    if webhookURL != "" && sender != nil {
                        msg := webhook.Message{
                            Content: "@everyone ADAMI SİKERİM BULDU LAN",
                            Embeds: []webhook.Embed{
                                {
                                    Title:       "🎉 BEYLER BULDUK 🎉",
                                    Description: "cetrefilli adamlar klübü siker",
                                    Color:       WEBHOOK_SUCCESS_COLOR,
                                    Fields: []webhook.Field{
                                        {Name: "Guild ID", Value: newGuild.ID, Inline: true},
                                        {Name: "Guild Name", Value: newGuild.Name, Inline: true},
                                        {Name: "Hash Value", Value: fmt.Sprintf("%d", hashValue), Inline: true},
                                        {Name: "Düşen hesap", Value: fmt.Sprintf("%s#%s (ID: %s)", user.Username, user.Discriminator, user.ID), Inline: false},
                                        {Name: "Discord sunucusu", Value: inviteLink, Inline: false},
                                        {Name: "Token Index", Value: fmt.Sprintf("%d", tokenIndex+1), Inline: false},
                                    },
                                    Timestamp: time.Now().Format(time.RFC3339),
                                    Image: &webhook.Image{URL: "https://media.discordapp.net/attachments/1192133027455844502/1279723107019653231/972B72F8-6C05-423C-96B3-58ADEA38A8AA.gif?ex=681c6e84&is=681b1d04&hm=0915bbe5b10603582395ad75fba6be8a02d5aa233d9acb0dbc5d4c768185fe86&=&width=318&height=300"},
                                },
                            },
                            Username:  "Finder Tool",
                            AvatarURL: "https://media.discordapp.net/attachments/1192133027455844502/1279723107019653231/972B72F8-6C05-423C-96B3-58ADEA38A8AA.gif?ex=681c6e84&is=681b1d04&hm=0915bbe5b10603582395ad75fba6be8a02d5aa233d9acb0dbc5d4c768185fe86&=&width=318&height=300",
                        }
                        if err := sender.Send(webhookURL, msg); err != nil {
                            log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Failed to send webhook notification: %v", err)))
                        } else {
                            log.Println(logPrefix("Success webhook notification sent."))
                        }
                    }

                    log.Println(console.ColorGreen + logPrefix("Found a guild (server) with tags, but continuing to search for more...") + console.ColorReset)
                    PrintFoundGuildsSummary(foundGuilds, tokenIndex)
                } else {
                    log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Guild (server) (ID: %s, Hash: %d) does not have the tag experiment. Scheduling deletion...", newGuild.ID, hashValue)))
                    go func(guildID, guildName string) {
                        defer func() {
                            if r := recover(); r != nil {
                                log.Printf(console.ColorRed+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Recovered from panic during guild deletion: %v", r)))
                            }
                        }()
                        time.Sleep(DELETE_DELAY_SECONDS * time.Second)
                        log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Deleting guild (server): %s (ID: %s)", guildName, guildID)))
                        if err := client.DeleteGuild(guildID); err != nil {
                            log.Printf(console.ColorRed+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Error deleting guild (server) (ID: %s): %v", guildID, err)))
                        } else {
                            log.Printf(console.ColorYellow+"%s"+console.ColorReset, logPrefix(fmt.Sprintf("Guild (server) (ID: %s) deleted.", guildID)))
                        }
                    }(newGuild.ID, newGuild.Name)
                }
            }()
        }
    }
}

func PrintFoundGuildsSummary(foundGuilds map[string]uint32, tokenIndex int) {
    if len(foundGuilds) == 0 {
        return
    }
    prefix := ""
    if tokenIndex >= 0 {
        prefix = fmt.Sprintf("[Token %d] ", tokenIndex+1)
    }
    log.Println(console.ColorGreen + prefix + "=== FOUND GUILDS SUMMARY ===" + console.ColorReset)
    for guildID, hashValue := range foundGuilds {
        log.Printf(console.ColorGreen+prefix+"Guild ID: %s, Hash: %d"+console.ColorReset, guildID, hashValue)
    }
    log.Println(console.ColorGreen + prefix + "==========================" + console.ColorReset)
}