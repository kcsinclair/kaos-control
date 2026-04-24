<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import AppHeader from '@/components/layout/AppHeader.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'

const route = useRoute()
const projectStore = useProjectStore()

async function syncProject() {
  const name = route.params.project as string
  if (!projectStore.projects.length) {
    await projectStore.fetchProjects()
  }
  projectStore.setCurrent(name)
}

onMounted(syncProject)
watch(() => route.params.project, syncProject)
</script>

<template>
  <div class="workspace">
    <AppHeader />
    <div class="workspace-body">
      <AppSidebar />
      <main class="workspace-main">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.workspace {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: var(--color-bg);
}
.workspace-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.workspace-main {
  flex: 1;
  overflow: auto;
}
</style>
