<template>
  <div class="page-container">
    <div class="page-header"><h1 class="page-title">系统设置</h1></div>
    <a-card>
      <a-tabs default-active-key="0" type="rounded">
        <a-tab-pane key="0" title="维护模式">
          <div class="tab-content">
            <a-alert style="margin-bottom:20px" type="warning">开启维护模式后，用户端（Web）将无法正常访问，管理后台不受影响。</a-alert>
            <a-form :model="maintenanceForm" layout="vertical" class="setting-form">
              <a-form-item field="maintenanceEnabled" label="维护模式">
                <a-switch v-model="maintenanceForm.maintenanceEnabled" :loading="saveLoading['maintenance_enabled']" @change="saveSetting('maintenance_enabled', maintenanceForm.maintenanceEnabled)" />
                <span class="ml-2" style="font-size:12px" :style="{ color: maintenanceForm.maintenanceEnabled ? '#f53f3f' : '#86909c' }">{{ maintenanceForm.maintenanceEnabled ? '维护模式已开启，用户端不可访问' : '维护模式已关闭，用户端正常运行' }}</span>
              </a-form-item>
              <a-form-item field="maintenanceMessage" label="维护提示消息">
                <div class="input-wrapper">
                  <a-textarea v-model="maintenanceForm.maintenanceMessage" placeholder="请输入展示给用户的维护提示内容" :auto-size="{ minRows: 3, maxRows: 6 }" allow-clear />
                  <a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['maintenance_message']" @click="saveSetting('maintenance_message', maintenanceForm.maintenanceMessage)"><template #icon><icon-save /></template></a-button>
                </div>
              </a-form-item>
            </a-form>
          </div>
        </a-tab-pane>

        <a-tab-pane key="1" title="基本设置">
          <div class="tab-content">
            <a-form :model="basicForm" layout="vertical" class="setting-form">
              <a-form-item field="accessUrl" label="访问 URL"><div class="input-wrapper"><a-input v-model="basicForm.accessUrl" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['access_url']" @click="saveSetting('access_url', basicForm.accessUrl)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="serverUrl" label="API 服务器地址"><div class="input-wrapper"><a-input v-model="basicForm.serverUrl" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['server_url']" @click="saveSetting('server_url', basicForm.serverUrl)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="siteName" label="系统名称"><div class="input-wrapper"><a-input v-model="basicForm.siteName" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['site_name']" @click="saveSetting('site_name', basicForm.siteName)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="chatKey" label="对话 Key"><div class="input-wrapper"><a-input v-model="basicForm.chatKey" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['chat_key']" @click="saveSetting('chat_key', basicForm.chatKey)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="drawKey" label="绘图 Key"><div class="input-wrapper"><a-input v-model="basicForm.drawKey" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['draw_key']" @click="saveSetting('draw_key', basicForm.drawKey)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="firecrawlApiKey" label="Firecrawl API Key"><div class="input-wrapper"><a-input-password v-model="basicForm.firecrawlApiKey" allow-clear placeholder="用于 Home Depot Listing 抓取" /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['firecrawl_api_key']" @click="saveSetting('firecrawl_api_key', basicForm.firecrawlApiKey)"><template #icon><icon-save /></template></a-button></div><div style="margin-top: 6px; color: #86909c; font-size: 12px; line-height: 1.6;">Home Depot Listing 评估会使用此 Key，通过 Internet Number 获取商品页信息。</div></a-form-item>
              <a-form-item field="firecrawlProxy" label="Firecrawl 代理地址"><div class="input-wrapper"><a-input v-model="basicForm.firecrawlProxy" allow-clear placeholder="http://10.10.65.24:10809" /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['firecrawl_proxy']" @click="saveSetting('firecrawl_proxy', basicForm.firecrawlProxy)"><template #icon><icon-save /></template></a-button></div><div style="margin-top: 6px; color: #86909c; font-size: 12px; line-height: 1.6;">仅用于后端访问 Firecrawl API。支持 HTTP、HTTPS 和 SOCKS5；留空表示直连。</div></a-form-item>
              <a-form-item field="sceneGenerationUserRunningImageLimit" label="场景图单用户运行中图片上限">
                <div class="input-wrapper">
                  <a-input-number v-model="basicForm.sceneGenerationUserRunningImageLimit" :min="1" :max="100" :step="1" style="width:200px" />
                  <a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['scene_generation_user_running_image_limit']" @click="saveSetting('scene_generation_user_running_image_limit', basicForm.sceneGenerationUserRunningImageLimit?.toString())"><template #icon><icon-save /></template></a-button>
                </div>
                <div style="margin-top: 6px; color: #86909c; font-size: 12px; line-height: 1.6;">控制单个用户在场景图任务中同时运行的图片总数。超过限制时，后台会自动等待执行，用户端无感知排队。</div>
              </a-form-item>
              <a-form-item field="videoKey" label="视频 Key"><div class="input-wrapper"><a-input v-model="basicForm.videoKey" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['video_key']" @click="saveSetting('video_key', basicForm.videoKey)"><template #icon><icon-save /></template></a-button></div></a-form-item>
            </a-form>
          </div>
        </a-tab-pane>

        <a-tab-pane key="6" title="邮件设置">
          <div class="tab-content">
            <a-form :model="smtpForm" layout="vertical" class="setting-form">
              <a-form-item field="smtpEnabled" label="启用 SMTP"><a-switch v-model="smtpForm.smtpEnabled" :loading="saveLoading['smtp_enabled']" @change="saveSetting('smtp_enabled', smtpForm.smtpEnabled)" /></a-form-item>
              <a-form-item field="smtpHost" label="SMTP 服务器"><div class="input-wrapper"><a-input v-model="smtpForm.smtpHost" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['smtp_host']" @click="saveSetting('smtp_host', smtpForm.smtpHost)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="smtpPort" label="SMTP 端口"><div class="input-wrapper"><a-input-number v-model="smtpForm.smtpPort" :min="1" :max="65535" style="width:100%" /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['smtp_port']" @click="saveSetting('smtp_port', smtpForm.smtpPort?.toString())"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="smtpUsername" label="SMTP 用户名"><div class="input-wrapper"><a-input v-model="smtpForm.smtpUsername" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['smtp_username']" @click="saveSetting('smtp_username', smtpForm.smtpUsername)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="smtpPassword" label="SMTP 密码/授权码"><div class="input-wrapper"><a-input-password v-model="smtpForm.smtpPassword" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['smtp_password']" @click="saveSetting('smtp_password', smtpForm.smtpPassword)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="smtpFromAddress" label="发件人地址"><div class="input-wrapper"><a-input v-model="smtpForm.smtpFromAddress" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['smtp_from_address']" @click="saveSetting('smtp_from_address', smtpForm.smtpFromAddress)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="smtpFromName" label="发件人名称"><div class="input-wrapper"><a-input v-model="smtpForm.smtpFromName" allow-clear /><a-button type="text" shape="circle" class="ml-2" :loading="saveLoading['smtp_from_name']" @click="saveSetting('smtp_from_name', smtpForm.smtpFromName)"><template #icon><icon-save /></template></a-button></div></a-form-item>
              <a-form-item field="smtpUseSSL" label="启用 SSL"><a-switch v-model="smtpForm.smtpUseSSL" :loading="saveLoading['smtp_use_ssl']" @change="saveSetting('smtp_use_ssl', smtpForm.smtpUseSSL)" /></a-form-item>
              <a-form-item field="emailRegisterEnabled" label="开启邮箱注册"><a-switch v-model="smtpForm.emailRegisterEnabled" :loading="saveLoading['email_register_enabled']" @change="saveSetting('email_register_enabled', smtpForm.emailRegisterEnabled)" /></a-form-item>
            </a-form>
          </div>
        </a-tab-pane>

        <a-tab-pane key="3" title="通知设置">
          <div class="tab-content">
            <a-list>
              <a-list-item><a-list-item-meta title="系统消息"><template #description>接收系统更新、维护等通知</template></a-list-item-meta><template #actions><a-switch v-model="notificationForm.system" /></template></a-list-item>
              <a-list-item><a-list-item-meta title="账户活动"><template #description>接收账户登录、密码修改等通知</template></a-list-item-meta><template #actions><a-switch v-model="notificationForm.account" /></template></a-list-item>
            </a-list>
          </div>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconSave } from '@arco-design/web-vue/es/icon'
import request from '../../utils/request'

const maintenanceForm = reactive({ maintenanceEnabled: false, maintenanceMessage: '' })
const basicForm = reactive({ accessUrl: '', serverUrl: '', siteName: '', chatKey: '', drawKey: '', firecrawlApiKey: '', firecrawlProxy: '', videoKey: '', sceneGenerationUserRunningImageLimit: 8 })
const smtpForm = reactive({ smtpEnabled: false, smtpHost: '', smtpPort: 465, smtpUsername: '', smtpPassword: '', smtpFromAddress: '', smtpFromName: '', smtpUseSSL: true, emailRegisterEnabled: false })
const saveLoading = reactive({})
const notificationForm = reactive({ system: true, account: false })

const fetchSettings = async () => {
  try {
    const res = await request.get('/settings')
    const data = res.data || {}
    if (data.maintenance_enabled) maintenanceForm.maintenanceEnabled = data.maintenance_enabled === 'true'
    if (data.maintenance_message) maintenanceForm.maintenanceMessage = data.maintenance_message
    if (data.access_url) basicForm.accessUrl = data.access_url
    if (data.server_url) basicForm.serverUrl = data.server_url
    if (data.site_name) basicForm.siteName = data.site_name
    if (data.chat_key) basicForm.chatKey = data.chat_key
    if (data.draw_key) basicForm.drawKey = data.draw_key
    if (data.firecrawl_api_key) basicForm.firecrawlApiKey = data.firecrawl_api_key
    if (data.firecrawl_proxy !== undefined) basicForm.firecrawlProxy = data.firecrawl_proxy
    if (data.video_key) basicForm.videoKey = data.video_key
    if (data.scene_generation_user_running_image_limit !== undefined) basicForm.sceneGenerationUserRunningImageLimit = parseInt(data.scene_generation_user_running_image_limit) || 8
    if (data.smtp_enabled) smtpForm.smtpEnabled = data.smtp_enabled === 'true'
    if (data.smtp_host) smtpForm.smtpHost = data.smtp_host
    if (data.smtp_port) smtpForm.smtpPort = parseInt(data.smtp_port)
    if (data.smtp_username) smtpForm.smtpUsername = data.smtp_username
    if (data.smtp_password) smtpForm.smtpPassword = data.smtp_password
    if (data.smtp_from_address) smtpForm.smtpFromAddress = data.smtp_from_address
    if (data.smtp_from_name) smtpForm.smtpFromName = data.smtp_from_name
    if (data.smtp_use_ssl) smtpForm.smtpUseSSL = data.smtp_use_ssl === 'true'
    if (data.email_register_enabled) smtpForm.emailRegisterEnabled = data.email_register_enabled === 'true'
  } catch (error) {
    console.error('Failed to fetch settings:', error)
  }
}

const saveSetting = async (key, value) => {
  saveLoading[key] = true
  try {
    let payloadValue = value
    if (Array.isArray(value)) payloadValue = JSON.stringify(value)
    else if (typeof value === 'boolean') payloadValue = value.toString()
    await request.post('/settings', { [key]: payloadValue })
    Message.success('保存成功')
  } finally {
    saveLoading[key] = false
  }
}

onMounted(fetchSettings)
</script>

<style scoped>
.page-container { padding: 24px; }
.page-header { margin-bottom: 24px; }
.tab-content { padding: 20px 0; }
.setting-form { max-width: 600px; }
.input-wrapper { display: flex; align-items: center; width: 100%; }
.ml-2 { margin-left: 8px; }
</style>
