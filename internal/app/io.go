package app

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

func ReadTokensFromFile(filePath string) ([]string, error) {
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