<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  StartProxy,
  StopProxy,
  GetStatus,
  GetTasks,
  GetDownloadDir,
  SetDownloadDir,
  SelectFolder,
  OpenFolder,
  IsFirstRun,
  CompleteOnboarding,
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

interface ServiceStatus {
  running: boolean
  proxyOn: boolean
  apiUrl: string
  proxyPort: number
  localIps: string[]
}

interface Task {
  id: string
  title: string
  filename: string
  size: number
  path: string
  status: string
}

interface TaskListResult {
  tasks: Task[]
  total: number
}

const status = ref<ServiceStatus>({ running: false, proxyOn: false, apiUrl: '', proxyPort: 2023, localIps: [] })
const tasks = ref<Task[]>([])
const downloadDir = ref('')
const loading = ref(false)
const logs = ref<string[]>([])
const logVisible = ref(false)
const firstRun = ref(true)
const onboardingStep = ref(0)
let pollTimer: ReturnType<typeof setInterval> | null = null

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB'
}

async function refreshStatus() {
  try { status.value = await GetStatus() }
  catch { status.value.running = false }
}

async function refreshTasks() {
  try {
    const result = await GetTasks(1, 50) as TaskListResult
    tasks.value = result?.tasks || []
  } catch { /* 服务未运行 */ }
}

async function startService() {
  loading.value = true
  try {
    await StartProxy()
    await refreshStatus()
    startPolling()
  } catch (e: any) {
    logs.value.push(`[ERROR] ${e}`)
  } finally { loading.value = false }
}

async function stopService() {
  loading.value = true
  try {
    await StopProxy()
    stopPolling()
    await refreshStatus()
  } catch (e: any) {
    logs.value.push(`[ERROR] ${e}`)
  } finally { loading.value = false }
}

async function chooseFolder() {
  try {
    const path = await SelectFolder() as string
    if (path) {
      await SetDownloadDir(path)
      downloadDir.value = path
    }
  } catch { /* 用户取消 */ }
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    await refreshStatus()
    await refreshTasks()
  }, 3000)
}

function stopPolling() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

async function finishOnboarding() {
  try { await CompleteOnboarding(); firstRun.value = false }
  catch { /* ignore */ }
}

onMounted(async () => {
  EventsOn('log:line', (line: string) => {
    logs.value.push(line)
    if (logs.value.length > 500) logs.value = logs.value.slice(-500)
  })
  EventsOn('service:status', (s: ServiceStatus) => { status.value = s })

  try {
    downloadDir.value = await GetDownloadDir() as string
    firstRun.value = await IsFirstRun() as boolean
    if (firstRun.value) onboardingStep.value = 1
  } catch { /* ignore */ }

  await refreshStatus()
  if (status.value.running) { startPolling(); await refreshTasks() }
})

onUnmounted(() => { stopPolling() })
</script>

<template>
  <!-- 引导页 -->
  <div v-if="firstRun && onboardingStep === 1" class="onboarding">
    <div class="onboarding-card">
      <h1>Scribe Desktop</h1>
      <p class="desc">视频号下载工具 — 安装即用，一键下载</p>
      <div class="step">
        <h3>选择下载目录</h3>
        <div class="folder-row">
          <input :value="downloadDir" readonly class="folder-input" />
          <button @click="chooseFolder" class="btn btn-outline">浏览</button>
        </div>
      </div>
      <button @click="finishOnboarding" class="btn btn-primary btn-lg">开始使用</button>
    </div>
  </div>

  <!-- 主面板 -->
  <div v-else class="dashboard">
    <header class="header">
      <div class="header-left">
        <h1 class="app-title">Scribe Desktop</h1>
        <span class="status-badge" :class="status.running ? 'running' : 'stopped'">
          <span class="dot"></span>
          {{ status.running ? '运行中' : '已停止' }}
        </span>
      </div>
      <div class="header-right">
        <button v-if="!status.running" @click="startService" :disabled="loading" class="btn btn-primary">
          {{ loading ? '启动中...' : '▶ 启动服务' }}
        </button>
        <button v-else @click="stopService" :disabled="loading" class="btn btn-danger">
          {{ loading ? '停止中...' : '■ 停止服务' }}
        </button>
        <button @click="logVisible = !logVisible" class="btn btn-outline btn-sm">
          {{ logVisible ? '隐藏日志' : '日志' }}
        </button>
      </div>
    </header>

    <div v-if="status.running" class="info-bar">
      <div class="info-item">
        <span class="info-label">API</span>
        <code>{{ status.apiUrl }}</code>
      </div>
      <div class="info-item">
        <span class="info-label">代理端口</span>
        <code>{{ status.proxyPort }}</code>
      </div>
      <div class="info-item">
        <span class="info-label">下载目录</span>
        <span class="folder-link" @click="OpenFolder(downloadDir)">{{ downloadDir }}</span>
      </div>
      <button @click="chooseFolder" class="btn btn-outline btn-sm">更改</button>
    </div>

    <div v-if="status.running" class="tip-card">
      <p>✅ 代理已启动！浏览器打开 <strong>channels.weixin.qq.com</strong> 即可下载。</p>
    </div>

    <div class="section">
      <h2 class="section-title">下载任务</h2>
      <div v-if="tasks.length === 0" class="empty">
        <p v-if="status.running">暂无任务。打开微信视频号试试吧！</p>
        <p v-else>请先启动服务。</p>
      </div>
      <div v-else class="task-list">
        <div v-for="task in tasks" :key="task.id" class="task-card" @click="OpenFolder(task.path)">
          <div class="task-icon">🎬</div>
          <div class="task-info">
            <div class="task-title">{{ task.title || task.filename || '未知' }}</div>
            <div class="task-meta">
              <span>{{ formatSize(task.size) }}</span>
              <span class="task-status">{{ task.status }}</span>
            </div>
          </div>
          <button class="btn btn-outline btn-sm" @click.stop="OpenFolder(task.path)">打开</button>
        </div>
      </div>
    </div>

    <div v-if="logVisible" class="log-panel">
      <div class="log-header">
        <span>日志</span>
        <button @click="logs = []" class="btn btn-outline btn-sm">清空</button>
      </div>
      <pre class="log-content">{{ logs.join('\n') || '(暂无日志)' }}</pre>
    </div>
  </div>
</template>

<style scoped>
.onboarding, .dashboard {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
  color: #1a1a2e;
  height: 100vh;
  display: flex;
  flex-direction: column;
}
.onboarding {
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.onboarding-card {
  background: white;
  border-radius: 16px;
  padding: 48px;
  max-width: 480px;
  width: 90%;
  text-align: center;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
}
.onboarding-card h1 { margin: 0 0 8px; font-size: 24px; }
.desc { color: #666; margin: 0 0 32px; }
.step { margin: 24px 0; text-align: left; }
.step h3 { margin: 0 0 8px; font-size: 16px; }

.btn {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 8px 16px; border: none; border-radius: 8px;
  font-size: 14px; font-weight: 500; cursor: pointer; transition: all 0.2s;
}
.btn:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary { background: #667eea; color: white; }
.btn-primary:hover:not(:disabled) { background: #5a6fd6; }
.btn-danger { background: #e74c3c; color: white; }
.btn-danger:hover:not(:disabled) { background: #c0392b; }
.btn-outline { background: transparent; border: 1px solid #ddd; color: #555; }
.btn-outline:hover { background: #f5f5f5; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-lg { padding: 12px 32px; font-size: 16px; margin-top: 24px; }

.header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px 24px; border-bottom: 1px solid #eee; background: white;
}
.header-left { display: flex; align-items: center; gap: 12px; }
.app-title { margin: 0; font-size: 20px; font-weight: 700; }
.header-right { display: flex; gap: 8px; align-items: center; }

.status-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;
}
.status-badge.running { background: #e8f8f0; color: #27ae60; }
.status-badge.stopped { background: #f5f5f5; color: #999; }
.dot { width: 8px; height: 8px; border-radius: 50%; }
.running .dot { background: #27ae60; animation: pulse 2s infinite; }
.stopped .dot { background: #ccc; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }

.info-bar {
  display: flex; align-items: center; gap: 16px;
  padding: 12px 24px; background: #f8f9fa; border-bottom: 1px solid #eee; flex-wrap: wrap;
}
.info-item { display: flex; flex-direction: column; gap: 2px; }
.info-label { font-size: 11px; color: #999; text-transform: uppercase; }
.info-bar code { font-size: 13px; color: #333; background: #e9ecef; padding: 2px 6px; border-radius: 4px; }
.folder-link { font-size: 13px; cursor: pointer; text-decoration: underline; }
.folder-link:hover { color: #667eea; }

.tip-card {
  margin: 16px 24px; padding: 12px 16px;
  background: #e8f8f0; border: 1px solid #c3e6cb; border-radius: 8px; font-size: 14px;
}
.tip-card p { margin: 0; }

.section { flex: 1; padding: 16px 24px; overflow-y: auto; }
.section-title { font-size: 16px; font-weight: 600; margin: 0 0 12px; }
.empty { text-align: center; padding: 40px; color: #999; }

.task-list { display: flex; flex-direction: column; gap: 8px; }
.task-card {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 16px; background: white; border: 1px solid #eee;
  border-radius: 8px; cursor: pointer; transition: all 0.15s;
}
.task-card:hover { border-color: #667eea; box-shadow: 0 2px 8px rgba(102,126,234,0.1); }
.task-icon { font-size: 24px; }
.task-info { flex: 1; min-width: 0; }
.task-title { font-weight: 500; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.task-meta { display: flex; gap: 12px; font-size: 12px; color: #999; margin-top: 2px; }
.task-status { color: #27ae60; font-weight: 500; }

.folder-row { display: flex; gap: 8px; }
.folder-input {
  flex: 1; padding: 8px 12px; border: 1px solid #ddd;
  border-radius: 6px; font-size: 14px; background: #f9f9f9;
}

.log-panel { border-top: 1px solid #eee; background: #1a1a2e; max-height: 200px; }
.log-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 12px; color: #aaa; font-size: 12px; border-bottom: 1px solid #333;
}
.log-content {
  margin: 0; padding: 8px 12px; color: #a0d0a0; font-size: 12px;
  font-family: 'Cascadia Code', 'Fira Code', monospace;
  max-height: 160px; overflow-y: auto; white-space: pre-wrap; word-break: break-all;
}
</style>
