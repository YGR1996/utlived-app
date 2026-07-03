// Bilibili live-stream parser.
// Bilibili's live APIs are relatively stable and require no request signing,
// which makes this the simplest of the supported platforms.
package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type BilibiliParser struct{}

func (p *BilibiliParser) Platform() string { return "bilibili" }

func (p *BilibiliParser) Match(url string) bool {
	return strings.Contains(url, "live.bilibili.com")
}

// biliRoomInitResp is the room_init API response.
type biliRoomInitResp struct {
	Code int `json:"code"`
	Data struct {
		UID        int   `json:"uid"`
		LiveStatus int   `json:"live_status"` // 0=offline 1=live 2=carousel
		LiveTime   int64 `json:"live_time"`
	} `json:"data"`
}

// biliMasterInfoResp is the Master/info API response.
type biliMasterInfoResp struct {
	Data struct {
		Info struct {
			Uname string `json:"uname"`
		} `json:"info"`
	} `json:"data"`
}

// biliH5InfoResp is the H5 room info API response (used for the title).
type biliH5InfoResp struct {
	Data struct {
		RoomInfo struct {
			Title string `json:"title"`
		} `json:"room_info"`
	} `json:"data"`
}

// biliPlayURLResp is the getRoomPlayInfo API response.
type biliPlayURLResp struct {
	Code int `json:"code"`
	Data struct {
		LiveStatus  int `json:"live_status"`
		PlayURLInfo struct {
			PlayURL struct {
				Stream []struct {
					Format []struct {
						FormatName string `json:"format_name"`
						Codec      []struct {
							CurrentQN int    `json:"current_qn"`
							BaseURL   string `json:"base_url"`
							URLInfo   []struct {
								Host  string `json:"host"`
								Extra string `json:"extra"`
							} `json:"url_info"`
						} `json:"codec"`
					} `json:"format"`
				} `json:"stream"`
			} `json:"playurl"`
		} `json:"playurl_info"`
	} `json:"data"`
}

func (p *BilibiliParser) extractRoomID(url string) string {
	// https://live.bilibili.com/12345?... → 12345
	url = strings.Split(url, "?")[0]
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func (p *BilibiliParser) GetRoomInfo(ctx context.Context, url string, cookies string) (*RoomInfo, error) {
	client := newClient()
	roomID := p.extractRoomID(url)

	headers := map[string]string{
		"Accept-Language": "zh-CN,zh;q=0.8",
		"Referer":         "https://live.bilibili.com",
	}
	if cookies != "" {
		headers["Cookie"] = cookies
	}

	// 1. Room status
	initAPI := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/room_init?id=%s", roomID)
	initBody, err := client.Get(ctx, initAPI, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch bilibili room info failed: %w", err)
	}

	var initResp biliRoomInitResp
	if err := json.Unmarshal(initBody, &initResp); err != nil {
		return nil, fmt.Errorf("parse bilibili room info failed: %w", err)
	}

	uid := initResp.Data.UID
	isLive := initResp.Data.LiveStatus == 1
	isReplay := false
	if initResp.Data.LiveStatus == 2 {
		// Official rooms (e.g. 2233) report live_status 2 with a negative/near-zero
		// live_time when offline; a real carousel has live_time > 0.
		if initResp.Data.LiveTime > 0 {
			isReplay = true
		} else {
			isReplay = false
			isLive = false
		}
	}

	// 2. Streamer name
	masterAPI := fmt.Sprintf("https://api.live.bilibili.com/live_user/v1/Master/info?uid=%d", uid)
	masterBody, err := client.Get(ctx, masterAPI, headers)
	if err != nil {
		return &RoomInfo{IsLive: isLive, IsReplay: isReplay}, nil
	}

	var masterResp biliMasterInfoResp
	json.Unmarshal(masterBody, &masterResp)

	// 3. Title
	h5API := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/index/getH5InfoByRoom?room_id=%s", roomID)
	h5Headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Linux; Android 11) AppleWebKit/537.36 Chrome/87.0.4280.141 Mobile Safari/537.36",
		"Referer":    "https://live.bilibili.com",
	}
	if cookies != "" {
		h5Headers["Cookie"] = cookies
	}
	h5Body, _ := client.Get(ctx, h5API, h5Headers)
	var h5Resp biliH5InfoResp
	json.Unmarshal(h5Body, &h5Resp)

	return &RoomInfo{
		AnchorName: masterResp.Data.Info.Uname,
		IsLive:     isLive,
		IsReplay:   isReplay,
		Title:      h5Resp.Data.RoomInfo.Title,
	}, nil
}

func (p *BilibiliParser) GetStreamURL(ctx context.Context, url string, quality string, cookies string) (*StreamInfo, error) {
	client := newClient()
	roomID := p.extractRoomID(url)

	// Quality code mapping.
	qualityMap := map[string]string{
		"OD": "10000", "BD": "400", "UHD": "250", "HD": "150", "SD": "80", "LD": "80",
	}
	qn := qualityMap["OD"]
	q := strings.ToUpper(quality)
	if v, ok := qualityMap[q]; ok {
		qn = v
	}

	headers := map[string]string{
		"Accept-Language": "zh-CN,zh;q=0.8",
		"Referer":         "https://live.bilibili.com",
		"Origin":          "https://live.bilibili.com",
	}
	if cookies != "" {
		headers["Cookie"] = cookies
	}

	// Prefer the v1 endpoint (more permissive for logged-out clients).
	recordURL := p.tryV1API(ctx, client, roomID, qn, headers)

	// Fall back to the v2 endpoint.
	if recordURL == "" {
		playAPI := fmt.Sprintf(
			"https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo?room_id=%s&protocol=0,1&format=0,1,2&codec=0,1,2&qn=%s&platform=web&ptype=8&dolby=5&panorama=1&hdr_type=0,1",
			roomID, qn,
		)
		body, err := client.Get(ctx, playAPI, headers)
		if err != nil {
			return nil, fmt.Errorf("fetch bilibili stream failed: %w", err)
		}

		var playResp biliPlayURLResp
		if err := json.Unmarshal(body, &playResp); err != nil {
			return nil, fmt.Errorf("parse bilibili stream failed: %w", err)
		}

		if playResp.Data.LiveStatus == 0 {
			return nil, fmt.Errorf("streamer is offline")
		}

		streams := playResp.Data.PlayURLInfo.PlayURL.Stream
		if len(streams) == 0 {
			return nil, fmt.Errorf("bilibili returned no stream data")
		}

		// Pick the highest current_qn codec, preferring non-fmp4 (flv/ts) when tied.
		var bestCodec *struct {
			CurrentQN int    `json:"current_qn"`
			BaseURL   string `json:"base_url"`
			URLInfo   []struct {
				Host  string `json:"host"`
				Extra string `json:"extra"`
			} `json:"url_info"`
		}

		for _, s := range streams {
			for _, f := range s.Format {
				for _, c := range f.Codec {
					cObj := c // avoid capturing the loop variable's address
					if bestCodec == nil {
						bestCodec = &cObj
						continue
					}
					if c.CurrentQN > bestCodec.CurrentQN {
						bestCodec = &cObj
					} else if c.CurrentQN == bestCodec.CurrentQN && f.FormatName != "fmp4" {
						bestCodec = &cObj
					}
				}
			}
		}

		if bestCodec == nil || len(bestCodec.URLInfo) == 0 {
			return nil, fmt.Errorf("no usable codec stream found")
		}

		recordURL = bestCodec.URLInfo[0].Host + bestCodec.BaseURL + bestCodec.URLInfo[0].Extra
	}

	return &StreamInfo{
		RecordURL: recordURL,
		Quality:   q,
		Headers: map[string]string{
			"Referer": "https://live.bilibili.com",
		},
	}, nil
}

// tryV1API tries the legacy v1 endpoint, which can hand logged-out clients an
// unauthenticated high-quality FLV stream (e.g. qn 10000 原画 or 400 蓝光).
func (p *BilibiliParser) tryV1API(ctx context.Context, client *httpClient, roomID, qn string, headers map[string]string) string {
	v1API := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/playUrl?cid=%s&qn=%s&platform=web", roomID, qn)
	body, err := client.Get(ctx, v1API, headers)
	if err != nil {
		return ""
	}

	var v1Resp struct {
		Code int `json:"code"`
		Data struct {
			Durl []struct {
				URL string `json:"url"`
			} `json:"durl"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &v1Resp); err != nil || v1Resp.Code != 0 || len(v1Resp.Data.Durl) == 0 {
		return ""
	}

	for _, durl := range v1Resp.Data.Durl {
		if strings.Contains(durl.URL, "d1--cn-gotcha") {
			return durl.URL
		}
	}
	return v1Resp.Data.Durl[len(v1Resp.Data.Durl)-1].URL
}
