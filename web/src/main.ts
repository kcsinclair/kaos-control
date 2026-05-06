import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from '@/router'
import App from '@/App.vue'
import '@/styles/tokens.css'
import '@/styles/main.css'
import '@/components/dashboard/registerWidgets'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
