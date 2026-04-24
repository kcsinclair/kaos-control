<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'

const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()

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
      <span v-if="auth.me" class="header-user">{{ auth.me.display_name }}</span>
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
</style>
