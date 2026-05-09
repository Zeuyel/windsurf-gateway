<template>
  <div class="page">
    <h1>Token Pool</h1>
    <form class="card" @submit.prevent="create">
      <input v-model="form.name" placeholder="Name" />
      <input v-model="form.token" placeholder="Windsurf token" />
      <input v-model="form.tenant_address" placeholder="https://server.codeium.com" />
      <button>Add Token</button>
    </form>
    <div class="card"><button @click="load">Refresh</button><pre>{{ tokens }}</pre></div>
  </div>
</template>
<script setup>
import { reactive, ref, onMounted } from 'vue'
import client from '../api/client'
const tokens = ref([])
const form = reactive({ name: '', token: '', tenant_address: 'https://server.codeium.com' })
async function load() { const r = await client.get('/tokens'); tokens.value = r.data.data?.list || [] }
async function create() { await client.post('/tokens', form); await load() }
onMounted(load)
</script>
