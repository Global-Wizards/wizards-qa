package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/scout"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"github.com/spf13/cobra"
)

func newScoutCmd() *cobra.Command {
	var (
		gameURL      string
		output       string
		jsonOutput   bool
		saveFlows    bool
		configPath   string
		headless     bool
		timeout      int
		agentMode    bool
		agentSteps   int
		modelFlag    string
		maxTokens    int
		temperature  float64
	)

	cmd := &cobra.Command{
		Use:   "scout",
		Short: "Scout a game URL and auto-generate Maestro test flows",
		Long: `Automatically scout a game page, detect its framework, analyze game mechanics
with AI, and generate ready-to-run Maestro test flows — no spec file needed.

Pipeline:
1. Fetch page and extract metadata (framework, canvas, scripts, structure)
2. Analyze game mechanics and UI with AI
3. Generate test scenarios and Maestro YAML flows

Example:
  wizards-qa scout --game https://game.example.com
  wizards-qa scout --game https://game.example.com --json
  wizards-qa scout --game https://game.example.com --headless
  wizards-qa scout --game https://game.example.com --output ./my-flows`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if gameURL == "" {
				return fmt.Errorf("--game URL is required")
			}

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			if !jsonOutput {
				fmt.Printf("%s Scouting %s...\n", util.EmojiTarget, gameURL)
			}

			// Step 1: Scout the page
			ctx := context.Background()
			timeoutDur := time.Duration(timeout) * time.Second

			var pageMeta *scout.PageMeta

			// Agent mode handles its own scouting (with keep-alive browser) below,
			// so skip the initial scout when agent mode is enabled.
			if !agentMode {
				if headless {
					pageMeta, err = scout.ScoutURLHeadless(ctx, gameURL, scout.HeadlessConfig{
						Enabled: true,
						Width:   cfg.Browser.Viewport.Width,
						Height:  cfg.Browser.Viewport.Height,
						Timeout: timeoutDur,
					})
				} else {
					pageMeta, err = scout.ScoutURL(ctx, gameURL, timeoutDur)
				}
				if err != nil {
					return fmt.Errorf("page scout failed: %w", err)
				}

				// Auto-fallback to headless if HTTP scout got minimal results
				if !headless && pageMeta.Framework == "unknown" && !pageMeta.CanvasFound && len(pageMeta.ScriptSrcs) <= 2 {
					if !jsonOutput {
						fmt.Printf("   Minimal page detected, retrying with headless Chrome...\n")
					} else {
						fmt.Fprintf(os.Stderr, "PROGRESS:fallback:Minimal page detected, retrying with headless Chrome...\n")
					}
					headlessMeta, headlessErr := scout.ScoutURLHeadless(ctx, gameURL, scout.HeadlessConfig{
						Enabled: true,
						Width:   cfg.Browser.Viewport.Width,
						Height:  cfg.Browser.Viewport.Height,
						Timeout: timeoutDur,
					})
					if headlessErr == nil {
						pageMeta = headlessMeta
						headless = true
					} else if !jsonOutput {
						fmt.Printf("   Headless fallback failed: %v\n", headlessErr)
					}
				}

				if !jsonOutput {
					fmt.Printf("   Title: %s\n", pageMeta.Title)
					fmt.Printf("   Framework: %s\n", pageMeta.Framework)
					fmt.Printf("   Canvas: %v\n", pageMeta.CanvasFound)
					fmt.Printf("   Scripts: %d\n", len(pageMeta.ScriptSrcs))
					if pageMeta.ScreenshotB64 != "" {
						screenshotKB := len(pageMeta.ScreenshotB64) * 3 / 4 / 1024
						fmt.Printf("   Screenshot: %d KB\n", screenshotKB)
					}
					if len(pageMeta.JSGlobals) > 0 {
						fmt.Printf("   JS Globals: %v\n", pageMeta.JSGlobals)
					}
					fmt.Println()
				}
			}

			if err := validateAPIKey(cfg); err != nil {
				return err
			}

			analyzer, err := newAnalyzer(cfg, modelFlag, maxTokens, temperature)
			if err != nil {
				return err
			}

			// Emit scouting progress for --json mode
			if jsonOutput && !agentMode {
				fmt.Fprintf(os.Stderr, "PROGRESS:scouting:Scouting page %s\n", gameURL)
				detail := fmt.Sprintf("%s | Canvas: %v | Scripts: %d", pageMeta.Framework, pageMeta.CanvasFound, len(pageMeta.ScriptSrcs))
				fmt.Fprintf(os.Stderr, "PROGRESS:scouted:%s\n", detail)

				// Emit detailed scouting info as JSON
				scoutDetail := map[string]interface{}{
					"framework":          pageMeta.Framework,
					"canvasFound":        pageMeta.CanvasFound,
					"jsGlobals":          pageMeta.JSGlobals,
					"screenshotCaptured": pageMeta.ScreenshotB64 != "",
					"scriptCount":        len(pageMeta.ScriptSrcs),
					"title":              pageMeta.Title,
				}
				if detailJSON, err := json.Marshal(scoutDetail); err == nil {
					fmt.Fprintf(os.Stderr, "PROGRESS:scouted_detail:%s\n", string(detailJSON))
				}
			}

			if jsonOutput && agentMode {
				fmt.Fprintf(os.Stderr, "PROGRESS:scouting:Scouting page with agent mode %s\n", gameURL)
			}

			if !jsonOutput {
				fmt.Printf("%s Analyzing game with AI...\n", util.EmojiClip)
			}

			// Step 2: Full pipeline — reuse pre-fetched pageMeta to avoid double-fetch
			var onProgress ai.ProgressFunc
			if jsonOutput {
				onProgress = func(step, message string) {
					fmt.Fprintf(os.Stderr, "PROGRESS:%s:%s\n", step, message)
				}
			}

			var result *ai.AnalysisResult
			var flows []*ai.MaestroFlow
			var agentStepsResult []ai.AgentStep

			if agentMode {
				// Agent mode: use ScoutURLHeadlessKeepAlive + agentic exploration
				agentPageMeta, browserPage, cleanup, agentErr := scout.ScoutURLHeadlessKeepAlive(ctx, gameURL, scout.HeadlessConfig{
					Enabled: true,
					Width:   cfg.Browser.Viewport.Width,
					Height:  cfg.Browser.Viewport.Height,
					Timeout: timeoutDur,
				})
				if agentErr != nil {
					return fmt.Errorf("agent scout failed: %w", agentErr)
				}
				defer cleanup()

				// Use the agent-scouted pageMeta (has initial screenshot)
				pageMeta = agentPageMeta

				if jsonOutput {
					detail := fmt.Sprintf("%s | Canvas: %v | Scripts: %d", pageMeta.Framework, pageMeta.CanvasFound, len(pageMeta.ScriptSrcs))
					fmt.Fprintf(os.Stderr, "PROGRESS:scouted:%s\n", detail)
				} else {
					fmt.Printf("   Title: %s\n", pageMeta.Title)
					fmt.Printf("   Framework: %s\n", pageMeta.Framework)
					fmt.Printf("   Canvas: %v\n", pageMeta.CanvasFound)
					fmt.Println()
				}

				// Scale exploration timeout: steps × 30s avg + 5min buffer, clamped 5-20min
				explorationTimeout := time.Duration(agentSteps)*30*time.Second + 5*time.Minute
				if explorationTimeout < 5*time.Minute {
					explorationTimeout = 5 * time.Minute
				}
				if explorationTimeout > 20*time.Minute {
					explorationTimeout = 20 * time.Minute
				}

				// Synthesis needs at least 8192 tokens for full JSON; ensure low-token profiles don't truncate
				synthTokens := 0
				if maxTokens > 0 && maxTokens < 8192 {
					synthTokens = 8192
				}

				agentCfg := ai.AgentConfig{
					MaxSteps:           agentSteps,
					StepTimeout:        30 * time.Second,
					TotalTimeout:       explorationTimeout,
					SynthesisMaxTokens: synthTokens,
				}

				// When launched by the backend (--json + --agent), read user hints from stdin
				if jsonOutput {
					userMsgs := make(chan string, 10)
					agentCfg.UserMessages = userMsgs

					// Create screenshot sub-dir for live streaming
					screenshotDir := filepath.Join(output, "agent-screenshots")
					if err := os.MkdirAll(screenshotDir, 0755); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to create screenshot dir: %v\n", err)
					}
					agentCfg.ScreenshotDir = screenshotDir

					go func() {
						scanner := bufio.NewScanner(os.Stdin)
						for scanner.Scan() {
							line := scanner.Text()
							var msg struct {
								Type    string `json:"type"`
								Message string `json:"message"`
							}
							if err := json.Unmarshal([]byte(line), &msg); err == nil && msg.Type == "user_hint" && msg.Message != "" {
								select {
								case userMsgs <- msg.Message:
								default:
									// Channel full, drop hint
								}
							}
						}
					}()
				}

				_, result, flows, agentStepsResult, err = analyzer.AnalyzeFromURLWithAgent(
					ctx, browserPage, pageMeta, gameURL, agentCfg, onProgress,
				)
				if err != nil {
					return fmt.Errorf("agent analysis failed: %w", err)
				}
			} else {
				// Standard 2-call pipeline (unchanged)
				_, result, flows, err = analyzer.AnalyzeFromURLWithMetaProgress(ctx, gameURL, pageMeta, onProgress)
				if err != nil {
					return fmt.Errorf("analysis failed: %w", err)
				}
			}

			// Emit detailed analysis info for --json mode
			if jsonOutput {
				analysisDetail := map[string]interface{}{
					"mechanicsCount":  len(result.Mechanics),
					"uiElementsCount": len(result.UIElements),
					"userFlowsCount":  len(result.UserFlows),
					"edgeCasesCount":  len(result.EdgeCases),
				}
				if detailJSON, err := json.Marshal(analysisDetail); err == nil {
					fmt.Fprintf(os.Stderr, "PROGRESS:analyzed_detail:%s\n", string(detailJSON))
				}
			}

			if !jsonOutput {
				if result.GameInfo.Name != "" {
					fmt.Printf("   Game: %s\n", result.GameInfo.Name)
					fmt.Printf("   Genre: %s\n", result.GameInfo.Genre)
				}
				fmt.Printf("   Mechanics: %d\n", len(result.Mechanics))
				fmt.Printf("   UI Elements: %d\n", len(result.UIElements))
				fmt.Printf("   User Flows: %d\n\n", len(result.UserFlows))
				fmt.Printf("%s Generating %d test flow(s)...\n", util.EmojiHammer, len(flows))
			}

			// Step 3: Save flows
			if saveFlows {
				flowsDir := output
				if !jsonOutput {
					gameName := deriveGameName(gameURL)
					flowsDir = fmt.Sprintf("%s/%s", output, gameName)
				}
				if err := ai.WriteFlowsToFiles(flows, flowsDir); err != nil {
					return fmt.Errorf("failed to save flows: %w", err)
				}
				if !jsonOutput {
					fmt.Printf("%s Flows saved to: %s\n\n", util.EmojiPassed, flowsDir)
				}
			}

			// Step 4: JSON output
			if jsonOutput {
				out := map[string]interface{}{
					"pageMeta": pageMeta,
					"analysis": result,
					"flows":    flows,
				}
				if agentMode {
					out["mode"] = "agent"
					out["agentSteps"] = agentStepsResult
				}
				data, err := json.Marshal(out)
				if err != nil {
					return fmt.Errorf("JSON marshal failed: %w", err)
				}
				fmt.Println(string(data))
			} else {
				fmt.Printf("%s Done! %d flow(s) generated.\n", util.EmojiRocket, len(flows))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&gameURL, "game", "g", "", "Game URL to analyze (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "./flows", "Output directory for generated flows")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output full result as JSON to stdout")
	cmd.Flags().BoolVar(&saveFlows, "save-flows", true, "Write generated flows to disk")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")
	cmd.Flags().BoolVar(&headless, "headless", false, "Use headless Chrome for JS-rendered pages")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "HTTP fetch timeout in seconds")
	cmd.Flags().BoolVar(&agentMode, "agent", false, "Enable agent mode: AI actively explores the game via browser tools")
	cmd.Flags().IntVar(&agentSteps, "agent-steps", 20, "Max exploration steps in agent mode")
	cmd.Flags().StringVar(&modelFlag, "model", "", "Override AI model (e.g. claude-sonnet-4-5-20250929)")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "Override max tokens for AI responses")
	cmd.Flags().Float64Var(&temperature, "temperature", -1, "Override AI temperature (0.0-1.0, unset by default)")

	cmd.MarkFlagRequired("game")

	return cmd
}
