<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted } from 'vue'
import Toast from '@/components/common/Toast.vue'
import { useThemeStore } from '@/stores/theme'

const theme = useThemeStore()
onMounted(() => theme.init())
</script>

<template>
  <RouterView />
  <Toast />
</template>
