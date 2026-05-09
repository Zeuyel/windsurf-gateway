import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../store/auth'
import Layout from '../components/Layout.vue'
import Login from '../views/Login.vue'
import Dashboard from '../views/Dashboard.vue'
import Tokens from '../views/Tokens.vue'
import Admin from '../views/Admin.vue'

const routes = [
  { 
    path: '/', 
    redirect: '/dashboard' 
  },
  { 
    path: '/login', 
    component: Login,
    meta: { standalone: true }
  },
  {
    path: '/dashboard',
    component: Layout,
    children: [
      { path: '', component: Dashboard, meta: { auth: true } }
    ]
  },
  {
    path: '/tokens',
    component: Layout,
    children: [
      { path: '', component: Tokens, meta: { auth: true, admin: true } }
    ]
  },
  {
    path: '/admin',
    component: Layout,
    children: [
      { path: '', component: Admin, meta: { auth: true, admin: true } }
    ]
  }
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  const auth = useAuthStore()
  auth.init()
  if (to.meta.auth && !auth.isLoggedIn) return '/login'
  if (to.meta.admin && !auth.isAdmin) return '/dashboard'
})

export default router
