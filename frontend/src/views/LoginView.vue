<template>
  <div class="login-container" :class="{ 'login-light': theme === 'light' }">
    <canvas ref="canvasRef" class="login-bg"></canvas>
    <div class="login-card-wrapper">
      <div class="login-logo">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="logo-icon">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          <path d="M9 12l2 2 4-4"/>
        </svg>
        <span class="login-title">Proxy</span>
      </div>
      <n-card class="login-card" :bordered="false">
        <n-form ref="formRef" :model="form" :rules="rules">
          <n-form-item label="密码" path="password">
            <n-input
              v-model:value="form.password"
              type="password"
              show-password-on="click"
              placeholder="请输入密码"
              @keyup.enter="handleSubmit"
            />
          </n-form-item>
          <n-form-item v-if="isSetup" label="确认密码" path="confirmPassword">
            <n-input
              v-model:value="form.confirmPassword"
              type="password"
              show-password-on="click"
              placeholder="再次输入密码"
              @keyup.enter="handleSubmit"
            />
          </n-form-item>
        </n-form>
        <n-button
          type="primary"
          block
          :loading="loading"
          @click="handleSubmit"
          style="margin-top: 8px"
        >
          {{ isSetup ? '设置密码并登录' : '登录' }}
        </n-button>
        <div style="text-align: center; margin-top: 12px">
          <n-button text size="small" @click="toggleTheme">
            {{ theme === 'dark' ? '☀️ 亮色模式' : '🌙 暗色模式' }}
          </n-button>
        </div>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { useSettingsStore } from '../stores/settings'

const router = useRouter()
const message = useMessage()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const theme = computed(() => settingsStore.theme)
const isSetup = ref(false)
const loading = ref(false)
const form = ref({ password: '', confirmPassword: '' })
const canvasRef = ref<HTMLCanvasElement | null>(null)

const rules = {
  password: { required: true, min: 6, message: '密码至少6个字符', trigger: 'blur' },
  confirmPassword: {
    required: true,
    validator: (_rule: any, value: string) => {
      if (value !== form.value.password) return new Error('两次密码不一致')
      return true
    },
    trigger: 'blur',
  },
}

function toggleTheme() {
  settingsStore.setTheme(theme.value === 'dark' ? 'light' : 'dark')
}

// --- Animated background ---
let animId = 0
interface Particle { x: number; y: number; vx: number; vy: number; r: number }

function initCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  let w = 0, h = 0
  let mouseX = -1000, mouseY = -1000
  const particles: Particle[] = []
  const count = 60
  const connectDist = 150
  const mouseDist = 200

  const isDark = () => settingsStore.theme === 'dark'

  function resize() {
    w = canvas!.width = window.innerWidth
    h = canvas!.height = window.innerHeight
  }

  function createParticles() {
    particles.length = 0
    for (let i = 0; i < count; i++) {
      particles.push({
        x: Math.random() * w,
        y: Math.random() * h,
        vx: (Math.random() - 0.5) * 0.5,
        vy: (Math.random() - 0.5) * 0.5,
        r: Math.random() * 1.5 + 0.5,
      })
    }
  }

  function draw() {
    const dark = isDark()
    const accent = dark ? '99, 226, 183' : '24, 160, 88'

    ctx!.clearRect(0, 0, w, h)

    for (const p of particles) {
      p.x += p.vx
      p.y += p.vy
      if (p.x < 0 || p.x > w) p.vx *= -1
      if (p.y < 0 || p.y > h) p.vy *= -1

      const dx = p.x - mouseX
      const dy = p.y - mouseY
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < mouseDist) {
        const force = (mouseDist - dist) / mouseDist * 0.02
        p.vx += dx * force
        p.vy += dy * force
      }
      p.vx *= 0.99
      p.vy *= 0.99

      ctx!.beginPath()
      ctx!.arc(p.x, p.y, p.r, 0, Math.PI * 2)
      ctx!.fillStyle = `rgba(${accent}, ${dark ? 0.5 : 0.4})`
      ctx!.fill()
    }

    ctx!.lineWidth = 0.5
    for (let i = 0; i < particles.length; i++) {
      for (let j = i + 1; j < particles.length; j++) {
        const dx = particles[i].x - particles[j].x
        const dy = particles[i].y - particles[j].y
        const dist = Math.sqrt(dx * dx + dy * dy)
        if (dist < connectDist) {
          const alpha = (1 - dist / connectDist) * (dark ? 0.15 : 0.1)
          ctx!.strokeStyle = `rgba(${accent}, ${alpha})`
          ctx!.beginPath()
          ctx!.moveTo(particles[i].x, particles[i].y)
          ctx!.lineTo(particles[j].x, particles[j].y)
          ctx!.stroke()
        }
      }
    }

    for (const p of particles) {
      const dx = p.x - mouseX
      const dy = p.y - mouseY
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < mouseDist) {
        const alpha = (1 - dist / mouseDist) * (dark ? 0.3 : 0.2)
        ctx!.strokeStyle = `rgba(${accent}, ${alpha})`
        ctx!.lineWidth = 0.8
        ctx!.beginPath()
        ctx!.moveTo(p.x, p.y)
        ctx!.lineTo(mouseX, mouseY)
        ctx!.stroke()
      }
    }

    animId = requestAnimationFrame(draw)
  }

  resize()
  createParticles()
  draw()

  const onResize = () => { resize(); createParticles() }
  const onMouseMove = (e: MouseEvent) => { mouseX = e.clientX; mouseY = e.clientY }
  const onMouseLeave = () => { mouseX = -1000; mouseY = -1000 }

  window.addEventListener('resize', onResize)
  canvas.addEventListener('mousemove', onMouseMove)
  canvas.addEventListener('mouseleave', onMouseLeave)

  return () => {
    cancelAnimationFrame(animId)
    window.removeEventListener('resize', onResize)
    canvas.removeEventListener('mousemove', onMouseMove)
    canvas.removeEventListener('mouseleave', onMouseLeave)
  }
}

let cleanup: (() => void) | undefined

onMounted(async () => {
  cleanup = initCanvas()
  await authStore.checkAuth()
  if (authStore.needsSetup) isSetup.value = true
  if (authStore.isAuthenticated) router.push('/')
})

onUnmounted(() => { cleanup?.() })

async function handleSubmit() {
  if (!form.value.password) { message.error('请输入密码'); return }
  loading.value = true
  try {
    if (isSetup.value) {
      if (form.value.password !== form.value.confirmPassword) { message.error('两次密码不一致'); return }
      await authStore.setup(form.value.password)
      message.success('密码设置成功')
    } else {
      await authStore.login(form.value.password)
      message.success('登录成功')
    }
    router.push('/')
  } catch (err: any) {
    message.error(err.response?.data?.error || '操作失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  position: relative;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: #181818;
  transition: background 0.3s;
}
.login-light {
  background: #f5f5f5;
}
.login-bg {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
}
.login-card-wrapper {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
}
.login-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.logo-icon {
  color: #63e2b7;
}
.login-light .logo-icon {
  color: #18a058;
}
.login-title {
  font-size: 28px;
  font-weight: 700;
  color: #e0e0e0;
  letter-spacing: 2px;
}
.login-light .login-title {
  color: #333;
}
.login-card {
  width: 400px;
  backdrop-filter: blur(12px);
  background: rgba(30, 30, 30, 0.75) !important;
}
.login-light .login-card {
  background: rgba(255, 255, 255, 0.85) !important;
}
</style>
