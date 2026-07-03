// Package parser resolves live-stream playback URLs and room metadata for
// supported streaming platforms.
//
// It is the open-source stream-resolution layer of the UTLIVED desktop app.
// Each platform implements the Parser interface and self-registers via init(),
// so callers can either look a parser up by platform name (Get) or auto-detect
// it from a room URL (Match).
package parser

import (
	"context"
	"fmt"
	"sync"
)

// RoomInfo describes a live room's basic metadata.
type RoomInfo struct {
	AnchorName string // 主播名 / streamer name
	Title      string // 直播标题 / stream title
	IsLive     bool   // 是否正在直播 / currently live
	IsReplay   bool   // 是否为轮播 / carousel-replay (not a real live)
	Cover      string // 封面图 URL / cover image URL
}

// StreamInfo describes a resolved playback stream.
type StreamInfo struct {
	RecordURL string            // 推荐用于录制的地址 / recommended URL for recording
	FlvURL    string            // FLV 地址（如有）/ FLV URL if available
	Quality   string            // 画质标识 / quality tag
	Headers   map[string]string // 拉流所需 HTTP 头（Referer/UA 等）/ headers required to pull the stream
}

// Parser resolves room info and stream URLs for a single platform.
type Parser interface {
	// Platform returns the short platform identifier, e.g. "bilibili".
	Platform() string
	// Match reports whether this parser handles the given room URL.
	Match(url string) bool
	// GetRoomInfo fetches the room's live status and metadata. cookies is optional.
	GetRoomInfo(ctx context.Context, url, cookies string) (*RoomInfo, error)
	// GetStreamURL resolves the playback stream for the requested quality. cookies is optional.
	GetStreamURL(ctx context.Context, url, quality, cookies string) (*StreamInfo, error)
}

var (
	mu      sync.RWMutex
	parsers = map[string]Parser{}
)

// Register adds a platform parser to the registry.
func Register(p Parser) {
	mu.Lock()
	defer mu.Unlock()
	parsers[p.Platform()] = p
}

// Get returns the parser for a platform identifier.
func Get(platform string) (Parser, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := parsers[platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	return p, nil
}

// Match auto-detects the platform from a room URL and returns its parser.
func Match(url string) (Parser, error) {
	mu.RLock()
	defer mu.RUnlock()
	for _, p := range parsers {
		if p.Match(url) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unrecognized live-stream URL: %s", url)
}

// ListPlatforms returns all registered platform identifiers.
func ListPlatforms() []string {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]string, 0, len(parsers))
	for k := range parsers {
		result = append(result, k)
	}
	return result
}

func init() {
	Register(&BilibiliParser{})
	Register(&DouyuParser{})
}
