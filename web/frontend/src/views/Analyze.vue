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
          <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
            <input type="checkbox" v-model="useAgentMode" class="rounded border-gray-300" />
            Agent Mode
            <span class="text-xs text-muted-foreground">(AI explores the game interactively)</span>
          </label>
        </div>

        <!-- Agent Modules (only when Agent Mode is on) -->
        <div v-if="useAgentMode" class="space-y-3 pt-2 border-t">
          <div class="flex items-center gap-3">
            <Zap class="h-4 w-4 text-muted-foreground shrink-0" />
            <label class="text-sm font-medium">Agent Modules</label>
          </div>
          <div class="ml-7 grid gap-2 sm:grid-cols-2">
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" v-model="moduleDynamicSteps" class="rounded border-gray-300" />
              <TrendingUp class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Dynamic Steps</span>
              <span class="text-xs text-muted-foreground">(AI requests more steps as needed)</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" v-model="moduleDynamicTimeout" class="rounded border-gray-300" />
              <Timer class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Dynamic Timeout</span>
              <span class="text-xs text-muted-foreground">(AI extends time for thorough testing)</span>
            </label>
          </div>
        </div>

        <!-- Analysis Modules -->
        <div class="space-y-3 pt-2 border-t">
          <div class="flex items-center gap-3">
            <Sparkles class="h-4 w-4 text-muted-foreground shrink-0" />
            <label class="text-sm font-medium">Analysis Modules</label>
          </div>
          <div class="ml-7 grid gap-2 sm:grid-cols-2">
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" v-model="moduleUiux" class="rounded border-gray-300" />
              <Eye class="h-3.5 w-3.5 text-muted-foreground" />
              <span>UI/UX Analysis</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" v-model="moduleWording" class="rounded border-gray-300" />
              <Type class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Wording Check</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" v-model="moduleGameDesign" class="rounded border-gray-300" />
              <Gamepad2 class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Game Design</span>
            </label>
            <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" v-model="moduleTestFlows" class="rounded border-gray-300" />
              <PlayCircle class="h-3.5 w-3.5 text-muted-foreground" />
              <span>Test Flows</span>
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
    <div v-else-if="status === 'scouting' || status === 'analyzing' || status === 'generating'" class="space-y-4">
      <Card>
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle class="truncate max-w-[calc(100%-6rem)]" :title="gameUrl">Analyzing: {{ truncateUrl(gameUrl) }}</CardTitle>
            <span v-if="elapsedSeconds > 0" class="text-sm text-muted-foreground">
              {{ formatElapsed(elapsedSeconds) }}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <div class="space-y-1">
            <ProgressStep
              :status="granularStepStatus('scouting')"
              label="Scouting page"
              :detail="stepDuration('scouting') ? `Completed in ${stepDuration('scouting')}s` : 'Fetching page and extracting metadata...'"
              :sub-details="scoutingDetails"
            />
            <!-- Agent Live Exploration Panel -->
            <template v-if="agentMode && (agentExplorationStatus === 'active' || liveAgentSteps.length > 0)">
              <div class="rounded-lg border bg-card p-4 my-2">
                <!-- Header -->
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-2">
                    <Loader2 v-if="agentExplorationStatus === 'active'" class="h-4 w-4 text-primary animate-spin" />
                    <CheckCircle v-else class="h-4 w-4 text-green-500" />
                    <span class="text-sm font-medium">Agent Exploring Game</span>
                  </div>
                  <div class="flex items-center gap-3 text-xs text-muted-foreground">
                    <span v-if="agentStepCurrent">Step {{ agentStepCurrent }}/{{ agentStepTotal }}</span>
                    <span v-if="elapsedSeconds > 0">{{ formatElapsed(elapsedSeconds) }}</span>
                  </div>
                </div>

                <!-- Screenshot + Reasoning Row -->
                <div class="flex gap-4 mb-3" v-if="latestScreenshot || agentReasoning">
                  <img
                    v-if="latestScreenshot"
                    :src="'data:image/jpeg;base64,' + latestScreenshot"
                    class="w-[300px] h-auto rounded border cursor-pointer shrink-0 object-contain"
                    alt="Live screenshot"
                    @click="expandLiveScreenshot"
                  />
                  <div v-if="agentReasoning" class="flex-1 min-w-0">
                    <p class="text-xs text-muted-foreground mb-1 font-medium">Latest thinking:</p>
                    <div class="max-h-40 overflow-y-auto text-xs text-muted-foreground leading-relaxed">
                      {{ agentReasoning.length > 500 ? agentReasoning.slice(-500) + '...' : agentReasoning }}
                    </div>
                  </div>
                </div>

                <!-- Step Timeline -->
                <div v-if="liveAgentSteps.length" ref="liveTimelineRef" class="max-h-48 overflow-y-auto space-y-1 mb-3 rounded-md bg-muted/50 p-2">
                  <div
                    v-for="(entry, i) in liveAgentSteps"
                    :key="i"
                    :class="[
                      'flex items-start gap-2 p-1.5 rounded text-xs',
                      entry.type === 'hint' ? 'bg-blue-50 dark:bg-blue-950/30' : ''
                    ]"
                  >
                    <template v-if="entry.type === 'hint'">
                      <MessageCircle class="h-3.5 w-3.5 text-blue-500 shrink-0 mt-0.5" />
                      <span class="text-blue-600 dark:text-blue-400">You: {{ entry.message }}</span>
                    </template>
                    <template v-else>
                      <Badge variant="outline" class="shrink-0 text-[10px] px-1 py-0">{{ entry.stepNumber }}</Badge>
                      <div class="min-w-0 flex-1">
                        <span class="font-medium">{{ entry.toolName }}</span>
                        <span class="text-muted-foreground ml-1">{{ entry.durationMs }}ms</span>
                        <p class="text-muted-foreground truncate" :title="entry.result">{{ entry.result }}</p>
                        <p v-if="entry.error" class="text-destructive">{{ entry.error }}</p>
                      </div>
                      <Badge
                        v-if="entry.hasScreenshot"
                        variant="secondary"
                        class="shrink-0 text-[10px] px-1 py-0 cursor-pointer"
                        @click="expandLiveScreenshot"
                      >screenshot</Badge>
                    </template>
                  </div>
                </div>

                <!-- Hint Input Bar -->
                <div v-if="agentExplorationStatus === 'active'" class="flex gap-2">
                  <Input
                    v-model="hintInput"
                    placeholder="Send a hint to the agent..."
                    :disabled="hintCooldown"
                    class="flex-1 text-sm"
                    @keyup.enter="handleSendHint"
                  />
                  <Button
                    size="sm"
                    :disabled="!hintInput.trim() || hintCooldown"
                    @click="handleSendHint"
                  >
                    <Send class="h-3.5 w-3.5 mr-1" />
                    {{ hintSent ? 'Sent!' : hintCooldown ? 'Wait...' : 'Send' }}
                  </Button>
                </div>
                <p v-else class="text-xs text-muted-foreground">Exploration complete ({{ liveAgentSteps.filter(s => s.type === 'tool').length }} steps)</p>
              </div>
            </template>
            <!-- Fallback: simple ProgressStep when no live data yet -->
            <ProgressStep
              v-else-if="agentMode"
              :status="agentExplorationStatus"
              label="Agent exploring game"
              :detail="agentExplorationDetail"
            />
            <ProgressStep
              :status="granularStepStatus('analyzing')"
              :label="agentMode ? 'Synthesizing analysis' : 'Analyzing game mechanics'"
              :detail="analyzingDetail"
              :sub-details="analysisDetails"
            />
            <ProgressStep
              :status="granularStepStatus('scenarios')"
              label="Generating test scenarios"
              :detail="scenariosDetail"
            />
            <ProgressStep
              :status="granularStepStatus('flows')"
              label="Generating Maestro test flows"
              :detail="flowsDetail"
            />
          </div>

          <Separator class="my-4" />

          <div ref="logContainer" class="max-h-40 overflow-y-auto rounded-md bg-muted p-3">
            <p v-for="(line, i) in logs" :key="i" class="text-xs font-mono text-muted-foreground">
              {{ line }}
            </p>
            <p v-if="!logs.length" class="text-xs text-muted-foreground">Waiting for output...</p>
          </div>

          <div class="flex justify-end mt-4">
            <Button variant="outline" @click="handleReset">Cancel</Button>
          </div>
        </CardContent>
      </Card>
    </div>

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
            <Button @click="navigateToNewPlan">Create Test Plan</Button>
            <Button variant="secondary" @click="runFlowsNow">Run Flows Now</Button>
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
    <Card v-else-if="status === 'error'">
      <CardContent class="pt-6">
        <Alert variant="destructive">
          <AlertTitle>Analysis Failed</AlertTitle>
          <AlertDescription>
            {{ analysisError }}
            <span v-if="failedPhaseLabel" class="block mt-1 text-xs opacity-80">
              Failed during: {{ failedPhaseLabel }}
            </span>
          </AlertDescription>
        </Alert>

        <!-- Show progress steps so user sees where it failed -->
        <div class="mt-4 space-y-1" v-if="currentStep || Object.keys(stepTimings).length">
          <ProgressStep
            :status="granularStepStatus('scouting')"
            label="Scouting page"
            :detail="stepDuration('scouting') ? `${stepDuration('scouting')}s` : ''"
            :sub-details="scoutingDetails"
          />
          <ProgressStep
            v-if="agentMode"
            :status="agentExplorationStatus"
            label="Agent exploring game"
            :detail="agentExplorationDetail"
          />
          <ProgressStep
            :status="granularStepStatus('analyzing')"
            :label="agentMode ? 'Synthesizing analysis' : 'Analyzing game mechanics'"
            :detail="stepDuration('analyzing') ? `${stepDuration('analyzing')}s` : ''"
            :sub-details="analysisDetails"
          />
          <ProgressStep
            :status="granularStepStatus('scenarios')"
            label="Generating test scenarios"
            :detail="stepDuration('scenarios') ? `${stepDuration('scenarios')}s` : ''"
          />
          <ProgressStep
            :status="granularStepStatus('flows')"
            label="Generating Maestro test flows"
            :detail="stepDuration('flows') ? `${stepDuration('flows')}s` : ''"
          />
        </div>

        <!-- Show collected logs -->
        <div v-if="logs.length" class="mt-4">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-muted-foreground font-medium">Log ({{ logs.length }} lines)</span>
            <Button variant="outline" size="sm" class="h-7 text-xs gap-1" @click="copyDebugLog">
              <Copy class="h-3 w-3" />
              {{ logCopied ? 'Copied!' : 'Copy Full Log' }}
            </Button>
          </div>
          <div class="max-h-40 overflow-y-auto rounded-md bg-muted p-3">
            <p v-for="(line, i) in logs" :key="i" class="text-xs font-mono text-muted-foreground">{{ line }}</p>
          </div>
        </div>

        <!-- Agent steps before failure -->
        <div v-if="navigatorSteps.length" class="mt-4">
          <p class="text-sm font-medium mb-2">Agent Steps Before Failure ({{ navigatorSteps.length }})</p>
          <AgentStepNavigator
            v-if="currentAnalysisId || analysisId"
            :analysis-id="currentAnalysisId || analysisId"
            :initial-steps="navigatorSteps"
          />
        </div>

        <!-- Elapsed time at failure -->
        <p v-if="elapsedSeconds > 0" class="mt-2 text-xs text-muted-foreground">
          Failed after {{ formatElapsed(elapsedSeconds) }}
        </p>

        <div class="mt-4 flex gap-2">
          <Button v-if="canContinue" @click="handleContinueAnalysis">
            <PlayCircle class="h-4 w-4 mr-1" />
            Continue Analysis
          </Button>
          <Button @click="retryAnalysis" :variant="canContinue ? 'secondary' : 'default'">
            <RefreshCw class="h-4 w-4 mr-1" />
            Retry Analysis
          </Button>
          <Button variant="outline" @click="handleReset">Start Over</Button>
        </div>
      </CardContent>
    </Card>

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
            v-if="agentScreenshotStep?.screenshotB64"
            :src="'data:image/jpeg;base64,' + agentScreenshotStep.screenshotB64"
            class="w-full rounded-md border"
            alt="Agent step screenshot"
          />
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAnalysis } from '@/composables/useAnalysis'
import { truncateUrl, isValidUrl, severityVariant } from '@/lib/utils'
import { ANALYSIS_PROFILES, getProfileByName } from '@/lib/profiles'
import { testsApi, analysesApi, projectsApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { useClipboard } from '@/composables/useClipboard'
import { useProject } from '@/composables/useProject'
import { RefreshCw, Trash2, Download, MessageCircle, Send, Loader2, CheckCircle, Bug, Copy, AlertCircle, Settings2, ExternalLink, Sparkles, Eye, Type, Gamepad2, PlayCircle, Zap, TrendingUp, Timer } from 'lucide-vue-next'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import ProgressStep from '@/components/ProgressStep.vue'
import AgentStepNavigator from '@/components/AgentStepNavigator.vue'

const router = useRouter()
const route = useRoute()
const { currentProject } = useProject()
const projectId = computed(() => route.params.projectId || '')
const gameUrl = ref('')
const analyzing = ref(false)
const useAgentMode = ref(true)
const selectedProfile = ref('balanced')
const showCustomFields = ref(false)
const defaultProfile = getProfileByName('balanced')
const customModel = ref(defaultProfile.model)
const customMaxTokens = ref(defaultProfile.maxTokens)
const customAgentSteps = ref(defaultProfile.agentSteps)
const customTemperature = ref(defaultProfile.temperature)
const customMaxTotalSteps = ref(35)
const customMaxTotalTimeout = ref(25)
const moduleDynamicSteps = ref(false)
const moduleDynamicTimeout = ref(false)
const moduleUiux = ref(true)
const moduleWording = ref(true)
const moduleGameDesign = ref(true)
const moduleTestFlows = ref(true)
const recentAnalyses = ref([])
const logContainer = ref(null)
const currentAnalysisId = ref(null)

// Flow preview state
const flowDialogOpen = ref(false)
const previewFlowData = ref(null)
const previewFlowYaml = ref('')
const { copied: flowCopied, copy: copyFlowToClipboard } = useClipboard()

// Agent screenshot preview state
const agentScreenshotOpen = ref(false)
const agentScreenshotStep = ref(null)

// Debug log clipboard
const { copied: logCopied, copy: copyLogToClipboard } = useClipboard()

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
  // Failed step tracking
  failedStep,
  // Persisted agent steps
  persistedAgentSteps,
  loadPersistedSteps,
} = useAnalysis()

// Hint input state
const hintInput = ref('')
const hintSent = ref(false)
let hintSentTimeout = null

async function handleSendHint() {
  if (!hintInput.value.trim()) return
  await sendHint(hintInput.value)
  hintInput.value = ''
  hintSent.value = true
  if (hintSentTimeout) clearTimeout(hintSentTimeout)
  hintSentTimeout = setTimeout(() => { hintSent.value = false }, 2000)
}

// Auto-scroll live timeline
const liveTimelineRef = ref(null)

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
  const dur = stepDuration('flows')
  if (flowList.value.length) {
    return `${flowList.value.length} flow(s) generated${dur ? ` in ${dur}s` : ''}`
  }
  return dur ? `Working... (${dur}s)` : ''
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
    flows_retry: 'Flow Generation',
    flows_done: 'Flow Generation',
    scouting: 'Page Scouting',
    scouted: 'Page Scouting',
    scenarios: 'Scenario Generation',
  }
  return map[failedStep.value] || failedStep.value
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
      screenshotB64: latestScreenshot.value,
      stepNumber: agentStepCurrent.value || '?',
      toolName: 'Live Screenshot',
      result: 'Current game state',
    }
    agentScreenshotOpen.value = true
  }
}

// Ordered step names for granular progress
const STEP_ORDER = ['scouting', 'scouted', 'agent_start', 'agent_step', 'agent_action', 'agent_adaptive', 'agent_timeout_extend', 'agent_done', 'agent_synthesize', 'synthesis_retry', 'analyzing', 'analyzed', 'scenarios', 'scenarios_done', 'flows', 'flows_retry', 'flows_done', 'saving', 'complete']

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
    flows: { start: 'flows', end: 'complete' },
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
  }
  try {
    await start(gameUrl.value, projectId.value, useAgentMode.value, profileParams.value, modules)
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
  useAgentMode.value = false
  selectedProfile.value = 'balanced'
  showCustomFields.value = false
  gameUrl.value = ''
  currentAnalysisId.value = null
  loadRecentAnalyses()
}

function navigateToNewPlan() {
  const flowNames = flowList.value.map((f) => f.name).join(',')
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push({ path: `${basePath}/tests/new`, query: { flows: flowNames, gameUrl: gameUrl.value } })
}

function navigateToFlows() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/flows`)
}

async function runFlowsNow() {
  try {
    await testsApi.run({ gameUrl: gameUrl.value })
    const basePath = projectId.value ? `/projects/${projectId.value}` : ''
    router.push(`${basePath}/tests`)
  } catch (err) {
    console.error('Failed to run flows:', err)
  }
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
  canContinue.value = false
  reset()
  analyzing.value = false
  gameUrl.value = url
  useAgentMode.value = agent
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

onUnmounted(() => {
  if (hintSentTimeout != null) clearTimeout(hintSentTimeout)
})

// Auto-scroll log area when new logs arrive (only if near bottom)
watch(logs, () => {
  nextTick(() => {
    const el = logContainer.value
    if (!el) return
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60
    if (isNearBottom) {
      el.scrollTop = el.scrollHeight
    }
  })
})

// Auto-scroll live agent timeline when new steps arrive (only if near bottom)
watch(liveAgentSteps, () => {
  nextTick(() => {
    const el = liveTimelineRef.value
    if (!el) return
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60
    if (isNearBottom) {
      el.scrollTop = el.scrollHeight
    }
  })
})

onMounted(async () => {
  // Pre-fill game URL from project context
  if (currentProject.value?.gameUrl && !gameUrl.value) {
    gameUrl.value = currentProject.value.gameUrl
  }

  // Try to recover a running or completed analysis from localStorage
  const recovery = await tryRecover()
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
