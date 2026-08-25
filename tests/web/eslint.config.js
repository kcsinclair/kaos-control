import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

export default defineConfigWithVueTs(
  {
    name: 'tests/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },
  {
    name: 'tests/files-to-ignore',
    ignores: ['**/test-results/**'],
  },
  pluginVue.configs['flat/base'],
  vueTsConfigs.base,
  {
    name: 'tests/correctness-rules',
    rules: {
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      'vue/no-unused-components': 'error',
      'vue/no-mutating-props': 'error',
      'vue/no-v-html': 'error',
      eqeqeq: ['error', 'always'],
      'prefer-const': 'error',
    },
  },
  {
    // FR-4: testing ergonomics — mocks/stubs/fixtures routinely use `any`
    // and intentionally-unused placeholders, so unused-vars/args and
    // explicit-any are relaxed here relative to the production rule set.
    name: 'tests/ergonomics-overrides',
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
    },
  },
)
