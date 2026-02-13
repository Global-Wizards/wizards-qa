<script setup>
import { provide } from 'vue'
import { useField } from 'vee-validate'

const props = defineProps({
  name: { type: String, required: true },
})

const { value, errorMessage, handleBlur, handleChange, meta } = useField(() => props.name)

provide('form-field', {
  name: props.name,
  value,
  errorMessage,
  handleBlur,
  handleChange,
  meta,
})
</script>

<template>
  <slot
    :componentField="{ modelValue: value, 'onUpdate:modelValue': handleChange, onBlur: handleBlur }"
    :value="value"
    :errorMessage="errorMessage"
    :meta="meta"
  />
</template>
