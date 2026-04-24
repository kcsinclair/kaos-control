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
          component: () => import('@/views/project/PlaceholderView.vue'),
        },
        {
          path: 'parse-errors',
          name: 'parse-errors',
          component: () => import('@/views/project/PlaceholderView.vue'),
        },
        {
          path: 'config',
          name: 'config',
          component: () => import('@/views/project/PlaceholderView.vue'),
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
