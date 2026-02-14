// Model pricing table — mirrored from pkg/ai/types.go ModelPricingTable
// Prices are per million tokens (MTok). Updated 2026-02-14.
const MODEL_PRICING = {
  'claude-sonnet-4-5-20250929': { input: 3.0, output: 15.0, cacheRead: 0.30 },
  'claude-haiku-4-5-20251001':  { input: 1.0, output: 5.0, cacheRead: 0.10 },
  'claude-opus-4-6':            { input: 5.0, output: 25.0, cacheRead: 0.50 },
  'gemini-2.0-flash':           { input: 0.10, output: 0.40, cacheRead: 0.025 },
  'gemini-2.5-flash':           { input: 0.30, output: 2.50, cacheRead: 0.03 },
  'gemini-2.5-pro':             { input: 1.25, output: 10.0, cacheRead: 0.125 },
  'gemini-3-flash-preview':     { input: 0.50, output: 3.00, cacheRead: 0.05 },
}

// Per-step token estimates (low / high)
const TOKENS_INPUT_LOW = 3000
const TOKENS_INPUT_HIGH = 8000
const TOKENS_OUTPUT_LOW = 600
const TOKENS_OUTPUT_HIGH = 1500

// Synthesis/flow generation token estimates (2 API calls)
const SYNTHESIS_INPUT_TOKENS = 6000
const SYNTHESIS_OUTPUT_TOKENS = 4000

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
 * @param {string} [opts.synthesisModel] - Secondary model for synthesis/flow gen
 * @param {number} opts.steps - Base agent steps
 * @param {number} opts.maxSteps - Max adaptive steps (0 = not adaptive)
 * @param {number} opts.moduleCount - Number of enabled modules
 * @param {number} opts.jurisdictionCount - Number of GLI jurisdictions selected
 * @param {number} opts.deviceCount - Number of devices
 * @param {boolean} opts.agentMode - Whether agent mode is enabled
 * @returns {{ low: number, high: number }}
 */
export function estimateCredits({ model, synthesisModel, steps, maxSteps, moduleCount, jurisdictionCount = 0, deviceCount, agentMode }) {
  if (!agentMode) {
    return {
      low: NON_AGENT_LOW * deviceCount,
      high: NON_AGENT_HIGH * deviceCount,
    }
  }

  const pricing = MODEL_PRICING[model] || MODEL_PRICING['claude-sonnet-4-5-20250929']
  const synthPricing = synthesisModel ? (MODEL_PRICING[synthesisModel] || pricing) : pricing

  // Module multiplier: each module adds ~4% more tokens
  const moduleMult = 1 + moduleCount * 0.04

  // GLI output multiplier: each jurisdiction beyond the first adds ~8% more output tokens
  // (12 compliance categories evaluated per jurisdiction)
  const gliOutputMult = 1 + Math.max(0, jurisdictionCount - 1) * 0.08

  // Low estimate: base steps, low tokens, cache discount
  const lowSteps = steps
  const lowInputCost = TOKENS_INPUT_LOW * ((1 - CACHE_DISCOUNT) * pricing.input + CACHE_DISCOUNT * (pricing.cacheRead || pricing.input)) / 1_000_000
  const lowOutputCost = TOKENS_OUTPUT_LOW * pricing.output / 1_000_000 * gliOutputMult
  const lowPerStep = (lowInputCost + lowOutputCost) * moduleMult
  let lowUsd = lowSteps * lowPerStep

  // High estimate: max adaptive steps (or base), high tokens, no cache discount
  const highSteps = maxSteps > steps ? maxSteps : steps
  const highInputCost = TOKENS_INPUT_HIGH * pricing.input / 1_000_000
  const highOutputCost = TOKENS_OUTPUT_HIGH * pricing.output / 1_000_000 * gliOutputMult
  const highPerStep = (highInputCost + highOutputCost) * moduleMult
  let highUsd = highSteps * highPerStep

  // Synthesis + flow generation cost (2 API calls via secondary model if configured)
  const synthCost = 2 * (SYNTHESIS_INPUT_TOKENS * synthPricing.input + SYNTHESIS_OUTPUT_TOKENS * synthPricing.output) / 1_000_000
  lowUsd += synthCost
  highUsd += synthCost

  // Convert USD to credits (1 credit = $0.01)
  const low = Math.max(1, Math.round(lowUsd / 0.01 * deviceCount))
  const high = Math.max(1, Math.round(highUsd / 0.01 * deviceCount))

  return { low, high }
}
