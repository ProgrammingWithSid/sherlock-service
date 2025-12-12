import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Initialize auth on app start
import { useAuthStore } from './stores/auth'
const authStore = useAuthStore()
authStore.init()

app.mount('#app')
