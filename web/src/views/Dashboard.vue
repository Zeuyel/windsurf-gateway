<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :md="8" :lg="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #409eff;">
              <el-icon><DataLine /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">总请求数</div>
              <div class="stat-value">{{ stats.totalRequests || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8" :lg="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #67c23a;">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">可用 Token</div>
              <div class="stat-value">{{ stats.availableTokens || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8" :lg="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #e6a23c;">
              <el-icon><User /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">活跃用户</div>
              <div class="stat-value">{{ stats.activeUsers || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8" :lg="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #f56c6c;">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">失败请求</div>
              <div class="stat-value">{{ stats.failedRequests || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :lg="16">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span>请求趋势</span>
              <el-radio-group v-model="timeRange" size="small">
                <el-radio-button label="24h">24小时</el-radio-button>
                <el-radio-button label="7d">7天</el-radio-button>
                <el-radio-button label="30d">30天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div ref="chartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="8">
        <el-card class="user-info-card">
          <template #header>
            <span>用户信息</span>
          </template>
          <div class="user-info">
            <el-descriptions :column="1" border>
              <el-descriptions-item label="用户名">
                {{ authStore.user?.username }}
              </el-descriptions-item>
              <el-descriptions-item label="API Token">
                <code class="token-code">{{ authStore.user?.api_token || 'N/A' }}</code>
              </el-descriptions-item>
              <el-descriptions-item label="已用请求">
                {{ authStore.user?.used_requests || 0 }}
              </el-descriptions-item>
              <el-descriptions-item label="最大请求">
                {{ authStore.user?.max_requests || 0 }}
              </el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="authStore.user?.status === 'active' ? 'success' : 'danger'">
                  {{ authStore.user?.status || 'unknown' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
            <div class="usage-progress">
              <div class="progress-label">使用进度</div>
              <el-progress 
                :percentage="usagePercentage" 
                :color="progressColor"
              />
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="content-row">
      <el-col :span="24">
        <el-card class="recent-requests-card">
          <template #header>
            <div class="card-header">
              <span>最近请求</span>
              <el-button size="small" @click="loadRecentRequests">刷新</el-button>
            </div>
          </template>
          <el-table :data="recentRequests" stripe style="width: 100%">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="method" label="方法" width="80">
              <template #default="{ row }">
                <el-tag :type="getMethodType(row.method)" size="small">
                  {{ row.method }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="path" label="路径" min-width="200" show-overflow-tooltip />
            <el-table-column prop="status_code" label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status_code === 200 ? 'success' : 'danger'" size="small">
                  {{ row.status_code }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="latency" label="延迟" width="100">
              <template #default="{ row }">
                {{ (row.latency / 1000).toFixed(2) }}ms
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '../store/auth'
import client from '../api/client'
import * as echarts from 'echarts'
import dayjs from 'dayjs'

const authStore = useAuthStore()
const chartRef = ref(null)
const timeRange = ref('24h')
const stats = ref({
  totalRequests: 0,
  availableTokens: 0,
  activeUsers: 0,
  failedRequests: 0
})
const recentRequests = ref([])
let chart = null

const usagePercentage = computed(() => {
  if (!authStore.user?.max_requests) return 0
  return Math.round((authStore.user.used_requests / authStore.user.max_requests) * 100)
})

const progressColor = computed(() => {
  const percentage = usagePercentage.value
  if (percentage < 50) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
})

const getMethodType = (method) => {
  const types = {
    GET: 'success',
    POST: 'warning',
    PUT: 'info',
    DELETE: 'danger'
  }
  return types[method] || 'info'
}

const formatTime = (time) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const loadStats = async () => {
  try {
    const res = await client.get('/stats/overview')
    if (res.data.code === 200) {
      stats.value = res.data.data
    }
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const loadRecentRequests = async () => {
  try {
    const res = await client.get('/request-records', {
      params: { limit: 10 }
    })
    if (res.data.code === 200) {
      recentRequests.value = res.data.data?.list || []
    }
  } catch (error) {
    console.error('Failed to load recent requests:', error)
  }
}

const initChart = () => {
  if (!chartRef.value) return
  
  chart = echarts.init(chartRef.value)
  const option = {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: []
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '请求数',
        type: 'line',
        smooth: true,
        data: [],
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64, 158, 255, 0.3)' },
            { offset: 1, color: 'rgba(64, 158, 255, 0.05)' }
          ])
        },
        lineStyle: {
          color: '#409eff'
        },
        itemStyle: {
          color: '#409eff'
        }
      }
    ]
  }
  chart.setOption(option)
}

const updateChart = async () => {
  try {
    const res = await client.get('/stats/trend', {
      params: { range: timeRange.value }
    })
    if (res.data.code === 200) {
      const data = res.data.data || []
      const xData = data.map(item => dayjs(item.time).format('HH:mm'))
      const yData = data.map(item => item.count)
      
      chart.setOption({
        xAxis: { data: xData },
        series: [{ data: yData }]
      })
    }
  } catch (error) {
    console.error('Failed to load chart data:', error)
  }
}

const handleResize = () => {
  chart?.resize()
}

onMounted(async () => {
  await Promise.all([loadStats(), loadRecentRequests()])
  initChart()
  await updateChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  chart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.dashboard {
  padding: 0;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  cursor: pointer;
  transition: transform 0.3s, box-shadow 0.3s;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.stat-content {
  display: flex;
  align-items: center;
  padding: 10px 0;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  color: white;
  font-size: 28px;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #999;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #333;
}

.content-row {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-card {
  min-height: 400px;
}

.chart-container {
  width: 100%;
  height: 320px;
}

.user-info-card {
  min-height: 400px;
}

.user-info {
  padding: 10px 0;
}

.token-code {
  background: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  word-break: break-all;
}

.usage-progress {
  margin-top: 24px;
}

.progress-label {
  margin-bottom: 8px;
  font-size: 14px;
  color: #666;
}

.recent-requests-card {
  min-height: 400px;
}
</style>

