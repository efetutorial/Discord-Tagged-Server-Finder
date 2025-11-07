package proxy

import (
    "bufio"
    "fmt"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"
)

type Manager struct {
    list  []string
    mu    sync.Mutex
    index int
}

func FromFile(filePath string) (*Manager, error) {
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
    return &Manager{list: proxies}, nil
}

func (m *Manager) Next() string {
    m.mu.Lock()
    defer m.mu.Unlock()
    if len(m.list) == 0 {
        return ""
    }
    p := m.list[m.index%len(m.list)]
    m.index++
    return p
}

func (m *Manager) Count() int {
    return len(m.list)
}

func NewHTTPClient(proxyAddr string) *http.Client {
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