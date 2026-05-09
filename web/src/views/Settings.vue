<template>
  <div class="settings-page">
    <el-row :gutter="20">
      <el-col :xs="24" :lg="8">
        <el-card class="profile-card" shadow="never">
          <template #header>
            <span>个人资料</span>
          </template>
          <div class="profile-content">
            <div class="avatar-section">
              <el-avatar :size="80" :icon="UserFilled" />
            </div>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="用户名">{{ authStore.user?.username }}</el-descriptions-item>
              <el-descriptions-item label="邮箱">{{ authStore.user?.email || 'N/A' }}</el-descriptions-item>
              <el-descriptions-item label="角色">
                <el-tag :type="authStore.user?.role === 'admin' ? 'danger' : 'primary'" size="small">
                  {{ authStore.user?.role }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="authStore.user?.status === 'active' ? 'success' : 'danger'" size="small">
                  {{ authStore.user?.status }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="16">
        <el-card class="settings-card" shadow="never">
          <template #header>
            <span>账户设置</span>
          </template>
          <el-tabs v-model="activeTab">
            <el-tab-pane label="基本信息" name="basic">
              <el-form ref="profileFormRef" :model="profileForm" :rules="profileRules" label-width="100px">
                <el-form-item label="用户名">
                  <el-input v-model="profileForm.username" disabled />
                </el-form-item>
                <el-form-item label="邮箱" prop="email">
                  <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" :loading="updating" @click="updateProfile">保存修改</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="修改密码" name="password" :disabled="authStore.isAdmin">
              <el-alert
                v-if="authStore.isAdmin"
                title="管理员账户暂不支持在此页面修改密码"
                type="info"
                show-icon
                :closable="false"
              />
              <el-form v-else ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="100px">
                <el-form-item label="当前密码" prop="oldPassword">
                  <el-input v-model="passwordForm.oldPassword" type="password" show-password />
                </el-form-item>
                <el-form-item label="新密码" prop="newPassword">
                  <el-input v-model="passwordForm.newPassword" type="password" show-password />
                </el-form-item>
                <el-form-item label="确认密码" prop="confirmPassword">
                  <el-input v-model="passwordForm.confirmPassword" type="password" show-password />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" :loading="changing" @click="changePassword">修改密码</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="API Token" name="token" :disabled="authStore.isAdmin">
              <el-alert
                v-if="authStore.isAdmin"
                title="管理员账户不使用网关 API Token"
                type="info"
                show-icon
                :closable="false"
              />
              <el-form v-else label-width="100px">
                <el-form-item label="当前 Token">
                  <el-input :model-value="authStore.user?.api_token || ''" readonly type="textarea" :rows="3">
                    <template #append>
                      <el-button @click="copyToken">
                        <el-icon><CopyDocument /></el-icon>
                      </el-button>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item>
                  <el-button type="warning" :loading="regenerating" @click="regenerateToken">重新生成 Token</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="使用情况" name="usage">
              <div class="usage-section">
                <div class="usage-item">
                  <div class="usage-label">已用请求</div>
                  <div class="usage-value">{{ authStore.user?.used_requests || 0 }}</div>
                </div>
                <div class="usage-item">
                  <div class="usage-label">最大请求</div>
                  <div class="usage-value">{{ authStore.user?.max_requests || '无限制' }}</div>
                </div>
                <div class="usage-item">
                  <div class="usage-label">使用进度</div>
                  <el-progress :percentage="usagePercentage" :color="usageColor" :stroke-width="20" />
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, UserFilled } from '@element-plus/icons-vue'
import client from '../api/client'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()
const activeTab = ref('basic')
const updating = ref(false)
const changing = ref(false)
const regenerating = ref(false)
const profileFormRef = ref(null)
const passwordFormRef = ref(null)

const profileForm = reactive({
  username: '',
  email: '',
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const profileRules = {
  email: [{ type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }],
}

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.newPassword) {
    callback(new Error('两次输入的密码不一致'))
    return
  }
  callback()
}

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' },
  ],
}

const usagePercentage = computed(() => {
  if (!authStore.user?.max_requests || authStore.user.max_requests === 0) return 0
  return Math.round((authStore.user.used_requests / authStore.user.max_requests) * 100)
})

const usageColor = computed(() => {
  if (usagePercentage.value < 50) return '#67c23a'
  if (usagePercentage.value < 80) return '#e6a23c'
  return '#f56c6c'
})

const loadProfile = () => {
  profileForm.username = authStore.user?.username || ''
  profileForm.email = authStore.user?.email || ''
}

const updateProfile = async () => {
  await profileFormRef.value?.validate(async (valid) => {
    if (!valid) return
    updating.value = true
    try {
      const endpoint = authStore.isAdmin ? '/auth/profile' : '/user/profile'
      const res = await client.put(endpoint, { email: profileForm.email })
      if (res.data.code === 200) {
        ElMessage.success('更新成功')
        await authStore.fetchUser()
        loadProfile()
      } else {
        ElMessage.error(res.data.msg || '更新失败')
      }
    } catch (error) {
      ElMessage.error('更新失败')
    } finally {
      updating.value = false
    }
  })
}

const changePassword = async () => {
  if (authStore.isAdmin) {
    ElMessage.warning('管理员账户暂不支持在此页面修改密码')
    return
  }

  await passwordFormRef.value?.validate(async (valid) => {
    if (!valid) return
    changing.value = true
    try {
      const res = await client.post('/user/change-password', {
        old_password: passwordForm.oldPassword,
        new_password: passwordForm.newPassword,
      })
      if (res.data.code === 200) {
        ElMessage.success('密码修改成功，请重新登录')
        authStore.logout()
        window.location.href = '/login'
      } else {
        ElMessage.error(res.data.msg || '密码修改失败')
      }
    } catch (error) {
      ElMessage.error('密码修改失败')
    } finally {
      changing.value = false
    }
  })
}

const copyToken = async () => {
  const token = authStore.user?.api_token
  if (!token) return
  await navigator.clipboard.writeText(token)
  ElMessage.success('Token 已复制到剪贴板')
}

const regenerateToken = async () => {
  if (authStore.isAdmin) {
    ElMessage.warning('管理员账户不使用网关 API Token')
    return
  }

  try {
    await ElMessageBox.confirm('重新生成 Token 后，旧 Token 将立即失效，确定继续吗？', '确认操作', {
      type: 'warning',
    })
    regenerating.value = true
    const res = await client.post('/user/regenerate-token')
    if (res.data.code === 200) {
      ElMessage.success('Token 已重新生成')
      await authStore.fetchUser()
    } else {
      ElMessage.error(res.data.msg || 'Token 重新生成失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Token 重新生成失败')
    }
  } finally {
    regenerating.value = false
  }
}

onMounted(loadProfile)
</script>

<style scoped>
.settings-page {
  padding: 0;
}

.profile-card,
.settings-card {
  margin-bottom: 20px;
}

.profile-content,
.usage-section {
  padding: 20px 0;
}

.avatar-section {
  text-align: center;
  margin-bottom: 24px;
}

.usage-item {
  margin-bottom: 24px;
}

.usage-label {
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.usage-value {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin-bottom: 16px;
}
</style>
