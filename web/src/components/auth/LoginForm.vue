<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'

const router = useRouter()
const auth = useAuthStore()

const email = ref('')
const password = ref('')
const errorMsg = ref('')
const submitting = ref(false)

async function submit() {
  errorMsg.value = ''
  submitting.value = true
  try {
    await auth.login(email.value, password.value)
    router.push('/projects')
  } catch (err) {
    if (err instanceof ApiError) {
      errorMsg.value = err.message
    } else {
      errorMsg.value = 'An unexpected error occurred'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="login-form" @submit.prevent="submit">
    <div class="field">
      <label class="label" for="email">Email</label>
      <input
        id="email"
        v-model="email"
        type="email"
        class="input"
        autocomplete="email"
        required
        :disabled="submitting"
      />
    </div>
    <div class="field">
      <label class="label" for="password">Password</label>
      <input
        id="password"
        v-model="password"
        type="password"
        class="input"
        autocomplete="current-password"
        required
        :disabled="submitting"
      />
    </div>
    <p v-if="errorMsg" class="error-msg" role="alert">{{ errorMsg }}</p>
    <button type="submit" class="btn-primary" :disabled="submitting">
      {{ submitting ? 'Signing in…' : 'Sign in' }}
    </button>
  </form>
</template>

<style scoped>
.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-base);
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.input:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-subtle);
}
.input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.error-msg {
  font-size: var(--text-sm);
  color: var(--color-error);
  margin: 0;
}
.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-base);
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn-primary:hover:not(:disabled) {
  opacity: 0.88;
}
.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
