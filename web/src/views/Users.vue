<template>
  <div class="users-page">
    <el-card class="operation-card">
      <template #header>
        <div class="card-header">
          <span>用户管理</span>
          <el-button type="primary" @click="openCreateDialog">添加用户</el-button>
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
            <el-option label="封禁" value="banned" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadUsers">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card">
      <div class="table-tip">
        <strong>调度说明：</strong>
        这里控制的是“这个用户请求到来时，Gateway 挑哪类 Backend Token”，不是给该用户单独记一套个人日限额或周限额。
        默认只会挑选仍有 Windsurf 日/周额度的 Token；开启忽略额度检查后，会放宽这层筛选。
      </div>

      <el-table :data="users" stripe v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="email" label="邮箱" min-width="200" show-overflow-tooltip />
        <el-table-column prop="api_token" label="网关 Token" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="token-text">{{ maskToken(row.api_token) }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'" size="small">
              {{ row.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="接入策略" min-width="170">
          <template #default="{ row }">
            <el-tag :type="row.unlimited_access ? 'warning' : 'success'" size="small">
              {{ row.unlimited_access ? '忽略 Token 额度检查' : '仅走有额度 Token' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rate_limit_per_minute" label="每分钟限速" width="120">
          <template #default="{ row }">
            {{ row.rate_limit_per_minute || 30 }} 次
          </template>
        </el-table-column>
        <el-table-column prop="used_requests" label="累计请求" width="120" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button size="small" @click="openEditDialog(row)">编辑</el-button>
              <el-button size="small" type="warning" @click="toggleStatus(row)">
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </el-button>
              <el-button size="small" type="danger" @click="deleteUser(row)">删除</el-button>
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

    <el-dialog v-model="showDialog" :title="dialogTitle" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="isEditing" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item v-if="!isEditing" label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item v-else label="重设密码">
          <el-input v-model="form.password" type="password" placeholder="留空则不修改" show-password />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="form.role" placeholder="请选择角色">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="账户状态" prop="status">
          <el-select v-model="form.status" placeholder="请选择状态">
            <el-option label="活跃" value="active" />
            <el-option label="禁用" value="disabled" />
            <el-option label="封禁" value="banned" />
          </el-select>
        </el-form-item>
        <el-form-item label="额度调度">
          <div class="switch-row">
            <el-switch v-model="form.unlimited_access" />
            <span class="switch-copy">
              {{
                form.unlimited_access
                  ? '开启后，请求调度时不再检查 Backend Token 的 Windsurf 日/周额度。'
                  : '关闭后，只会挑选仍有 Windsurf 日/周额度的 Backend Token。'
              }}
            </span>
          </div>
        </el-form-item>
        <el-form-item label="每分钟限速" prop="rate_limit_per_minute">
          <el-input-number v-model="form.rate_limit_per_minute" :min="1" :max="600" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import client from '../api/client'
import dayjs from 'dayjs'

const loading = ref(false)
const users = ref([])
const showDialog = ref(false)
const submitting = ref(false)
const formRef = ref(null)
const editingUserId = ref(null)

const searchForm = reactive({
  username: '',
  status: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const baseForm = () => ({
  username: '',
  email: '',
  password: '',
  role: 'user',
  status: 'active',
  unlimited_access: false,
  rate_limit_per_minute: 30
})

const form = reactive(baseForm())

const isEditing = computed(() => editingUserId.value !== null)
const dialogTitle = computed(() => (isEditing.value ? '编辑用户' : '添加用户'))

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

const maskToken = (token) => {
  if (!token) return ''
  if (token.length <= 12) return token
  return token.substring(0, 8) + '...' + token.substring(token.length - 4)
}

const formatTime = (time) => {
  if (!time) return '-'
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const statusLabel = (status) => {
  switch (status) {
    case 'active':
      return '活跃'
    case 'disabled':
      return '禁用'
    case 'banned':
      return '封禁'
    default:
      return status || '-'
  }
}

const statusTagType = (status) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'disabled':
      return 'warning'
    case 'banned':
      return 'danger'
    default:
      return 'info'
  }
}

const resetForm = () => {
  editingUserId.value = null
  Object.assign(form, baseForm())
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
    } else {
      ElMessage.error(res.data.msg || '加载用户列表失败')
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

const openCreateDialog = () => {
  resetForm()
  showDialog.value = true
}

const openEditDialog = (row) => {
  editingUserId.value = row.id
  Object.assign(form, {
    username: row.username,
    email: row.email,
    password: '',
    role: row.role,
    status: row.status,
    unlimited_access: !!row.unlimited_access,
    rate_limit_per_minute: row.rate_limit_per_minute || 30
  })
  showDialog.value = true
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      const payload = {
        username: form.username,
        email: form.email,
        password: form.password,
        role: form.role,
        status: form.status,
        unlimited_access: form.unlimited_access,
        rate_limit_per_minute: form.rate_limit_per_minute
      }

      const res = isEditing.value
        ? await client.put(`/users/${editingUserId.value}`, {
            email: payload.email,
            password: payload.password || undefined,
            role: payload.role,
            status: payload.status,
            unlimited_access: payload.unlimited_access,
            rate_limit_per_minute: payload.rate_limit_per_minute
          })
        : await client.post('/users', payload)

      if (res.data.code === 200) {
        ElMessage.success(isEditing.value ? '用户已更新' : '用户已创建')
        showDialog.value = false
        resetForm()
        loadUsers()
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

const toggleStatus = async (row) => {
  const newStatus = row.status === 'active' ? 'disabled' : 'active'
  try {
    await ElMessageBox.confirm(
      `确定要将用户 ${row.username} ${newStatus === 'active' ? '启用' : '禁用'} 吗？`,
      '确认操作',
      { type: 'warning' }
    )
    const res = await client.put(`/users/${row.id}`, { status: newStatus })
    if (res.data.code === 200) {
      ElMessage.success('状态已更新')
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
      `确定要删除用户 ${row.username} 吗？此操作不可恢复。`,
      '确认删除',
      { type: 'warning' }
    )
    const res = await client.delete(`/users/${row.id}`)
    if (res.data.code === 200) {
      ElMessage.success('用户已删除')
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

.table-tip {
  margin-bottom: 16px;
  padding: 14px 16px;
  border-radius: 12px;
  background: #f4f8ff;
  border: 1px solid #d8e6ff;
  line-height: 1.7;
  color: #44536a;
}

.token-text {
  background: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Courier New', monospace;
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 12px;
  line-height: 1.6;
}

.switch-copy {
  color: #666;
  font-size: 13px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
