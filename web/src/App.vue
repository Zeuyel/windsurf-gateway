<template>
  <div id="app-container">
    <nav class="navbar" v-if="auth.isLoggedIn">
      <div class="logo">Windsurf Gateway</div>
      <div class="nav-links">
        <router-link to="/dashboard">Dashboard</router-link>
        <router-link v-if="auth.isAdmin" to="/tokens">Tokens</router-link>
        <router-link v-if="auth.isAdmin" to="/admin">Admin</router-link>
        <span>{{ auth.user?.username }}</span>
        <button @click="logout">Logout</button>
      </div>
    </nav>
    <router-view />
  </div>
</template>
<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from './store/auth'
const router = useRouter()
const auth = useAuthStore()
function logout() { auth.logout(); router.push('/login') }
</script>
