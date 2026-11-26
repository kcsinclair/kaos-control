<script lang="ts">
import { defineComponent } from 'vue'
import HelperWidget from './HelperWidget.support.vue'

// Violates vue/no-unused-components: HelperWidget is registered via the
// Options API `components` map but never referenced in the template.
export default defineComponent({
  components: { HelperWidget },
})
</script>

<template>
  <div>static content, no HelperWidget usage</div>
</template>
