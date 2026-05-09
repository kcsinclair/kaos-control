<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{
  html?: string
  source?: string
  project: string
}>()

const md = new MarkdownIt({ html: false, linkify: true, typographer: true })

// Wiki-link inline rule: [[slug]] or [[slug|display]]
md.core.ruler.push('wiki_links', (state) => {
  for (const token of state.tokens) {
    if (token.type !== 'inline' || !token.children) continue
    const next: typeof token.children = []
    for (const t of token.children) {
      if (t.type !== 'text') { next.push(t); continue }
      const parts = t.content.split(/(\[\[[^\]]+\]\])/)
      if (parts.length === 1) { next.push(t); continue }
      for (const part of parts) {
        const m = part.match(/^\[\[([^\]|]+)(?:\|([^\]]+))?\]\]$/)
        if (m) {
          const slug = m[1].trim()
          const display = m[2]?.trim() ?? slug
          const open = new state.Token('html_inline', '', 0)
          open.content = `<a href="/p/${props.project}/artifacts?lineage=${encodeURIComponent(slug)}" class="wiki-link">${display}</a>`
          next.push(open)
        } else if (part) {
          const txt = new state.Token('text', '', 0)
          txt.content = part
          next.push(txt)
        }
      }
    }
    token.children = next
  }
})

const rendered = computed(() => {
  if (props.html) return props.html
  if (props.source) return md.render(props.source)
  return ''
})
</script>

<template>
  <div class="md-preview" v-html="rendered" />
</template>

<style scoped>
.md-preview {
  line-height: 1.7;
  color: var(--color-text);
  font-size: var(--text-base);
  max-width: 72ch;
}
.md-preview :deep(h1),
.md-preview :deep(h2),
.md-preview :deep(h3),
.md-preview :deep(h4) {
  font-weight: 600;
  margin: 1.5em 0 0.5em;
  line-height: 1.3;
  color: var(--color-text);
}
.md-preview :deep(h1) { font-size: var(--text-2xl); }
.md-preview :deep(h2) { font-size: var(--text-xl); }
.md-preview :deep(h3) { font-size: var(--text-lg); }
.md-preview :deep(p) { margin: 0.75em 0; }
.md-preview :deep(ul),
.md-preview :deep(ol) { padding-left: 1.5em; margin: 0.75em 0; }
.md-preview :deep(li) { margin: 0.25em 0; }
.md-preview :deep(code) {
  font-family: monospace;
  font-size: 0.875em;
  background: var(--color-border);
  padding: 1px 5px;
  border-radius: 3px;
}
.md-preview :deep(pre) {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  overflow-x: auto;
}
.md-preview :deep(pre code) {
  background: none;
  padding: 0;
}
.md-preview :deep(blockquote) {
  border-left: 3px solid var(--color-border);
  margin: 0.75em 0;
  padding-left: var(--space-4);
  color: var(--color-text-muted);
}
.md-preview :deep(a) { color: var(--color-accent); text-decoration: none; }
.md-preview :deep(a:hover) { text-decoration: underline; }
.md-preview :deep(.wiki-link) { color: var(--color-accent); font-style: italic; }
.md-preview :deep(table) { border-collapse: collapse; width: 100%; margin: 1em 0; }
.md-preview :deep(th),
.md-preview :deep(td) {
  border: 1px solid var(--color-border);
  padding: var(--space-2) var(--space-3);
  text-align: left;
}
.md-preview :deep(th) { background: var(--color-surface); font-weight: 600; }
.md-preview :deep(hr) { border: none; border-top: 1px solid var(--color-border); margin: 1.5em 0; }
</style>
