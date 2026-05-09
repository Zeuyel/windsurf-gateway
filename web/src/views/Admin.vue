<template>
  <div class="page"><h1>Admin</h1><div class="card"><button @click="load">Load Stats</button><pre>{{ stats }}</pre></div></div>
</template>
<script setup>
import { ref } from 'vue'
import client from '../api/client'
const stats = ref({})
async function load() { const r = await client.get('/stats/overview'); stats.value = r.data.data }
</script>
