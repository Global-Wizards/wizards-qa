// Model pricing table — mirrored from pkg/ai/types.go ModelPricingTable
// Prices are per million tokens (MTok)
const MODEL_PRICING = {
  'claude-sonnet-4-5-20250929': { input: 3.0, output: 15.0, cacheRead: 0.30 },
  'claude-haiku-4-5-20251001':  { input: 0.80, output: 4.0, cacheRead: 0.08 },
  'claude-opus-4-6':            { input: 15.0, output: 75.0, cacheRead: 1.50 },
  'gemini-2.0-flash':               { input: 0.10, output: 0.40, cacheRead: 0 },
  'gemini-2.5-flash-preview-05-20': { input: 0.15, output: 0.60, cacheRead: 0 },
  'gemini-3-flash-preview':          { input: 0.25, output: 1.50, cacheRead: 0.05 },
  'gemini-pro':                      { input: 0.50, output: 1.50, cacheRead: 0 },
}

// Per-step token estimates (low / high)
const TOKENS_INPUT_LOW = 3000
const TOKENS_INPUT_HIGH = 8000
const TOKENS_OUTPUT_LOW = 600
const TOKENS_OUTPUT_HIGH = 1500

// Average cache hit ratio for low estimate
const CACHE_DISCOUNT = 0.40

// Non-agent flat estimate per device
const NON_AGENT_LOW = 1
const NON_AGENT_HIGH = 3

/**
 * Estimate credits for an analysis run.
 * 1 credit = $0.01 USD.
 *
 * @param {Object} opts
 * @param {string} opts.model - Model ID
 * @param {number} opts.steps - Base agent steps
 * @param {number} opts.maxSteps - Max adaptive steps (0 = not adaptive)
 * @param {number} opts.moduleCount - Number of enabled modules
 * @param {number} opts.deviceCount - Number of devices
 * @param {boolean} opts.agentMode - Whether agent mode is enabled
 * @returns {{ low: number, high: number }}
 */
export function estimateCredits({ model, steps, maxSteps, moduleCount, deviceCount, agentMode }) {
  if (!agentMode) {
    return {
      low: NON_AGENT_LOW * deviceCount,
      high: NON_AGENT_HIGH * deviceCount,
    }
  }

  const pricing = MODEL_PRICING[model] || MODEL_PRICING['claude-sonnet-4-5-20250929']

  // Module multiplier: each module adds ~4% more tokens
  const moduleMult = 1 + moduleCount * 0.04

  // Low estimate: base steps, low tokens, cache discount
  const lowSteps = steps
  const lowInputCost = TOKENS_INPUT_LOW * ((1 - CACHE_DISCOUNT) * pricing.input + CACHE_DISCOUNT * (pricing.cacheRead || pricing.input)) / 1_000_000
  const lowOutputCost = TOKENS_OUTPUT_LOW * pricing.output / 1_000_000
  const lowPerStep = (lowInputCost + lowOutputCost) * moduleMult
  const lowUsd = lowSteps * lowPerStep

  // High estimate: max adaptive steps (or base), high tokens, no cache discount
  const highSteps = maxSteps > steps ? maxSteps : steps
  const highInputCost = TOKENS_INPUT_HIGH * pricing.input / 1_000_000
  const highOutputCost = TOKENS_OUTPUT_HIGH * pricing.output / 1_000_000
  const highPerStep = (highInputCost + highOutputCost) * moduleMult
  const highUsd = highSteps * highPerStep

  // Convert USD to credits (1 credit = $0.01)
  const low = Math.max(1, Math.round(lowUsd / 0.01 * deviceCount))
  const high = Math.max(1, Math.round(highUsd / 0.01 * deviceCount))

  return { low, high }
}
