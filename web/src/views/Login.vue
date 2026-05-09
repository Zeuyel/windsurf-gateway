<template>
  <div class="login-page">
    <form class="card" @submit.prevent="submit">
      <h1>Windsurf Gateway</h1>
      <input v-model="username" placeholder="Username" />
      <input v-model="password" type="password" placeholder="Password" />
      <label><input v-model="admin" type="checkbox" /> Admin login</label>
      <button type="submit">Login</button>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </div>
</template>
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../store/auth'
const router = useRouter(); const auth = useAuthStore()
const username = ref('admin'); const password = ref('admin123'); const admin = ref(true); const error = ref('')
async function submit() { try { await auth.login(username.value, password.value, admin.value); router.push('/dashboard') } catch(e) { error.value = e.message } }
</script>
