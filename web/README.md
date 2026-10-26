# 绘梦 - AI智能图像生成平台

新一代AI智能图像生成平台，支持产品效果图、多角度商品展示、智能背景消除、图片编辑等功能。

## 技术栈

- Vue 3
- Vite
- Tailwind CSS v4
- Headless UI
- Heroicons
- Vue Router
- Pinia

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## 项目结构

```
src/
├── assets/          # 静态资源
├── components/      # 组件
│   ├── layout/      # 布局组件
│   └── ui/          # UI 组件
├── composables/     # 组合式函数
├── router/          # 路由配置
├── stores/          # Pinia 状态管理
└── views/           # 页面视图
```

## 页面说明

- **首页** (`/`) - 展示产品特色和热门模板
- **功能** (`/features`) - 平台功能介绍
- **模板库** (`/templates`) - 浏览和选择图像模板
- **AI生图** (`/generate`) - AI图像生成
- **产品效果图** (`/product-image`) - 产品图制作
- **多角度展示** (`/multi-angle`) - 多角度产品图生成
- **图片工具** (`/tools`) - 背景消除、擦除等工具
- **我的作品** (`/my-designs`) - 管理已保存的作品
