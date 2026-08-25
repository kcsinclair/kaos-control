<script setup lang="ts">
// Positive case: HelperWidget is imported AND used in the template, props
// are read-only, and markup is static (no v-html).
import HelperWidget from './HelperWidget.support.vue'

const props = defineProps<{ count: number }>()
</script>

<template>
  <div>
    <HelperWidget />
    <span>{{ props.count }}</span>
  </div>
</template>
