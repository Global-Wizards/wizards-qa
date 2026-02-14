<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Analyze Game</h2>
        <p class="text-muted-foreground">
          Enter a game URL and our AI will automatically detect the framework, analyze game mechanics, and generate test flows.
        </p>
      </div>
    </div>

    <!-- State 1: Input -->
    <Card v-if="status === 'idle'">
      <CardHeader>
        <CardTitle>Game URL</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="space-y-2">
          <Input
            v-model="gameUrl"
            placeholder="https://your-game.example.com"
            @keyup.enter="handleAnalyze"
          />
          <p v-if="gameUrl && !isValidUrl(gameUrl)" class="text-xs text-destructive">
            Enter a valid URL starting with http:// or https://
          </p>
        </div>
        <Alert v-if="tokenWarning" variant="destructive" class="flex items-start gap-2">
          <AlertCircle class="h-4 w-4 mt-0.5 shrink-0" />
          <div>
            <AlertTitle>Expired Token Detected</AlertTitle>
            <AlertDescription>{{ tokenWarning }}</AlertDescription>
          </div>
        </Alert>
        <div class="flex items-center gap-4">
          <Button :disabled="!isValidUrl(gameUrl) || analyzing" @click="handleAnalyze">
            {{ analyzing ? 'Starting...' : 'Analyze Game' }}
          </Button>
          <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="AI explores the game interactively via browser tools">
            <input type="checkbox" v-model="useAgentMode" class="rounded border-gray-300" aria-label="Agent Mode" />
            Agent Mode
          </label>
        </div>

        <!-- Agent Modules (only when Agent Mode is on) -->
        <div v-if="useAgentMode" class="space-y-3 pt-2 border-t">
          <div class="flex items-center gap-3">
            <Zap class="h-4 w-4 text-muted-foreground shrink-0" />
            <label class="text-sm font-medium">Agent Modules</label>
          </div>
          <div class="ml-7 grid gap-2 sm:grid-cols-2">
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="AI can request additional exploration steps when it finds unexplored areas">
              <input type="checkbox" v-model="moduleDynamicSteps" class="rounded border-gray-300" aria-label="Dynamic Steps" />
              <TrendingUp class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Dynamic Steps</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="AI can extend exploration time for more thorough testing">
              <input type="checkbox" v-model="moduleDynamicTimeout" class="rounded border-gray-300" aria-label="Dynamic Timeout" />
              <Timer class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Dynamic Timeout</span>
            </label>
          </div>
        </div>

        <!-- Device Viewport -->
        <div class="space-y-3 pt-2 border-t">
          <div class="flex items-center gap-3">
            <Monitor class="h-4 w-4 text-muted-foreground shrink-0" />
            <div class="flex items-center gap-2 flex-1">
              <label class="text-sm font-medium whitespace-nowrap">Device</label>
              <template v-if="!multiDeviceMode">
                <Select :model-value="selectedViewport" @update:model-value="selectedViewport = $event">
                  <SelectTrigger class="w-[220px]">
                    <SelectValue placeholder="Select device" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectLabel>Recommended</SelectLabel>
                      <SelectItem v-for="p in getRecommendedViewports()" :key="p.name" :value="p.name">
                        {{ p.label }}
                      </SelectItem>
                    </SelectGroup>
                    <SelectGroup v-for="cat in getViewportCategories()" :key="cat.name">
                      <SelectLabel>{{ cat.name }}</SelectLabel>
                      <SelectItem v-for="p in cat.presets" :key="p.name" :value="p.name">
                        {{ p.label }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <span v-if="activeViewport" class="text-xs text-muted-foreground">
                  {{ activeViewport.width }}&times;{{ activeViewport.height }}
                </span>
              </template>
            </div>
            <label class="flex items-center gap-2 text-xs cursor-pointer select-none text-muted-foreground whitespace-nowrap">
              <input type="checkbox" v-model="multiDeviceMode" class="rounded border-gray-300" />
              Multi-Device
            </label>
          </div>

          <!-- Multi-device category selectors -->
          <div v-if="multiDeviceMode" class="ml-7 space-y-2">
            <div v-for="(cfg, category) in multiDevices" :key="category" class="flex items-center gap-3">
              <label class="flex items-center gap-2 text-sm cursor-pointer select-none w-24">
                <input type="checkbox" v-model="cfg.enabled" class="rounded border-gray-300" />
                <span class="capitalize">{{ category === 'ios' ? 'iOS' : category === 'android' ? 'Android' : 'Desktop' }}</span>
              </label>
              <Select v-if="cfg.enabled" :model-value="cfg.viewport" @update:model-value="cfg.viewport = $event">
                <SelectTrigger class="w-[200px] h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup v-for="cat in viewportCategoriesForDevice(category)" :key="cat.name">
                    <SelectLabel>{{ cat.name }}</SelectLabel>
                    <SelectItem v-for="p in cat.presets" :key="p.name" :value="p.name">
                      {{ p.label }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
              <span v-if="cfg.enabled" class="text-[10px] text-muted-foreground">
                {{ getViewportByName(cfg.viewport)?.width }}&times;{{ getViewportByName(cfg.viewport)?.height }}
              </span>
            </div>
          </div>
        </div>

        <!-- Analysis Modules -->
        <div class="space-y-3 pt-2 border-t">
          <div class="flex items-center gap-3">
            <Sparkles class="h-4 w-4 text-muted-foreground shrink-0" />
            <label class="text-sm font-medium">Analysis Modules</label>
          </div>
          <div class="ml-7 grid gap-2 sm:grid-cols-2">
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="Evaluate alignments, spacing, color harmony, typography, visual hierarchy, and accessibility">
              <input type="checkbox" v-model="moduleUiux" class="rounded border-gray-300" />
              <Eye class="h-3.5 w-3.5 text-muted-foreground" />
              <span>UI/UX Analysis</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="Check for grammar, spelling, inconsistent terminology, truncation, and placeholder text">
              <input type="checkbox" v-model="moduleWording" class="rounded border-gray-300" />
              <Type class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Wording Check</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="Analyze reward systems, balance, progression, difficulty curve, and monetization fairness">
              <input type="checkbox" v-model="moduleGameDesign" class="rounded border-gray-300" />
              <Gamepad2 class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Game Design</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none" title="Generate runnable Maestro YAML test flows from the analysis">
              <input type="checkbox" v-model="moduleTestFlows" class="rounded border-gray-300" />
              <PlayCircle class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Test Flows</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none"
                   title="Automatically run generated test flows in headless browser after analysis"
                   :class="{ 'opacity-50': !moduleTestFlows }">
              <input type="checkbox" v-model="moduleRunTests" :disabled="!moduleTestFlows" class="rounded border-gray-300" />
              <FlaskConical class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Run Tests</span>
            </label>
          </div>
        </div>

        <!-- Analysis Profile Selector -->
        <div class="space-y-3 pt-2 border-t">
          <div class="flex items-center gap-3">
            <Settings2 class="h-4 w-4 text-muted-foreground shrink-0" />
            <div class="flex items-center gap-2 flex-1">
              <label class="text-sm font-medium whitespace-nowrap">Analysis Profile</label>
              <Select :model-value="selectedProfile" @update:model-value="onProfileChange">
                <SelectTrigger class="w-[180px]">
                  <SelectValue placeholder="Select profile" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="p in ANALYSIS_PROFILES" :key="p.name" :value="p.name">
                    {{ p.label }}
                  </SelectItem>
                  <SelectItem value="custom">Custom</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <p v-if="activeProfile" class="text-xs text-muted-foreground ml-7">
            {{ activeProfile.description }}
            <span class="text-muted-foreground/70">
              &mdash; {{ activeProfile.model.split('-').slice(0, 2).join('-') }}
              / {{ activeProfile.maxTokens }} tokens
              / {{ activeProfile.agentSteps }} steps
            </span>
            <span v-if="activeProfile.adaptive" class="text-muted-foreground/70">
              (adaptive, up to {{ activeProfile.maxTotalSteps }})
            </span>
            <span v-if="activeProfile.cost" class="text-muted-foreground/70">
              &middot; {{ activeProfile.cost }} cost &middot; ~{{ activeProfile.time }}
            </span>
          </p>

          <!-- Custom fields -->
          <div v-if="showCustomFields" class="ml-7 grid gap-3 sm:grid-cols-2">
            <div class="space-y-1">
              <label class="text-xs font-medium">Model</label>
              <Input v-model="customModel" placeholder="claude-sonnet-4-5-20250929" class="text-sm" />
            </div>
            <div class="space-y-1">
              <label class="text-xs font-medium">Max Tokens</label>
              <Input v-model.number="customMaxTokens" type="number" :min="512" :max="32768" class="text-sm" />
            </div>
            <div class="space-y-1">
              <label class="text-xs font-medium">Agent Steps</label>
              <Input v-model.number="customAgentSteps" type="number" :min="1" :max="50" class="text-sm" />
            </div>
            <div class="space-y-1">
              <label class="text-xs font-medium">Temperature</label>
              <Input v-model.number="customTemperature" type="number" :min="0" :max="1" step="0.1" class="text-sm" />
            </div>
            <div v-if="moduleDynamicSteps" class="space-y-1">
              <label class="text-xs font-medium">Max Total Steps</label>
              <Input v-model.number="customMaxTotalSteps" type="number" :min="customAgentSteps || 5" :max="100" class="text-sm" />
            </div>
            <div v-if="moduleDynamicTimeout" class="space-y-1">
              <label class="text-xs font-medium">Max Total Timeout (minutes)</label>
              <Input v-model.number="customMaxTotalTimeout" type="number" :min="1" :max="60" class="text-sm" />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Recent Analyses (idle state) -->
    <Card v-if="status === 'idle' && recentAnalyses.length" class="mt-4">
      <CardHeader class="flex flex-row items-center justify-between">
        <CardTitle class="text-lg">Recent Analyses</CardTitle>
        <Button variant="link" size="sm" class="text-xs" @click="navigateToAnalysesList">
          View All Analyses
          <ExternalLink class="h-3 w-3 ml-1" />
        </Button>
      </CardHeader>
      <CardContent>
        <div class="space-y-2">
          <div
            v-for="item in recentAnalyses"
            :key="item.id"
            class="flex items-center justify-between p-3 rounded-md border hover:bg-muted/50 transition-colors"
          >
            <div class="min-w-0 cursor-pointer flex-1" @click="viewAnalysis(item)">
              <p class="text-sm font-medium truncate" :title="item.gameUrl">{{ item.gameName || truncateUrl(item.gameUrl, 60) }}</p>
              <p class="text-xs text-muted-foreground">
                {{ item.framework }} &middot; {{ item.flowCount }} flow(s) &middot; {{ formatDate(item.createdAt) }}
              </p>
            </div>
            <div class="flex items-center gap-2 shrink-0 ml-2">
              <Badge variant="secondary">{{ item.status }}</Badge>
              <Button variant="ghost" size="sm" @click="reAnalyze(item)">
                <RefreshCw class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="sm" @click="deleteAnalysis(item)">
                <Trash2 class="h-3 w-3 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- State 2: Progress -->
    <AnalysisProgressPanel
      v-else-if="status === 'queued' || status === 'scouting' || status === 'analyzing' || status === 'generating' || status === 'creating_test_plan' || status === 'testing'"
      mode="progress"
      :game-url="gameUrl"
      :elapsed-seconds="elapsedSeconds"
      :format-elapsed="formatElapsed"
      :agent-mode="agentMode"
      :phases="progressPhases"
      :show-agent-panel="agentMode && (agentExplorationStatus === 'active' || liveAgentSteps.length > 0)"
      :show-testing-panel="testStepScreenshots.length > 0 || testFlowProgress.length > 0"
      :logs="logs"
      :device-label="deviceLabel"
      @cancel="handleReset"
      @copy-log="copyDebugLog"
    >
      <template #agent-exploration>
        <AgentExplorationPanel
          :steps="liveAgentSteps"
          :step-current="agentStepCurrent"
          :step-total="agentStepTotal"
          :exploration-status="agentExplorationStatus"
          :elapsed-seconds="elapsedSeconds"
          :hint-cooldown="hintCooldown"
          :format-elapsed="formatElapsed"
          :device-label="deviceLabel"
          @send-hint="sendHint"
          @expand-screenshot="expandStepScreenshot"
        />
      </template>
      <template #test-execution>
        <TestStepNavigator
          v-if="testStepScreenshots.length"
          :steps="testStepScreenshots"
          :live="status === 'testing'"
        />
      </template>
    </AnalysisProgressPanel>

    <!-- State 3: Results -->
    <div v-else-if="status === 'complete'" class="space-y-4">
      <Card>
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle>Analysis Complete</CardTitle>
            <Badge variant="secondary">{{ flowCount }} flow(s) generated</Badge>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <!-- Summary -->
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <span class="text-sm text-muted-foreground">Game</span>
              <p class="font-medium">{{ gameName }}</p>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Framework</span>
              <p class="font-medium capitalize">{{ framework }}</p>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Canvas</span>
              <p class="font-medium">{{ pageMeta?.canvasFound ? 'Yes' : 'No' }}</p>
            </div>
          </div>

          <!-- Multi-device summary -->
          <div v-if="devices.length > 0" class="grid gap-2 sm:grid-cols-3">
            <div
              v-for="d in devices"
              :key="d.device"
              class="rounded-md border p-3 text-center"
              :class="d.status === 'failed' ? 'border-destructive/50 bg-destructive/5' : 'border-border'"
            >
              <p class="text-sm font-medium capitalize">{{ d.device }}</p>
              <p class="text-[10px] text-muted-foreground">{{ d.viewport }}</p>
              <div class="mt-1">
                <Badge v-if="d.status === 'completed'" variant="secondary">{{ d.flowCount }} flow(s)</Badge>
                <Badge v-else variant="destructive" class="text-[10px]">Failed</Badge>
              </div>
              <p v-if="d.error" class="text-[10px] text-destructive mt-1 truncate" :title="d.error">{{ d.error }}</p>
            </div>
          </div>

          <Separator />

          <!-- Page Metadata -->
          <details class="group">
            <summary class="cursor-pointer text-sm font-medium">Page Metadata</summary>
            <div class="mt-2 space-y-2 text-sm">
              <div v-if="pageMeta?.title">
                <span class="text-muted-foreground">Title:</span> {{ pageMeta.title }}
              </div>
              <div v-if="pageMeta?.description">
                <span class="text-muted-foreground">Description:</span> {{ pageMeta.description }}
              </div>
              <div v-if="pageMeta?.scriptSrcs?.length">
                <span class="text-muted-foreground">Scripts ({{ pageMeta.scriptSrcs.length }}):</span>
                <ul class="ml-4 list-disc text-xs text-muted-foreground">
                  <li v-for="src in pageMeta.scriptSrcs.slice(0, 10)" :key="src">{{ src }}</li>
                </ul>
              </div>
            </div>
          </details>

          <!-- Game Analysis -->
          <details v-if="analysis" class="group">
            <summary class="cursor-pointer text-sm font-medium">Game Analysis</summary>
            <div class="mt-2 space-y-2 text-sm">
              <div v-if="analysis.mechanics?.length">
                <span class="text-muted-foreground">Mechanics ({{ analysis.mechanics.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="m in analysis.mechanics" :key="m.name">{{ m.name }}: {{ m.description }}</li>
                </ul>
              </div>
              <div v-if="analysis.uiElements?.length">
                <span class="text-muted-foreground">UI Elements ({{ analysis.uiElements.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="el in analysis.uiElements" :key="el.name">{{ el.name }} ({{ el.type }})</li>
                </ul>
              </div>
              <div v-if="analysis.userFlows?.length">
                <span class="text-muted-foreground">User Flows ({{ analysis.userFlows.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="f in analysis.userFlows" :key="f.name">{{ f.name }}: {{ f.description }}</li>
                </ul>
              </div>
              <div v-if="analysis.edgeCases?.length">
                <span class="text-muted-foreground">Edge Cases ({{ analysis.edgeCases.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="ec in analysis.edgeCases" :key="ec.name">{{ ec.name }}: {{ ec.description }}</li>
                </ul>
              </div>
            </div>
          </details>

          <!-- UI/UX Analysis -->
          <details v-if="analysis?.uiuxAnalysis?.length" class="group">
            <summary class="cursor-pointer text-sm font-medium">UI/UX Analysis ({{ analysis.uiuxAnalysis.length }})</summary>
            <div class="mt-2 space-y-2">
              <div v-for="(finding, i) in analysis.uiuxAnalysis" :key="i" class="rounded-md border p-3 text-sm space-y-1">
                <div class="flex items-center gap-2">
                  <Badge :variant="severityVariant(finding.severity)">{{ finding.severity }}</Badge>
                  <Badge variant="outline">{{ finding.category }}</Badge>
                  <span v-if="finding.location" class="text-xs text-muted-foreground">{{ finding.location }}</span>
                </div>
                <p>{{ finding.description }}</p>
                <p v-if="finding.suggestion" class="text-muted-foreground text-xs">Suggestion: {{ finding.suggestion }}</p>
              </div>
            </div>
          </details>

          <!-- Wording/Translation Check -->
          <details v-if="analysis?.wordingCheck?.length" class="group">
            <summary class="cursor-pointer text-sm font-medium">Wording/Translation Check ({{ analysis.wordingCheck.length }})</summary>
            <div class="mt-2 space-y-2">
              <div v-for="(finding, i) in analysis.wordingCheck" :key="i" class="rounded-md border p-3 text-sm space-y-1">
                <div class="flex items-center gap-2">
                  <Badge :variant="severityVariant(finding.severity)">{{ finding.severity }}</Badge>
                  <Badge variant="outline">{{ finding.category }}</Badge>
                  <span v-if="finding.location" class="text-xs text-muted-foreground">{{ finding.location }}</span>
                </div>
                <p v-if="finding.text" class="font-mono text-xs bg-muted px-2 py-1 rounded">"{{ finding.text }}"</p>
                <p>{{ finding.description }}</p>
                <p v-if="finding.suggestion" class="text-muted-foreground text-xs">Suggestion: {{ finding.suggestion }}</p>
              </div>
            </div>
          </details>

          <!-- Game Design Analysis -->
          <details v-if="analysis?.gameDesign?.length" class="group">
            <summary class="cursor-pointer text-sm font-medium">Game Design Analysis ({{ analysis.gameDesign.length }})</summary>
            <div class="mt-2 space-y-2">
              <div v-for="(finding, i) in analysis.gameDesign" :key="i" class="rounded-md border p-3 text-sm space-y-1">
                <div class="flex items-center gap-2">
                  <Badge :variant="severityVariant(finding.severity)">{{ finding.severity }}</Badge>
                  <Badge variant="outline">{{ finding.category }}</Badge>
                </div>
                <p>{{ finding.description }}</p>
                <p v-if="finding.impact" class="text-xs"><span class="text-muted-foreground">Impact:</span> {{ finding.impact }}</p>
                <p v-if="finding.suggestion" class="text-muted-foreground text-xs">Suggestion: {{ finding.suggestion }}</p>
              </div>
            </div>
          </details>

          <!-- Agent Exploration (agent mode only) -->
          <details v-if="navigatorSteps.length" class="group" open>
            <summary class="cursor-pointer text-sm font-medium">Agent Exploration ({{ navigatorSteps.length }} steps)</summary>
            <div class="mt-2">
              <AgentStepNavigator
                v-if="currentAnalysisId"
                :analysis-id="currentAnalysisId"
                :initial-steps="navigatorSteps"
              />
            </div>
          </details>

          <!-- Generated Flows -->
          <details v-if="flowList.length" class="group">
            <summary class="cursor-pointer text-sm font-medium">Generated Flows ({{ flowList.length }})</summary>
            <div class="mt-2 flex flex-wrap gap-2">
              <Badge
                v-for="flow in flowList"
                :key="flow.name"
                variant="outline"
                class="cursor-pointer hover:bg-accent"
                @click="previewFlow(flow)"
              >
                {{ flow.name }}
              </Badge>
            </div>
          </details>

          <!-- Debug Info -->
          <details class="group">
            <summary class="cursor-pointer text-sm font-medium flex items-center gap-1.5">
              <Bug class="h-4 w-4" />
              Debug Info
            </summary>
            <div class="mt-2 space-y-3 text-sm">
              <!-- Progress Log -->
              <div>
                <div class="flex items-center justify-between mb-1">
                  <span class="text-muted-foreground font-medium">Progress Log ({{ logs.length }} lines):</span>
                  <Button variant="outline" size="sm" class="h-7 text-xs gap-1" @click="copyDebugLog">
                    <Copy class="h-3 w-3" />
                    {{ logCopied ? 'Copied!' : 'Copy Full Log' }}
                  </Button>
                </div>
                <div class="max-h-48 overflow-y-auto rounded-md bg-muted p-3">
                  <p v-for="(line, i) in logs" :key="i" class="text-xs font-mono text-muted-foreground">{{ line }}</p>
                  <p v-if="!logs.length" class="text-xs text-muted-foreground">No log entries recorded.</p>
                </div>
              </div>

              <!-- Screenshot -->
              <div v-if="pageMeta?.screenshotB64">
                <span class="text-muted-foreground font-medium">Screenshot:</span>
                <img
                  :src="'data:image/jpeg;base64,' + pageMeta.screenshotB64"
                  class="mt-1 rounded-md border max-w-md"
                  alt="Game screenshot"
                />
              </div>

              <!-- JS Globals -->
              <div v-if="pageMeta?.jsGlobals?.length">
                <span class="text-muted-foreground font-medium">JS Globals:</span>
                <div class="mt-1 flex flex-wrap gap-1">
                  <Badge v-for="g in pageMeta.jsGlobals" :key="g" variant="secondary" class="text-xs">{{ g }}</Badge>
                </div>
              </div>

              <!-- URL Hints -->
              <div v-if="gameUrl">
                <span class="text-muted-foreground font-medium">URL Hints:</span>
                <div class="mt-1 space-y-0.5">
                  <div v-for="(value, key) in parseUrlHints(gameUrl)" :key="key" class="text-xs font-mono">
                    <span class="text-muted-foreground">{{ key }}:</span> {{ value }}
                  </div>
                </div>
              </div>

              <!-- Step Timings -->
              <div v-if="formatStepTimingSummary()">
                <span class="text-muted-foreground font-medium">Step Timings:</span>
                <p class="text-xs font-mono mt-1">{{ formatStepTimingSummary() }}</p>
              </div>

              <!-- Body Snippet -->
              <div v-if="pageMeta?.bodySnippet">
                <span class="text-muted-foreground font-medium">Body Snippet:</span>
                <pre class="mt-1 max-h-32 overflow-auto rounded-md bg-muted p-2 text-xs">{{ pageMeta.bodySnippet.slice(0, 500) }}</pre>
              </div>

              <!-- Raw AI Response (shown when JSON parsing failed) -->
              <div v-if="analysis?.rawResponse">
                <span class="text-muted-foreground font-medium">Raw AI Response:</span>
                <pre class="mt-1 max-h-48 overflow-auto rounded-md bg-muted p-2 text-xs">{{ analysis.rawResponse }}</pre>
              </div>

              <!-- Script Sources (full list) -->
              <div v-if="pageMeta?.scriptSrcs?.length">
                <span class="text-muted-foreground font-medium">Script Sources ({{ pageMeta.scriptSrcs.length }}):</span>
                <ul class="mt-1 ml-4 list-disc text-xs text-muted-foreground">
                  <li v-for="src in pageMeta.scriptSrcs" :key="src">{{ src }}</li>
                </ul>
              </div>
            </div>
          </details>

          <Separator />

          <!-- Actions -->
          <div class="flex flex-wrap gap-2">
            <Button v-if="autoTestPlanId" @click="navigateToTestPlan">View Test Plan</Button>
            <Button v-else @click="navigateToNewPlan">Create Test Plan</Button>
            <Button variant="outline" @click="viewCurrentAnalysis">
              <ExternalLink class="h-4 w-4 mr-1" />
              View Full Analysis
            </Button>
            <Button variant="outline" @click="navigateToFlows">View Flows</Button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="outline">
                  <Download class="h-4 w-4 mr-1" />
                  Export
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem @click="exportAnalysis('json')">Export as JSON</DropdownMenuItem>
                <DropdownMenuItem @click="exportAnalysis('markdown')">Export as Markdown</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <Button variant="outline" @click="handleReset">Analyze Another</Button>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Error State -->
    <AnalysisProgressPanel
      v-else-if="status === 'error'"
      mode="error"
      :game-url="gameUrl"
      :elapsed-seconds="elapsedSeconds"
      :format-elapsed="formatElapsed"
      :agent-mode="agentMode"
      :phases="progressPhases"
      :logs="logs"
      :error-message="analysisError"
      :failed-phase-label="failedPhaseLabel"
      :can-continue="canContinue"
      :device-label="deviceLabel"
      @retry="retryAnalysis"
      @continue="handleContinueAnalysis"
      @start-over="handleReset"
      @copy-log="copyDebugLog"
    />
    <!-- Agent steps navigator (stays outside component, error state only) -->
    <div v-if="status === 'error' && navigatorSteps.length" class="mt-4">
      <AgentStepNavigator
        v-if="currentAnalysisId || analysisId"
        :analysis-id="currentAnalysisId || analysisId"
        :initial-steps="navigatorSteps"
      />
    </div>

    <!-- Flow Preview Dialog -->
    <Dialog :open="flowDialogOpen" @update:open="flowDialogOpen = $event">
      <DialogContent class="max-w-3xl max-h-[80vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>{{ previewFlowData?.name }}</DialogTitle>
          <DialogDescription>Generated flow YAML</DialogDescription>
        </DialogHeader>
        <div class="mt-4 relative">
          <Button
            variant="outline"
            size="sm"
            class="absolute top-2 right-2 z-10"
            @click="copyFlowYaml"
          >
            {{ flowCopied ? 'Copied!' : 'Copy' }}
          </Button>
          <pre class="bg-muted rounded-md p-4 text-sm overflow-auto max-h-[60vh]"><code>{{ previewFlowYaml }}</code></pre>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Agent Screenshot Dialog -->
    <Dialog :open="agentScreenshotOpen" @update:open="agentScreenshotOpen = $event">
      <DialogContent class="max-w-4xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>Step {{ agentScreenshotStep?.stepNumber }}: {{ agentScreenshotStep?.toolName }}</DialogTitle>
          <DialogDescription>{{ agentScreenshotStep?.result }}</DialogDescription>
        </DialogHeader>
        <div class="mt-4">
          <img
            v-if="agentScreenshotStep?.screenshotB64 || agentScreenshotStep?.screenshotUrl"
            :src="agentScreenshotStep.screenshotB64
              ? 'data:image/jpeg;base64,' + agentScreenshotStep.screenshotB64
              : agentScreenshotStep.screenshotUrl"
            class="w-full rounded-md border"
            alt="Agent step screenshot"
          />
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useStorage } from '@vueuse/core'
import { useRouter, useRoute } from 'vue-router'
import { useAnalysis } from '@/composables/useAnalysis'
import { truncateUrl, isValidUrl, severityVariant } from '@/lib/utils'
import { ANALYSIS_PROFILES, getProfileByName } from '@/lib/profiles'
import { DEFAULT_VIEWPORT, getViewportByName, getViewportCategories, getRecommendedViewports } from '@/lib/viewports'
import { analysesApi, analyzeApi, projectsApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { useClipboard } from '@vueuse/core'
import { useProject } from '@/composables/useProject'
import { RefreshCw, Trash2, Download, Bug, Copy, AlertCircle, Settings2, ExternalLink, Sparkles, Eye, Type, Gamepad2, PlayCircle, Zap, TrendingUp, Timer, Monitor, FlaskConical } from 'lucide-vue-next'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem, SelectGroup, SelectLabel } from '@/components/ui/select'
import AnalysisProgressPanel from '@/components/AnalysisProgressPanel.vue'
import AgentStepNavigator from '@/components/AgentStepNavigator.vue'
import AgentExplorationPanel from '@/components/AgentExplorationPanel.vue'
import TestStepNavigator from '@/components/TestStepNavigator.vue'

const router = useRouter()
const route = useRoute()
const { currentProject } = useProject()
const projectId = computed(() => route.params.projectId || '')
const gameUrl = ref('')
const analyzing = ref(false)
const useAgentMode = useStorage('analyze-agent-mode', true)
const selectedProfile = useStorage('analyze-profile', 'balanced')
const showCustomFields = ref(false)
const defaultProfile = getProfileByName('balanced')
const customModel = ref(defaultProfile.model)
const customMaxTokens = ref(defaultProfile.maxTokens)
const customAgentSteps = ref(defaultProfile.agentSteps)
const customTemperature = ref(defaultProfile.temperature)
const customMaxTotalSteps = ref(35)
const customMaxTotalTimeout = ref(25)
const moduleDynamicSteps = useStorage('analyze-dynamic-steps', true)
const moduleDynamicTimeout = useStorage('analyze-dynamic-timeout', true)
const moduleUiux = useStorage('analyze-module-uiux', true)
const moduleWording = useStorage('analyze-module-wording', true)
const moduleGameDesign = useStorage('analyze-module-game-design', true)
const moduleTestFlows = useStorage('analyze-module-test-flows', true)
const moduleRunTests = useStorage('analyze-module-run-tests', false)
watch(moduleTestFlows, (val) => { if (!val) moduleRunTests.value = false })
const selectedViewport = useStorage('analyze-viewport', DEFAULT_VIEWPORT)

// Multi-device mode state
const multiDeviceMode = useStorage('analyze-multi-device', false)
const multiDevices = useStorage('analyze-multi-devices', {
  desktop: { enabled: true, viewport: 'desktop-std' },
  ios: { enabled: true, viewport: 'iphone-16' },
  android: { enabled: true, viewport: 'pixel-9' },
})
const recentAnalyses = ref([])
const currentAnalysisId = ref(null)

// Flow preview state
const flowDialogOpen = ref(false)
const previewFlowData = ref(null)
const previewFlowYaml = ref('')
const { copied: flowCopied, copy: copyFlowToClipboard } = useClipboard({ copiedDuring: 2000 })

// Agent screenshot preview state
const agentScreenshotOpen = ref(false)
const agentScreenshotStep = ref(null)

// Debug log clipboard
const { copied: logCopied, copy: copyLogToClipboard } = useClipboard({ copiedDuring: 2000 })

const {
  status,
  currentStep,
  analysisId,
  pageMeta,
  analysis,
  flows: flowList,
  agentSteps: agentStepsList,
  agentMode,
  error: analysisError,
  logs,
  elapsedSeconds,
  stepTimings,
  formatElapsed,
  start,
  reset,
  tryRecover,
  // Live agent exploration
  liveAgentSteps,
  latestScreenshot,
  agentReasoning,
  hintCooldown,
  agentStepCurrent,
  agentStepTotal,
  sendHint,
  // Continue from checkpoint
  continueAnalysis,
  // Start batch tracking
  startBatch,
  // Failed step tracking
  failedStep,
  // Persisted agent steps
  persistedAgentSteps,
  loadPersistedSteps,
  // Live step message
  latestStepMessage,
  // Auto-created test plan
  autoTestPlanId,
  // Inline browser test execution
  testRunId,
  testStepScreenshots,
  testFlowProgress,
  // Multi-device batch
  devices,
  // Multi-device progress tracking
  currentDeviceIndex,
  currentDeviceTotal,
  currentDeviceCategory,
} = useAnalysis()


const activeViewport = computed(() => getViewportByName(selectedViewport.value))

// Format device category name for display
function formatDeviceName(category) {
  if (!category) return ''
  if (category === 'ios') return 'iOS'
  if (category === 'android') return 'Android'
  return category.charAt(0).toUpperCase() + category.slice(1)
}

// Device label for progress panels during batch analysis
const deviceLabel = computed(() => {
  if (currentDeviceTotal.value <= 1) return ''
  const name = formatDeviceName(currentDeviceCategory.value)
  return `${name} ${currentDeviceIndex.value}/${currentDeviceTotal.value}`
})

// Filter viewport categories for multi-device mode based on device type
function viewportCategoriesForDevice(deviceCategory) {
  const allCats = getViewportCategories()
  switch (deviceCategory) {
    case 'desktop':
      return allCats.filter(c => c.name === 'Desktop')
    case 'ios':
      return allCats.filter(c => c.name === 'iPhone' || c.name === 'iPad')
    case 'android':
      return allCats.filter(c => c.name === 'Android' || c.name === 'Android Tablet')
    default:
      return allCats
  }
}

const activeProfile = computed(() => {
  if (selectedProfile.value === 'custom') return null
  return getProfileByName(selectedProfile.value)
})

const profileParams = computed(() => {
  if (selectedProfile.value === 'custom') {
    const params = {
      model: customModel.value || undefined,
      maxTokens: customMaxTokens.value || undefined,
      temperature: customTemperature.value,
    }
    if (useAgentMode.value) {
      params.agentSteps = customAgentSteps.value || undefined
      if (moduleDynamicSteps.value) {
        params.adaptive = true
        params.maxTotalSteps = customMaxTotalSteps.value || undefined
      }
      if (moduleDynamicTimeout.value) {
        params.adaptiveTimeout = true
        params.maxTotalTimeout = customMaxTotalTimeout.value || undefined
      }
    }
    return params
  }
  const p = activeProfile.value
  if (!p) return {}
  const params = {
    model: p.model,
    maxTokens: p.maxTokens,
    temperature: p.temperature,
  }
  if (useAgentMode.value) {
    params.agentSteps = p.agentSteps
    // Adaptive steps — always from agent module toggle
    if (moduleDynamicSteps.value) {
      params.adaptive = true
      params.maxTotalSteps = p.maxTotalSteps || 35
    }
    // Adaptive timeout
    if (moduleDynamicTimeout.value) {
      params.adaptiveTimeout = true
      params.maxTotalTimeout = p.maxTotalTimeout || 25
    }
  }
  return params
})

function onProfileChange(val) {
  selectedProfile.value = val
  showCustomFields.value = val === 'custom'
  if (val !== 'custom') {
    const p = getProfileByName(val)
    if (p) {
      customModel.value = p.model
      customMaxTokens.value = p.maxTokens
      customAgentSteps.value = p.agentSteps
      customTemperature.value = p.temperature
      customMaxTotalSteps.value = p.maxTotalSteps || 35
      customMaxTotalTimeout.value = p.maxTotalTimeout || 25
      // Sync agent module toggles from profile defaults
      moduleDynamicSteps.value = p.adaptive || false
      moduleDynamicTimeout.value = p.adaptiveTimeout || false
    }
  }
}

const gameName = computed(() => {
  return analysis.value?.gameInfo?.name || pageMeta.value?.title || 'Unknown Game'
})

const framework = computed(() => {
  return pageMeta.value?.framework || 'unknown'
})

const flowCount = computed(() => {
  return flowList.value?.length || 0
})

// JWT token expiration warning
const tokenWarning = computed(() => {
  if (!gameUrl.value || !isValidUrl(gameUrl.value)) return ''
  try {
    const u = new URL(gameUrl.value)
    const warnings = []
    for (const [param, value] of u.searchParams) {
      const parts = value.split('.')
      if (parts.length !== 3) continue
      try {
        const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')))
        if (typeof payload.exp !== 'number') continue
        const now = Date.now() / 1000
        if (payload.exp < now) {
          const agoSec = Math.round(now - payload.exp)
          let agoStr
          if (agoSec < 60) agoStr = `${agoSec}s ago`
          else if (agoSec < 3600) agoStr = `${Math.round(agoSec / 60)}m ago`
          else agoStr = `${Math.round(agoSec / 3600)}h ago`
          warnings.push(`The ${param} in this URL expired ${agoStr}.`)
        }
      } catch {
        // Not a valid JWT payload — skip
      }
    }
    if (warnings.length === 0) return ''
    return warnings.join(' ') + ' The game may not load during analysis.'
  } catch {
    return ''
  }
})

// --- Rich progress detail computeds ---

const scoutingDetails = computed(() => {
  if (!pageMeta.value) return []
  const details = []
  if (pageMeta.value.title) details.push({ label: 'Title', value: pageMeta.value.title })
  details.push({ label: 'Framework', value: pageMeta.value.framework || 'unknown' })
  details.push({ label: 'Canvas', value: pageMeta.value.canvasFound ? 'Detected' : 'Not found' })
  details.push({ label: 'Scripts', value: `${pageMeta.value.scriptSrcs?.length || 0} found` })
  if (pageMeta.value.jsGlobals?.length) {
    details.push({ label: 'JS Globals', value: pageMeta.value.jsGlobals.join(', ') })
  }
  if (pageMeta.value.screenshotB64) {
    const sizeKB = Math.round(pageMeta.value.screenshotB64.length * 3 / 4 / 1024)
    details.push({ label: 'Screenshot', value: `Captured (${sizeKB} KB)` })
  }
  return details
})

const analyzingDetail = computed(() => {
  const dur = stepDuration('analyzing')
  if (analysis.value) {
    const mode = pageMeta.value?.screenshotB64 ? 'multimodal' : 'text-only'
    return `${mode} analysis${dur ? ` in ${dur}s` : ''}`
  }
  // Show the latest log message for the analyzing step
  const lastAnalyzingLog = logs.value.filter(l => l.includes('AI') || l.includes('multimodal') || l.includes('Sending')).pop()
  return lastAnalyzingLog || 'Waiting for AI response...'
})

const analysisDetails = computed(() => {
  if (!analysis.value) return []
  const details = []
  if (analysis.value.gameInfo?.name) {
    details.push({ label: 'Game', value: `${analysis.value.gameInfo.name}${analysis.value.gameInfo.genre ? ' (' + analysis.value.gameInfo.genre + ')' : ''}` })
  }
  if (analysis.value.gameInfo?.technology) {
    details.push({ label: 'Technology', value: analysis.value.gameInfo.technology })
  }
  if (analysis.value.mechanics?.length) {
    details.push({ label: 'Mechanics', value: `${analysis.value.mechanics.length} found` })
  }
  if (analysis.value.uiElements?.length) {
    details.push({ label: 'UI Elements', value: `${analysis.value.uiElements.length} found` })
  }
  if (analysis.value.userFlows?.length) {
    details.push({ label: 'User Flows', value: `${analysis.value.userFlows.length} identified` })
  }
  if (analysis.value.edgeCases?.length) {
    details.push({ label: 'Edge Cases', value: `${analysis.value.edgeCases.length} identified` })
  }
  if (analysis.value.uiuxAnalysis?.length) {
    details.push({ label: 'UI/UX Issues', value: `${analysis.value.uiuxAnalysis.length} found` })
  }
  if (analysis.value.wordingCheck?.length) {
    details.push({ label: 'Wording Issues', value: `${analysis.value.wordingCheck.length} found` })
  }
  if (analysis.value.gameDesign?.length) {
    details.push({ label: 'Game Design', value: `${analysis.value.gameDesign.length} observations` })
  }
  return details
})

const scenariosDetail = computed(() => {
  const dur = stepDuration('scenarios')
  if (stepOrder(currentStep.value) > stepOrder('scenarios_done')) {
    return `Scenarios generated${dur ? ` in ${dur}s` : ''}`
  }
  if (currentStep.value === 'scenarios_done') {
    return `Scenarios generated${dur ? ` in ${dur}s` : ''}`
  }
  return dur ? `Working... (${dur}s)` : ''
})

const flowsDetail = computed(() => {
  if (flowList.value.length) {
    const dur = stepDuration('flows')
    return `${flowList.value.length} flow(s) generated${dur ? ` in ${dur}s` : ''}`
  }
  const flowSteps = ['flows', 'flows_prompt', 'flows_calling', 'flows_parsing', 'flows_validating', 'flows_retry']
  if (flowSteps.includes(currentStep.value) && latestStepMessage.value) {
    return latestStepMessage.value
  }
  const dur = stepDuration('flows')
  return dur ? `Working... (${dur}s)` : ''
})

const flowsSubDetails = computed(() => {
  const details = []
  if (analysis.value?.scenarios?.length) {
    details.push({ label: 'Scenarios', value: `${analysis.value.scenarios.length}` })
  }
  const flowsLog = logs.value.find(l => l.startsWith('Converting') && l.includes('scenarios'))
  if (flowsLog) {
    const match = flowsLog.match(/scenarios to Maestro flows: (.+)/)
    if (match) {
      const names = match[1].split(', ')
      names.forEach(name => {
        details.push({ label: 'Flow', value: name.trim() })
      })
    }
  }
  return details
})

const testPlanDetail = computed(() => {
  if (autoTestPlanId.value) {
    const dur = stepDuration('test_plan')
    return `Test plan created${dur ? ` in ${dur}s` : ''}`
  }
  const planSteps = ['test_plan', 'test_plan_checking', 'test_plan_flows', 'test_plan_saving', 'test_plan_done']
  if (planSteps.includes(currentStep.value) && latestStepMessage.value) {
    return latestStepMessage.value
  }
  const dur = stepDuration('test_plan')
  return dur ? `Working... (${dur}s)` : ''
})

const testPlanSubDetails = computed(() => {
  const details = []
  if (flowList.value.length) {
    details.push({ label: 'Flows', value: `${flowList.value.length} included` })
  }
  flowList.value.forEach(flow => {
    details.push({ label: 'Flow', value: flow.name })
  })
  if (analysis.value?.gameInfo?.name) {
    details.push({ label: 'Game', value: analysis.value.gameInfo.name })
  }
  return details
})

const testingDetail = computed(() => {
  const passed = testFlowProgress.value.filter(f => f.status === 'passed').length
  const failed = testFlowProgress.value.filter(f => f.status === 'failed').length
  const total = testFlowProgress.value.length
  if (total > 0) {
    const dur = stepDuration('testing')
    return `${passed} passed, ${failed} failed of ${total} flow(s)${dur ? ` (${dur}s)` : ''}`
  }
  if (testStepScreenshots.value.length > 0) {
    return `Running... ${testStepScreenshots.value.length} step(s) captured`
  }
  const dur = stepDuration('testing')
  return dur ? `Running browser tests... (${dur}s)` : 'Starting browser tests...'
})

// Prefer persisted steps (have screenshots via URL), fall back to live or agentStepsList
const navigatorSteps = computed(() => {
  if (persistedAgentSteps.value.length) return persistedAgentSteps.value
  if (liveAgentSteps.value.length) {
    return liveAgentSteps.value.filter(s => s.type === 'tool')
  }
  return agentStepsList.value || []
})

const agentExplorationStatus = computed(() => {
  const agentSteps = ['agent_start', 'agent_step', 'agent_action', 'agent_adaptive', 'agent_timeout_extend']
  const doneSteps = ['agent_done', 'agent_synthesize', 'synthesis_retry', 'analyzing', 'analyzed', 'flows', 'flows_retry', 'flows_done', 'complete']
  if (doneSteps.includes(currentStep.value)) return 'complete'
  if (agentSteps.includes(currentStep.value)) return 'active'
  if (stepOrder(currentStep.value) < stepOrder('agent_start')) return 'pending'
  return 'pending'
})

const failedPhaseLabel = computed(() => {
  if (!failedStep.value) return ''
  const map = {
    agent_step: 'Exploration',
    agent_action: 'Exploration',
    agent_start: 'Exploration',
    agent_adaptive: 'Exploration',
    agent_timeout_extend: 'Exploration',
    agent_synthesize: 'Synthesis',
    synthesis_retry: 'Synthesis',
    analyzing: 'Analysis',
    analyzed: 'Analysis',
    flows: 'Flow Generation',
    flows_prompt: 'Flow Generation',
    flows_calling: 'Flow Generation',
    flows_parsing: 'Flow Generation',
    flows_validating: 'Flow Generation',
    flows_retry: 'Flow Generation',
    flows_done: 'Flow Generation',
    saving: 'Saving Flows',
    test_plan: 'Test Plan Creation',
    test_plan_checking: 'Test Plan Creation',
    test_plan_flows: 'Test Plan Creation',
    test_plan_saving: 'Test Plan Creation',
    test_plan_done: 'Test Plan Creation',
    testing: 'Browser Testing',
    testing_started: 'Browser Testing',
    testing_done: 'Browser Testing',
    scouting: 'Page Scouting',
    scouted: 'Page Scouting',
    scenarios: 'Scenario Generation',
  }
  return map[failedStep.value] || failedStep.value
})

const progressPhases = computed(() => {
  const isQueued = status.value === 'queued'
  // Append device suffix to active phase labels during batch analysis
  const suffix = deviceLabel.value ? ` [${deviceLabel.value}]` : ''
  const withSuffix = (label, phaseStatus) => {
    return phaseStatus === 'active' && suffix ? label + suffix : label
  }

  const scoutingStatus = isQueued ? 'active' : granularStepStatus('scouting')
  const phases = [{
    id: 'scouting', label: withSuffix(isQueued ? 'Queued' : 'Scouting page', scoutingStatus), icon: isQueued ? 'Clock' : 'Radar', color: isQueued ? 'amber' : 'blue',
    status: scoutingStatus,
    detail: isQueued ? 'Another analysis is running. Waiting in queue...' : stepDuration('scouting') ? `Completed in ${stepDuration('scouting')}s` : 'Fetching page and extracting metadata...',
    durationSeconds: stepDuration('scouting'),
    subDetails: scoutingDetails.value,
  }]
  if (agentMode.value) {
    const agentStatus = agentExplorationStatus.value
    phases.push({
      id: 'agent', label: withSuffix('Agent exploring game', agentStatus), icon: 'Bot', color: 'purple',
      status: agentStatus,
      detail: agentExplorationDetail.value,
      durationSeconds: null, subDetails: [], isAgentSlot: true,
    })
  }
  const analyzingStatus = granularStepStatus('analyzing')
  const scenariosStatus = granularStepStatus('scenarios')
  const flowsStatus = granularStepStatus('flows')
  const testPlanStatus = granularStepStatus('test_plan')
  phases.push(
    { id: 'analyzing', label: withSuffix(agentMode.value ? 'Synthesizing analysis' : 'Analyzing game mechanics', analyzingStatus),
      icon: 'Brain', color: 'amber', status: analyzingStatus,
      detail: analyzingDetail.value, durationSeconds: stepDuration('analyzing'),
      subDetails: analysisDetails.value },
    { id: 'scenarios', label: withSuffix('Generating test scenarios', scenariosStatus), icon: 'ListTree', color: 'emerald',
      status: scenariosStatus, detail: scenariosDetail.value,
      durationSeconds: stepDuration('scenarios'), subDetails: [] },
    { id: 'flows', label: withSuffix('Generating test flows', flowsStatus), icon: 'PlayCircle', color: 'rose',
      status: flowsStatus, detail: flowsDetail.value,
      durationSeconds: stepDuration('flows'), subDetails: flowsSubDetails.value },
    { id: 'test_plan', label: withSuffix('Creating test plan', testPlanStatus), icon: 'ClipboardCheck', color: 'sky',
      status: testPlanStatus, detail: testPlanDetail.value,
      durationSeconds: stepDuration('test_plan'), subDetails: testPlanSubDetails.value },
  )
  if (moduleRunTests.value) {
    const testingStatus = granularStepStatus('testing')
    phases.push({
      id: 'testing', label: withSuffix('Running browser tests', testingStatus), icon: 'FlaskConical', color: 'violet',
      status: testingStatus,
      detail: testingDetail.value,
      durationSeconds: stepDuration('testing'),
      subDetails: [],
      isTestingSlot: true,
    })
  }
  return phases
})

const agentExplorationDetail = computed(() => {
  if (!agentMode.value) return ''
  const lastAgentLog = logs.value.filter(l => l.includes('Step') || l.includes('agent')).pop()
  if (currentStep.value === 'agent_done' || stepOrder(currentStep.value) > stepOrder('agent_done')) {
    return `Exploration complete (${agentStepsList.value?.length || 0} steps)`
  }
  return lastAgentLog || 'AI is exploring the game...'
})

function expandAgentScreenshot(step) {
  agentScreenshotStep.value = step
  agentScreenshotOpen.value = true
}

function expandLiveScreenshot() {
  if (latestScreenshot.value) {
    agentScreenshotStep.value = {
      screenshotUrl: latestScreenshot.value,
      stepNumber: agentStepCurrent.value || '?',
      toolName: 'Live Screenshot',
      result: 'Current game state',
    }
    agentScreenshotOpen.value = true
  }
}

function expandStepScreenshot(entry) {
  agentScreenshotStep.value = {
    screenshotUrl: entry.screenshotUrl || entry.screenshotB64,
    stepNumber: entry.stepNumber,
    toolName: entry.toolName,
    result: entry.result,
  }
  agentScreenshotOpen.value = true
}

// Ordered step names for granular progress
const STEP_ORDER = ['scouting', 'scouted', 'device_transition', 'agent_start', 'agent_step', 'agent_action', 'agent_adaptive', 'agent_timeout_extend', 'agent_done', 'agent_synthesize', 'synthesis_retry', 'analyzing', 'analyzed', 'scenarios', 'scenarios_done', 'flows', 'flows_prompt', 'flows_calling', 'flows_parsing', 'flows_validating', 'flows_retry', 'flows_done', 'saving', 'test_plan', 'test_plan_checking', 'test_plan_flows', 'test_plan_saving', 'test_plan_done', 'testing', 'testing_started', 'testing_done', 'complete']

function stepOrder(step) {
  const idx = STEP_ORDER.indexOf(step)
  return idx >= 0 ? idx : -1
}

/**
 * Determine the status of a progress step group based on the current granular step.
 * Each ProgressStep represents a group of granular steps:
 *   scouting  → scouting, scouted
 *   analyzing → analyzing, analyzed
 *   scenarios → scenarios, scenarios_done
 *   flows     → flows, flows_done, saving
 */
function granularStepStatus(groupStart) {
  const groupMap = {
    scouting: { start: 'scouting', end: 'scouted' },
    analyzing: { start: 'analyzing', end: 'analyzed' },
    scenarios: { start: 'scenarios', end: 'scenarios_done' },
    flows: { start: 'flows', end: 'saving' },
    test_plan: { start: 'test_plan', end: 'test_plan_done' },
    testing: { start: 'testing', end: 'testing_done' },
  }
  const group = groupMap[groupStart]
  if (!group) return 'pending'

  const current = stepOrder(currentStep.value)
  const groupStartIdx = stepOrder(group.start)
  const groupEndIdx = stepOrder(group.end)

  if (current < 0 || current < groupStartIdx) return 'pending'
  if (current > groupEndIdx) return 'complete'
  return 'active'
}

function stepDuration(stepName) {
  const timing = stepTimings.value[stepName]
  if (!timing || !timing.start) return null
  const end = timing.end || Date.now()
  return ((end - timing.start) / 1000).toFixed(1)
}

function formatStepTimingSummary() {
  const labels = { scouting: 'Scouting', analyzing: 'Analyzing', scenarios: 'Scenarios', flows: 'Flows' }
  return Object.entries(labels)
    .map(([key, label]) => {
      const d = stepDuration(key)
      return d ? `${label}: ${d}s` : null
    })
    .filter(Boolean)
    .join(' | ')
}

function parseUrlHints(urlStr) {
  try {
    const url = new URL(urlStr)
    const hints = {}
    hints.domain = url.hostname
    const interestingParams = ['game_type', 'mode', 'game_id', 'operator_id', 'gameType', 'gameid']
    for (const [key, value] of url.searchParams) {
      if (interestingParams.includes(key) || value) {
        hints[key] = value
      }
    }
    return hints
  } catch {
    return {}
  }
}

async function handleAnalyze() {
  if (!isValidUrl(gameUrl.value)) return
  analyzing.value = true
  const modules = {
    uiux: moduleUiux.value,
    wording: moduleWording.value,
    gameDesign: moduleGameDesign.value,
    testFlows: moduleTestFlows.value,
    runTests: moduleRunTests.value,
  }
  const params = { ...profileParams.value }

  // Multi-device mode: launch batch analysis
  if (multiDeviceMode.value) {
    const devices = []
    for (const [category, cfg] of Object.entries(multiDevices.value)) {
      if (cfg.enabled) {
        devices.push({ category, viewport: cfg.viewport })
      }
    }
    if (devices.length === 0) {
      analyzing.value = false
      return
    }
    try {
      const batchReq = {
        gameUrl: gameUrl.value,
        projectId: projectId.value || '',
        agentMode: useAgentMode.value,
        modules,
        devices,
        ...params,
      }
      const result = await analyzeApi.batchAnalyze(batchReq)
      // Track the single unified analysis
      if (result.analysisId) {
        startBatch(result.analysisId, gameUrl.value, useAgentMode.value)
      }
    } catch {
      analyzing.value = false
    }
    return
  }

  // Single device mode
  if (selectedViewport.value && selectedViewport.value !== DEFAULT_VIEWPORT) {
    params.viewport = selectedViewport.value
  }
  try {
    await start(gameUrl.value, projectId.value, useAgentMode.value, params, modules)
  } catch {
    analyzing.value = false
  }
}

// Track the analysis ID for export
watch(analysisId, (val) => {
  if (val) currentAnalysisId.value = val
})

// Reset analyzing flag when status changes from idle
watch(status, (val) => {
  if (val !== 'idle') {
    analyzing.value = false
  }
  if (val === 'error') {
    checkCanContinue()
  }
})

function handleReset() {
  reset()
  analyzing.value = false
  showCustomFields.value = false
  gameUrl.value = ''
  currentAnalysisId.value = null
  loadRecentAnalyses()
}

function navigateToNewPlan() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  const query = { gameUrl: gameUrl.value }
  if (currentAnalysisId.value) {
    query.analysisId = currentAnalysisId.value
  } else {
    query.flows = flowList.value.map((f) => f.name).join(',')
  }
  router.push({ path: `${basePath}/tests/new`, query })
}

function navigateToTestPlan() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/tests/plans/${autoTestPlanId.value}`)
}

function navigateToFlows() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/flows`)
}

const canContinue = ref(false)

// Check if a failed analysis has checkpoint data for resume
async function checkCanContinue() {
  canContinue.value = false
  const id = currentAnalysisId.value || analysisId.value
  if (!id) return
  try {
    const data = await analysesApi.get(id)
    canContinue.value = !!(data.partialResult)
  } catch {
    // ignore
  }
}

async function handleContinueAnalysis() {
  const id = currentAnalysisId.value || analysisId.value
  if (!id) return
  canContinue.value = false
  await continueAnalysis(id)
}

function retryAnalysis() {
  const url = gameUrl.value
  const agent = agentMode.value
  // Preserve multi-device state before reset
  const wasMultiDevice = multiDeviceMode.value
  const savedDevices = wasMultiDevice
    ? JSON.parse(JSON.stringify(multiDevices.value))
    : null
  canContinue.value = false
  reset()
  analyzing.value = false
  gameUrl.value = url
  useAgentMode.value = agent
  // Restore multi-device settings
  if (wasMultiDevice && savedDevices) {
    multiDeviceMode.value = true
    for (const [category, cfg] of Object.entries(savedDevices)) {
      if (multiDevices.value[category]) {
        multiDevices.value[category].enabled = cfg.enabled
        multiDevices.value[category].viewport = cfg.viewport
      }
    }
  }
  nextTick(() => handleAnalyze())
}

function reAnalyze(item) {
  gameUrl.value = item.gameUrl
  handleAnalyze()
}

async function deleteAnalysis(item) {
  if (!confirm(`Delete analysis "${item.gameName || item.id}"? This cannot be undone.`)) return
  try {
    await analysesApi.delete(item.id)
    recentAnalyses.value = recentAnalyses.value.filter((a) => a.id !== item.id)
  } catch (err) {
    console.error('Failed to delete analysis:', err)
  }
}

function exportAnalysis(format) {
  if (!currentAnalysisId.value) return
  analysesApi.export(currentAnalysisId.value, format).catch((err) => {
    console.error('Failed to export analysis:', err)
  })
}

function previewFlow(flow) {
  previewFlowData.value = flow
  flowCopied.value = false

  // Build a simple YAML representation from the flow object
  let yaml = ''
  if (flow.url) yaml += `url: ${flow.url}\n`
  if (flow.appId) yaml += `appId: ${flow.appId}\n`
  if (flow.tags?.length) {
    yaml += 'tags:\n'
    flow.tags.forEach((t) => { yaml += `  - ${t}\n` })
  }
  yaml += '---\n'
  if (flow.commands?.length) {
    flow.commands.forEach((cmd) => {
      for (const [key, val] of Object.entries(cmd)) {
        if (key === 'comment') {
          yaml += `# ${val}\n`
        } else if (typeof val === 'object' && val !== null) {
          yaml += `- ${key}:\n`
          for (const [sk, sv] of Object.entries(val)) {
            yaml += `    ${sk}: ${sv}\n`
          }
        } else {
          yaml += `- ${key}: ${val}\n`
        }
      }
    })
  }
  previewFlowYaml.value = yaml || 'No YAML content available'
  flowDialogOpen.value = true
}

async function copyFlowYaml() {
  await copyFlowToClipboard(previewFlowYaml.value)
}

function buildDebugLogText() {
  const lines = []
  lines.push('=== Wizards QA Debug Log ===')
  lines.push(`Date: ${new Date().toISOString()}`)
  if (currentAnalysisId.value) lines.push(`Analysis ID: ${currentAnalysisId.value}`)
  if (gameUrl.value) lines.push(`Game URL: ${gameUrl.value}`)
  lines.push(`Status: ${status.value}`)
  lines.push(`Agent Mode: ${agentMode.value ? 'yes' : 'no'}`)
  lines.push(`Framework: ${framework.value}`)
  lines.push(`Flows: ${flowCount.value}`)
  if (tokenWarning.value) {
    lines.push(`Token Warning: ${tokenWarning.value}`)
  }
  lines.push('')

  // Duration and step timings
  if (elapsedSeconds.value > 0) {
    lines.push(`Duration: ${formatElapsed(elapsedSeconds.value)}`)
  }
  const timingSummary = formatStepTimingSummary()
  if (timingSummary) {
    lines.push(`Step Timings: ${timingSummary}`)
  }
  lines.push('')

  // Error
  if (analysisError.value) {
    lines.push(`Error: ${analysisError.value}`)
    lines.push('')
  }

  // Full progress log
  lines.push(`--- Progress Log (${logs.value.length} lines) ---`)
  logs.value.forEach((line) => lines.push(line))
  lines.push('')

  // Analysis summary
  if (analysis.value) {
    lines.push('--- Analysis Summary ---')
    lines.push(`Mechanics: ${analysis.value.mechanics?.length || 0}`)
    lines.push(`UI Elements: ${analysis.value.uiElements?.length || 0}`)
    lines.push(`User Flows: ${analysis.value.userFlows?.length || 0}`)
    lines.push(`Edge Cases: ${analysis.value.edgeCases?.length || 0}`)
    lines.push(`UI/UX Issues: ${analysis.value.uiuxAnalysis?.length || 0}`)
    lines.push(`Wording Issues: ${analysis.value.wordingCheck?.length || 0}`)
    lines.push(`Game Design: ${analysis.value.gameDesign?.length || 0}`)
    lines.push('')
  }

  // Agent steps
  const stepsSource = persistedAgentSteps.value.length
    ? persistedAgentSteps.value
    : liveAgentSteps.value.filter(s => s.type === 'tool')
  if (stepsSource.length) {
    lines.push(`--- Agent Steps (${stepsSource.length}) ---`)
    stepsSource.forEach((step) => {
      const num = step.stepNumber || '?'
      const tool = step.toolName || 'unknown'
      const dur = step.durationMs != null ? `${step.durationMs}ms` : ''
      const err = step.error ? ` [ERROR: ${step.error}]` : ''
      lines.push(`  Step ${num}: ${tool} ${dur}${err}`)
      if (step.input) {
        const inputTrunc = step.input.length > 200 ? step.input.slice(0, 200) + '...' : step.input
        lines.push(`    Input: ${inputTrunc}`)
      }
      if (step.result) {
        const resultTrunc = step.result.length > 200 ? step.result.slice(0, 200) + '...' : step.result
        lines.push(`    Result: ${resultTrunc}`)
      }
      if (step.reasoning) {
        const reasoningTrunc = step.reasoning.length > 150 ? step.reasoning.slice(0, 150) + '...' : step.reasoning
        lines.push(`    Reasoning: ${reasoningTrunc}`)
      }
    })
    lines.push('')
  }

  // Test flow progress
  if (testFlowProgress.value.length) {
    lines.push(`--- Test Flow Results (${testFlowProgress.value.length}) ---`)
    testFlowProgress.value.forEach((f) => {
      const dur = f.duration ? ` (${f.duration})` : ''
      lines.push(`  ${f.flowName}: ${f.status}${dur}`)
    })
    lines.push('')
  }

  // Test step details (skip base64 screenshots)
  if (testStepScreenshots.value.length) {
    lines.push(`--- Test Steps (${testStepScreenshots.value.length}) ---`)
    testStepScreenshots.value.forEach((s) => {
      const reason = s.reasoning ? ` | ${s.reasoning.slice(0, 100)}` : ''
      lines.push(`  ${s.flowName} step ${s.stepIndex}: ${s.command} → ${s.status}${reason}`)
    })
    lines.push('')
  }

  // Last agent reasoning
  if (agentReasoning.value) {
    lines.push('--- Last Agent Reasoning ---')
    lines.push(agentReasoning.value.length > 500 ? agentReasoning.value.slice(0, 500) + '...' : agentReasoning.value)
    lines.push('')
  }

  // Flow names + command counts
  if (flowList.value.length) {
    lines.push('--- Generated Flows ---')
    flowList.value.forEach((f) => {
      lines.push(`  ${f.name}: ${f.commands?.length || 0} commands`)
    })
    lines.push('')
  }

  lines.push('=== End Debug Log ===')
  return lines.join('\n')
}

async function copyDebugLog() {
  await copyLogToClipboard(buildDebugLogText())
}

function navigateToAnalysesList() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/analyses`)
}

function viewAnalysis(item) {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/analyses/${item.id}`)
}

function viewCurrentAnalysis() {
  if (!currentAnalysisId.value) return
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/analyses/${currentAnalysisId.value}`)
}

async function loadRecentAnalyses() {
  try {
    const data = projectId.value
      ? await projectsApi.analyses(projectId.value)
      : await analysesApi.list()
    const all = data.analyses || []
    recentAnalyses.value = all.slice(-3).reverse()
  } catch {
    // Silently ignore — analyses endpoint may not have data yet
  }
}



onMounted(async () => {
  // Pre-fill game URL from project context
  if (currentProject.value?.gameUrl && !gameUrl.value) {
    gameUrl.value = currentProject.value.gameUrl
  }

  // Use explicit analysisId from query param (e.g., navigated from list),
  // or fall back to localStorage recovery
  const explicitId = route.query.analysisId || null
  const recovery = await tryRecover(explicitId)
  if (recovery) {
    if (recovery.gameUrl) {
      gameUrl.value = recovery.gameUrl
    }
    if (recovery.status === 'running') {
      logs.value = [...logs.value, 'Reconnected to running analysis...']
    }
    // If recovered as completed, the status/data refs are already set
    return
  }

  loadRecentAnalyses()
})
</script>
