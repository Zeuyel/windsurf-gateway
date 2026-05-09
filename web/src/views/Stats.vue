<template>
  <div class="stats-page">
    <el-row :gutter="16" class="summary-grid">
      <el-col :xs="24" :sm="12" :lg="4" v-for="card in summaryCards" :key="card.label">
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
              <span>成功 / 失败趋势</span>
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
        <el-card shadow="never">
          <template #header>
            <span>请求方法分布</span>
          </template>
          <div ref="methodChartRef" class="chart-box small"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-grid">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>Token 使用次数</span>
          </template>
          <div ref="usageChartRef" class="chart-box small"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>Token 失败次数</span>
          </template>
          <div ref="failureChartRef" class="chart-box small"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-grid">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>最近错误</span>
          </template>
          <el-table :data="stats.recent_errors || []" stripe>
            <el-table-column prop="created_at" label="时间" min-width="160" />
            <el-table-column prop="token_name" label="Token" min-width="120" show-overflow-tooltip />
            <el-table-column prop="status_code" label="状态" width="90" />
            <el-table-column prop="failure_category" label="分类" min-width="140" />
            <el-table-column prop="error_message" label="原因" min-width="220" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>高频用户</span>
          </template>
          <el-table :data="stats.top_users || []" stripe>
            <el-table-column prop="username" label="用户" min-width="160" />
            <el-table-column prop="requests" label="请求数" width="120" />
          </el-table>
          <div class="category-list">
            <div class="category-title">最近错误分类</div>
            <div v-for="item in stats.error_categories || []" :key="item.failure_category" class="category-item">
              <span>{{ item.failure_category }}</span>
              <strong>{{ item.count }}</strong>
            </div>
          </div>
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
const timeRange = ref('7d')
const trendChartRef = ref(null)
const methodChartRef = ref(null)
const usageChartRef = ref(null)
const failureChartRef = ref(null)

let trendChart = null
let methodChart = null
let usageChart = null
let failureChart = null

const summaryCards = computed(() => [
  {
    label: '总请求',
    value: stats.value.total_requests || 0,
    meta: `Token ${stats.value.total_tokens || 0}`,
  },
  {
    label: '成功请求',
    value: stats.value.success_requests || 0,
    meta: `可用 ${stats.value.available_tokens || 0}`,
  },
  {
    label: '失败请求',
    value: stats.value.failed_requests || 0,
    meta: `冷却 ${stats.value.cooldown_tokens || 0}`,
  },
  {
    label: '平均延迟',
    value: `${Math.round(stats.value.avg_latency_ms || 0)} ms`,
    meta: `并发 ${stats.value.total_active_requests || 0}`,
  },
  {
    label: '负载策略',
    value: stats.value.load_balancer_strategy || 'round_robin',
    meta: `耗尽 ${stats.value.exhausted_tokens || 0}`,
  },
  {
    label: '活跃用户',
    value: stats.value.active_users || 0,
    meta: `总用户 ${stats.value.total_users || 0}`,
  },
])

const loadOverview = async () => {
  const res = await client.get('/stats/overview')
  if (res.data.code === 200) {
    stats.value = res.data.data || {}
    updateStaticCharts()
  }
}

const initCharts = () => {
  if (trendChartRef.value) {
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

  if (methodChartRef.value) {
    methodChart = echarts.init(methodChartRef.value)
  }

  if (usageChartRef.value) {
    usageChart = echarts.init(usageChartRef.value)
  }

  if (failureChartRef.value) {
    failureChart = echarts.init(failureChartRef.value)
  }
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

const updateStaticCharts = () => {
  if (methodChart) {
    methodChart.setOption({
      tooltip: { trigger: 'item' },
      series: [
        {
          type: 'pie',
          radius: ['35%', '65%'],
          data: (stats.value.method_distribution || []).map((item) => ({
            name: item.method,
            value: item.count,
          })),
        },
      ],
    })
  }

  if (usageChart) {
    const usage = stats.value.token_usage || []
    usageChart.setOption({
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 24, right: 24, top: 16, bottom: 48, containLabel: true },
      xAxis: {
        type: 'category',
        data: usage.map((item) => item.token_name),
        axisLabel: { rotate: 20 },
      },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: usage.map((item) => item.requests), itemStyle: { color: '#2563eb' } }],
    })
  }

  if (failureChart) {
    const failures = stats.value.token_failures || []
    failureChart.setOption({
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 24, right: 24, top: 16, bottom: 48, containLabel: true },
      xAxis: {
        type: 'category',
        data: failures.map((item) => item.token_name),
        axisLabel: { rotate: 20 },
      },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: failures.map((item) => item.failures), itemStyle: { color: '#dc2626' } }],
    })
  }
}

const handleResize = () => {
  trendChart?.resize()
  methodChart?.resize()
  usageChart?.resize()
  failureChart?.resize()
}

onMounted(async () => {
  initCharts()
  await loadOverview()
  await loadTrend()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  trendChart?.dispose()
  methodChart?.dispose()
  usageChart?.dispose()
  failureChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.stats-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.summary-card {
  border: 1px solid #dbe4f0;
  background: linear-gradient(180deg, #f7fbff 0%, #ffffff 100%);
}

.summary-label {
  color: #5b7087;
  font-size: 13px;
}

.summary-value {
  margin-top: 10px;
  color: #132238;
  font-size: 26px;
  font-weight: 700;
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
  height: 340px;
}

.chart-box.small {
  height: 300px;
}

.category-list {
  margin-top: 16px;
  border-top: 1px solid #eef3f8;
  padding-top: 16px;
}

.category-title {
  font-size: 13px;
  color: #5b7087;
  margin-bottom: 10px;
}

.category-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f2f5f9;
}

.category-item:last-child {
  border-bottom: none;
}
</style>
