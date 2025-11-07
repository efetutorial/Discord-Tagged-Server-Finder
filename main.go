//@efetutorial
package main

import (
    "bufio"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"
    "projem/internal/app"
    "projem/internal/console"
    "projem/internal/proxy"
    "projem/internal/webhook"
)

func main() {
    console.EnableVT()
    fmt.Println(console.ColorRed + `
██████╗ ██╗███████╗ ██████╗ ██████╗ ██████╗ ██████╗    ████████╗  █████╗  ██████╗ 
██╔══██╗██║██╔════╝██╔════╝██╔═══██╗██╔══██╗██╔══██╗   ╚══██╔══╝ ██╔══██╗██╔════╝ 
██║  ██║██║███████╗██║     ██║   ██║██████╔╝██║  ██║      ██║    ███████║███████╗ 
██║  ██║██║╚════██║██║     ██║   ██║██╔══██╗██║  ██║      ██║    ██╔══██║██╔═══██╗
██████╔╝██║███████║╚██████╗╚██████╔╝██║  ██║██████╔╝      ██║    ██║  ██║╚██████╔╝
╚═════╝ ╚═╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚═════╝       ╚═╝    ╚═╝  ╚═╝ ╚═════╝ 
                                  Tool
` + console.ColorReset)
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

    var pm *proxy.Manager
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
        pm, err = proxy.FromFile(proxyFile)
        if err != nil {
            log.Fatalf(console.ColorRed+"Failed to read proxies from file: %v"+console.ColorReset, err)
            return
        }
        log.Printf(console.ColorGreen+"%d proxy uploaded."+console.ColorReset, pm.Count())
    }

    switch choice {
    case "1":
        fmt.Print("Enter your Discord token: ")
        tokenInput, _ := reader.ReadString('\n')
        token := strings.TrimSpace(tokenInput)
        if token == "" {
            log.Fatalln(console.ColorRed + "Token cannot be empty." + console.ColorReset)
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
        tokens, err = app.ReadTokensFromFile(filePath)
        if err != nil {
            log.Fatalf(console.ColorRed+"Failed to read tokens from file: %v"+console.ColorReset, err)
            return
        }
        log.Printf(console.ColorGreen+"Successfully loaded %d tokens from %s"+console.ColorReset, len(tokens), filePath)
    default:
        log.Fatalln(console.ColorRed + "Invalid choice. Please restart the program and enter either 1 or 2." + console.ColorReset)
        return
    }

    var wg sync.WaitGroup
    sender := webhook.NewSender()
    for i, token := range tokens {
        wg.Add(1)
        go func(tok string, index int) {
            defer wg.Done()
            var httpClient *http.Client
            if pm != nil {
                httpClient = proxy.NewHTTPClient(pm.Next())
            } else {
                httpClient = &http.Client{Timeout: 10 * time.Second}
            }
            app.RunForToken(tok, webhookURL, index, httpClient, sender)
        }(token, i)
        time.Sleep(2 * time.Second)
    }
    wg.Wait()
}