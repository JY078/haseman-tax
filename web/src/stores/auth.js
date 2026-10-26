import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import request from '@/utils/request'

const IDLE_TIMEOUT    = 30 * 60 * 1000  // 30 minutes → auto-logout
const HEARTBEAT_MS    = 30 * 1000       // heartbeat every 30 s
const ACTIVITY_EVENTS = ['mousemove', 'mousedown', 'keydown', 'scroll', 'touchstart']

export const useAuthStore = defineStore('auth', () => {
  const user  = ref(null)
  const token = ref(localStorage.getItem('token') || null)

  const isLoggedIn = computed(() => !!token.value)

  // ── timers ──────────────────────────────────────────────────────────────
  let idleTimer      = null
  let heartbeatTimer = null
  let lastActivityAt = Date.now()

  function _idleExpired() {
    return !user.value?.long_login && Date.now() - lastActivityAt >= IDLE_TIMEOUT
  }

  function _resetIdleTimer() {
    if (!token.value) return
    if (_idleExpired()) {
      _idleLogout()
      return
    }
    lastActivityAt = Date.now()
    if (idleTimer) clearTimeout(idleTimer)
    if (user.value?.long_login) return
    idleTimer = setTimeout(_idleLogout, IDLE_TIMEOUT)
  }

  async function _idleLogout() {
    await logout()
    // Redirect to home; works even if router is not available here
    if (typeof window !== 'undefined') {
      window.location.href = '/'
    }
  }

  async function _sendHeartbeat() {
    if (!token.value) return
    if (_idleExpired()) {
      await _idleLogout()
      return
    }
    const checkedToken = token.value
    try {
      await request.post('/heartbeat')
      // Heartbeats verify the session; only user activity resets the idle timer.
    } catch (error) {
      if (error.response?.status === 401 && token.value === checkedToken) {
        _clearLogin()
      }
      // A network failure does not prove the login has expired.
    }
  }

  function _checkOnVisible() {
    if (document.visibilityState === 'visible') _sendHeartbeat()
  }

  function _startTracking() {
    // Idle detection
    lastActivityAt = Date.now()
    _resetIdleTimer()
    ACTIVITY_EVENTS.forEach(e => window.addEventListener(e, _resetIdleTimer, { passive: true }))
    document.addEventListener('visibilitychange', _checkOnVisible)
    // Heartbeat
    if (!heartbeatTimer) {
      _sendHeartbeat()
      heartbeatTimer = setInterval(_sendHeartbeat, HEARTBEAT_MS)
    }
  }

  function _stopTracking() {
    if (idleTimer)      { clearTimeout(idleTimer);   idleTimer = null }
    if (heartbeatTimer) { clearInterval(heartbeatTimer); heartbeatTimer = null }
    ACTIVITY_EVENTS.forEach(e => window.removeEventListener(e, _resetIdleTimer))
    document.removeEventListener('visibilitychange', _checkOnVisible)
  }

  // ── public API ───────────────────────────────────────────────────────────
  function login(userData, authToken) {
    user.value  = userData
    token.value = authToken
    localStorage.setItem('token', authToken)
    localStorage.setItem('user', JSON.stringify(userData))
    _startTracking()
  }

  async function logout() {
    const logoutToken = token.value
    _clearLogin()
    if (logoutToken) {
      try {
        await request.post('/logout', {}, { headers: { Authorization: 'Bearer ' + logoutToken } })
      } catch { /* ignore */ }
    }
  }

  function _clearLogin() {
    _stopTracking()
    user.value  = null
    token.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  function init() {
    const savedToken = localStorage.getItem('token')
    const savedUser  = localStorage.getItem('user')
    if (savedToken && savedUser) {
      token.value = savedToken
      try { user.value = JSON.parse(savedUser) } catch { /* ignore */ }
      _startTracking()
    }
  }

  return { user, token, isLoggedIn, login, logout, init }
})
