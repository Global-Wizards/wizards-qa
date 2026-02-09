export const ANALYSIS_PROFILES = [
  {
    name: 'quick',
    label: 'Quick Scan',
    description: 'Fast & cheap. Good for checking if a game loads.',
    model: 'claude-haiku-4-5-20251001',
    maxTokens: 2048,
    agentSteps: 8,
    temperature: 0.3,
  },
  {
    name: 'balanced',
    label: 'Balanced',
    description: 'Default. Good balance of speed, cost, and quality.',
    model: 'claude-sonnet-4-5-20250929',
    maxTokens: 4096,
    agentSteps: 15,
    temperature: 0.5,
  },
  {
    name: 'thorough',
    label: 'Thorough',
    description: 'More exploration steps and larger output.',
    model: 'claude-sonnet-4-5-20250929',
    maxTokens: 8192,
    agentSteps: 20,
    temperature: 0.7,
  },
  {
    name: 'maximum',
    label: 'Maximum',
    description: 'Best quality. Uses most capable model.',
    model: 'claude-opus-4-6',
    maxTokens: 8192,
    agentSteps: 25,
    temperature: 0.7,
  },
  {
    name: 'debug',
    label: 'Debug',
    description: 'Minimal. For debugging the pipeline quickly.',
    model: 'claude-haiku-4-5-20251001',
    maxTokens: 1024,
    agentSteps: 5,
    temperature: 0.1,
  },
]

export function getProfileByName(name) {
  return ANALYSIS_PROFILES.find((p) => p.name === name) || null
}
