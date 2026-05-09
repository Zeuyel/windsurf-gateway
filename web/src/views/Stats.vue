<template>
  <div class="stats-page">
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
              <el-icon><SuccessFilled /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">成功请求</div>
              <div class="stat-value">{{ stats.successRequests || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8" :lg="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #f56c6c;">
              <el-icon><CircleCloseFilled /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">失败请求</div>
              <div class="stat-value">{{ stats.failedRequests || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8" :lg="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #e6a23c;">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">平均延迟</div>
              <div class="stat-value">{{ avgLatency }}ms</div>
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
              <el-radio-group v-model="timeRange" size="small" @change="updateChart">
                <el-radio-button label="24h">24小时</el-radio-button>
                <el-radio-button label="7d">7天</el-radio-button>
                <el-radio-button label="30d">30天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div ref="trendChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="8">
        <el-card class="chart-card">
          <template #header>
            <span>请求分布</span>
          </template>
          <div ref="pieChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :lg="12">
        <el-card class="chart-card">
          <template #header>
            <span>Token 使用排名</span>
          </template>
          <div ref="tokenChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card class="chart-card">
          <template #header>
            <span>用户活跃度</span>
          </template>
          <div ref="userChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import client from '../api/client'
import * as echarts from 'echarts'

const stats = ref({
  totalRequests: 0,
  successRequests: 0,
  failedRequests: 0,
  totalLatency: 0
})

const timeRange = ref('24h')
const trendChartRef = ref(null)
const pieChartRef = ref(null)
const tokenChartRef = ref(null)
const userChartRef = ref(null)

let trendChart = null
let pieChart = null
let tokenChart = null
let userChart = null

const avgLatency = computed(() => {
  if (!stats.value.totalRequests || stats.value.totalRequests === 0) return 0
  return Math.round(stats.value.totalLatency / stats.value.totalRequests)
})

const loadStats = async () => {
  try {
    const res = await client.get('/stats/overview')
    if (res.data.code === 200) {
      stats.value = res.data.data || {}
    }
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const initTrendChart = () => {
  if (!trendChartRef.value) return
  
  trendChart = echarts.init(trendChartRef.value)
  const option = {
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: ['成功', '失败']
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
        name: '成功',
        type: 'line',
        smooth: true,
        data: [],
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(103, 194, 58, 0.3)' },
            { offset: 1, color: 'rgba(103, 194, 58, 0.05)' }
          ])
        },
        lineStyle: { color: '#67c23a' },
        itemStyle: { color: '#67c23a' }
      },
      {
        name: '失败',
        type: 'line',
        smooth: true,
        data: [],
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(245, 108, 108, 0.3)' },
            { offset: 1, color: 'rgba(245, 108, 108, 0.05)' }
          ])
        },
        lineStyle: { color: '#f56c6c' },
        itemStyle: { color: '#f56c6c' }
      }
    ]
  }
  trendChart.setOption(option)
}

const initPieChart = () => {
  if (!pieChartRef.value) return
  
  pieChart = echarts.init(pieChartRef.value)
  const option = {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '请求类型',
        type: 'pie',
        radius: '50%',
        data: [
          { value: 0, name: 'GET' },
          { value: 0, name: 'POST' },
          { value: 0, name: 'PUT' },
          { value: 0, name: 'DELETE' }
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
  pieChart.setOption(option)
}

const initTokenChart = () => {
  if (!tokenChartRef.value) return
  
  tokenChart = echarts.init(tokenChartRef.value)
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: []
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '请求数',
        type: 'bar',
        data: [],
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#83bff6' },
            { offset: 0.5, color: '#188df0' },
            { offset: 1, color: '#188df0' }
          ])
        }
      }
    ]
  }
  tokenChart.setOption(option)
}

const initUserChart = () => {
  if (!userChartRef.value) return
  
  userChart = echarts.init(userChartRef.value)
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
      data: []
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '活跃度',
        type: 'bar',
        data: [],
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#a0cfff' },
            { offset: 0.5, color: '#409eff' },
            { offset: 1, color: '#409eff' }
          ])
        }
      }
    ]
  }
  userChart.setOption(option)
}

const updateChart = async () => {
  try {
    const res = await client.get('/stats/trend', {
      params: { range: timeRange.value }
    })
    if (res.data.code === 200) {
      const data = res.data.data || []
      const xData = data.map(item => item.time)
      const successData = data.map(item => item.success || 0)
      const failedData = data.map(item => item.failed || 0)
      
      trendChart?.setOption({
        xAxis: { data: xData },
        series: [
          { data: successData },
          { data: failedData }
        ]
      })
    }
  } catch (error) {
    console.error('Failed to load chart data:', error)
  }
}

const handleResize = () => {
  trendChart?.resize()
  pieChart?.resize()
  tokenChart?.resize()
  userChart?.resize()
}

onMounted(async () => {
  await loadStats()
  initTrendChart()
  initPieChart()
  initTokenChart()
  initUserChart()
  await updateChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  trendChart?.dispose()
  pieChart?.dispose()
  tokenChart?.dispose()
  userChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.stats-page {
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
</style>
