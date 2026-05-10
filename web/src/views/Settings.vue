<template>
  <div class="settings-page">
    <el-row :gutter="20">
      <el-col :xs="24" :lg="8">
        <el-card class="overview-card" shadow="never">
          <template #header>
            <span>账户概览</span>
          </template>

          <div class="overview-top">
            <el-avatar :size="76" :icon="UserFilled" />
            <div>
              <div class="overview-name">{{ authStore.user?.username || '-' }}</div>
              <div class="overview-role">{{ authStore.isAdmin ? '管理员账户' : 'Gateway 用户账户' }}</div>
            </div>
          </div>

          <el-descriptions :column="1" border>
            <el-descriptions-item label="邮箱">{{ authStore.user?.email || '未填写' }}</el-descriptions-item>
            <el-descriptions-item label="角色">
              <el-tag :type="authStore.isAdmin ? 'danger' : 'primary'" size="small">
                {{ authStore.isAdmin ? 'admin' : 'user' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="账户状态">
              <el-tag :type="userStatusType" size="small">{{ userStatusLabel }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="令牌状态" v-if="!authStore.isAdmin">
              <el-tag :type="tokenStatusType" size="small">{{ tokenStatusLabel }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="接入模式" v-if="!authStore.isAdmin">
              <el-tag :type="authStore.user?.unlimited_access ? 'warning' : 'success'" size="small">
                {{ accessModeLabel }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="累计请求" v-if="!authStore.isAdmin">
              {{ authStore.user?.used_requests || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="每分钟限速" v-if="!authStore.isAdmin">
              {{ rateLimitLabel }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="16">
        <el-card class="settings-card" shadow="never">
          <template #header>
            <span>{{ authStore.isAdmin ? '管理设置' : '个人设置' }}</span>
          </template>

          <el-tabs v-model="activeTab" class="settings-tabs">
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

            <el-tab-pane v-if="!authStore.isAdmin" label="网关令牌" name="token">
              <div class="info-panel">
                <strong>如何使用</strong>
                <p>如果管理员开启了 Gateway 用户鉴权，请在 patcher 中选择“网关用户令牌”模式，并填入下面这个以 <code>ws-</code> 开头的令牌。</p>
              </div>

              <el-form label-width="110px">
                <el-form-item label="当前令牌">
                  <el-input :model-value="authStore.user?.api_token || ''" readonly type="textarea" :rows="3" />
                </el-form-item>
                <el-form-item>
                  <el-button @click="copyToken">
                    <el-icon><CopyDocument /></el-icon>
                    <span>复制令牌</span>
                  </el-button>
                  <el-button type="warning" :loading="regenerating" @click="regenerateToken">重新生成令牌</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="接入与额度" name="usage">
              <template v-if="authStore.isAdmin">
                <el-alert
                  title="管理员账户只用于登录管理后台，不作为 Gateway 客户端令牌。"
                  type="info"
                  show-icon
                  :closable="false"
                />
                <div class="info-panel admin-panel">
                  <strong>管理职责</strong>
                  <p>在这里控制 Gateway 是否要求客户端必须携带 ws 用户令牌；真正的 Windsurf 日限额与周限额由 Backend Token 池同步并调度。</p>
                </div>
              </template>

              <template v-else>
                <div class="usage-grid">
                  <div class="usage-card">
                    <span>接入模式</span>
                    <strong>{{ accessModeLabel }}</strong>
                    <p>{{ usageModeHint }}</p>
                  </div>
                  <div class="usage-card">
                    <span>累计请求</span>
                    <strong>{{ authStore.user?.used_requests || 0 }}</strong>
                    <p>这里记录 Gateway 侧累计转发次数。</p>
                  </div>
                  <div class="usage-card">
                    <span>每分钟限速</span>
                    <strong>{{ rateLimitLabel }}</strong>
                    <p>超过这个速度会在 Gateway 侧被限流。</p>
                  </div>
                  <div class="usage-card">
                    <span>令牌状态</span>
                    <strong>{{ tokenStatusLabel }}</strong>
                    <p>只有令牌状态为活跃时，客户端才能正常接入 Gateway。</p>
                  </div>
                </div>

                <el-alert
                  :title="usageAlertTitle"
                  type="info"
                  show-icon
                  :closable="false"
                />
              </template>
            </el-tab-pane>

            <el-tab-pane v-if="!authStore.isAdmin" label="修改密码" name="password">
              <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="100px">
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

            <el-tab-pane v-if="authStore.isAdmin" label="Gateway 接入" name="gateway">
              <el-skeleton :loading="gatewayConfigLoading" animated>
                <template #template>
                  <el-skeleton-item variant="rect" style="width: 100%; height: 180px; border-radius: 16px;" />
                </template>
                <template #default>
                  <div class="gateway-card">
                    <div class="gateway-head">
                      <div>
                        <h3>客户端是否必须先鉴权</h3>
                        <p>开启后，只知道 Gateway 地址不足以使用。客户端必须携带后台创建的 <code>ws-...</code> Gateway 用户令牌。</p>
                      </div>
                      <el-switch v-model="gatewayConfig.require_user_auth_proxy" />
                    </div>

                    <el-alert
                      :title="gatewayAuthHint"
                      :type="gatewayConfig.require_user_auth_proxy ? 'success' : 'warning'"
                      show-icon
                      :closable="false"
                    />

                    <div class="info-panel compact-panel">
                      <strong>建议搭配方式</strong>
                      <p>打开该开关后，patcher 应选择“网关用户令牌”模式，让不同用户拿各自的 <code>ws-...</code> 令牌接入。这样管理员可以单独禁用用户、切换无限模式，并利用 Windsurf credit 池做统一调度。</p>
                    </div>

                    <div class="gateway-actions">
                      <el-button type="primary" :loading="gatewayConfigSaving" @click="saveGatewayConfig">保存 Gateway 接入规则</el-button>
                    </div>
                  </div>
                </template>
              </el-skeleton>
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
const activeTab = ref(authStore.isAdmin ? 'gateway' : 'token')
const updating = ref(false)
const changing = ref(false)
const regenerating = ref(false)
const gatewayConfigLoading = ref(false)
const gatewayConfigSaving = ref(false)
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

const gatewayConfig = reactive({
  require_user_auth_proxy: false,
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

const userStatusLabel = computed(() => {
  switch (authStore.user?.status) {
    case 'active':
      return '活跃'
    case 'disabled':
      return '禁用'
    case 'banned':
      return '封禁'
    default:
      return authStore.user?.status || '-'
  }
})

const userStatusType = computed(() => {
  switch (authStore.user?.status) {
    case 'active':
      return 'success'
    case 'disabled':
      return 'warning'
    case 'banned':
      return 'danger'
    default:
      return 'info'
  }
})

const tokenStatusLabel = computed(() => {
  switch (authStore.user?.token_status) {
    case 'active':
      return '活跃'
    case 'disabled':
      return '禁用'
    default:
      return authStore.user?.token_status || '未设置'
  }
})

const tokenStatusType = computed(() => {
  switch (authStore.user?.token_status) {
    case 'active':
      return 'success'
    case 'disabled':
      return 'danger'
    default:
      return 'info'
  }
})

const accessModeLabel = computed(() => (authStore.user?.unlimited_access ? '无限模式' : '受 Windsurf Credit 限制'))
const rateLimitLabel = computed(() => `${authStore.user?.rate_limit_per_minute || 30} 次/分钟`)
const usageModeHint = computed(() => {
  if (authStore.user?.unlimited_access) {
    return '该用户不会因为 Gateway 的 Windsurf credit 约束被拦截，但仍取决于当前可调度的 Backend Token。'
  }
  return '该用户会优先使用仍有 Windsurf 日/周 credit 的 Backend Token。'
})
const usageAlertTitle = computed(() => {
  if (authStore.user?.unlimited_access) {
    return '无限模式只绕过 Gateway 的 Windsurf credit 策略，不会绕过每分钟限速，也不会绕过后端 token 自身失效或冷却。'
  }
  return '当前模式下，Gateway 会优先挑选仍有 Windsurf 日限额和周限额的 Backend Token。'
})
const gatewayAuthHint = computed(() => {
  if (gatewayConfig.require_user_auth_proxy) {
    return '已开启：只有携带有效 ws 用户令牌的客户端才能通过 Gateway。'
  }
  return '未开启：知道 Gateway 地址的客户端仍可匿名接入。'
})

const loadProfile = () => {
  profileForm.username = authStore.user?.username || ''
  profileForm.email = authStore.user?.email || ''
}

const loadGatewayConfig = async () => {
  if (!authStore.isAdmin) {
    return
  }
  gatewayConfigLoading.value = true
  try {
    const res = await client.get('/system-config')
    if (res.data.code !== 200) {
      throw new Error(res.data.msg || '加载系统配置失败')
    }
    const configs = Array.isArray(res.data.data) ? res.data.data : []
    const requireAuth = configs.find((item) => item.key === 'require_user_auth_proxy')
    gatewayConfig.require_user_auth_proxy = String(requireAuth?.value || 'false').toLowerCase() === 'true'
  } catch (error) {
    ElMessage.error(error.message || '加载系统配置失败')
  } finally {
    gatewayConfigLoading.value = false
  }
}

const saveGatewayConfig = async () => {
  gatewayConfigSaving.value = true
  try {
    const res = await client.put('/system-config', {
      require_user_auth_proxy: String(gatewayConfig.require_user_auth_proxy),
    })
    if (res.data.code === 200) {
      ElMessage.success('Gateway 接入规则已保存')
    } else {
      ElMessage.error(res.data.msg || '保存失败')
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    gatewayConfigSaving.value = false
  }
}

const updateProfile = async () => {
  await profileFormRef.value?.validate(async (valid) => {
    if (!valid) return
    updating.value = true
    try {
      const endpoint = authStore.isAdmin ? '/auth/profile' : '/user/settings'
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
  if (!token) {
    ElMessage.warning('当前没有可复制的令牌')
    return
  }
  try {
    await navigator.clipboard.writeText(token)
    ElMessage.success('令牌已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

const regenerateToken = async () => {
  try {
    await ElMessageBox.confirm('重新生成令牌后，旧令牌会立即失效，确定继续吗？', '确认操作', {
      type: 'warning',
    })
    regenerating.value = true
    const res = await client.post('/user/regenerate-token')
    if (res.data.code === 200) {
      ElMessage.success('令牌已重新生成')
      await authStore.fetchUser()
      loadProfile()
    } else {
      ElMessage.error(res.data.msg || '令牌重新生成失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('令牌重新生成失败')
    }
  } finally {
    regenerating.value = false
  }
}

onMounted(() => {
  loadProfile()
  loadGatewayConfig()
})
</script>

<style scoped>
.settings-page {
  padding: 0;
}

.overview-card,
.settings-card {
  margin-bottom: 20px;
}

.overview-top {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.overview-name {
  font-size: 22px;
  font-weight: 700;
  color: #1f2a37;
}

.overview-role {
  margin-top: 6px;
  color: #6b7280;
}

.settings-tabs {
  min-height: 420px;
}

.info-panel {
  margin-bottom: 18px;
  padding: 16px 18px;
  border-radius: 14px;
  background: #f5f8ff;
  border: 1px solid #dbe6ff;
}

.info-panel strong {
  display: block;
  margin-bottom: 8px;
  color: #22314d;
}

.info-panel p {
  margin: 0;
  line-height: 1.75;
  color: #516075;
}

.compact-panel {
  margin-top: 16px;
  margin-bottom: 0;
}

.admin-panel {
  margin-top: 18px;
}

.usage-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 18px;
}

.usage-card {
  padding: 18px;
  border-radius: 16px;
  background: #fbfcfe;
  border: 1px solid #e5ebf3;
}

.usage-card span {
  display: block;
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 10px;
}

.usage-card strong {
  display: block;
  font-size: 22px;
  color: #111827;
  margin-bottom: 10px;
}

.usage-card p {
  margin: 0;
  line-height: 1.7;
  color: #64748b;
}

.gateway-card {
  display: grid;
  gap: 18px;
}

.gateway-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
  border-radius: 18px;
  border: 1px solid #e5ebf3;
  background: #fbfcfe;
}

.gateway-head h3 {
  margin: 0 0 8px;
  font-size: 20px;
  color: #1f2937;
}

.gateway-head p {
  margin: 0;
  line-height: 1.75;
  color: #5f6b7a;
}

.gateway-actions {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 960px) {
  .usage-grid {
    grid-template-columns: 1fr;
  }

  .gateway-head {
    flex-direction: column;
  }
}
</style>
