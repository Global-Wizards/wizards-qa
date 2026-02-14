package scout

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// PageMeta contains metadata extracted from scouting a game URL.
type PageMeta struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Framework     string            `json:"framework"`
	CanvasFound   bool              `json:"canvasFound"`
	ScriptSrcs    []string          `json:"scriptSrcs"`
	MetaTags      map[string]string `json:"metaTags"`
	BodySnippet   string            `json:"bodySnippet"`
	Links         []string          `json:"links"`
	Error         string            `json:"error,omitempty"`
	ScreenshotB64 string            `json:"screenshotB64,omitempty"`
	JSGlobals     []string          `json:"jsGlobals,omitempty"`
	// ClickStrategy records which click dispatch method was selected for this page.
	ClickStrategy string `json:"clickStrategy,omitempty"`
	// Screenshots holds base64-encoded JPEG screenshots of multiple game states
	// (initial load, after canvas click, after button click). The first entry
	// is the same as ScreenshotB64 for backward compatibility.
	Screenshots []string `json:"screenshots,omitempty"`
}

// HeadlessConfig configures headless browser scouting.
type HeadlessConfig struct {
	Enabled          bool
	Width            int
	Height           int
	DevicePixelRatio float64
	Timeout          time.Duration
	ScreenshotPath   string // if non-empty, save screenshot PNG here
	DeviceCategory   string // viewport device category (e.g. "iPhone", "iPad", "Desktop")
}

const (
	maxScriptSrcs      = 20
	maxLinks           = 20
	maxBodySnippet     = 2000
	defaultFetchTimeout = 10 * time.Second
	maxBodyRead        = 5 * 1024 * 1024 // 5MB
)

// ScoutURL fetches a URL and extracts page metadata for game analysis.
// Pass 0 for timeout to use the default (10s).
func ScoutURL(ctx context.Context, gameURL string, timeout time.Duration) (*PageMeta, error) {
	if timeout <= 0 {
		timeout = defaultFetchTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", gameURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "WizardsQA-Scout/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, gameURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyRead))
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return ParseHTML(string(body)), nil
}

// ParseHTML extracts page metadata from raw HTML. Exported for testing.
func ParseHTML(rawHTML string) *PageMeta {
	meta := &PageMeta{
		MetaTags: make(map[string]string),
	}

	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		meta.Error = fmt.Sprintf("HTML parse error: %v", err)
		return meta
	}

	var bodyText strings.Builder
	var allScriptContent strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					meta.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				handleMeta(n, meta)
			case "script":
				HandleScript(n, meta, &allScriptContent)
			case "canvas":
				meta.CanvasFound = true
			case "a":
				handleLink(n, meta)
			}

			// Collect body inner text for snippet
			if n.Data == "body" {
				collectBodyText(n, &bodyText)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	snippet := bodyText.String()
	if len(snippet) > maxBodySnippet {
		snippet = snippet[:maxBodySnippet]
	}
	meta.BodySnippet = strings.TrimSpace(snippet)

	meta.Framework = DetectFramework(meta.ScriptSrcs, allScriptContent.String())

	return meta
}

func handleMeta(n *html.Node, meta *PageMeta) {
	var name, content, property string
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "name":
			name = strings.ToLower(a.Val)
		case "content":
			content = a.Val
		case "property":
			property = strings.ToLower(a.Val)
		}
	}

	if name != "" && content != "" {
		meta.MetaTags[name] = content
		switch name {
		case "description":
			meta.Description = content
		case "generator":
			meta.MetaTags["generator"] = content
		}
	}
	if property != "" && content != "" {
		meta.MetaTags[property] = content
	}
}

// HandleScript processes a <script> node, collecting src attributes and inline text.
// Exported for testing.
func HandleScript(n *html.Node, meta *PageMeta, allScriptContent *strings.Builder) {
	for _, a := range n.Attr {
		if strings.ToLower(a.Key) == "src" && a.Val != "" {
			if len(meta.ScriptSrcs) < maxScriptSrcs {
				meta.ScriptSrcs = append(meta.ScriptSrcs, a.Val)
			}
			return
		}
	}
	// Inline script — walk all children for framework detection (capped at 512KB)
	const maxScriptContentBytes = 512 * 1024
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			if allScriptContent.Len() >= maxScriptContentBytes {
				return
			}
			allScriptContent.WriteString(c.Data)
			allScriptContent.WriteString("\n")
		}
	}
}

func handleLink(n *html.Node, meta *PageMeta) {
	for _, a := range n.Attr {
		if strings.ToLower(a.Key) == "href" && a.Val != "" {
			if len(meta.Links) < maxLinks {
				meta.Links = append(meta.Links, a.Val)
			}
			return
		}
	}
}

func collectBodyText(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if sb.Len() < maxBodySnippet {
			collectBodyText(c, sb)
		}
	}
}

// DetectFramework identifies a game framework from script sources and inline content.
// Exported for testing.
func DetectFramework(scriptSrcs []string, inlineScripts string) string {
	combined := strings.ToLower(inlineScripts)
	for _, src := range scriptSrcs {
		combined += " " + strings.ToLower(src)
	}

	switch {
	case strings.Contains(combined, "phaser"):
		return "phaser"
	case strings.Contains(combined, "/assets/index-") && strings.Contains(combined, ".js"):
		return "vite-spa"
	case strings.Contains(combined, "pixi.js") || strings.Contains(combined, "pixi.min.js") || strings.Contains(combined, "pixijs"):
		return "pixi"
	case strings.Contains(combined, "unityloader") || strings.Contains(combined, "unityprogress") || strings.Contains(combined, "unityinstance"):
		return "unity"
	case strings.Contains(combined, "godot") || strings.Contains(combined, "engine.wasm"):
		return "godot"
	case strings.Contains(combined, "three.js") || strings.Contains(combined, "three.min.js"):
		return "threejs"
	case strings.Contains(combined, "babylon") || strings.Contains(combined, "babylonjs"):
		return "babylon"
	case strings.Contains(combined, "construct") || strings.Contains(combined, "c3runtime"):
		return "construct"
	case strings.Contains(combined, "playcanvas"):
		return "playcanvas"
	case strings.Contains(combined, "cocos") || strings.Contains(combined, "cc.game"):
		return "cocos"
	case strings.Contains(combined, "createjs") || strings.Contains(combined, "easeljs"):
		return "createjs"
	default:
		return "unknown"
	}
}
