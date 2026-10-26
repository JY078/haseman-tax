<template>
  <footer class="bg-neutral-900 text-neutral-400">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-8">
        <div class="col-span-2 md:col-span-1">
          <div class="flex items-center gap-2 mb-4">
            <div class="w-8 h-8 bg-white rounded-lg flex items-center justify-center">
              <span class="text-neutral-900 font-bold text-xs leading-none">DH</span>
            </div>
            <span class="text-lg font-semibold text-white">DH</span>
          </div>
          <p class="text-sm">
            AI智能图像生成平台，让创意变为现实。
          </p>
        </div>

        <div>
          <h4 class="text-white font-semibold mb-4">产品</h4>
          <ul class="space-y-2 text-sm">
            <li><router-link to="/templates" class="hover:text-white transition-colors">模板库</router-link></li>
            <!-- <li><a href="#" class="hover:text-white transition-colors">定价</a></li> -->
            <li><router-link to="/features" class="hover:text-white transition-colors">功能介绍</router-link></li>
          </ul>
        </div>

        <div>
          <h4 class="text-white font-semibold mb-4">资源</h4>
          <ul class="space-y-2 text-sm">
            <li><a href="#" class="hover:text-white transition-colors">帮助文档</a></li>
            <li><a href="#" class="hover:text-white transition-colors">教程</a></li>
            <li><a href="#" class="hover:text-white transition-colors">博客</a></li>
            <li><a href="#" class="hover:text-white transition-colors">客服支持</a></li>
          </ul>
        </div>

        <div>
          <h4 class="text-white font-semibold mb-4">关于</h4>
          <ul class="space-y-2 text-sm">
            <li><a href="#" class="hover:text-white transition-colors">关于我们</a></li>
            <li><a href="#" class="hover:text-white transition-colors">加入我们</a></li>
            <li><a href="#" class="hover:text-white transition-colors">联系我们</a></li>
            <li><a href="#" class="hover:text-white transition-colors">隐私政策</a></li>
          </ul>
        </div>
      </div>

      <div class="border-t border-neutral-800 mt-12 pt-8 flex flex-col md:flex-row justify-between items-center gap-4">
        <p class="text-sm">&copy; 2026 DH. 保留所有权利.</p>
      </div>
    </div>
  </footer>
</template>
