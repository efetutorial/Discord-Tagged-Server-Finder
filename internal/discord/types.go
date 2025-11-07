package discord

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