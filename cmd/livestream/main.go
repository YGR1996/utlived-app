// Command livestream resolves a live-stream playback URL from a room URL.
//
//	livestream https://live.bilibili.com/12345
//	livestream -q HD -json https://www.douyu.com/9999
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/YGR1996/utlived-app/parser"
)

func main() {
	quality := flag.String("q", "OD", "quality: OD/BD/UHD/HD/SD/LD")
	cookie := flag.String("cookie", "", "optional cookie string")
	asJSON := flag.Bool("json", false, "output as JSON")
	timeout := flag.Duration("timeout", 30*time.Second, "request timeout")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: livestream [flags] <room-url>\n\n")
		fmt.Fprintf(os.Stderr, "Resolve the real playback stream URL for a live room.\n")
		fmt.Fprintf(os.Stderr, "Supported platforms: %v\n\n", parser.ListPlatforms())
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	roomURL := flag.Arg(0)

	p, err := parser.Match(roomURL)
	if err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	info, err := p.GetRoomInfo(ctx, roomURL, *cookie)
	if err != nil {
		fatal(err)
	}

	out := result{
		Platform:   p.Platform(),
		AnchorName: info.AnchorName,
		Title:      info.Title,
		IsLive:     info.IsLive,
		IsReplay:   info.IsReplay,
	}

	if info.IsLive {
		stream, err := p.GetStreamURL(ctx, roomURL, *quality, *cookie)
		if err != nil {
			fatal(err)
		}
		out.Quality = stream.Quality
		out.StreamURL = stream.RecordURL
		out.Headers = stream.Headers
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	fmt.Printf("platform : %s\n", out.Platform)
	fmt.Printf("anchor   : %s\n", out.AnchorName)
	if out.Title != "" {
		fmt.Printf("title    : %s\n", out.Title)
	}
	fmt.Printf("live     : %v\n", out.IsLive)
	if !out.IsLive {
		fmt.Println("(streamer is offline — no stream URL)")
		return
	}
	fmt.Printf("quality  : %s\n", out.Quality)
	fmt.Printf("stream   : %s\n", out.StreamURL)
	if len(out.Headers) > 0 {
		fmt.Println("headers  :")
		for k, v := range out.Headers {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

type result struct {
	Platform   string            `json:"platform"`
	AnchorName string            `json:"anchor_name"`
	Title      string            `json:"title,omitempty"`
	IsLive     bool              `json:"is_live"`
	IsReplay   bool              `json:"is_replay"`
	Quality    string            `json:"quality,omitempty"`
	StreamURL  string            `json:"stream_url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
