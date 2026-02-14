// Shared constants used across composables
export const MAX_LOGS = 500

// Centralized localStorage key strings to avoid magic strings scattered across files
export const STORAGE_KEYS = {
  accessToken: 'accessToken',
  refreshToken: 'refreshToken',
  theme: 'theme',
  lastProject: 'wizards-qa-last-project',
}

// Analysis status constants — avoids magic strings in templates and composables
export const STATUS = {
  IDLE: 'idle',
  QUEUED: 'queued',
  SCOUTING: 'scouting',
  ANALYZING: 'analyzing',
  GENERATING: 'generating',
  CREATING_TEST_PLAN: 'creating_test_plan',
  TESTING: 'testing',
  COMPLETE: 'complete',
  ERROR: 'error',
}
