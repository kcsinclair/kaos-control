import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**'],
  },
  pluginVue.configs['flat/base'],
  vueTsConfigs.base,
  {
    name: 'app/correctness-rules',
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      'vue/no-unused-components': 'error',
      'vue/no-mutating-props': 'error',
      'vue/no-v-html': 'error',
      eqeqeq: ['error', 'always'],
      'prefer-const': 'error',
    },
  },
)
