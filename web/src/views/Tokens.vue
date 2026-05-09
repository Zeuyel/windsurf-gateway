<template>
  <div class="tokens-page">
    <el-card class="operation-card">
      <template #header>
        <div class="card-header">
          <span>Token 管理</span>
          <el-button type="primary" @click="showCreateDialog = true">
            <el-icon><Plus /></el-icon> 添加 Token
          </el-button>
        </div>
      </template>
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="全部状态" clearable>
            <el-option label="全部" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="禁用" value="disabled" />
            <el-option label="过期" value="expired" />
          </el-select>
        </el-form-item>
        <el-form-item label="池状态">
          <el-select v-model="searchForm.poolStatus" placeholder="全部" clearable>
            <el-option label="全部" value="" />
            <el-option label="可用" value="available" />
            <el-option label="已分配" value="allocated" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadTokens">
            <el-icon><Search /></el-icon> 搜索
          </el-button>
          <el-button @click="resetSearch">
            <el-icon><Refresh /></el-icon> 重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card">
      <el-table :data="tokens" stripe v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="token" label="Token" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="token-text">{{ maskToken(row.token) }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="tenant_address" label="租户地址" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="pool_status" label="池状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getPoolStatusType(row.pool_status)" size="small">
              {{ row.pool_status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="使用量" width="120">
          <template #default="{ row }">
            <el-progress 
              :percentage="getUsagePercentage(row)" 
              :color="getUsageColor(row)"
              :stroke-width="8"
            />
          </template>
        </el-table-column>
        <el-table-column prop="used_requests" label="已用/上限" width="120">
          <template #default="{ row }">
            {{ row.used_requests }} / {{ row.max_requests || '∞' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button size="small" @click="editToken(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button size="small" type="warning" @click="toggleStatus(row)">
                <el-icon><Switch /></el-icon>
              </el-button>
              <el-button size="small" type="danger" @click="deleteToken(row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadTokens"
          @current-change="loadTokens"
        />
      </div>
    </el-card>

    <el-dialog v-model="showCreateDialog" title="添加 Token" width="500px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入 Token 名称" />
        </el-form-item>
        <el-form-item label="Token" prop="token">
          <el-input 
            v-model="form.token" 
            type="textarea" 
            :rows="3"
            placeholder="请输入 Windsurf Token"
          />
        </el-form-item>
        <el-form-item label="租户地址" prop="tenant_address">
          <el-input v-model="form.tenant_address" placeholder="https://server.codeium.com" />
        </el-form-item>
        <el-form-item label="最大请求" prop="max_requests">
          <el-input-number v-model="form.max_requests" :min="0" />
        </el-form-item>
        <el-form-item label="过期时间" prop="expires_at">
          <el-date-picker
            v-model="form.expires_at"
            type="datetime"
            placeholder="选择过期时间"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import client from '../api/client'
import dayjs from 'dayjs'

const loading = ref(false)
const tokens = ref([])
const showCreateDialog = ref(false)
const submitting = ref(false)
const formRef = ref(null)

const searchForm = reactive({
  status: '',
  poolStatus: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const form = reactive({
  name: '',
  token: '',
  tenant_address: 'https://server.codeium.com',
  max_requests: 0,
  expires_at: null
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  token: [{ required: true, message: '请输入 Token', trigger: 'blur' }],
  tenant_address: [{ required: true, message: '请输入租户地址', trigger: 'blur' }]
}

const maskToken = (token) => {
  if (!token) return ''
  if (token.length <= 8) return token
  return token.substring(0, 4) + '****' + token.substring(token.length - 4)
}

const getStatusType = (status) => {
  const types = {
    active: 'success',
    disabled: 'danger',
    expired: 'warning'
  }
  return types[status] || 'info'
}

const getPoolStatusType = (poolStatus) => {
  const types = {
    available: 'success',
    allocated: 'warning',
    disabled: 'danger'
  }
  return types[poolStatus] || 'info'
}

const getUsagePercentage = (row) => {
  if (!row.max_requests || row.max_requests === 0) return 0
  return Math.round((row.used_requests / row.max_requests) * 100)
}

const getUsageColor = (row) => {
  const percentage = getUsagePercentage(row)
  if (percentage < 50) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

const loadTokens = async () => {
  loading.value = true
  try {
    const res = await client.get('/tokens', {
      params: {
        page: pagination.page,
        page_size: pagination.pageSize,
        status: searchForm.status,
        pool_status: searchForm.poolStatus
      }
    })
    if (res.data.code === 200) {
      tokens.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    }
  } catch (error) {
    ElMessage.error('加载 Token 列表失败')
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.status = ''
  searchForm.poolStatus = ''
  pagination.page = 1
  loadTokens()
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        const data = { ...form }
        if (data.expires_at) {
          data.expires_at = dayjs(data.expires_at).format('YYYY-MM-DD HH:mm:ss')
        }
        const res = await client.post('/tokens', data)
        if (res.data.code === 200) {
          ElMessage.success('添加成功')
          showCreateDialog.value = false
          Object.assign(form, {
            name: '',
            token: '',
            tenant_address: 'https://server.codeium.com',
            max_requests: 0,
            expires_at: null
          })
          loadTokens()
        } else {
          ElMessage.error(res.data.msg || '添加失败')
        }
      } catch (error) {
        ElMessage.error('添加失败')
      } finally {
        submitting.value = false
      }
    }
  })
}

const editToken = (row) => {
  Object.assign(form, {
    name: row.name,
    token: row.token,
    tenant_address: row.tenant_address,
    max_requests: row.max_requests,
    expires_at: row.expires_at ? new Date(row.expires_at) : null
  })
  showCreateDialog.value = true
}

const toggleStatus = async (row) => {
  const newStatus = row.status === 'active' ? 'disabled' : 'active'
  try {
    await ElMessageBox.confirm(
      `确定要${newStatus === 'active' ? '启用' : '禁用'}该 Token 吗？`,
      '确认操作',
      { type: 'warning' }
    )
    const res = await client.put(`/tokens/${row.id}`, { status: newStatus })
    if (res.data.code === 200) {
      ElMessage.success('操作成功')
      loadTokens()
    } else {
      ElMessage.error(res.data.msg || '操作失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

const deleteToken = async (row) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除该 Token 吗？此操作不可恢复。',
      '确认删除',
      { type: 'warning' }
    )
    const res = await client.delete(`/tokens/${row.id}`)
    if (res.data.code === 200) {
      ElMessage.success('删除成功')
      loadTokens()
    } else {
      ElMessage.error(res.data.msg || '删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

onMounted(() => {
  loadTokens()
})
</script>

<style scoped>
.tokens-page {
  padding: 0;
}

.operation-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-form {
  margin-bottom: 0;
}

.table-card {
  min-height: 600px;
}

.token-text {
  background: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Courier New', monospace;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>

