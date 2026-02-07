package scout

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestParseHTML(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantTitle   string
		wantDesc    string
		wantCanvas  bool
		wantScripts int
		wantLinks   int
	}{
		{
			name:       "basic page with title and meta",
			html:       `<html><head><title>My Game</title><meta name="description" content="A fun game"></head><body></body></html>`,
			wantTitle:  "My Game",
			wantDesc:   "A fun game",
			wantCanvas: false,
		},
		{
			name:       "page with canvas",
			html:       `<html><head><title>Canvas Game</title></head><body><canvas id="game"></canvas></body></html>`,
			wantTitle:  "Canvas Game",
			wantCanvas: true,
		},
		{
			name:        "page with scripts",
			html:        `<html><head><script src="phaser.min.js"></script><script src="game.js"></script></head><body></body></html>`,
			wantScripts: 2,
		},
		{
			name:      "page with links",
			html:      `<html><body><a href="/play">Play</a><a href="/about">About</a><a href="/help">Help</a></body></html>`,
			wantLinks: 3,
		},
		{
			name:       "page with og:title meta",
			html:       `<html><head><meta property="og:title" content="OG Game Title"></head><body></body></html>`,
			wantCanvas: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := ParseHTML(tt.html)

			if tt.wantTitle != "" && meta.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", meta.Title, tt.wantTitle)
			}
			if tt.wantDesc != "" && meta.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", meta.Description, tt.wantDesc)
			}
			if meta.CanvasFound != tt.wantCanvas {
				t.Errorf("CanvasFound = %v, want %v", meta.CanvasFound, tt.wantCanvas)
			}
			if tt.wantScripts > 0 && len(meta.ScriptSrcs) != tt.wantScripts {
				t.Errorf("ScriptSrcs count = %d, want %d", len(meta.ScriptSrcs), tt.wantScripts)
			}
			if tt.wantLinks > 0 && len(meta.Links) != tt.wantLinks {
				t.Errorf("Links count = %d, want %d", len(meta.Links), tt.wantLinks)
			}
		})
	}
}

func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name       string
		scriptSrcs []string
		inline     string
		want       string
	}{
		{name: "phaser from src", scriptSrcs: []string{"phaser.min.js"}, want: "phaser"},
		{name: "pixi from src", scriptSrcs: []string{"pixi.js"}, want: "pixi"},
		{name: "pixi min from src", scriptSrcs: []string{"pixi.min.js"}, want: "pixi"},
		{name: "pixijs from src", scriptSrcs: []string{"libs/pixijs/bundle.js"}, want: "pixi"},
		{name: "unity from unityloader", scriptSrcs: []string{"UnityLoader.js"}, want: "unity"},
		{name: "unity from unityprogress", inline: "var UnityProgress = function() {}", want: "unity"},
		{name: "unity from unityinstance", inline: "new UnityInstance()", want: "unity"},
		{name: "godot from src", scriptSrcs: []string{"godot.js"}, want: "godot"},
		{name: "godot from engine.wasm", scriptSrcs: []string{"engine.wasm"}, want: "godot"},
		{name: "threejs from src", scriptSrcs: []string{"three.js"}, want: "threejs"},
		{name: "threejs min from src", scriptSrcs: []string{"three.min.js"}, want: "threejs"},
		{name: "babylon from src", scriptSrcs: []string{"babylon.js"}, want: "babylon"},
		{name: "babylonjs from src", scriptSrcs: []string{"babylonjs.loaders.js"}, want: "babylon"},
		{name: "construct from src", scriptSrcs: []string{"c3runtime.js"}, want: "construct"},
		{name: "construct from inline", inline: "new Construct()", want: "construct"},
		{name: "playcanvas from src", scriptSrcs: []string{"playcanvas.min.js"}, want: "playcanvas"},
		{name: "cocos from inline", inline: "cc.game.run()", want: "cocos"},
		{name: "cocos from src", scriptSrcs: []string{"cocos2d.js"}, want: "cocos"},
		{name: "createjs from src", scriptSrcs: []string{"createjs.min.js"}, want: "createjs"},
		{name: "easeljs from src", scriptSrcs: []string{"easeljs.min.js"}, want: "createjs"},
		{name: "unknown", scriptSrcs: []string{"app.js"}, inline: "console.log('hello')", want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFramework(tt.scriptSrcs, tt.inline)
			if got != tt.want {
				t.Errorf("DetectFramework() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleScriptMultiChild(t *testing.T) {
	// Simulate a <script> node with multiple text children
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
	}
	child1 := &html.Node{Type: html.TextNode, Data: "var a = 1;"}
	child2 := &html.Node{Type: html.TextNode, Data: "var b = 2;"}
	child3 := &html.Node{Type: html.TextNode, Data: "var c = 3;"}

	scriptNode.FirstChild = child1
	child1.NextSibling = child2
	child2.NextSibling = child3

	meta := &PageMeta{MetaTags: make(map[string]string)}
	var sb strings.Builder
	HandleScript(scriptNode, meta, &sb)

	got := sb.String()
	if !strings.Contains(got, "var a = 1;") {
		t.Errorf("missing child 1 content in output: %q", got)
	}
	if !strings.Contains(got, "var b = 2;") {
		t.Errorf("missing child 2 content in output: %q", got)
	}
	if !strings.Contains(got, "var c = 3;") {
		t.Errorf("missing child 3 content in output: %q", got)
	}
}

func TestHandleScriptWithSrc(t *testing.T) {
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: []html.Attribute{
			{Key: "src", Val: "game.js"},
		},
	}
	// Even if there's a text child, src takes priority
	child := &html.Node{Type: html.TextNode, Data: "inline code"}
	scriptNode.FirstChild = child

	meta := &PageMeta{MetaTags: make(map[string]string)}
	var sb strings.Builder
	HandleScript(scriptNode, meta, &sb)

	if len(meta.ScriptSrcs) != 1 || meta.ScriptSrcs[0] != "game.js" {
		t.Errorf("ScriptSrcs = %v, want [game.js]", meta.ScriptSrcs)
	}
	if sb.Len() != 0 {
		t.Errorf("expected no inline content for src script, got: %q", sb.String())
	}
}

func TestParseHTMLEmpty(t *testing.T) {
	// Empty string shouldn't panic
	meta := ParseHTML("")
	if meta == nil {
		t.Fatal("ParseHTML('') returned nil")
	}

	// Malformed HTML shouldn't panic
	meta = ParseHTML("<html><head><title>")
	if meta == nil {
		t.Fatal("ParseHTML(malformed) returned nil")
	}

	// Just whitespace
	meta = ParseHTML("   \n\t  ")
	if meta == nil {
		t.Fatal("ParseHTML(whitespace) returned nil")
	}
}

func TestParseHTMLBodySnippet(t *testing.T) {
	body := strings.Repeat("Hello World ", 200)
	htmlStr := "<html><body>" + body + "</body></html>"
	meta := ParseHTML(htmlStr)

	if len(meta.BodySnippet) > maxBodySnippet {
		t.Errorf("BodySnippet length = %d, want <= %d", len(meta.BodySnippet), maxBodySnippet)
	}
}
