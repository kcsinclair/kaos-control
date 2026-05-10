<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { useRoute } from 'vue-router'
import LoginForm from '@/components/auth/LoginForm.vue'

const route = useRoute()
const sessionExpired = route.query.expired === '1'
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">kaos-control</h1>
      <p class="login-subtitle">Sign in to continue</p>
      <p v-if="sessionExpired" class="expiry-notice" role="alert">
        Your session has expired. Please log in again.
      </p>
      <LoginForm />
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
}
.login-card {
  width: 100%;
  max-width: 400px;
  padding: var(--space-8);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
}
.login-title {
  font-size: var(--text-2xl);
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 var(--space-1);
}
.login-subtitle {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0 0 var(--space-6);
}
.expiry-notice {
  font-size: var(--text-sm);
  color: var(--color-warning, #b45309);
  background: var(--color-warning-subtle, #fef3c7);
  border: 1px solid var(--color-warning-border, #fcd34d);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  margin: 0 0 var(--space-4);
}
</style>
