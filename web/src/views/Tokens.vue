<template>
  <div class="tokens-page">
    <el-row :gutter="16" class="summary-grid">
      <el-col :xs="24" :sm="12" :lg="6" v-for="card in tokenCards" :key="card.label">
        <el-card class="summary-card" shadow="hover">
          <div class="summary-label">{{ card.label }}</div>
          <div class="summary-value">{{ card.value }}</div>
          <div class="summary-meta">{{ card.meta }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="panel-header">
          <span>Backend Token 池</span>
          <div class="header-actions">
            <el-button :loading="syncingAllQuota" @click="syncAllQuota">同步全部额度</el-button>
            <el-button @click="openSmartDialog">Smart Login 导入</el-button>
            <el-button type="primary" @click="openCreateDialog">添加 Token</el-button>
          </div>
        </div>
      </template>

      <div class="sync-tip">
        Backend Token 的 Windsurf 配额会在客户端命中 `GetUserStatus` 时被动刷新。
        后台不会再主动构造 `GetUserStatus` 请求；如果需要刷新，请让真实 Windsurf 客户端经过 Gateway 完成一次启动/状态请求。
      </div>

      <el-form :inline="true" :model="filters" class="filters">
        <el-form-item label="状态">
          <el-select v-model="filters.status" clearable placeholder="全部状态" style="width: 150px">
            <el-option label="活跃" value="active" />
            <el-option label="禁用" value="disabled" />
            <el-option label="过期" value="expired" />
          </el-select>
        </el-form-item>
        <el-form-item label="池状态">
          <el-select v-model="filters.pool_status" clearable placeholder="全部池状态" style="width: 180px">
            <el-option label="可用" value="available" />
            <el-option label="冷却中" value="cooldown" />
            <el-option label="已耗尽" value="exhausted" />
            <el-option label="已过期" value="expired" />
            <el-option label="已禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadAll">搜索</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tokens" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="id" label="ID" min-width="120" show-overflow-tooltip />
        <el-table-column label="Token" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <code>{{ maskToken(row.token) }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="tenant_address" label="租户地址" min-width="180" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="pool_status" label="池状态" width="110">
          <template #default="{ row }">
            <el-tag :type="poolTag(row.pool_status)">{{ row.pool_status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="active_requests" label="并发" width="80" />
        <el-table-column label="套餐/额度" min-width="220">
          <template #default="{ row }">
            <div class="quota-cell">
              <div class="quota-plan">{{ row.plan_name || '未同步 Windsurf 配额' }}</div>
              <div v-for="line in quotaCreditLines(row)" :key="line" class="quota-sub">{{ line }}</div>
              <div class="quota-meta">
                同步时间 {{ formatTime(row.quota_updated_at) }}
                <span v-if="!row.quota_updated_at">，正常登录时会被动同步</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="日/周剩余" min-width="120">
          <template #default="{ row }">
            <div class="quota-sub">日 {{ formatQuotaPercent(row.daily_quota_remaining_percent, row.hide_daily_quota, row.quota_updated_at) }}</div>
            <div class="quota-sub">周 {{ formatQuotaPercent(row.weekly_quota_remaining_percent, row.hide_weekly_quota, row.quota_updated_at) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="重置时间" min-width="180">
          <template #default="{ row }">
            <div class="quota-sub">日 {{ formatQuotaReset(row.daily_quota_reset_at, row.hide_daily_quota, row.quota_updated_at) }}</div>
            <div class="quota-sub">周 {{ formatQuotaReset(row.weekly_quota_reset_at, row.hide_weekly_quota, row.quota_updated_at) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="请求计数" min-width="120">
          <template #default="{ row }">{{ row.used_requests }} / {{ row.max_requests || '∞' }}</template>
        </el-table-column>
        <el-table-column prop="total_successes" label="成功" width="90" />
        <el-table-column prop="total_failures" label="失败" width="90" />
        <el-table-column prop="consecutive_failures" label="连续失败" width="110" />
        <el-table-column prop="cooldown_until" label="冷却到期" min-width="170">
          <template #default="{ row }">{{ formatTime(row.cooldown_until) }}</template>
        </el-table-column>
        <el-table-column prop="last_error" label="最近错误" min-width="220" show-overflow-tooltip />
        <el-table-column prop="last_used_at" label="最近使用" min-width="170">
          <template #default="{ row }">{{ formatTime(row.last_used_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="132">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button size="small" @click="openEditDialog(row)">编辑</el-button>
              <el-dropdown @command="(command) => handleRowCommand(row, command)">
                <el-button size="small" :loading="syncingTokenId === row.id">
                  操作
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="sync">同步额度</el-dropdown-item>
                    <el-dropdown-item command="toggle-status">
                      {{ row.status === 'active' ? '禁用' : '启用' }}
                    </el-dropdown-item>
                    <el-dropdown-item v-if="row.pool_status === 'cooldown'" command="unlock">
                      解除冷却
                    </el-dropdown-item>
                    <el-dropdown-item command="delete">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="loadTokens"
          @size-change="loadTokens"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '添加 Token' : '编辑 Token'" width="720px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="名称" prop="name">
              <el-input v-model="form.name" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="租户地址" prop="tenant_address">
              <el-input v-model="form.tenant_address" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="Backend Token" prop="token">
              <el-input v-model="form.token" type="textarea" :rows="3" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="描述">
              <el-input v-model="form.description" type="textarea" :rows="2" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="Proxy URL">
              <el-input v-model="form.proxy_url" placeholder="可选" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="权重">
              <el-input-number v-model="form.weight" :min="1" :max="100" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="手动上限">
              <el-input-number v-model="form.max_requests" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="过期时间">
              <el-date-picker
                v-model="form.expires_at"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                placeholder="可选"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="smartDialogVisible" title="Smart Login 导入" width="620px">
      <el-form ref="smartFormRef" :model="smartForm" :rules="smartRules" label-width="120px">
        <el-alert
          v-if="smartHint"
          :title="smartHint"
          type="info"
          :closable="false"
          show-icon
          class="smart-alert"
        />
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="smartForm.email" autocomplete="username" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="密码" prop="password">
              <el-input v-model="smartForm.password" type="password" show-password autocomplete="current-password" />
            </el-form-item>
          </el-col>
          <el-col :span="24" v-if="smartOrgOptions.length">
            <el-form-item label="组织" prop="org_id">
              <el-select v-model="smartForm.org_id" placeholder="请选择组织" style="width: 100%">
                <el-option
                  v-for="org in smartOrgOptions"
                  :key="org.id"
                  :label="org.name || org.id"
                  :value="org.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="名称">
              <el-input v-model="smartForm.name" placeholder="默认用邮箱生成" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="权重">
              <el-input-number v-model="smartForm.weight" :min="1" :max="100" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="手动上限">
              <el-input-number v-model="smartForm.max_requests" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="Proxy URL">
              <el-input v-model="smartForm.proxy_url" placeholder="可选" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="备注">
              <el-input v-model="smartForm.description" type="textarea" :rows="2" placeholder="可选" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="smartDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="smartSubmitting" @click="submitSmartLogin">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import client from '../api/client'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref('create')
const editingId = ref('')
const formRef = ref(null)
const smartDialogVisible = ref(false)
const smartSubmitting = ref(false)
const syncingAllQuota = ref(false)
const syncingTokenId = ref('')
const smartFormRef = ref(null)
const smartOrgOptions = ref([])
const smartHint = ref('')
const tokens = ref([])
const tokenStats = ref({})

const filters = reactive({
  status: '',
  pool_status: '',
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
})

const form = reactive({
  name: '',
  token: '',
  description: '',
  tenant_address: 'https://server.codeium.com',
  proxy_url: '',
  weight: 1,
  max_requests: 0,
  expires_at: '',
})

const smartForm = reactive({
  email: '',
  password: '',
  org_id: '',
  name: '',
  description: '',
  proxy_url: '',
  weight: 1,
  max_requests: 0,
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  token: [{ required: true, message: '请输入 backend token', trigger: 'blur' }],
  tenant_address: [{ required: true, message: '请输入租户地址', trigger: 'blur' }],
}

const smartRules = {
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const tokenCards = computed(() => [
  {
    label: '总 Token',
    value: tokenStats.value.total || 0,
    meta: `活跃 ${tokenStats.value.active || 0}`,
  },
  {
    label: '可用 Token',
    value: tokenStats.value.available || 0,
    meta: `冷却 ${tokenStats.value.cooldown || 0}`,
  },
  {
    label: '已同步额度',
    value: tokenStats.value.quota_synced || 0,
    meta: `日低额 ${tokenStats.value.low_daily_quota || 0} / 周低额 ${tokenStats.value.low_weekly_quota || 0}`,
  },
  {
    label: '活跃并发',
    value: tokenStats.value.total_active_requests || 0,
    meta: `已耗尽 ${tokenStats.value.exhausted || 0}`,
  },
])

const loadTokenStats = async () => {
  const res = await client.get('/tokens/stats')
  if (res.data.code === 200) {
    tokenStats.value = res.data.data || {}
  }
}

const loadTokens = async () => {
  loading.value = true
  try {
    const res = await client.get('/tokens', {
      params: {
        page: pagination.page,
        page_size: pagination.page_size,
        status: filters.status,
        pool_status: filters.pool_status,
      },
    })
    if (res.data.code === 200) {
      tokens.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    } else {
      ElMessage.error(res.data.msg || '加载 Token 列表失败')
    }
  } catch (error) {
    ElMessage.error('加载 Token 列表失败')
  } finally {
    loading.value = false
  }
}

const loadAll = async () => {
  await Promise.all([loadTokenStats(), loadTokens()])
}

const syncQuota = async (row) => {
  syncingTokenId.value = row.id
  try {
    const res = await client.post(`/tokens/${row.id}/sync-quota`)
    if (res.data.code === 200) {
      ElMessage.success(res.data.msg || '已刷新本地 Token 状态，等待真实客户端被动同步额度')
      await loadAll()
    } else {
      ElMessage.error(res.data.msg || '同步额度失败')
    }
  } catch (error) {
    ElMessage.error('同步额度失败')
  } finally {
    syncingTokenId.value = ''
  }
}

const syncAllQuota = async () => {
  syncingAllQuota.value = true
  try {
    const res = await client.post('/tokens/sync-quota')
    if (res.data.code === 200) {
      const success = res.data.data?.success || 0
      const failed = res.data.data?.failed || 0
      ElMessage.success(res.data.msg || `已刷新 ${success} 个 Token 状态，失败 ${failed} 个；额度等待真实客户端被动同步`)
      await loadAll()
    } else {
      ElMessage.error(res.data.msg || '批量同步额度失败')
    }
  } catch (error) {
    ElMessage.error('批量同步额度失败')
  } finally {
    syncingAllQuota.value = false
  }
}

const resetFilters = async () => {
  filters.status = ''
  filters.pool_status = ''
  pagination.page = 1
  await loadAll()
}

const resetForm = () => {
  Object.assign(form, {
    name: '',
    token: '',
    description: '',
    tenant_address: 'https://server.codeium.com',
    proxy_url: '',
    weight: 1,
    max_requests: 0,
    expires_at: '',
  })
}

const resetSmartForm = () => {
  Object.assign(smartForm, {
    email: '',
    password: '',
    org_id: '',
    name: '',
    description: '',
    proxy_url: '',
    weight: 1,
    max_requests: 0,
  })
  smartOrgOptions.value = []
  smartHint.value = ''
}

const openCreateDialog = () => {
  dialogMode.value = 'create'
  editingId.value = ''
  resetForm()
  dialogVisible.value = true
}

const openSmartDialog = () => {
  resetSmartForm()
  smartDialogVisible.value = true
}

const openEditDialog = (row) => {
  dialogMode.value = 'edit'
  editingId.value = row.id
  Object.assign(form, {
    name: row.name,
    token: row.token,
    description: row.description || '',
    tenant_address: row.tenant_address,
    proxy_url: row.proxy_url || '',
    weight: row.weight || 1,
    max_requests: row.max_requests || 0,
    expires_at: row.expires_at ? dayjs(row.expires_at).format('YYYY-MM-DD HH:mm:ss') : '',
  })
  dialogVisible.value = true
}

const submitForm = async () => {
  await formRef.value?.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      const payload = { ...form }
      if (!payload.expires_at) {
        delete payload.expires_at
      }
      const res = dialogMode.value === 'create'
        ? await client.post('/tokens', payload)
        : await client.put(`/tokens/${editingId.value}`, payload)
      if (res.data.code === 200) {
        ElMessage.success(dialogMode.value === 'create' ? '添加成功' : '更新成功')
        dialogVisible.value = false
        await loadAll()
      } else {
        ElMessage.error(res.data.msg || '保存失败')
      }
    } catch (error) {
      ElMessage.error('保存失败')
    } finally {
      submitting.value = false
    }
  })
}

const submitSmartLogin = async () => {
  if (smartOrgOptions.value.length > 0 && !smartForm.org_id) {
    ElMessage.warning('请选择组织后再导入')
    return
  }

  await smartFormRef.value?.validate(async (valid) => {
    if (!valid) return
    smartSubmitting.value = true
    try {
      const payload = { ...smartForm }
      const res = await client.post('/tokens/smart-login', payload)
      if (res.data.code === 200) {
        const tokenKind = res.data.data?.backend_token_kind || 'token'
        ElMessage.success(`导入成功，已生成 ${tokenKind}`)
        smartDialogVisible.value = false
        await loadAll()
        return
      }

      if (res.data.code === 409 && res.data.data?.requires_org_selection) {
        smartOrgOptions.value = res.data.data?.orgs || []
        smartHint.value = res.data.data?.reason || '该账号有多个组织，请选择组织后重试'
        ElMessage.warning('该账号需要先选择组织')
        return
      }

      smartHint.value = res.data.data?.reason || ''
      ElMessage.error(res.data.msg || 'Smart Login 导入失败')
    } catch (error) {
      ElMessage.error('Smart Login 导入失败')
    } finally {
      smartSubmitting.value = false
    }
  })
}

const toggleStatus = async (row) => {
  const status = row.status === 'active' ? 'disabled' : 'active'
  try {
    await ElMessageBox.confirm(`确定要将 ${row.name} 标记为 ${status} 吗？`, '确认操作', { type: 'warning' })
    const res = await client.put(`/tokens/${row.id}`, { status })
    if (res.data.code === 200) {
      ElMessage.success('状态已更新')
      await loadAll()
    } else {
      ElMessage.error(res.data.msg || '更新失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('更新失败')
    }
  }
}

const deleteToken = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除 ${row.name} 吗？`, '确认删除', { type: 'warning' })
    const res = await client.delete(`/tokens/${row.id}`)
    if (res.data.code === 200) {
      ElMessage.success('删除成功')
      await loadAll()
    } else {
      ElMessage.error(res.data.msg || '删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const unlockCooldown = async (row) => {
  try {
    await ElMessageBox.confirm(`确定解除 ${row.name} 的冷却状态吗？`, '确认操作', { type: 'warning' })
    const res = await client.post(`/tokens/${row.id}/unlock`)
    if (res.data.code === 200) {
      ElMessage.success('冷却状态已解除')
      await loadAll()
    } else {
      ElMessage.error(res.data.msg || '解除冷却失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('解除冷却失败')
    }
  }
}

const handleRowCommand = async (row, command) => {
  switch (command) {
    case 'sync':
      await syncQuota(row)
      break
    case 'toggle-status':
      await toggleStatus(row)
      break
    case 'unlock':
      await unlockCooldown(row)
      break
    case 'delete':
      await deleteToken(row)
      break
  }
}

const maskToken = (value) => {
  if (!value) return ''
  if (value.length <= 12) return value
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

const formatCreditUsage = (used, available, total) => {
  const usedValue = Number(used || 0)
  const availableValue = Number(available || 0)
  const totalValue = Number(total || 0)
  if (usedValue <= 0 && availableValue <= 0 && totalValue <= 0) {
    return ''
  }

  const resolvedTotal = totalValue > 0 ? totalValue : usedValue + availableValue
  if (resolvedTotal > 0) {
    return `${usedValue}/${resolvedTotal}`
  }
  return `${usedValue}/${availableValue}`
}

const quotaCreditLines = (row) => {
  const lines = []
  const prompt = formatCreditUsage(row.used_prompt_credits, row.available_prompt_credits, row.monthly_prompt_credits)
  const flow = formatCreditUsage(row.used_flow_credits, row.available_flow_credits, row.monthly_flow_credits)
  const flex = formatCreditUsage(row.used_flex_credits, row.available_flex_credits, row.monthly_flex_credits)

  if (prompt) lines.push(`Prompt ${prompt}`)
  if (flow) lines.push(`Flow ${flow}`)
  if (flex) lines.push(`Flex ${flex}`)

  return lines.length > 0 ? lines : ['尚未从 GetUserStatus 同步额度']
}

const formatQuotaPercent = (value, hidden, quotaUpdatedAt) => {
  if (!quotaUpdatedAt) return '-'
  if (hidden) return '隐藏'
  return `${Number(value || 0)}%`
}

const formatQuotaReset = (value, hidden, quotaUpdatedAt) => {
  if (!quotaUpdatedAt) return '-'
  if (hidden) return '隐藏'
  return formatTime(value)
}

const formatTime = (value) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-')
const statusTag = (status) => ({ active: 'success', disabled: 'danger', expired: 'warning' }[status] || 'info')
const poolTag = (status) => ({ available: 'success', cooldown: 'warning', exhausted: 'danger', expired: 'info', disabled: 'danger' }[status] || 'info')

onMounted(loadAll)
</script>

<style scoped>
.tokens-page {
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
  font-size: 28px;
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

.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.sync-tip {
  margin-bottom: 14px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid #dbe6ff;
  background: #f5f8ff;
  color: #52637a;
  line-height: 1.65;
}

.filters {
  margin-bottom: 12px;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

code {
  color: #1d4ed8;
  font-size: 12px;
}

.quota-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.quota-plan {
  color: #132238;
  font-weight: 600;
}

.quota-sub {
  color: #4f647c;
  font-size: 12px;
  line-height: 1.4;
}

.quota-meta {
  color: #8a9aad;
  font-size: 12px;
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.smart-alert {
  margin-bottom: 16px;
}
</style>
