const API_URL = 'https://application-api.radium.net.cn'
const SECRET_KEY = 'aea3d21ab344aba07c83c768c2529da9ee9bd32d20b171407e26ebdf91ec34d5'

function getClientId() {
  let id = localStorage.getItem('_cid')
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem('_cid', id)
  }
  return id
}

export function recordUsage() {
  fetch(`${API_URL}/api/application-usage`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Secret-Key': SECRET_KEY,
    },
    body: JSON.stringify({ client_id: getClientId() }),
  }).catch(() => {})
}
