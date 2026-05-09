<template>
  <div class="dashboard-page">
    <el-row :gutter="16" class="summary-grid">
      <el-col :xs="24" :sm="12" :lg="6" v-for="card in summaryCards" :key="card.label">
        <el-card class="summary-card" shadow="hover">
          <div class="summary-label">{{ card.label }}</div>
          <div class="summary-value">{{ card.value }}</div>
          <div class="summary-meta">{{ card.meta }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-grid">
      <el-col :xs="24" :lg="16">
        <el-card shadow="never">
          <template #header>
            <div class="panel-header">
              <span>请求趋势</span>
              <el-radio-group v-model="timeRange" size="small" @change="loadTrend">
                <el-radio-button label="24h">24h</el-radio-button>
                <el-radio-button label="7d">7d</el-radio-button>
                <el-radio-button label="30d">30d</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div ref="trendChartRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="8">
        <el-card shadow="never" class="gateway-card">
          <template #header>
            <span>网关状态</span>
          </template>
          <div class="gateway-stat">
            <span>负载策略</span>
            <strong>{{ stats.load_balancer_strategy || 'round_robin' }}</strong>
          </div>
          <div class="gateway-stat">
            <span>可用 Token</span>
            <strong>{{ stats.available_tokens || 0 }}</strong>
          </div>
          <div class="gateway-stat">
            <span>冷却 Token</span>
            <strong>{{ stats.cooldown_tokens || 0 }}</strong>
          </div>
          <div class="gateway-stat">
            <span>摘除 Token</span>
            <strong>{{ stats.disabled_tokens || 0 }}</strong>
          </div>
          <div class="gateway-stat">
            <span>活跃并发</span>
            <strong>{{ stats.total_active_requests || 0 }}</strong>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-grid">
      <el-col :xs="24" :lg="24">
        <el-card shadow="never">
          <template #header>
            <div class="panel-header">
              <span>最近请求</span>
              <el-button size="small" @click="loadRecentRequests">刷新</el-button>
            </div>
          </template>
          <el-table :data="recentRequests" stripe>
            <el-table-column prop="created_at" label="时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column prop="request_id" label="Request ID" min-width="180" show-overflow-tooltip />
            <el-table-column prop="username" label="用户" min-width="120" />
            <el-table-column prop="token_name" label="Backend Token" min-width="140" show-overflow-tooltip />
            <el-table-column prop="method" label="方法" width="90">
              <template #default="{ row }">
                <el-tag :type="methodTag(row.method)" size="small">{{ row.method }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="path" label="路径" min-width="220" show-overflow-tooltip />
            <el-table-column prop="status_code" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusTag(row.status_code)" size="small">{{ row.status_code }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="failure_category" label="错误分类" min-width="140" />
            <el-table-column prop="latency" label="延迟" width="110">
              <template #default="{ row }">{{ formatLatency(row.latency) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import client from '../api/client'

const stats = ref({})
const recentRequests = ref([])
const timeRange = ref('7d')
const trendChartRef = ref(null)
let trendChart = null

const summaryCards = computed(() => [
  {
    label: '总请求',
    value: stats.value.total_requests || 0,
    meta: `成功 ${stats.value.success_requests || 0}`,
  },
  {
    label: '失败请求',
    value: stats.value.failed_requests || 0,
    meta: `错误分类 ${stats.value.error_categories?.length || 0}`,
  },
  {
    label: '活跃用户',
    value: stats.value.active_users || 0,
    meta: `总用户 ${stats.value.total_users || 0}`,
  },
  {
    label: '平均延迟',
    value: `${Math.round(stats.value.avg_latency_ms || 0)} ms`,
    meta: `总并发 ${stats.value.total_active_requests || 0}`,
  },
])

const loadOverview = async () => {
  const res = await client.get('/stats/overview')
  if (res.data.code === 200) {
    stats.value = res.data.data || {}
  }
}

const loadRecentRequests = async () => {
  const res = await client.get('/request-records', {
    params: { page: 1, page_size: 10 },
  })
  if (res.data.code === 200) {
    recentRequests.value = res.data.data?.list || []
  }
}

const initTrendChart = () => {
  if (!trendChartRef.value) return
  trendChart = echarts.init(trendChartRef.value)
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['总请求', '成功', '失败'] },
    grid: { left: 24, right: 24, top: 48, bottom: 24, containLabel: true },
    xAxis: { type: 'category', data: [] },
    yAxis: { type: 'value' },
    series: [
      { name: '总请求', type: 'line', smooth: true, data: [], lineStyle: { color: '#2563eb' } },
      { name: '成功', type: 'line', smooth: true, data: [], lineStyle: { color: '#16a34a' } },
      { name: '失败', type: 'line', smooth: true, data: [], lineStyle: { color: '#dc2626' } },
    ],
  })
}

const loadTrend = async () => {
  const res = await client.get('/stats/trend', { params: { range: timeRange.value } })
  if (res.data.code !== 200 || !trendChart) return

  const data = res.data.data || []
  trendChart.setOption({
    xAxis: {
      data: data.map((item) =>
        timeRange.value === '24h'
          ? dayjs(item.time).format('HH:mm')
          : dayjs(item.time).format('MM-DD')
      ),
    },
    series: [
      { data: data.map((item) => item.total_count || 0) },
      { data: data.map((item) => item.success_count || 0) },
      { data: data.map((item) => item.failed_count || 0) },
    ],
  })
}

const formatTime = (value) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-')
const formatLatency = (microseconds) => `${((microseconds || 0) / 1000).toFixed(1)} ms`
const statusTag = (statusCode) => {
  if (statusCode >= 500) return 'danger'
  if (statusCode >= 400) return 'warning'
  return 'success'
}
const methodTag = (method) => ({ GET: 'success', POST: 'primary', PUT: 'warning', DELETE: 'danger' }[method] || 'info')
const handleResize = () => trendChart?.resize()

onMounted(async () => {
  await Promise.all([loadOverview(), loadRecentRequests()])
  initTrendChart()
  await loadTrend()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  trendChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.summary-grid,
.panel-grid {
  margin: 0;
}

.summary-card {
  border: 1px solid #dbe4f0;
  background: linear-gradient(180deg, #f8fbff 0%, #ffffff 100%);
}

.summary-label {
  color: #5b7087;
  font-size: 13px;
}

.summary-value {
  margin-top: 10px;
  font-size: 28px;
  font-weight: 700;
  color: #132238;
}

.summary-meta {
  margin-top: 8px;
  color: #789;
  font-size: 12px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chart-box {
  height: 320px;
}

.gateway-card {
  height: 100%;
}

.gateway-stat {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #eef3f8;
  color: #425466;
}

.gateway-stat:last-child {
  border-bottom: none;
}

.gateway-stat strong {
  color: #0f172a;
}
</style>
