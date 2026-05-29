package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CachingClient implements Client and caches CompletionRequests.
type CachingClient struct {
	client   Client
	cacheDir string
	cacheMap map[string]CompletionResponse
	mu       sync.RWMutex
}

// Ensure CachingClient implements Client.
var _ Client = (*CachingClient)(nil)

type cacheKeyData struct {
	Model       string
	Messages    []Message
	Temperature *float64
	Seed        *int
	JSONMode    bool
}

// NewCachingClient creates a CachingClient that wraps an underlying Client.
func NewCachingClient(client Client, cacheDir string) (*CachingClient, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	cc := &CachingClient{
		client:   client,
		cacheDir: cacheDir,
		cacheMap: make(map[string]CompletionResponse),
	}

	cacheFilePath := filepath.Join(cacheDir, "llm.json")
	if _, err := os.Stat(cacheFilePath); err == nil {
		data, err := os.ReadFile(cacheFilePath)
		if err != nil {
			return nil, fmt.Errorf("read cache file: %w", err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &cc.cacheMap); err != nil {
				return nil, fmt.Errorf("unmarshal cache file: %w", err)
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat cache file: %w", err)
	}

	if err := updateGitignore(); err != nil {
		// Log a warning or return error; returning error is safer to ensure it succeeds
		return nil, fmt.Errorf("update gitignore: %w", err)
	}

	return cc, nil
}

// Complete implements Client by caching requests and returning cached responses if possible.
func (cc *CachingClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	hash, err := hashRequest(req)
	if err != nil {
		return nil, fmt.Errorf("hash request: %w", err)
	}

	cc.mu.RLock()
	resp, ok := cc.cacheMap[hash]
	cc.mu.RUnlock()

	if ok {
		return &resp, nil
	}

	actualResp, err := cc.client.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.cacheMap[hash] = *actualResp

	cacheFilePath := filepath.Join(cc.cacheDir, "llm.json")
	data, err := json.MarshalIndent(cc.cacheMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal cache: %w", err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return nil, fmt.Errorf("write cache file: %w", err)
	}

	return actualResp, nil
}

func hashRequest(req *CompletionRequest) (string, error) {
	kd := cacheKeyData{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		Seed:        req.Seed,
		JSONMode:    req.JSONMode,
	}
	data, err := json.Marshal(kd)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func findGitignore() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, ".gitignore")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func updateGitignore() error {
	gitIgnorePath := findGitignore()
	if gitIgnorePath == "" {
		return nil
	}

	data, err := os.ReadFile(gitIgnorePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	hasSugo := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".sugo" || trimmed == ".sugo/" {
			hasSugo = true
			break
		}
	}

	if !hasSugo {
		content := string(data)
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += ".sugo/\n"
		if err := os.WriteFile(gitIgnorePath, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}
