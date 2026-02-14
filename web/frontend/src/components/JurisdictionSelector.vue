<template>
  <div class="space-y-2">
    <!-- Collapsed summary / toggle -->
    <button
      type="button"
      class="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
      @click="expanded = !expanded"
    >
      <ChevronRight class="h-3.5 w-3.5 transition-transform" :class="expanded && 'rotate-90'" />
      <span v-if="modelValue.length === 0">No jurisdictions selected</span>
      <span v-else>{{ modelValue.length }} jurisdiction{{ modelValue.length !== 1 ? 's' : '' }} selected</span>
    </button>

    <!-- Expanded panel -->
    <div v-if="expanded" class="rounded-md border p-3 space-y-3">
      <!-- Search -->
      <Input
        v-model="search"
        placeholder="Search jurisdictions..."
        class="h-8 text-xs"
      />

      <!-- Region quick-select buttons -->
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="region in regions"
          :key="region"
          type="button"
          class="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-medium transition-colors"
          :class="isRegionFullySelected(region)
            ? 'bg-primary text-primary-foreground'
            : isRegionPartiallySelected(region)
              ? 'bg-primary/30 text-primary-foreground'
              : 'bg-muted text-muted-foreground hover:bg-muted/80'"
          @click="toggleRegion(region)"
        >
          {{ region }}
        </button>
      </div>

      <!-- Checkbox tree -->
      <div class="max-h-64 overflow-y-auto space-y-2">
        <div v-for="node in filteredTree" :key="node.region">
          <!-- Region header -->
          <div class="flex items-center gap-2 py-1">
            <input
              type="checkbox"
              :checked="isRegionFullySelected(node.region)"
              :indeterminate.prop="isRegionPartiallySelected(node.region)"
              class="rounded border-gray-300"
              @change="toggleRegion(node.region)"
            />
            <span class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ node.region }}</span>
          </div>

          <!-- Country groups -->
          <div v-for="c in node.countries" :key="c.country" class="ml-4">
            <!-- Country header (only if country has multiple jurisdictions) -->
            <div v-if="c.jurisdictions.length > 1" class="flex items-center gap-2 py-0.5">
              <input
                type="checkbox"
                :checked="isCountryFullySelected(c.country)"
                :indeterminate.prop="isCountryPartiallySelected(c.country)"
                class="rounded border-gray-300"
                @change="toggleCountry(c.country)"
              />
              <span class="text-xs font-medium">{{ c.country }}</span>
            </div>

            <!-- Individual jurisdictions -->
            <div :class="c.jurisdictions.length > 1 ? 'ml-4' : ''" class="space-y-0.5">
              <label
                v-for="j in c.jurisdictions"
                :key="j.id"
                class="flex items-center gap-2 py-0.5 text-xs cursor-pointer select-none"
              >
                <input
                  type="checkbox"
                  :checked="modelValue.includes(j.id)"
                  class="rounded border-gray-300"
                  @change="toggleJurisdiction(j.id)"
                />
                <span>{{ c.jurisdictions.length > 1 ? j.name : c.country }}</span>
                <span class="text-muted-foreground">{{ j.regulator }}</span>
              </label>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ChevronRight } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import {
  getJurisdictionTree,
  getJurisdictionIdsByRegion,
  getJurisdictionIdsByCountry,
} from '@/lib/jurisdictions'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue'])

const expanded = ref(false)
const search = ref('')

const tree = getJurisdictionTree()
const regions = tree.map(n => n.region)

const filteredTree = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return tree
  return tree
    .map(node => ({
      ...node,
      countries: node.countries
        .map(c => ({
          ...c,
          jurisdictions: c.jurisdictions.filter(j =>
            j.name.toLowerCase().includes(q) ||
            j.country.toLowerCase().includes(q) ||
            j.regulator.toLowerCase().includes(q) ||
            j.id.toLowerCase().includes(q)
          ),
        }))
        .filter(c => c.jurisdictions.length > 0),
    }))
    .filter(node => node.countries.length > 0)
})

function isRegionFullySelected(region) {
  const ids = getJurisdictionIdsByRegion(region)
  return ids.length > 0 && ids.every(id => props.modelValue.includes(id))
}

function isRegionPartiallySelected(region) {
  const ids = getJurisdictionIdsByRegion(region)
  const selected = ids.filter(id => props.modelValue.includes(id))
  return selected.length > 0 && selected.length < ids.length
}

function isCountryFullySelected(country) {
  const ids = getJurisdictionIdsByCountry(country)
  return ids.length > 0 && ids.every(id => props.modelValue.includes(id))
}

function isCountryPartiallySelected(country) {
  const ids = getJurisdictionIdsByCountry(country)
  const selected = ids.filter(id => props.modelValue.includes(id))
  return selected.length > 0 && selected.length < ids.length
}

function toggleRegion(region) {
  const ids = getJurisdictionIdsByRegion(region)
  if (isRegionFullySelected(region)) {
    emit('update:modelValue', props.modelValue.filter(id => !ids.includes(id)))
  } else {
    const set = new Set([...props.modelValue, ...ids])
    emit('update:modelValue', [...set])
  }
}

function toggleCountry(country) {
  const ids = getJurisdictionIdsByCountry(country)
  if (isCountryFullySelected(country)) {
    emit('update:modelValue', props.modelValue.filter(id => !ids.includes(id)))
  } else {
    const set = new Set([...props.modelValue, ...ids])
    emit('update:modelValue', [...set])
  }
}

function toggleJurisdiction(id) {
  if (props.modelValue.includes(id)) {
    emit('update:modelValue', props.modelValue.filter(x => x !== id))
  } else {
    emit('update:modelValue', [...props.modelValue, id])
  }
}
</script>
