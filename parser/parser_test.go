package parser

import "testing"

func TestMatchRouting(t *testing.T) {
	cases := []struct {
		url  string
		want string // expected platform, "" means no match
	}{
		{"https://live.bilibili.com/12345", "bilibili"},
		{"https://live.bilibili.com/12345?spm_id_from=x", "bilibili"},
		{"https://www.douyu.com/9999", "douyu"},
		{"https://www.douyu.com/topic/xyz?rid=9999", "douyu"},
		{"https://www.twitch.tv/someone", ""},
	}
	for _, c := range cases {
		p, err := Match(c.url)
		if c.want == "" {
			if err == nil {
				t.Errorf("Match(%q): expected no match, got %s", c.url, p.Platform())
			}
			continue
		}
		if err != nil {
			t.Errorf("Match(%q): unexpected error: %v", c.url, err)
			continue
		}
		if p.Platform() != c.want {
			t.Errorf("Match(%q): got %s, want %s", c.url, p.Platform(), c.want)
		}
	}
}

func TestExtractRoomID(t *testing.T) {
	b := &BilibiliParser{}
	if got := b.extractRoomID("https://live.bilibili.com/12345?a=b"); got != "12345" {
		t.Errorf("bilibili extractRoomID: got %q, want 12345", got)
	}

	d := &DouyuParser{}
	if got := d.extractRoomID("https://www.douyu.com/9999?rid=9999"); got != "9999" {
		t.Errorf("douyu extractRoomID: got %q, want 9999", got)
	}
}

func TestListPlatforms(t *testing.T) {
	got := ListPlatforms()
	if len(got) != 2 {
		t.Fatalf("ListPlatforms: got %d platforms, want 2 (%v)", len(got), got)
	}
}
