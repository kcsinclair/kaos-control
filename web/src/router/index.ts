import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/projects' },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/projects',
      name: 'projects',
      component: () => import('@/views/ProjectPickerView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/p/:project',
      component: () => import('@/views/project/WorkspaceView.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: (to) => `/p/${to.params.project}/graph`,
        },
        {
          path: 'artifacts',
          name: 'artifacts',
          component: () => import('@/views/project/ArtifactListView.vue'),
        },
        {
          path: 'artifacts/board',
          name: 'kanban-board',
          component: () => import('@/views/project/KanbanBoardView.vue'),
        },
        {
          path: 'artifacts/:pathMatch(.*)+',
          name: 'artifact-editor',
          component: () => import('@/views/project/ArtifactEditorView.vue'),
        },
        {
          path: 'graph',
          name: 'graph',
          component: () => import('@/views/project/GraphView.vue'),
        },
        {
          path: 'agents',
          name: 'agents',
          component: () => import('@/views/project/AgentsRunsView.vue'),
        },
        {
          path: 'scheduler',
          name: 'scheduler',
          component: () => import('@/views/project/SchedulerListView.vue'),
        },
        {
          path: 'scheduler/:name',
          name: 'scheduler-detail',
          component: () => import('@/views/project/SchedulerDetailView.vue'),
        },
        {
          path: 'feed',
          name: 'feed',
          component: () => import('@/views/project/ProjectFeedView.vue'),
        },
        {
          path: 'parse-errors',
          name: 'parse-errors',
          component: () => import('@/views/project/ParseErrorsView.vue'),
        },
        {
          path: 'config',
          name: 'config',
          component: () => import('@/views/project/ProjectConfigView.vue'),
        },
        {
          path: 'settings/ollama',
          name: 'ollama-settings',
          component: () => import('@/views/project/OllamaSettingsView.vue'),
        },
        {
          path: 'devops',
          name: 'devops',
          component: () => import('@/views/project/DevOpsView.vue'),
          meta: { roles: ['product-owner', 'devops'] },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.initialized) {
    await auth.fetchMe()
  }
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return '/login'
  }
  if (to.path === '/login' && auth.isAuthenticated) {
    return '/projects'
  }
})

export default router
