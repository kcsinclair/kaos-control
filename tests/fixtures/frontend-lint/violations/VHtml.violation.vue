<script setup lang="ts">
// Violates vue/no-v-html: raw v-html is an XSS vector unless explicitly
// reviewed and disabled inline for a specific, justified case.
const rawMarkup = '<strong>bold</strong>'
</script>

<template>
  <div v-html="rawMarkup"></div>
</template>
