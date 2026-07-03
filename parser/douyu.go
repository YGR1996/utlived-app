// Douyu live-stream parser.
// The playback URL is obtained via the getH5PlayV1 endpoint, which requires an
// auth signature computed from parameters returned by the getEncryption endpoint.
package parser

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// guestDeviceID is a well-known anonymous device id used by logged-out web
// clients. It is not a credential; passing a stale login cookie's dy_did here
// instead would clash with the signature and trigger HTTP 403.
const guestDeviceID = "10000000000000000000000000003306"

type DouyuParser struct{}

func (p *DouyuParser) Platform() string { return "douyu" }

func (p *DouyuParser) Match(urlStr string) bool {
	return strings.Contains(urlStr, "douyu.com")
}

var douyuRoomIDRe = regexp.MustCompile(`douyu\.com[^\d]*(\d+)`)

func (p *DouyuParser) extractRoomID(urlStr string) string {
	m := douyuRoomIDRe.FindStringSubmatch(urlStr)
	if len(m) >= 2 {
		return m[1]
	}
	// Fallback: last numeric path segment.
	parts := strings.Split(strings.Split(urlStr, "?")[0], "/")
	return parts[len(parts)-1]
}

// douyuBetardResp is the room field of the betard API response.
type douyuBetardResp struct {
	Room struct {
		OwnerName  string      `json:"owner_name"`
		Nickname   string      `json:"nickname"`
		ShowStatus interface{} `json:"show_status"` // 1=live 2=offline
		VideoLoop  interface{} `json:"videoLoop"`   // 1=carousel
		RoomPic    string      `json:"room_pic"`
	} `json:"room"`
}

func (p *DouyuParser) GetRoomInfo(ctx context.Context, urlStr string, cookies string) (*RoomInfo, error) {
	client := newClient()
	roomID := p.extractRoomID(urlStr)

	// The betard API needs a full browser UA and Referer, otherwise it returns an
	// HTML challenge page. Public streams need no cookie; a stale one causes 403.
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Referer":    fmt.Sprintf("https://www.douyu.com/%s", roomID),
	}

	betardURL := fmt.Sprintf("https://www.douyu.com/betard/%s", roomID)
	body, err := client.Get(ctx, betardURL, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch douyu room info failed: %w", err)
	}

	// Detect WAF interception (HTML instead of JSON).
	if len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("douyu betard API blocked by risk control (returned HTML), retry later")
	}

	var resp douyuBetardResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse douyu room info failed: %w", err)
	}

	name := resp.Room.OwnerName
	if name == "" {
		name = resp.Room.Nickname
	}

	showStatus := toIntValue(resp.Room.ShowStatus)
	isLive := showStatus == 1
	// videoLoop=1 means carousel, not a real live.
	if isLive && toIntValue(resp.Room.VideoLoop) == 1 {
		isLive = false
	}

	return &RoomInfo{
		AnchorName: name,
		IsLive:     isLive,
		IsReplay:   showStatus == 1 && toIntValue(resp.Room.VideoLoop) == 1,
		Cover:      resp.Room.RoomPic,
	}, nil
}

// md5Hex returns the hex-encoded MD5 digest of s.
func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}

// douyuEncryptionResp is the getEncryption API response.
type douyuEncryptionResp struct {
	Error int `json:"error"`
	Data  struct {
		RandStr   string `json:"rand_str"`
		Key       string `json:"key"`
		EncTime   int    `json:"enc_time"`
		EncData   string `json:"enc_data"`
		IsSpecial int    `json:"is_special"`
	} `json:"data"`
}

// getDouyuSignParams fetches signature parameters from the getEncryption
// endpoint, then derives the auth signature via iterated MD5.
func getDouyuSignParams(ctx context.Context, roomID string, did string) (map[string]string, error) {
	client := newClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Referer":    fmt.Sprintf("https://www.douyu.com/%s", roomID),
	}

	keyURL := fmt.Sprintf("https://www.douyu.com/wgapi/livenc/liveweb/websec/getEncryption?did=%s", did)
	body, err := client.Get(ctx, keyURL, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch douyu signature key failed: %w", err)
	}

	var encResp douyuEncryptionResp
	if err := json.Unmarshal(body, &encResp); err != nil {
		return nil, fmt.Errorf("parse douyu signature response failed: %w", err)
	}
	if encResp.Error != 0 {
		return nil, fmt.Errorf("douyu signature API error (%d)", encResp.Error)
	}

	enc := encResp.Data
	ts := time.Now().Unix()

	// Non-special mode signs rid+ts.
	signStr := ""
	if enc.IsSpecial != 1 {
		signStr = fmt.Sprintf("%s%d", roomID, ts)
	}

	auth := enc.RandStr
	for i := 0; i < enc.EncTime; i++ {
		auth = md5Hex(auth + enc.Key)
	}
	auth = md5Hex(auth + enc.Key + signStr)

	return map[string]string{
		"enc_data": enc.EncData,
		"did":      did,
		"ts":       fmt.Sprintf("%d", ts),
		"auth":     auth,
	}, nil
}

func (p *DouyuParser) GetStreamURL(ctx context.Context, urlStr string, quality string, cookies string) (*StreamInfo, error) {
	client := newClient()
	roomID := p.extractRoomID(urlStr)
	did := guestDeviceID

	// 1. Signature parameters.
	signParams, err := getDouyuSignParams(ctx, roomID, did)
	if err != nil {
		return nil, fmt.Errorf("douyu signing failed: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Referer":      fmt.Sprintf("https://www.douyu.com/%s", roomID),
	}

	// Quality → rate code.
	rateMap := map[string]string{
		"OD": "0", "BD": "0", "UHD": "3", "HD": "2", "SD": "1", "LD": "1",
	}
	q := strings.ToUpper(quality)
	rate := rateMap["OD"]
	if v, ok := rateMap[q]; ok {
		rate = v
	}

	// 2. Call getH5PlayV1 with the signature.
	postValues := url.Values{
		"enc_data": {signParams["enc_data"]},
		"tt":       {signParams["ts"]},
		"did":      {signParams["did"]},
		"auth":     {signParams["auth"]},
		"cdn":      {""},
		"rate":     {rate},
		"hevc":     {"0"},
		"fa":       {"0"},
		"ive":      {"0"},
	}

	streamAPI := fmt.Sprintf("https://www.douyu.com/lapi/live/getH5PlayV1/%s", roomID)
	body, err := client.Post(ctx, streamAPI, headers, []byte(postValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("fetch douyu stream failed: %w", err)
	}

	// Two-phase parse: data is a string on error, an object on success.
	var raw struct {
		Error int             `json:"error"`
		Msg   string          `json:"msg"`
		Data  json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse douyu stream response failed: %w", err)
	}

	if raw.Error != 0 {
		return nil, fmt.Errorf("douyu stream API error (%d): %s", raw.Error, raw.Msg)
	}

	var data struct {
		RTMPURL  string `json:"rtmp_url"`
		RTMPLive string `json:"rtmp_live"`
		RTMPCdn  string `json:"rtmp_cdn"`
	}

	if err := json.Unmarshal(raw.Data, &data); err != nil {
		return nil, fmt.Errorf("parse douyu stream URL failed: %w", err)
	}

	if data.RTMPURL == "" || data.RTMPLive == "" {
		return nil, fmt.Errorf("empty douyu stream URL (streamer may be offline)")
	}

	flvURL := data.RTMPURL + "/" + data.RTMPLive
	flvURL = strings.Replace(flvURL, "http://", "https://", 1)

	return &StreamInfo{
		FlvURL:    flvURL,
		RecordURL: flvURL,
		Quality:   q,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Referer":    "https://www.douyu.com/",
		},
	}, nil
}

// toIntValue converts an interface{} to int, tolerating JSON's float64/string forms.
func toIntValue(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	}
	return 0
}
