<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useThemeStore } from '@/stores/theme'
import { useAgentsStore } from '@/stores/agents'
import { ApiError } from '@/api/client'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const ui = useUiStore()
const theme = useThemeStore()
const agentsStore = useAgentsStore()

const project = computed(() => route.params.project as string | undefined)
const activeCount = computed(() => agentsStore.activeRuns.length)

async function handleLogout() {
  try {
    await auth.logout()
    router.push('/login')
  } catch (err) {
    if (err instanceof ApiError) {
      ui.error(err.message)
    }
  }
}
</script>

<template>
  <header class="app-header">
    <div class="header-brand">
      <RouterLink to="/projects" class="brand-link">kaos-control</RouterLink>
    </div>
    <div class="header-actions">
      <RouterLink
        v-if="project && activeCount > 0"
        :to="`/p/${project}/agents`"
        class="header-run-indicator"
        :aria-label="`${activeCount} running agent${activeCount === 1 ? '' : 's'} — click to view`"
      >
        <span class="run-dot" />
        {{ activeCount }}<span class="run-label"> running agent{{ activeCount === 1 ? '' : 's' }}</span>
      </RouterLink>
      <span v-if="auth.me" class="header-user">{{ auth.me.display_name }}</span>
      <button
        class="btn-icon"
        :title="theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        :aria-label="theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        @click="theme.toggle()"
      >
        <!-- Sun icon (shown in dark mode to switch to light) -->
        <svg v-if="theme.isDark" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="4"/>
          <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
        </svg>
        <!-- Moon icon (shown in light mode to switch to dark) -->
        <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>
        </svg>
      </button>
      <button v-if="auth.isAuthenticated" class="btn-ghost" @click="handleLogout">
        Sign out
      </button>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 52px;
  padding: 0 var(--space-4);
  background: var(--color-sidebar);
  border-bottom: 1px solid var(--color-border-dark);
  flex-shrink: 0;
  z-index: 10;
}
.brand-link {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-sidebar-text);
  text-decoration: none;
  letter-spacing: -0.02em;
}
.brand-link:hover {
  color: #fff;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.header-user {
  font-size: var(--text-sm);
  color: var(--color-sidebar-text-muted);
}
.btn-ghost {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  border: 1px solid var(--color-border-dark);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-sidebar-text-muted);
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.btn-ghost:hover {
  color: #fff;
  border-color: var(--color-sidebar-text);
}
.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: transparent;
  border: 1px solid var(--color-border-dark);
  border-radius: var(--radius-md);
  color: var(--color-sidebar-text-muted);
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s, background 0.15s;
}
.btn-icon:hover {
  color: #fff;
  border-color: var(--color-sidebar-text);
  background: rgba(255,255,255,0.08);
}
.header-run-indicator {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--color-border-dark);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-sidebar-text-muted);
  text-decoration: none;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.header-run-indicator:hover {
  color: #fff;
  border-color: var(--color-sidebar-text);
  background: rgba(255,255,255,0.08);
}
.run-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  flex-shrink: 0;
  animation: pulse 1.5s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
@media (prefers-reduced-motion: reduce) {
  .run-dot {
    animation: none;
  }
}
@media (max-width: 768px) {
  .run-label {
    display: none;
  }
}
</style>
