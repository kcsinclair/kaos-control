<script setup lang="ts">
// Violates vue/no-mutating-props: child components must treat props as
// read-only; direct mutation breaks one-way data flow.
const props = defineProps<{ count: number }>()

function increment() {
  props.count++
}
</script>

<template>
  <button @click="increment">{{ props.count }}</button>
</template>
