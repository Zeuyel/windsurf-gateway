<template>
  <div class="users-page">
    <el-card class="operation-card">
      <template #header>
        <div class="card-header">
          <span>用户管理</span>
          <el-button type="primary" @click="showCreateDialog = true">
            <el-icon><Plus /></el-icon> 添加用户
          </el-button>
        </div>
      </template>
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="用户名">
          <el-input v-model="searchForm.username" placeholder="搜索用户名" clearable />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="全部状态" clearable>
            <el-option label="全部" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadUsers">
            <el-icon><Search /></el-icon> 搜索
          </el-button>
          <el-button @click="resetSearch">
            <el-icon><Refresh /></el-icon> 重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card">
      <el-table :data="users" stripe v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" show-overflow-tooltip />
        <el-table-column prop="api_token" label="API Token" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="token-text">{{ maskToken(row.api_token) }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'" size="small">
              {{ row.role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status }}
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
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button size="small" @click="editUser(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button size="small" type="warning" @click="toggleStatus(row)">
                <el-icon><Switch /></el-icon>
              </el-button>
              <el-button size="small" type="danger" @click="deleteUser(row)">
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
          @size-change="loadUsers"
          @current-change="loadUsers"
        />
      </div>
    </el-card>

    <el-dialog v-model="showCreateDialog" title="添加用户" width="500px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="form.role" placeholder="请选择角色">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="最大请求" prop="max_requests">
          <el-input-number v-model="form.max_requests" :min="0" />
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
const users = ref([])
const showCreateDialog = ref(false)
const submitting = ref(false)
const formRef = ref(null)

const searchForm = reactive({
  username: '',
  status: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const form = reactive({
  username: '',
  email: '',
  password: '',
  role: 'user',
  max_requests: 0
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

const maskToken = (token) => {
  if (!token) return ''
  if (token.length <= 8) return token
  return token.substring(0, 4) + '****' + token.substring(token.length - 4)
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

const formatTime = (time) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await client.get('/users', {
      params: {
        page: pagination.page,
        page_size: pagination.pageSize,
        username: searchForm.username,
        status: searchForm.status
      }
    })
    if (res.data.code === 200) {
      users.value = res.data.data?.list || []
      pagination.total = res.data.data?.total || 0
    }
  } catch (error) {
    ElMessage.error('加载用户列表失败')
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.username = ''
  searchForm.status = ''
  pagination.page = 1
  loadUsers()
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        const res = await client.post('/users', form)
        if (res.data.code === 200) {
          ElMessage.success('添加成功')
          showCreateDialog.value = false
          Object.assign(form, {
            username: '',
            email: '',
            password: '',
            role: 'user',
            max_requests: 0
          })
          loadUsers()
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

const editUser = (row) => {
  Object.assign(form, {
    username: row.username,
    email: row.email,
    password: '',
    role: row.role,
    max_requests: row.max_requests
  })
  showCreateDialog.value = true
}

const toggleStatus = async (row) => {
  const newStatus = row.status === 'active' ? 'disabled' : 'active'
  try {
    await ElMessageBox.confirm(
      `确定要${newStatus === 'active' ? '启用' : '禁用'}该用户吗？`,
      '确认操作',
      { type: 'warning' }
    )
    const res = await client.put(`/users/${row.id}`, { status: newStatus })
    if (res.data.code === 200) {
      ElMessage.success('操作成功')
      loadUsers()
    } else {
      ElMessage.error(res.data.msg || '操作失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

const deleteUser = async (row) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除该用户吗？此操作不可恢复。',
      '确认删除',
      { type: 'warning' }
    )
    const res = await client.delete(`/users/${row.id}`)
    if (res.data.code === 200) {
      ElMessage.success('删除成功')
      loadUsers()
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
  loadUsers()
})
</script>

<style scoped>
.users-page {
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
