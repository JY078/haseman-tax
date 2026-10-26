import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useDesignStore = defineStore('design', () => {
  const designs = ref([])
  const currentDesign = ref(null)
  const templates = ref([
    {
      id: 1,
      name: 'Instagram 快拍',
      category: 'social',
      width: 1080,
      height: 1920,
      thumbnail: 'https://via.placeholder.com/300x500/1a1a1a/ffffff?text=Instagram',
    },
    {
      id: 2,
      name: 'Facebook 帖子',
      category: 'social',
      width: 1200,
      height: 630,
      thumbnail: 'https://via.placeholder.com/300x158/404040/ffffff?text=Facebook',
    },
    {
      id: 3,
      name: '网页横幅',
      category: 'web',
      width: 728,
      height: 90,
      thumbnail: 'https://via.placeholder.com/300x37/525252/ffffff?text=Banner',
    },
    {
      id: 4,
      name: '商品卡片',
      category: 'ecommerce',
      width: 800,
      height: 800,
      thumbnail: 'https://via.placeholder.com/300x300/262626/ffffff?text=Product',
    },
    {
      id: 5,
      name: '小红书封面',
      category: 'social',
      width: 1080,
      height: 1440,
      thumbnail: 'https://via.placeholder.com/300x400/171717/ffffff?text=Xiaohongshu',
    },
    {
      id: 6,
      name: '淘宝主图',
      category: 'ecommerce',
      width: 800,
      height: 800,
      thumbnail: 'https://via.placeholder.com/300x300/333333/ffffff?text=Taobao',
    },
    {
      id: 7,
      name: '微信公众号',
      category: 'social',
      width: 900,
      height: 383,
      thumbnail: 'https://via.placeholder.com/300x127/0a0a0a/ffffff?text=WeChat',
    },
    {
      id: 8,
      name: '电商详情页',
      category: 'ecommerce',
      width: 750,
      height: 1000,
      thumbnail: 'https://via.placeholder.com/300x400/444444/ffffff?text=Detail',
    },
  ])

  const filteredTemplates = computed(() => templates.value)

  function setCurrentDesign(design) {
    currentDesign.value = design
  }

  function addDesign(design) {
    designs.value.push(design)
  }

  function updateDesign(id, updates) {
    const index = designs.value.findIndex(d => d.id === id)
    if (index !== -1) {
      designs.value[index] = { ...designs.value[index], ...updates }
    }
  }

  function deleteDesign(id) {
    designs.value = designs.value.filter(d => d.id !== id)
  }

  return {
    designs,
    currentDesign,
    templates,
    filteredTemplates,
    setCurrentDesign,
    addDesign,
    updateDesign,
    deleteDesign,
  }
})
