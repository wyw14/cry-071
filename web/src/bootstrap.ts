import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'

export function mountFeedbackWorkspace(target: string): void {
  const application = createApp(App)
  const workspaceState = createPinia()
  application.use(workspaceState)
  application.use(router)
  application.mount(target)
}
