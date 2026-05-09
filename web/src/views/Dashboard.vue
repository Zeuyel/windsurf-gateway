<template>
  <div class="page">
    <h1>Dashboard</h1>
    <div class="grid">
      <div class="card"><h3>API Token</h3><code>{{ auth.user?.api_token || 'N/A' }}</code></div>
      <div class="card"><h3>Quota</h3><p>{{ auth.user?.used_requests || 0 }} / {{ auth.user?.max_requests || 0 }}</p></div>
      <div class="card"><h3>Gateway URL</h3><code>/proxy/*</code></div>
    </div>
  </div>
</template>
<script setup>
import { useAuthStore } from '../store/auth'
const auth = useAuthStore()
</script>
