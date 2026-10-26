<template>
  <MainLayout>
    <div class="stp-white-bg workflow-editor">
      <input ref="fileInput" type="file" accept=".stp,.step,.obj,.fbx" class="hidden-input" @change="onFileSelect">
      <input ref="sceneFileInput" type="file" accept="image/*" class="hidden-input" @change="onSceneFileSelect">
      <input ref="generateFileInput" type="file" accept="image/*" class="hidden-input" @change="onGenerateFileSelect">

      <div v-if="!activeProjectId" class="project-create-page">
        <div class="project-create-card">
          <span class="project-create-card__eyebrow">专业模式</span>
          <h1>创建工作流项目</h1>
          <p>在画布里自由组合 3D、图片、控制图和生图节点，把不同输入串成完整的产品出图流程。</p>
          <div class="project-create-form">
            <input v-model.trim="newProjectName" type="text" placeholder="例如：浴室龙头场景图流程">
            <button type="button" @click="createProject">创建项目</button>
          </div>
          <div v-if="projects.length" class="project-create-list">
            <span>继续已有项目</span>
            <button
              v-for="project in projects"
              :key="project.id"
              type="button"
              @click="selectProject(project.id)"
            >
              {{ project.name }}
            </button>
          </div>
        </div>
      </div>

      <aside v-if="activeProjectId" class="project-sidebar">
        <div class="project-sidebar__header">
          <span>项目</span>
          <button type="button" title="新建项目" @click="openCreateProjectModal">+</button>
        </div>
        <div
          v-for="project in projects"
          :key="project.id"
          class="project-item"
          :class="{ active: project.id === activeProjectId }"
          @click="selectProject(project.id)"
        >
          <div class="project-item__text">
            <input
              v-if="renamingProjectId === project.id"
              v-model.trim="renameProjectName"
              type="text"
              class="project-rename-input"
              @click.stop
              @keydown.enter.prevent="confirmRenameProject(project.id)"
              @keydown.esc.prevent="cancelRenameProject"
              @blur="confirmRenameProject(project.id)"
            >
            <strong v-else @dblclick.stop="startRenameProject(project)">{{ project.name }}</strong>
          </div>
          <button type="button" class="project-item__delete" title="删除项目" @click.stop="deleteProject(project.id)">×</button>
        </div>
      </aside>

      <div v-if="showProjectNameModal" class="project-name-modal" @pointerdown.self="closeProjectNameModal">
        <div class="project-name-modal__card">
          <h3>新建项目</h3>
          <p>输入一个项目名称，创建后会进入画布。</p>
          <input
            ref="projectNameModalInput"
            v-model.trim="projectNameDraft"
            type="text"
            placeholder="例如：浴室龙头场景图流程"
            @keydown.enter.prevent="confirmCreateProject"
            @keydown.esc.prevent="closeProjectNameModal"
          >
          <div class="project-name-modal__actions">
            <button type="button" @click="closeProjectNameModal">取消</button>
            <button type="button" class="primary" @click="confirmCreateProject">创建</button>
          </div>
        </div>
      </div>

      <div v-if="deleteProjectTarget" class="project-name-modal" @pointerdown.self="closeDeleteProjectModal">
        <div class="project-name-modal__card project-delete-card">
          <h3>删除项目</h3>
          <p>确定删除“{{ deleteProjectTarget.name }}”吗？删除后项目内的节点和连线将无法恢复。</p>
          <div class="project-name-modal__actions">
            <button type="button" @click="closeDeleteProjectModal">取消</button>
            <button type="button" class="danger" @click="confirmDeleteProject">删除</button>
          </div>
        </div>
      </div>

      <div
        v-if="activeProjectId"
        ref="workflowCanvasRef"
        class="node-canvas"
        :class="{ 'node-canvas--ready': canvasReady, 'node-canvas--linking': !!draggingLink || !!pendingLinkFrom, 'node-canvas--batch-selecting': batchDownloadSelectMode }"
        @pointerdown="startCanvasPan"
        @pointermove="onCanvasPointerMove"
        @pointerup="onCanvasPointerUp"
        @pointercancel="onCanvasPointerCancel"
        @dragover.prevent
        @drop.prevent="onWorkflowCanvasDrop"
        @wheel="onCanvasWheel"
      >
        <div ref="workflowStageRef" class="node-canvas__stage">
        <div
          v-for="group in groupBoxes"
          :key="group.id"
          class="node-group-box"
          :data-workflow-group="group.id"
          :class="{ active: activeGroupId === group.id, running: groupRunState(group.id).running, failed: !!groupRunState(group.id).error }"
          :style="groupBoxStyle(group)"
          :title="groupRunState(group.id).error || ''"
          @pointerdown.stop.prevent="startGroupDrag($event, group.id)"
        >
          <div v-if="activeGroupId === group.id || groupRunState(group.id).running" class="node-group-toolbar" @pointerdown.stop @click.stop>
            <span v-if="groupRunState(group.id).error" class="node-group-toolbar__error" :title="groupRunState(group.id).error">运行已暂停</span>
            <button type="button" class="node-group-toolbar__run" :disabled="groupRunState(group.id).running" @click.stop="runGroupWorkflow(group.id)"><PlayIcon aria-hidden="true" />{{ groupRunState(group.id).paused ? '继续' : '运行' }}</button>
            <button v-if="groupRunState(group.id).paused && !groupRunState(group.id).running" type="button" class="node-group-toolbar__restart" @click.stop="restartGroupWorkflow(group.id)"><ArrowPathIcon aria-hidden="true" />重新开始</button>
            <button type="button" class="node-group-toolbar__pause" :disabled="!groupRunState(group.id).running || groupRunState(group.id).paused" @click.stop="pauseGroupWorkflow(group.id)"><span aria-hidden="true"></span>{{ groupRunState(group.id).paused ? '暂停中' : '暂停' }}</button>
            <template v-if="!workflowGroupIsPristine(group)">
              <button type="button" :disabled="groupRunState(group.id).running" @click.stop="ungroupNodes(group.id)">解除组合</button>
              <button type="button" class="node-group-toolbar__save" :disabled="groupRunState(group.id).running" @click.stop="saveGroupAsWorkflow(group.id)">保存为工作流</button>
            </template>
          </div>
        </div>
        <div
          v-if="selectionBoxVisible"
          class="node-selection-box"
          :class="{ fixed: selectionBox.fixed }"
          :style="selectionBoxStyle()"
          @pointerdown.stop.prevent="startSelectionMove($event)"
        >
          <div v-if="selectionBox.fixed && selectedNodeIds.length > 1" class="node-selection-toolbar" @pointerdown.stop @click.stop>
            <button type="button" @click.stop="groupSelectedNodes">组合节点</button>
          </div>
        </div>
        <svg class="node-links" viewBox="0 0 6000 4000" preserveAspectRatio="none" aria-hidden="true">
          <defs>
            <marker id="workflow-link-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto" markerUnits="strokeWidth">
              <path d="M 0 0 L 8 4 L 0 8 Z" fill="#94a3b8" />
            </marker>
            <marker id="workflow-link-arrow-active" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto" markerUnits="strokeWidth">
              <path d="M 0 0 L 8 4 L 0 8 Z" fill="#60a5fa" />
            </marker>
          </defs>
          <g
            v-for="link in workflowLinkViews"
            :key="link.key"
            class="node-link"
            :class="{ selected: activeWorkflowLink === link.key, transmitting: transmittingWorkflowLinkKeys.has(link.key) }"
            @pointerdown.stop.prevent="selectWorkflowLink(link)"
          >
            <path class="node-link__hit" :d="link.path" />
            <path
              class="node-link__line"
              :d="link.path"
              :marker-end="transmittingWorkflowLinkKeys.has(link.key) || activeWorkflowLink === link.key ? 'url(#workflow-link-arrow-active)' : 'url(#workflow-link-arrow)'"
            />
          </g>
          <path
            v-if="draggingLink"
            ref="draftLinkPathRef"
            class="node-links__draft"
            :d="draftLinkPath()"
          />
        </svg>

        <div v-if="isWorkflowEmpty()" class="empty-canvas-hint">
          <strong>空白工作流</strong>
          <span>点击底部 + 创建第一个节点</span>
        </div>

        <section
          v-for="modelNode in visibleModelWorkflowNodes()"
          :key="modelNode.id"
          class="canvas-node node-upload-3d"
          :data-workflow-node="modelNode.id"
          :class="{ selected: activeWorkflowNode === modelNode.id, done: modelNodeState(modelNode.id).modelLoaded, 'link-source': pendingLinkFrom === modelNode.id, 'link-candidate': isWorkflowLinkCandidate(pendingLinkFrom, modelNode.id), 'link-target': hoveredLinkTarget === modelNode.id, 'multi-selected': selectedNodeIds.includes(modelNode.id) }"
          :style="nodeStyle(modelNode.id)"
          @click="onWorkflowNodeClick($event, modelNode.id)"
          @pointerdown="startNodeDrag($event, modelNode.id)"
        >
          <span v-if="batchDownloadSelectMode && selectedNodeIds.includes(modelNode.id)" class="batch-selection-order">{{ batchDownloadSelectionOrder(modelNode.id) }}</span>
          <div class="node-label">3D 文件 / 预览节点</div>
          <div class="node-preview-shell">
            <div v-if="!modelNodeState(modelNode.id).modelLoaded && !modelNodeState(modelNode.id).loading && !modelNodeState(modelNode.id).stpFileUrl" class="node-empty-preview">
              <div class="model-file-icon" aria-hidden="true">
                <svg viewBox="0 0 24 24">
                  <path d="M12 3 5 7l7 4 7-4-7-4Z" />
                  <path d="M5 7v8l7 4 7-4V7" />
                  <path d="M12 11v8" />
                </svg>
              </div>
              <h3>上传 3D 文件</h3>
              <p>支持 STP / STEP / OBJ / FBX</p>
            </div>
            <div v-if="!modelNodeState(modelNode.id).modelLoaded && !modelNodeState(modelNode.id).loading && modelNodeState(modelNode.id).stpFileUrl" class="model-node-summary model-node-summary--pending">
              <div class="model-file-icon" aria-hidden="true">
                <svg viewBox="0 0 24 24">
                  <path d="M12 3 5 7l7 4 7-4-7-4Z" />
                  <path d="M5 7v8l7 4 7-4V7" />
                  <path d="M12 11v8" />
                </svg>
              </div>
              <strong>{{ modelNodeState(modelNode.id).fileName || '3D 文件' }}</strong>
              <span>点击控制角度加载模型</span>
            </div>
            <div v-if="modelNodeState(modelNode.id).loading" class="node-loading">
              <span class="loading-ring-sm"></span>
              {{ modelNodeState(modelNode.id).loadingMessage }}
            </div>
            <div
              :ref="el => setModelViewerRef(modelNode.id, el)"
              class="viewer-container"
              :class="{ visible: modelNodeViewerVisible(modelNode.id) }"
              title="点击右上角按钮预览 / 控制角度"
            ></div>
            <div
              v-if="modelNodeState(modelNode.id).modelLoaded && activeModelNodeId !== modelNode.id"
              class="model-node-summary"
            >
              <div class="model-file-icon" aria-hidden="true">
                <svg viewBox="0 0 24 24">
                  <path d="M12 3 5 7l7 4 7-4-7-4Z" />
                  <path d="M5 7v8l7 4 7-4V7" />
                  <path d="M12 11v8" />
                </svg>
              </div>
              <strong>{{ modelNodeState(modelNode.id).fileName || '3D 文件' }}</strong>
              <span>{{ modelNodeState(modelNode.id).modelLoaded ? '已上传，可以控制角度' : '已上传，点击控制角度加载' }}</span>
            </div>
            <div class="model-node-footer" @pointerdown.stop @click.stop>
              <div class="model-node-file">
                <strong>{{ modelNodeState(modelNode.id).fileName || '未上传文件' }}</strong>
                <span>{{ modelNodeState(modelNode.id).modelLoaded ? '3D 文件已就绪' : (modelNodeState(modelNode.id).stpFileUrl ? '点击控制角度加载模型' : '选择文件后可控制角度') }}</span>
              </div>
              <div class="model-node-actions">
                <button type="button" class="model-node-action" @click.stop="openModelUpload(modelNode.id)">
                  {{ modelNodeState(modelNode.id).modelLoaded ? '更换' : '上传' }}
                </button>
                <button type="button" class="model-node-action" @click.stop="openAssetPicker('model', modelNode.id)">
                  选模型
                </button>
                <button
                  v-if="modelNodeState(modelNode.id).modelLoaded || modelNodeState(modelNode.id).loading || modelNodeState(modelNode.id).error || modelNodeState(modelNode.id).stpFileUrl"
                  type="button"
                  class="model-node-action"
                  @click.stop="openModelPreview(modelNode.id)"
                >
                  控制角度
                </button>
                <button
                  v-if="modelNodeState(modelNode.id).modelLoaded"
                  type="button"
                  class="model-node-action model-node-action--primary"
                  :disabled="generating"
                  @click.stop="generateModelWorkflow(modelNode.id)"
                >
                  {{ modelGenerateButtonText(modelNode.id) }}
                </button>
              </div>
            </div>
          </div>
          <button
            type="button"
            class="node-port node-port--out"
            :class="{ active: pendingLinkFrom === modelNode.id }"
            :data-node="modelNode.id"
            data-port="out"
            title="从这里连接到下一个节点"
            @pointerdown.stop.prevent="startLinkDrag($event, modelNode.id)"
            @click.stop="handleNodePortClick(modelNode.id, 'out')"
          ></button>
        </section>

        <section
          v-if="workflowNodes.output && shouldRenderWorkflowNode('output')"
          class="canvas-node node-control-output"
          data-workflow-node="output"
          :class="{ selected: activeWorkflowNode === 'output', 'multi-selected': selectedNodeIds.includes('output'), done: resultUrls.white && resultUrls.depth }"
          :style="nodeStyle('output')"
          @click="onWorkflowNodeClick($event, 'output')"
          @pointerdown="startNodeDrag($event, 'output')"
        >
          <span v-if="batchDownloadSelectMode && selectedNodeIds.includes('output')" class="batch-selection-order">{{ batchDownloadSelectionOrder('output') }}</span>
          <div class="node-label">控制图输出节点</div>
          <div v-if="resultUrls.white || resultUrls.depth" class="control-output-grid">
            <button v-if="resultUrls.white" type="button" @click.stop="openPreview(resultUrls.white, '白底结构图')">
              <img v-lazy-src="thumbnailImage(resultUrls.white, 520)" alt="白底结构图">
              <span>白底结构图</span>
            </button>
            <button v-if="resultUrls.depth" type="button" @click.stop="openPreview(resultUrls.depth, '深度图')">
              <img v-lazy-src="thumbnailImage(resultUrls.depth, 520)" alt="深度图">
              <span>深度图</span>
            </button>
          </div>
          <div v-else class="node-body node-body--center">
            <div class="node-icon">OUT</div>
            <p>基于预览节点当前角度生成白底结构图和深度图。</p>
          </div>
          <button
            type="button"
            class="node-port node-port--in"
            data-node="output"
            data-port="in"
            title="连接到这个节点"
            @pointerdown.stop
            @click.stop="handleNodePortClick('output', 'in')"
          ></button>
          <button
            type="button"
            class="node-port node-port--out"
            :class="{ active: pendingLinkFrom === 'output' }"
            data-node="output"
            data-port="out"
            title="从这里连接到下一个节点"
            @pointerdown.stop.prevent="startLinkDrag($event, 'output')"
            @click.stop="handleNodePortClick('output', 'out')"
          ></button>
        </section>

        <section
          v-if="workflowNodes.scene && shouldRenderWorkflowNode('scene')"
          class="canvas-node node-scene-input"
          data-workflow-node="scene"
          :class="{ selected: activeWorkflowNode === 'scene', 'multi-selected': selectedNodeIds.includes('scene'), done: sceneImage.url }"
          :style="nodeStyle('scene')"
          @click="onWorkflowNodeClick($event, 'scene')"
          @pointerdown="startNodeDrag($event, 'scene')"
          @dragover.prevent
          @drop.prevent="onSceneDrop"
        >
          <span v-if="batchDownloadSelectMode && selectedNodeIds.includes('scene')" class="batch-selection-order">{{ batchDownloadSelectionOrder('scene') }}</span>
          <div class="node-label">场景图节点</div>
          <div v-if="sceneImage.url" class="scene-node-preview" role="button" tabindex="0" @click.stop="openWorkflowImageAsEditableNode('scene', sceneImage.url, sceneImage.name || '场景图', 'scene-input')" @keydown.enter.stop.prevent="openWorkflowImageAsEditableNode('scene', sceneImage.url, sceneImage.name || '场景图', 'scene-input')">
            <img v-lazy-src="thumbnailImage(sceneImage.url, 520)" alt="场景图">
            <p>{{ sceneImage.name }}</p>
          </div>
          <div v-else class="node-body node-body--center">
            <div class="node-icon">IMG</div>
            <p>上传浴室、台面或空间场景图。</p>
          </div>
          <button type="button" class="node-primary-btn" @click.stop="sceneFileInput?.click()">
            {{ sceneImage.url ? '更换场景图' : '上传场景图' }}
          </button>
          <button
            type="button"
            class="node-port node-port--out"
            :class="{ active: pendingLinkFrom === 'scene' }"
            data-node="scene"
            data-port="out"
            title="从这里连接到下一个节点"
            @pointerdown.stop.prevent="startLinkDrag($event, 'scene')"
            @click.stop="handleNodePortClick('scene', 'out')"
          ></button>
        </section>

        <section
          v-if="workflowNodes.final && shouldRenderWorkflowNode('final')"
          class="canvas-node node-final-output"
          data-workflow-node="final"
          :class="{ selected: activeWorkflowNode === 'final', 'multi-selected': selectedNodeIds.includes('final'), done: sceneImage.url && resultUrls.white }"
          :style="nodeStyle('final')"
          @click="onWorkflowNodeClick($event, 'final')"
          @pointerdown="startNodeDrag($event, 'final')"
        >
          <span v-if="batchDownloadSelectMode && selectedNodeIds.includes('final')" class="batch-selection-order">{{ batchDownloadSelectionOrder('final') }}</span>
          <div class="node-label">最终场景输出节点</div>
          <div v-if="sceneImage.url" ref="sceneStageRef" class="final-stage">
            <img v-lazy-src="thumbnailImage(sceneImage.url, 720)" class="scene-base-image" alt="场景图">
            <div
              v-if="resultUrls.white"
              class="scene-product-layer"
              :style="sceneProductStyle()"
              @pointerdown.stop.prevent="startPlacementDrag"
              @pointermove.stop.prevent="dragPlacement"
              @pointerup.stop.prevent="endPlacementDrag"
              @pointercancel.stop.prevent="endPlacementDrag"
            >
              <img v-lazy-src="thumbnailImage(resultUrls.white, 520)" alt="产品白底结构图">
              <span class="placement-handle placement-handle--tl"></span>
              <span class="placement-handle placement-handle--tr"></span>
              <span class="placement-handle placement-handle--bl"></span>
              <span class="placement-handle placement-handle--br"></span>
            </div>
          </div>
          <div v-else class="node-body node-body--center">
            <div class="node-icon">GEN</div>
            <p>连接场景图和控制图后，在这里做最终输出。</p>
          </div>
          <div class="placement-controls compact">
            <label><span>X</span><input v-model.number="placement.x" type="range" min="0" max="100" step="0.5" @input="clampPlacement"></label>
            <label><span>Y</span><input v-model.number="placement.y" type="range" min="0" max="100" step="0.5" @input="clampPlacement"></label>
            <label><span>缩放</span><input v-model.number="placement.scale" type="range" min="0.08" max="1.8" step="0.01" @input="clampPlacement"></label>
          </div>
          <button type="button" class="node-primary-btn" :disabled="!sceneImage.url || !resultUrls.white" @click.stop="renderSceneComposite">
            预览合成
          </button>
          <button
            type="button"
            class="node-port node-port--in"
            data-node="final"
            data-port="in"
            title="连接到这个节点"
            @pointerdown.stop
            @click.stop="handleNodePortClick('final', 'in')"
          ></button>
        </section>

        <section
          v-for="workflowNode in visibleNonGenerateDynamicNodes()"
          :key="workflowNode.id"
          class="canvas-node"
          :data-workflow-node="workflowNode.id"
          :class="{
            selected: activeWorkflowNode === workflowNode.id,
            done: workflowNode.type === 'video' && videoNodeState(workflowNode.id).videoUrl,
            'link-source': pendingLinkFrom === workflowNode.id,
            'link-candidate': isWorkflowLinkCandidate(pendingLinkFrom, workflowNode.id),
            'link-target': hoveredLinkTarget === workflowNode.id,
            'multi-selected': selectedNodeIds.includes(workflowNode.id),
            'node-generic-workflow': workflowNode.type !== 'video',
            'node-image-generate': workflowNode.type === 'video',
            'node-video-workflow': workflowNode.type === 'video',
            'node-text-workflow': workflowNode.type === 'text',
            'node-llm-workflow': workflowNode.type === 'llm'
          }"
          :style="nodeStyle(workflowNode.id)"
          @click="onWorkflowNodeClick($event, workflowNode.id)"
          @pointerdown="startNodeDrag($event, workflowNode.id)"
        >
          <span v-if="batchDownloadSelectMode && selectedNodeIds.includes(workflowNode.id)" class="batch-selection-order">{{ batchDownloadSelectionOrder(workflowNode.id) }}</span>
          <div class="node-label">{{ dynamicNodeTitle(workflowNode) }}</div>
          <div
            v-if="workflowNode.type === 'video' && videoNodeState(workflowNode.id).videoUrl && activeWorkflowNode === workflowNode.id"
            class="image-node-toolbar video-node-toolbar"
            @pointerdown.stop
            @click.stop
          >
            <button type="button" title="下载" aria-label="下载" @click.stop="downloadVideoNode(workflowNode.id)">
              <ArrowDownTrayIcon class="toolbar-icon" aria-hidden="true" />
            </button>
          </div>
          <div v-if="workflowNode.type === 'output' && outputNodeDone(workflowNode.id)" class="control-output-grid">
            <button v-if="outputNodeResultUrls(workflowNode.id).white" type="button" @click.stop="openPreview(outputNodeResultUrls(workflowNode.id).white, '白底结构图')">
              <img v-lazy-src="thumbnailImage(outputNodeResultUrls(workflowNode.id).white, 520)" alt="白底结构图">
              <span>白底结构图</span>
            </button>
            <button v-if="outputNodeResultUrls(workflowNode.id).depth" type="button" @click.stop="openPreview(outputNodeResultUrls(workflowNode.id).depth, '深度图')">
              <img v-lazy-src="thumbnailImage(outputNodeResultUrls(workflowNode.id).depth, 520)" alt="深度图">
              <span>深度图</span>
            </button>
          </div>
          <div v-else class="node-body node-body--center">
            <template v-if="workflowNode.type === 'llm'">
              <textarea
                v-if="!llmNodeState(workflowNode.id).loading"
                :value="llmNodeState(workflowNode.id).prompt"
                class="llm-node-textarea"
                :data-llm-node="workflowNode.id"
                :class="{ loading: llmNodeState(workflowNode.id).loading }"
                :placeholder="llmNodeState(workflowNode.id).loading ? '正在生成...' : '输入内容'"
                :disabled="llmNodeState(workflowNode.id).loading"
                :readonly="!llmNodeState(workflowNode.id).editing"
                title="双击编辑文本"
                @input="onLlmNodeInput(workflowNode.id, $event.target.value)"
                @dblclick.stop="enableLlmNodeEdit(workflowNode.id, $event)"
                @keydown="handleLlmNodeKeydown(workflowNode.id, $event)"
                @keyup="llmNodeState(workflowNode.id).editing && $event.stopPropagation()"
                @blur="disableLlmNodeEdit(workflowNode.id)"
              ></textarea>
              <div v-else class="text-node-loading-panel llm-node-loading-panel">
                <span class="loading-ring-sm"></span>
                <strong>{{ llmNodeState(workflowNode.id).analysisLabel || '模型处理中' }}</strong>
                <em>{{ llmNodeState(workflowNode.id).progress || 0 }}%</em>
                <div class="text-node-progress-track" aria-hidden="true">
                  <span :style="{ width: `${Math.min(100, Math.max(0, llmNodeState(workflowNode.id).progress || 0))}%` }"></span>
                </div>
              </div>
              <span v-if="!llmNodeState(workflowNode.id).loading && llmNodeState(workflowNode.id).error" class="llm-node-run-state error" :title="llmNodeState(workflowNode.id).error">{{ llmNodeState(workflowNode.id).error }}</span>
              <button
                v-if="!llmNodeState(workflowNode.id).loading"
                type="button"
                class="llm-node-run"
                :class="{ 'has-resize': !!llmNodeState(workflowNode.id).output }"
                :disabled="!effectiveLlmPrompt(workflowNode.id)"
                title="运行大模型"
                aria-label="运行大模型"
                @pointerdown.stop
                @dblclick.stop
                @click.stop="runLlmNode(workflowNode.id)"
              ><PlayIcon aria-hidden="true" /></button>
              <button
                v-if="!llmNodeState(workflowNode.id).loading && llmNodeState(workflowNode.id).output"
                type="button"
                class="text-node-resize llm-node-resize"
                title="调整大小"
                aria-label="调整大模型节点大小"
                @pointerdown.stop.prevent="startTextNodeResize($event, workflowNode.id)"
              ></button>
            </template>
            <template v-else-if="workflowNode.type === 'video'">
              <div class="generate-card">
                <div v-if="videoNodeState(workflowNode.id).loading" class="generate-node__loading">
                  <div class="generate-wave" aria-hidden="true"></div>
                  <div class="generate-progress-panel">
                    <button type="button" class="generate-progress-pill" @click.stop>
                      <strong>生成中 {{ videoNodeState(workflowNode.id).progress || 0 }}%</strong>
                      <span>视频</span>
                    </button>
                    <div class="generate-progress-track" aria-hidden="true">
                      <span :style="{ width: `${Math.min(100, Math.max(0, videoNodeState(workflowNode.id).progress || 0))}%` }"></span>
                    </div>
                  </div>
                </div>
                <div v-else-if="videoNodeState(workflowNode.id).error" class="generate-node__error">
                  <strong>生成失败</strong>
                  <span>{{ videoNodeState(workflowNode.id).error }}</span>
                </div>
                <div v-else-if="videoNodeState(workflowNode.id).videoUrl" class="generate-node__preview">
                  <div class="generate-node__image-button generate-node__image-button--video">
                    <video
                      :src="resolveImageUrl(videoNodeState(workflowNode.id).videoUrl)"
                      muted
                      playsinline
                      preload="metadata"
                      @loadedmetadata="rememberVideoNodeSize(workflowNode.id, $event)"
                    ></video>
                    <button
                      type="button"
                      class="video-node-play-button"
                      aria-label="播放视频"
                      title="播放视频"
                      @pointerdown.stop
                      @click.stop="onVideoPreviewClick(videoNodeState(workflowNode.id).videoUrl, '视频生成结果')"
                    >
                      <PlayIcon class="video-node-play-icon" aria-hidden="true" />
                    </button>
                  </div>
                </div>
                <div v-else class="generate-node__placeholder">
                  <div class="generate-node__placeholder-icon" aria-hidden="true">
                    <svg viewBox="0 0 24 24">
                      <rect x="4" y="6" width="12" height="12" rx="2"></rect>
                      <path d="m16 10 4-2.4v8.8L16 14"></path>
                    </svg>
                  </div>
                </div>
              </div>

              <div
                v-if="activeWorkflowNode === workflowNode.id"
                class="generate-editor-panel"
                :class="{ 'has-refs': videoEditorReferences(workflowNode.id).length }"
                @pointerdown.stop
                @click.stop
              >
                <div class="generate-input-summary">
                  <span>输入</span>
                  <strong>{{ videoReferenceSummary(workflowNode.id) }}</strong>
                </div>

                <div v-if="videoEditorReferences(workflowNode.id).length" class="generate-node__refs video-node__refs">
                  <div
                    v-for="(item, referenceIndex) in videoEditorReferences(workflowNode.id)"
                    :key="item.key"
                    class="video-ref-item"
                  >
                    <component
                      :is="item.type === 'text' ? 'div' : 'button'"
                      :type="item.type === 'text' ? undefined : 'button'"
                      class="generate-ref"
                      :class="{ 'generate-ref--text': item.type === 'text' }"
                      :title="item.type === 'text' ? item.text : item.label"
                      @click.stop="item.type === 'text' ? null : openPreview(item.url, item.label)"
                    >
                      <template v-if="item.type === 'text'">
                        <span class="generate-ref-text-icon" aria-hidden="true"></span>
                      </template>
                      <img v-else v-lazy-src="thumbnailImage(item.url, 260)" :alt="item.label">
                    </component>
                    <span v-if="item.type !== 'text'" class="video-ref-index">{{ referenceIndex + 1 }}</span>
                  </div>
                </div>

                <div class="video-media-inputs">
                  <button type="button" :class="{ active: videoNodeState(workflowNode.id).inputMode === 'reference' }" @click="updateVideoNode(workflowNode.id, { inputMode: 'reference' })">多参考图</button>
                  <button type="button" :class="{ active: videoNodeState(workflowNode.id).inputMode === 'frames' }" @click="updateVideoNode(workflowNode.id, { inputMode: 'frames' })">首尾帧</button>
                  <label v-if="videoNodeState(workflowNode.id).inputMode === 'frames'">首帧<input type="file" accept="image/*" @change="uploadVideoNodeMedia(workflowNode.id, 'firstFrame', $event)"></label>
                  <label v-if="videoNodeState(workflowNode.id).inputMode === 'frames'">尾帧<input type="file" accept="image/*" @change="uploadVideoNodeMedia(workflowNode.id, 'lastFrame', $event)"></label>
                  <label>视频 {{ videoNodeState(workflowNode.id).referenceVideos.length }}/3<input type="file" accept="video/*" multiple @change="uploadVideoNodeMedia(workflowNode.id, 'referenceVideos', $event)"></label>
                  <button v-if="videoNodeState(workflowNode.id).referenceVideos.length" type="button" @click="updateVideoNode(workflowNode.id, { referenceVideos: [] })">清空</button>
                  <label>音频 {{ videoNodeState(workflowNode.id).referenceAudios.length }}/3<input type="file" accept="audio/*" multiple @change="uploadVideoNodeMedia(workflowNode.id, 'referenceAudios', $event)"></label>
                  <button v-if="videoNodeState(workflowNode.id).referenceAudios.length" type="button" @click="updateVideoNode(workflowNode.id, { referenceAudios: [] })">清空</button>
                </div>

                <textarea
                  :value="videoNodeState(workflowNode.id).prompt"
                  class="generate-prompt video-editor-prompt"
                  placeholder="描述想生成的视频内容，例如：产品在柔和灯光下缓慢旋转，镜头平稳推进"
                  @pointerdown.stop
                  @input="updateVideoNode(workflowNode.id, { prompt: $event.target.value })"
                ></textarea>

                <div class="generate-node__bar">
                  <div class="generate-select generate-select--model" :class="{ open: videoNodeState(workflowNode.id).menu === 'model' }" @click.stop>
                    <button type="button" class="generate-select__trigger" @click="toggleVideoNodeMenu(workflowNode.id, 'model')">
                      <span class="generate-select__icon">∞</span>
                      <span class="generate-select__copy">
                        <strong>{{ videoModelLabel(videoNodeState(workflowNode.id).model) }}</strong>
                      </span>
                      <span class="generate-select__chevron">⌃</span>
                    </button>
                    <div v-if="videoNodeState(workflowNode.id).menu === 'model'" class="generate-select-menu generate-model-menu">
                      <button
                        v-for="model in videoModelOptions"
                        :key="model.value"
                        type="button"
                        class="generate-model-option"
                        :class="{ active: videoNodeState(workflowNode.id).model === model.value }"
                        @click="selectVideoNodeModel(workflowNode.id, model.value)"
                      >
                        <span class="generate-model-option__icon">✦</span>
                        <span>
                          <strong>{{ model.label }}</strong>
                        </span>
                      </button>
                    </div>
                  </div>

                  <div class="generate-select generate-select--quality" :class="{ open: videoNodeState(workflowNode.id).menu === 'qualityPanel' }" @click.stop>
                    <button type="button" class="generate-select__trigger" @click="toggleVideoNodeMenu(workflowNode.id, 'qualityPanel')">
                      <span class="generate-quality-check"></span>
                      <span class="generate-select__value">{{ videoNodeQualityLabel(workflowNode.id) }}</span>
                      <span class="generate-select__chevron">⌃</span>
                    </button>
                    <div v-if="videoNodeState(workflowNode.id).menu === 'qualityPanel'" class="generate-select-menu generate-quality-menu video-quality-menu">
                      <section class="generate-quality-section">
                        <p>比例</p>
                        <div class="generate-aspect-grid video-aspect-grid">
                          <button
                            v-for="item in videoAspectRatioOptions"
                            :key="item.value"
                            type="button"
                            :class="{ active: videoNodeState(workflowNode.id).aspectRatio === item.value }"
                            @click="updateVideoNode(workflowNode.id, { aspectRatio: item.value })"
                          >
                            <span class="generate-aspect-icon" :class="videoAspectIconClass(item.value)"></span>
                            <strong>{{ videoAspectShortLabel(item.value) }}</strong>
                          </button>
                        </div>
                      </section>
                      <section class="generate-quality-section">
                        <p>清晰度</p>
                        <div class="generate-quality-row video-resolution-row">
                          <button
                            v-for="item in videoResolutionOptionsForNode(workflowNode.id)"
                            :key="item"
                            type="button"
                            :class="{ active: videoNodeState(workflowNode.id).resolution === item }"
                            @click="updateVideoNode(workflowNode.id, { resolution: item })"
                          >
                            {{ item.toUpperCase() }}
                          </button>
                        </div>
                      </section>
                      <section class="generate-quality-section">
                        <p>视频时长</p>
                        <div v-if="videoNodeState(workflowNode.id).model === 'seedance-2-5'" class="video-duration-range">
                          <input :value="videoNodeState(workflowNode.id).duration" type="range" min="4" max="30" step="1" aria-label="视频时长" @input="updateVideoNode(workflowNode.id, { duration: Number($event.target.value) })">
                          <input :value="videoNodeState(workflowNode.id).duration" type="number" min="4" max="30" step="1" aria-label="视频时长秒数" @change="updateVideoNode(workflowNode.id, { duration: Number($event.target.value) })">
                          <span>秒</span>
                        </div>
                        <div v-else class="generate-quality-row video-resolution-row" role="group" aria-label="视频时长">
                          <button
                            v-for="item in videoDurationOptions"
                            :key="item"
                            type="button"
                            :class="{ active: videoNodeState(workflowNode.id).duration === item }"
                            @click="updateVideoNode(workflowNode.id, { duration: item })"
                          >
                            {{ item }}s
                          </button>
                        </div>
                      </section>
                      <section class="generate-quality-section">
                        <p>同步音频</p>
                        <div class="generate-quality-row video-resolution-row">
                          <button type="button" :class="{ active: videoNodeState(workflowNode.id).generateAudio }" @click="updateVideoNode(workflowNode.id, { generateAudio: true })">开启</button>
                          <button type="button" :class="{ active: !videoNodeState(workflowNode.id).generateAudio }" @click="updateVideoNode(workflowNode.id, { generateAudio: false })">关闭</button>
                        </div>
                      </section>
                    </div>
                  </div>

                  <button
                    type="button"
                    class="node-primary-btn"
                    :disabled="!canRunVideoNode(workflowNode.id)"
                    @pointerdown.stop
                    @click.stop="runVideoNode(workflowNode.id)"
                  >
                    {{ videoNodeState(workflowNode.id).loading ? '生成中...' : '生成视频' }}
                  </button>
                </div>
              </div>
              <button
                type="button"
                class="node-port node-port--in"
                :data-node="workflowNode.id"
                data-port="in"
                title="连接到这个节点"
                @pointerdown.stop
                @click.stop="handleNodePortClick(workflowNode.id, 'in')"
              ></button>
              <button
                type="button"
                class="node-port node-port--out"
                :class="{ active: pendingLinkFrom === workflowNode.id }"
                :data-node="workflowNode.id"
                data-port="out"
                title="从这里连接到下一个节点"
                @pointerdown.stop.prevent="startLinkDrag($event, workflowNode.id)"
                @click.stop="handleNodePortClick(workflowNode.id, 'out')"
              ></button>
            </template>
            <template v-else-if="workflowNode.type === 'text'">
              <div
                class="text-node-card"
                :class="{ 'text-node-card--loading': textNodeState(workflowNode.id).loading }"
                @pointerdown="setActiveWorkflowNode(workflowNode.id)"
                @click.stop
              >
                <textarea
                  v-if="!textNodeState(workflowNode.id).loading"
                  :value="textNodeTextValue(workflowNode.id)"
                  class="text-node-input"
                  :data-text-node="workflowNode.id"
                  placeholder="双击编辑文本"
                  :readonly="!textNodeState(workflowNode.id).resultEditing"
                  title="双击编辑文本"
                  @pointerdown="setActiveWorkflowNode(workflowNode.id)"
                  @dblclick.stop.prevent="enableTextNodeEdit(workflowNode.id, $event)"
                  @keydown="handleTextNodeInputKeydown(workflowNode.id, $event)"
                  @keyup="handleTextNodeInputKeyup(workflowNode.id, $event)"
                  @input="updateTextNodeText(workflowNode.id, $event.target.value)"
                ></textarea>
                <div v-else class="text-node-loading-panel">
                  <span class="loading-ring-sm"></span>
                  <strong>{{ textNodeState(workflowNode.id).analysisLabel || '模型分析中' }}</strong>
                  <em>{{ textNodeState(workflowNode.id).progress || 0 }}%</em>
                  <div class="text-node-progress-track" aria-hidden="true">
                    <span :style="{ width: `${Math.min(100, Math.max(0, textNodeState(workflowNode.id).progress || 0))}%` }"></span>
                  </div>
                </div>
                <p v-if="textNodeState(workflowNode.id).error" class="text-node-error">
                  {{ textNodeState(workflowNode.id).error }}
                </p>
                <button
                  type="button"
                  class="text-node-resize"
                  title="调整大小"
                  aria-label="调整文本节点大小"
                  @pointerdown.stop.prevent="startTextNodeResize($event, workflowNode.id)"
                ></button>
              </div>
            </template>
            <template v-else-if="workflowNode.type === 'batch-recolor'">
              <div class="batch-node-card batch-node-card--recolor">
                <div class="batch-node-head">
                  <span>批量换色</span>
                  <strong>{{ batchRecolorProgressText(workflowNode.id) }}</strong>
                </div>
                <label class="batch-field" @pointerdown.stop @click.stop>
                  <span>换色对象</span>
                  <input
                    :value="batchRecolorNodeState(workflowNode.id).target"
                    type="text"
                    placeholder="例如：水龙头"
                    @input="updateBatchRecolorNode(workflowNode.id, { target: $event.target.value })"
                  >
                </label>
                <div class="batch-status-row">
                  <span>{{ batchRecolorInputText(workflowNode.id) }}</span>
                </div>
                <div class="generate-select generate-select--quality batch-quality-select" :class="{ open: batchRecolorNodeState(workflowNode.id).menu === 'qualityPanel' }" @pointerdown.stop @click.stop>
                  <button type="button" class="generate-select__trigger" @click="toggleBatchRecolorMenu(workflowNode.id, 'qualityPanel')">
                    <span class="generate-quality-check"></span>
                    <span class="generate-select__value">{{ batchRecolorQualityLabel(workflowNode.id) }}</span>
                    <span class="generate-select__chevron">⌃</span>
                  </button>
                  <div v-if="batchRecolorNodeState(workflowNode.id).menu === 'qualityPanel'" class="generate-select-menu generate-quality-menu batch-quality-menu">
                    <section class="generate-quality-section">
                      <p>清晰度</p>
                      <div class="generate-quality-row">
                        <button
                          v-for="item in generateResolutionOptions"
                          :key="item"
                          type="button"
                          :class="{ active: batchRecolorNodeState(workflowNode.id).resolution === item }"
                          @click="updateBatchRecolorNode(workflowNode.id, { resolution: item, menu: '' })"
                        >
                          {{ item }}
                        </button>
                      </div>
                    </section>
                    <section class="generate-quality-section">
                      <p>比例</p>
                      <div class="generate-aspect-grid">
                        <button
                          v-for="ratio in generateAspectRatioOptions"
                          :key="ratio"
                          type="button"
                          :class="{ active: batchRecolorNodeState(workflowNode.id).aspectRatio === ratio }"
                          @click="updateBatchRecolorNode(workflowNode.id, { aspectRatio: ratio, menu: '' })"
                        >
                          <span class="generate-aspect-icon" :class="aspectIconClass(ratio)"></span>
                          <strong>{{ ratio }}</strong>
                        </button>
                      </div>
                    </section>
                  </div>
                </div>
                <div class="batch-material-grid">
                  <button
                    v-for="material in batchRecolorNodeState(workflowNode.id).materials"
                    :key="material.value"
                    type="button"
                    class="batch-material-card"
                    title="点击移除"
                    @pointerdown.stop
                    @click.stop="removeBatchRecolorMaterial(workflowNode.id, material.value)"
                  >
                    <img v-if="material.imageUrl" v-lazy-src="thumbnailImage(material.imageUrl, 96)" :alt="material.label">
                    <span v-else>{{ material.label.charAt(0) }}</span>
                    <strong>{{ material.label }}</strong>
                    <em>×</em>
                  </button>
                  <button type="button" class="batch-add-material" @pointerdown.stop @click.stop="openBatchMaterialLibrary(workflowNode.id)">
                    + 材质
                  </button>
                </div>
                <p v-if="batchRecolorNodeState(workflowNode.id).error" class="batch-node-error">
                  {{ batchRecolorNodeState(workflowNode.id).error }}
                </p>
                <button
                  type="button"
                  class="batch-run-btn"
                  :disabled="!canRunBatchRecolor(workflowNode.id)"
                  @pointerdown.stop
                  @click.stop="runBatchRecolorNode(workflowNode.id)"
                >
                  {{ batchRecolorNodeState(workflowNode.id).loading ? '换色中...' : '开始换色' }}
                </button>
              </div>
            </template>
            <template v-else-if="workflowNode.type === 'batch-translate'">
              <div class="batch-node-card batch-node-card--translate">
                <div class="batch-node-head">
                  <span>批量翻译</span>
                  <strong>{{ batchTranslateProgressText(workflowNode.id) }}</strong>
                </div>
                <label class="batch-field" @pointerdown.stop @click.stop>
                  <span>目标语言</span>
                  <div class="batch-language-grid">
                    <button
                      v-for="item in batchTranslateLanguageOptions"
                      :key="item.value"
                      type="button"
                      :class="{ active: isBatchTranslateLanguageSelected(workflowNode.id, item.value) }"
                      @click="toggleBatchTranslateLanguage(workflowNode.id, item.value)"
                    >
                      {{ item.label }}
                    </button>
                  </div>
                </label>
                <div class="generate-select generate-select--quality batch-quality-select" :class="{ open: batchTranslateNodeState(workflowNode.id).menu === 'qualityPanel' }" @pointerdown.stop @click.stop>
                  <button type="button" class="generate-select__trigger" @click="toggleBatchTranslateMenu(workflowNode.id, 'qualityPanel')">
                    <span class="generate-quality-check"></span>
                    <span class="generate-select__value">{{ batchTranslateQualityLabel(workflowNode.id) }}</span>
                    <span class="generate-select__chevron">⌃</span>
                  </button>
                  <div v-if="batchTranslateNodeState(workflowNode.id).menu === 'qualityPanel'" class="generate-select-menu generate-quality-menu batch-quality-menu">
                    <section class="generate-quality-section">
                      <p>清晰度</p>
                      <div class="generate-quality-row">
                        <button
                          v-for="item in generateResolutionOptions"
                          :key="item"
                          type="button"
                          :class="{ active: batchTranslateNodeState(workflowNode.id).resolution === item }"
                          @click="updateBatchTranslateNode(workflowNode.id, { resolution: item, menu: '' })"
                        >
                          {{ item }}
                        </button>
                      </div>
                    </section>
                    <section class="generate-quality-section">
                      <p>比例</p>
                      <div class="generate-aspect-grid">
                        <button
                          v-for="ratio in generateAspectRatioOptions"
                          :key="ratio"
                          type="button"
                          :class="{ active: batchTranslateNodeState(workflowNode.id).aspectRatio === ratio }"
                          @click="updateBatchTranslateNode(workflowNode.id, { aspectRatio: ratio, menu: '' })"
                        >
                          <span class="generate-aspect-icon" :class="aspectIconClass(ratio)"></span>
                          <strong>{{ ratio }}</strong>
                        </button>
                      </div>
                    </section>
                  </div>
                </div>
                <div v-if="batchTranslateNodeState(workflowNode.id).tasks.length" class="batch-task-panel">
                  <div
                    v-for="task in batchTranslateNodeState(workflowNode.id).tasks"
                    :key="task.index"
                    class="batch-task-row"
                    :title="task.error || task.materialName || `图${task.index + 1}`"
                  >
                    <img v-if="task.sourceImageUrl" v-lazy-src="thumbnailImage(task.sourceImageUrl, 80)" :alt="task.materialName || `图${task.index + 1}`">
                    <span v-else class="batch-task-thumb">{{ task.index + 1 }}</span>
                    <strong>{{ task.targetLanguageName || batchLanguageLabel(task.targetLanguage) }}</strong>
                    <small>{{ task.imageUrl ? '已生成翻译图' : '等待图片翻译' }}</small>
                    <em :class="`batch-status-badge is-${task.status || 'pending'}`">{{ batchTaskStatusText(task) }}</em>
                  </div>
                </div>
                <p v-if="batchTranslateNodeState(workflowNode.id).error" class="batch-node-error">
                  {{ batchTranslateNodeState(workflowNode.id).error }}
                </p>
                <button
                  type="button"
                  class="batch-run-btn"
                  :disabled="!canRunBatchTranslate(workflowNode.id)"
                  @pointerdown.stop
                  @click.stop="runBatchTranslateNode(workflowNode.id)"
                >
                  {{ batchTranslateNodeState(workflowNode.id).loading ? '翻译中...' : '开始翻译' }}
                </button>
              </div>
            </template>
            <template v-else>
              <div class="node-icon">{{ dynamicNodeIcon(workflowNode) }}</div>
              <p>{{ dynamicNodeDescription(workflowNode) }}</p>
            </template>
          </div>
          <button
            v-if="workflowNode.type !== 'upload' && workflowNode.type !== 'video'"
            type="button"
            class="node-port node-port--in"
            :data-node="workflowNode.id"
            data-port="in"
            title="连接到这个节点"
            @pointerdown.stop
            @click.stop="handleNodePortClick(workflowNode.id, 'in')"
          ></button>
          <button
            v-if="workflowNode.type !== 'video'"
            type="button"
            class="node-port node-port--out"
            :class="{ active: pendingLinkFrom === workflowNode.id }"
            :data-node="workflowNode.id"
            data-port="out"
            title="从这里连接到下一个节点"
            @pointerdown.stop.prevent="startLinkDrag($event, workflowNode.id)"
            @click.stop="handleNodePortClick(workflowNode.id, 'out')"
          ></button>
        </section>

        <section
          v-for="generateNode in visibleGenerateWorkflowNodes()"
          :key="generateNode.id"
          class="canvas-node node-image-generate"
          :data-workflow-node="generateNode.id"
          :class="{ selected: activeWorkflowNode === generateNode.id, done: primaryGenerationImage(generateNode.id), 'link-source': pendingLinkFrom === generateNode.id, 'link-candidate': isWorkflowLinkCandidate(pendingLinkFrom, generateNode.id), 'link-target': hoveredLinkTarget === generateNode.id, 'multi-selected': selectedNodeIds.includes(generateNode.id) }"
          :style="nodeStyle(generateNode.id)"
          @click="onGenerateNodeClick($event, generateNode.id)"
          @pointerdown="startNodeDrag($event, generateNode.id)"
        >
          <span v-if="batchDownloadSelectMode && selectedNodeIds.includes(generateNode.id)" class="batch-selection-order">{{ batchDownloadSelectionOrder(generateNode.id) }}</span>
          <div class="node-label">{{ generateNodeTitle(generateNode.id) }}</div>
          <div v-if="generateNodeImageSizeText(generateNode.id)" class="generate-node__size">
            {{ generateNodeImageSizeText(generateNode.id) }}
          </div>

          <button
            v-if="showGenerateUploadButton(generateNode.id)"
            type="button"
            class="generate-upload-btn"
            :class="{ 'generate-upload-btn--top': !generationUploadImages(generateNode.id).length }"
            @pointerdown.stop
            @click.stop="openGenerateUpload(generateNode.id)"
            aria-label="上传图片"
          >
            <ArrowUpTrayIcon v-if="generationUploadImages(generateNode.id).length" class="toolbar-icon" aria-hidden="true" />
            <span v-else>上传</span>
          </button>

          <div v-if="(primaryGenerationImage(generateNode.id) || canRegenerateGeneratedImageNode(generateNode.id)) && activeWorkflowNode === generateNode.id && !isGenerateImageEditing(generateNode.id)" class="image-node-toolbar" @pointerdown.stop @click.stop>
            <button v-if="canRegenerateGeneratedImageNode(generateNode.id)" type="button" title="重新生成" aria-label="重新生成" @click.stop="regenerateGeneratedImageNode(generateNode.id)">
              <ArrowPathIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id)" type="button" title="图片反推提示词" aria-label="图片反推提示词" @click.stop="createImageAnalysisTextNode(generateNode.id, 'prompt')">
              <DocumentTextIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id)" type="button" title="图片优化建议" aria-label="图片优化建议" @click.stop="createImageAnalysisTextNode(generateNode.id, 'suggestion')">
              <LightBulbIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id)" type="button" title="转化为景深图" aria-label="转化为景深图" @click.stop="createDepthOfFieldImageNode(generateNode.id)">
              <ViewfinderCircleIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id)" type="button" title="光影优化" aria-label="光影优化" @click.stop="createLightShadowImageNode(generateNode.id)">
              <SunIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id) && canEditGenerateNodeImage(generateNode.id)" type="button" title="编辑" aria-label="编辑" @click.stop="startGenerateImageEdit(generateNode.id)">
              <PencilSquareIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id)" type="button" title="下载" aria-label="下载" @click="downloadGeneratedImage(primaryGenerationImage(generateNode.id), 0, generateNode.id)">
              <ArrowDownTrayIcon class="toolbar-icon" aria-hidden="true" />
            </button>
            <button v-if="primaryGenerationImage(generateNode.id)" type="button" title="查看大图" aria-label="查看大图" @pointerdown.stop.prevent @click.stop.prevent="openGenerateNodePreview(generateNode.id)">
              <ArrowsPointingOutIcon class="toolbar-icon" aria-hidden="true" />
            </button>
          </div>

          <div v-if="isGenerateImageEditing(generateNode.id)" class="generate-edit-toolbar" @pointerdown.stop @click.stop>
            <button type="button" class="generate-edit-toolbar__exit" title="退出编辑" @click.stop="stopGenerateImageEdit">
              <XMarkIcon class="generate-edit-icon" aria-hidden="true" />
              <strong>退出编辑</strong>
            </button>
            <span class="generate-edit-toolbar__divider"></span>
            <button type="button" :class="{ active: generateImageEditState.tool === 'brush' }" title="画笔" @click.stop="setGenerateEditTool('brush')">
              <PaintBrushIcon class="generate-edit-icon" aria-hidden="true" />
            </button>
            <button type="button" :class="{ active: generateImageEditState.tool === 'rect' }" title="框选" @click.stop="setGenerateEditTool('rect')">
              <Squares2X2Icon class="generate-edit-icon" aria-hidden="true" />
            </button>
            <button type="button" :class="{ active: generateImageEditState.tool === 'line' }" title="直线" @click.stop="setGenerateEditTool('line')">
              <MinusIcon class="generate-edit-icon" aria-hidden="true" />
            </button>
            <button type="button" :class="{ active: generateImageEditState.tool === 'arrow' }" title="箭头" @click.stop="setGenerateEditTool('arrow')">
              <ArrowLongRightIcon class="generate-edit-icon" aria-hidden="true" />
            </button>
            <span class="generate-edit-toolbar__divider"></span>
            <label class="generate-edit-color" title="颜色">
              <input v-model="generateImageEditState.color" type="color" aria-label="颜色">
              <span :style="{ background: generateImageEditState.color }"></span>
            </label>
            <label class="generate-edit-size" title="笔触大小">
              <AdjustmentsHorizontalIcon class="generate-edit-icon" aria-hidden="true" />
              <input v-model.number="generateImageEditState.size" type="range" min="2" :max="IMAGE_EDIT_BRUSH_DEFAULTS.maxSize" aria-label="笔触大小">
            </label>
            <span class="generate-edit-toolbar__divider"></span>
            <button type="button" title="撤销" :disabled="!generateImageEditState.undoStack.length" @click.stop="undoGenerateImageEdit">
              <ArrowUturnLeftIcon class="generate-edit-icon" aria-hidden="true" />
            </button>
            <button type="button" title="重做" :disabled="!generateImageEditState.redoStack.length" @click.stop="redoGenerateImageEdit">
              <ArrowUturnLeftIcon class="generate-edit-icon generate-edit-icon--redo" aria-hidden="true" />
            </button>
            <button type="button" class="generate-edit-toolbar__save" @click.stop="submitGenerateImageEdit(generateNode.id)">
              保存
            </button>
          </div>
          <div class="generate-card">
            <div v-if="generateNode.generating" class="generate-node__loading">
              <div class="generate-wave" aria-hidden="true"></div>
              <div class="generate-progress-panel">
                <button type="button" class="generate-progress-pill" @click.stop="cancelWorkflowGeneration(generateNode.id)">
                  <strong>生成中 {{ generateNode.progress || 0 }}%</strong>
                  <span>取消</span>
                </button>
                <div class="generate-progress-track" aria-hidden="true">
                  <span :style="{ width: `${Math.min(100, Math.max(0, generateNode.progress || 0))}%` }"></span>
                </div>
              </div>
            </div>
            <div v-else-if="generateNode.error" class="generate-node__error">
              <strong>生成失败</strong>
              <span>{{ generateNode.error }}</span>
            </div>
          <div v-else-if="generationImages(generateNode.id).length" class="generate-node__preview">
              <button type="button" class="generate-node__image-button" @pointerdown="prepareGenerateImageNodeInteraction(generateNode.id)" @click.stop="onGeneratePreviewClick(generateNode.id)">
                <img
                  :key="primaryGenerationImage(generateNode.id)"
                  v-lazy-src="thumbnailImage(primaryGenerationImage(generateNode.id), 640)"
                  alt="生图结果"
                  @load="rememberImageSize(primaryGenerationImage(generateNode.id), $event)"
                >
              </button>
              <canvas
                v-if="isGenerateImageEditing(generateNode.id)"
                ref="generateEditCanvasRef"
                class="generate-edit-canvas"
                @pointerdown.stop.prevent="startGenerateEditDraw"
                @pointermove.stop.prevent="moveGenerateEditDraw"
                @pointerup.stop.prevent="endGenerateEditDraw"
                @pointercancel.stop.prevent="endGenerateEditDraw"
              ></canvas>
            </div>
            <div v-else class="generate-node__placeholder">
              <div class="generate-node__placeholder-icon" aria-hidden="true">
                <svg viewBox="0 0 24 24">
                  <rect x="4" y="5" width="16" height="14" rx="3"></rect>
                  <path d="m7 16 3.2-3.2 2.4 2.4 2.2-2.7L18 16"></path>
                  <circle cx="8.7" cy="9.3" r="1.1"></circle>
                </svg>
              </div>
            </div>
          </div>

          <div
            v-if="canShowGenerateEditorPanel(generateNode.id)"
            class="generate-editor-panel"
            :class="{ 'has-refs': generationReferenceImages(generateNode.id).length }"
            @pointerdown.stop
            @click.stop="setGenerateNodeMenu(generateNode.id, '')"
          >
            <div class="generate-input-summary">
              <span>输入</span>
              <strong>{{ generationReferenceSummary(generateNode.id) }}</strong>
            </div>
            <div v-if="generationReferenceImages(generateNode.id).length" class="generate-node__refs">
              <component
                v-for="item in generationReferenceImages(generateNode.id)"
                :is="item.type === 'text' ? 'div' : 'button'"
                :key="item.key"
                :type="item.type === 'text' ? undefined : 'button'"
                class="generate-ref"
                :class="{ 'generate-ref--text': item.type === 'text' }"
                @click.stop="item.type === 'text' ? null : openPreview(item.url, item.label)"
              >
                <template v-if="item.type === 'text'">
                  <span class="generate-ref-text-icon" aria-hidden="true"></span>
                </template>
                <img v-else v-lazy-src="thumbnailImage(item.url, 260)" :alt="item.label">
              </component>
            </div>

            <textarea
              :value="generateNode.prompt"
              class="generate-prompt"
              placeholder="可直接文字生图，或上传图片输入文字指令对图片进行编辑，如：将背景改为雪夜"
              @pointerdown.stop
              @input="updateGenerateNode(generateNode.id, { prompt: $event.target.value })"
            ></textarea>

            <p v-if="generateNode.error" class="generate-error">{{ generateNode.error }}</p>

            <div class="generate-node__bar">
              <div class="generate-select generate-select--model" :class="{ open: generateNode.menu === 'model' }" @click.stop>
                <button type="button" class="generate-select__trigger" @click="toggleWorkflowGenerateMenu(generateNode.id, 'model')">
                  <span class="generate-select__icon">∞</span>
                  <span class="generate-select__copy">
                    <strong>{{ selectedDrawingModel(generateNode.id)?.display_name || selectedDrawingModel(generateNode.id)?.name || effectiveDrawingModelName(generateNode.id) || '选择模型' }}</strong>
                  </span>
                  <span class="generate-select__chevron">⌃</span>
                </button>
                <div v-if="generateNode.menu === 'model'" class="generate-select-menu generate-model-menu">
                  <button
                    v-for="model in drawingModels"
                    :key="model.id || model.name"
                    type="button"
                    class="generate-model-option"
                    :class="{ active: effectiveDrawingModelName(generateNode.id) === model.name }"
                    @click="selectWorkflowGenerateOption(generateNode.id, 'model', model.name)"
                  >
                    <span class="generate-model-option__icon">✦</span>
                    <span>
                      <strong>{{ model.display_name || model.name }}</strong>
                    </span>
                    <em v-if="model.avg_time || model.estimated_time">{{ model.avg_time || model.estimated_time }}</em>
                  </button>
                  <div v-if="!drawingModels.length" class="generate-select-empty">暂无可用模型</div>
                </div>
              </div>

              <div class="generate-select generate-select--quality" :class="{ open: generateNode.menu === 'qualityPanel' }" @click.stop>
                <button type="button" class="generate-select__trigger" @click="toggleWorkflowGenerateMenu(generateNode.id, 'qualityPanel')">
                  <span class="generate-quality-check"></span>
                  <span class="generate-select__value">{{ workflowGenerateControlLabel(generateNode.id) }}</span>
                  <span class="generate-select__chevron">⌃</span>
                </button>
                <div v-if="generateNode.menu === 'qualityPanel'" class="generate-select-menu generate-quality-menu">
                  <section class="generate-quality-section">
                    <p>清晰度</p>
                    <div class="generate-quality-row">
                      <button
                        v-for="item in generateResolutionOptions"
                        :key="item"
                        type="button"
                        :class="{ active: generateNode.resolution === item }"
                        @click="selectWorkflowGenerateOption(generateNode.id, 'resolution', item)"
                      >
                        {{ item }}
                      </button>
                    </div>
                  </section>
                  <section class="generate-quality-section">
                    <p>比例</p>
                    <div class="generate-aspect-grid">
                      <button
                        v-for="ratio in generateAspectRatioOptionsForNode(generateNode.id)"
                        :key="ratio"
                        type="button"
                        :class="{ active: generateNode.aspectRatio === ratio }"
                        @click="selectWorkflowGenerateOption(generateNode.id, 'aspect', ratio)"
                      >
                        <span class="generate-aspect-icon" :class="aspectIconClass(ratio)"></span>
                        <strong>{{ ratio }}</strong>
                      </button>
                    </div>
                  </section>
                </div>
              </div>

              <div class="generate-select generate-select--count" :class="{ open: generateNode.menu === 'count' }" @click.stop>
                <button type="button" class="generate-select__trigger" @click="toggleWorkflowGenerateMenu(generateNode.id, 'count')">
                  <span>{{ generateNode.count }}张</span>
                  <small>数量</small>
                </button>
                <div v-if="generateNode.menu === 'count'" class="generate-select-menu generate-count-menu">
                  <button
                    v-for="item in generateCountOptions"
                    :key="item"
                    type="button"
                    :class="{ active: generateNode.count === item }"
                    @click="selectWorkflowGenerateOption(generateNode.id, 'count', item)"
                  >
                    {{ item }}张
                  </button>
                </div>
              </div>
              <button
                type="button"
                class="node-primary-btn"
                :disabled="!canGenerateWorkflowImage(generateNode.id)"
                @click.stop="generateWorkflowImage(generateNode.id)"
              >
                {{ generateNode.generating ? '生成中...' : '生成图片' }}
              </button>
            </div>
          </div>

          <button
            v-if="!isUploadOnlyGenerateNode(generateNode.id)"
            type="button"
            class="node-port node-port--in"
            :data-node="generateNode.id"
            data-port="in"
            title="连接到这个节点"
            @pointerdown.stop
            @click.stop="handleNodePortClick(generateNode.id, 'in')"
          ></button>
          <button
            type="button"
            class="node-port node-port--out"
            :class="{ active: pendingLinkFrom === generateNode.id }"
            :data-node="generateNode.id"
            data-port="out"
            title="从这里连接到下一个节点"
            @pointerdown.stop.prevent="startLinkDrag($event, generateNode.id)"
            @click.stop="handleNodePortClick(generateNode.id, 'out')"
          ></button>
        </section>
        </div>

        <div v-if="activeProjectId && !isWorkflowEmpty() && !hasNodesInViewport" class="viewport-empty-tip" @pointerdown.stop @click.stop>
          <span>当前视窗没有节点，可点击按钮快速回到内容区域</span>
          <button type="button" @click.stop="focusCanvasOnNodes">回到节点</button>
        </div>
        <div
          v-if="linkCreateMenu.open"
          class="link-create-menu"
          :style="linkCreateMenuStyle()"
          @pointerdown.stop
          @click.stop
        >
          <button type="button" @click="createLinkedNodeFromMenu('generate')">
            <span class="node-menu__icon node-menu__icon--image" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="5" width="16" height="14" rx="3" />
                <path d="m7 16 3.2-3.2 2.4 2.4 2.2-2.7L18 16" />
                <path d="M8.5 9.5h.01" />
              </svg>
            </span>
            <strong>图片节点</strong>
            <small>继续生成或作为参考图</small>
          </button>
          <button v-if="nodeType(linkCreateMenu.from) === 'generate'" type="button" @click="createLinkedNodeFromMenu('video')">
            <span class="node-menu__icon node-menu__icon--video" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="6" width="12" height="12" rx="2" />
                <path d="m16 10 4-2.4v8.8L16 14" />
              </svg>
            </span>
            <strong>视频节点</strong>
            <small>用提示词和参考图生成视频</small>
          </button>
          <button type="button" @click="createLinkedNodeFromMenu('text')">
            <span class="node-menu__icon node-menu__icon--text" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M5 6h14" />
                <path d="M8 6v12" />
                <path d="M16 6v12" />
                <path d="M6.5 18h3" />
                <path d="M14.5 18h3" />
              </svg>
            </span>
            <strong>文本节点</strong>
            <small>写文本，可作为提示词连接</small>
          </button>
          <button type="button" @click="createLinkedNodeFromMenu('batch-recolor')">
            <span class="node-menu__icon node-menu__icon--material" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="5" width="16" height="14" rx="3" />
                <path d="M8 9h8M8 13h5" />
                <path d="M15 16c1.8-1.8 3-3.2 3-4.6a2 2 0 0 0-4 0c0 1.4 1.2 2.8 3 4.6Z" />
              </svg>
            </span>
            <strong>批量换色</strong>
            <small>按材质批量生成换色图</small>
          </button>
          <button type="button" @click="createLinkedNodeFromMenu('batch-translate')">
            <span class="node-menu__icon node-menu__icon--text" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M5 5h8" />
                <path d="M9 5v14" />
                <path d="M5 19h8" />
                <path d="M15 9h4" />
                <path d="M14 19l3-8 3 8" />
                <path d="M15.2 16h3.6" />
              </svg>
            </span>
            <strong>批量翻译</strong>
            <small>复用图片翻译处理多图</small>
          </button>
        </div>
      </div>

      <div v-if="activeProjectId" class="canvas-toolbar">
        <button type="button" class="toolbar-main" :class="{ active: showNodeMenu }" :title="showNodeMenu ? '关闭节点菜单' : '添加节点'" :aria-label="showNodeMenu ? '关闭节点菜单' : '添加节点'" :aria-expanded="showNodeMenu" @click.stop="toggleNodeMenu">
          {{ showNodeMenu ? '×' : '+' }}
        </button>
        <div v-if="showNodeMenu" class="node-menu">
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('upload')">
            <span class="node-menu__icon node-menu__icon--model" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M12 3 4.5 7.2 12 11.4l7.5-4.2L12 3Z" />
                <path d="M4.5 7.2v8.4L12 21l7.5-5.4V7.2" />
                <path d="M12 11.4V21" />
              </svg>
            </span>
            <strong>3D 文件</strong>
            <small>上传模型并选择视角</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('generate')">
            <span class="node-menu__icon node-menu__icon--image" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="5" width="16" height="14" rx="3" />
                <path d="m7 16 3.2-3.2 2.4 2.4 2.2-2.7L18 16" />
                <path d="M8.5 9.5h.01" />
              </svg>
            </span>
            <strong>图片</strong>
            <small>用提示词和参考图生成图片</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('video')">
            <span class="node-menu__icon node-menu__icon--video" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="6" width="12" height="12" rx="2" />
                <path d="m16 10 4-2.4v8.8L16 14" />
              </svg>
            </span>
            <strong>视频</strong>
            <small>Seedance 2.5 固定视频模型</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="openAssetPicker('node')">
            <span class="node-menu__icon node-menu__icon--asset" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="5" width="7" height="6" rx="1.5" />
                <rect x="13" y="5" width="7" height="6" rx="1.5" />
                <rect x="4" y="14" width="7" height="5" rx="1.5" />
                <path d="M15 15.5h4M17 13.5v4" />
              </svg>
            </span>
            <strong>资产库</strong>
            <small>选择图片或模型，直接创建对应节点</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="openCanvasMaterialLibrary">
            <span class="node-menu__icon node-menu__icon--material" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="5" width="16" height="14" rx="3" />
                <path d="M8 8h8M8 11h8M8 14h5" />
                <path d="M16 15.5 18.5 18 21 13" />
              </svg>
            </span>
            <strong>材质</strong>
            <small>选择后台材质作为参考图</small>
          </button>
          <button
            type="button"
            class="node-menu__library"
            :class="{ open: showStyleLibraryMenu }"
            @click.stop="toggleStyleLibraryMenu"
            @mouseenter="showStyleLibraryMenu = true"
          >
            <span class="node-menu__icon node-menu__icon--style" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M12 3.5c3.8 0 7 2.7 7 6.1 0 2.2-1.2 3.4-3.1 3.4h-1.1c-.9 0-1.5.7-1.3 1.6l.1.5c.3 1.5-.8 2.9-2.4 2.9C7.7 18 5 15 5 11.4 5 7 8.1 3.5 12 3.5Z" />
                <path d="M8.7 10.1h.01M10.8 7.6h.01M14 7.8h.01M15.9 10.5h.01" />
              </svg>
            </span>
            <strong>风格库</strong>
            <small>管理常用风格参考</small>
          </button>
          <div v-if="showStyleLibraryMenu" class="node-menu-submenu" @mouseenter="showStyleLibraryMenu = true" @pointerdown.stop @click.stop>
            <button type="button" @click="openCanvasStyleLibrary('kitchen')">厨房场景</button>
            <button type="button" @click="openCanvasStyleLibrary('bathroom')">浴室场景</button>
          </div>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('text')">
            <span class="node-menu__icon node-menu__icon--text" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M5 6h14" />
                <path d="M8 6v12" />
                <path d="M16 6v12" />
                <path d="M6.5 18h3" />
                <path d="M14.5 18h3" />
              </svg>
            </span>
            <strong>文本工具</strong>
            <small>写文本，可作为提示词连接</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('llm')">
            <span class="node-menu__icon node-menu__icon--llm" aria-hidden="true">
              <svg viewBox="0 0 24 24"><path d="M8 4h8a4 4 0 0 1 4 4v5a4 4 0 0 1-4 4h-4l-4 3v-3a4 4 0 0 1-4-4V8a4 4 0 0 1 4-4Z"/><path d="M8 9h8M8 13h5"/></svg>
            </span>
            <strong>大模型</strong>
            <small>理解上游图片与文字，输出可连接的文本</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('batch-recolor')">
            <span class="node-menu__icon node-menu__icon--material" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <rect x="4" y="5" width="16" height="14" rx="3" />
                <path d="M8 9h8M8 13h5" />
                <path d="M15 16c1.8-1.8 3-3.2 3-4.6a2 2 0 0 0-4 0c0 1.4 1.2 2.8 3 4.6Z" />
              </svg>
            </span>
            <strong>批量换色</strong>
            <small>输入换色对象，批量套用材质</small>
          </button>
          <button type="button" @mouseenter="showStyleLibraryMenu = false" @click="addWorkflowNode('batch-translate')">
            <span class="node-menu__icon node-menu__icon--text" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M5 5h8" />
                <path d="M9 5v14" />
                <path d="M5 19h8" />
                <path d="M15 9h4" />
                <path d="M14 19l3-8 3 8" />
                <path d="M15.2 16h3.6" />
              </svg>
            </span>
            <strong>批量翻译</strong>
            <small>将多张图片文字翻译成目标语言</small>
          </button>
        </div>
        <button type="button" class="toolbar-action" title="一键整理画布" aria-label="一键整理画布" @click.stop="autoArrangeWorkflow">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <rect x="4" y="5" width="6" height="5" rx="1.5" />
            <rect x="14" y="5" width="6" height="5" rx="1.5" />
            <rect x="9" y="15" width="6" height="5" rx="1.5" />
            <path d="M10 7.5h4M17 10v2.5L12 15M7 10v2.5L12 15" />
          </svg>
        </button>
        <button type="button" class="toolbar-action" :class="{ active: showWorkflowTemplateModal }" title="工作流模板" aria-label="工作流模板" @click.stop="openWorkflowTemplateModal">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <rect x="4" y="4" width="6" height="6" rx="1.5" />
            <rect x="14" y="4" width="6" height="6" rx="1.5" />
            <rect x="4" y="14" width="6" height="6" rx="1.5" />
            <rect x="14" y="14" width="6" height="6" rx="1.5" />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-action toolbar-download"
          :class="{ active: batchDownloadSelectMode || batchDownloadState.loading }"
          :disabled="batchDownloadState.loading"
          :title="batchDownloadState.loading ? `正在打包 ${batchDownloadState.completed}/${batchDownloadState.total}` : batchDownloadSelectMode ? (selectedDownloadImageCount ? `下载已选 ${selectedDownloadImageCount} 张图片` : '点击节点进行选择，再次点击此按钮开始下载') : '进入批量下载选择模式'"
          aria-label="批量下载图片"
          @click.stop="handleBatchDownloadButton"
        >
          <ArrowDownTrayIcon aria-hidden="true" />
        </button>
        <Teleport to="body">
        <Transition name="workflow-library-pop">
          <div v-if="showWorkflowTemplateModal" class="workflow-template-modal" role="dialog" aria-modal="true" aria-labelledby="workflow-library-title" @pointerdown.self="closeWorkflowTemplateModal" @keydown.esc="closeWorkflowTemplateModal">
            <div class="workflow-template-modal__panel" @pointerdown.stop @click.stop>
              <header class="workflow-template-modal__header">
                <div><h3 id="workflow-library-title">工作流</h3><p>从模板开始，或继续使用你保存的流程</p></div>
                <button type="button" aria-label="关闭工作流" @click="closeWorkflowTemplateModal"><XMarkIcon /></button>
              </header>
              <div class="workflow-template-controls">
                <div class="workflow-template-tabs" role="tablist" aria-label="工作流分类">
                  <button type="button" role="tab" :aria-selected="workflowTemplateScope === 'system'" :class="{ active: workflowTemplateScope === 'system' }" @click="workflowTemplateScope = 'system'"><span>工作流模板</span><small>{{ systemWorkflowTemplates.length }}</small></button>
                  <button type="button" role="tab" :aria-selected="workflowTemplateScope === 'user'" :class="{ active: workflowTemplateScope === 'user' }" @click="workflowTemplateScope = 'user'"><span>我的工作流</span><small>{{ userWorkflowTemplates.length }}</small></button>
                </div>
                <label class="workflow-template-search">
                  <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6"/><path d="m16 16 4 4"/></svg>
                  <input ref="workflowTemplateSearchInput" v-model.trim="workflowTemplateSearch" type="search" placeholder="搜索名称或描述" aria-label="搜索工作流">
                </label>
              </div>
              <div class="workflow-template-grid">
                <template v-if="workflowTemplatesLoading">
                  <div v-for="index in 6" :key="index" class="workflow-template-skeleton"><i></i><span></span><span></span><button></button></div>
                </template>
                <div v-else-if="!visibleWorkflowTemplates.length" class="workflow-template-empty"><span class="workflow-template-empty__icon"><Squares2X2Icon /></span><strong>{{ workflowTemplateSearch ? '没有匹配的工作流' : workflowTemplateScope === 'user' ? '还没有保存的工作流' : '暂无工作流模板' }}</strong><small>{{ workflowTemplateSearch ? '换个关键词试试' : workflowTemplateScope === 'user' ? '组合画布节点后即可保存' : '系统模板配置后会显示在这里' }}</small></div>
                <article v-for="template in visibleWorkflowTemplates" v-else :key="template.id" class="workflow-template-card">
                  <div class="workflow-template-card__topline">
                    <span :class="{ official: template.scope === 'system' }">{{ template.scope === 'system' ? '系统模板' : '我的工作流' }}</span>
                    <button v-if="template.scope === 'user' || isWorkflowAdmin" type="button" class="workflow-template-card__delete" title="删除工作流" aria-label="删除工作流" @click="deleteWorkflowTemplate(template)"><TrashIcon /></button>
                  </div>
                  <div class="workflow-template-card__preview">
                    <div class="workflow-preview-flow">
                      <div v-for="node in workflowTemplatePreviewNodes(template)" :key="node.id" class="workflow-preview-item">
                        <span class="workflow-preview-node" :class="node.className">{{ node.label }}</span>
                      </div>
                      <span v-if="workflowTemplateNodeCount(template) > 4" class="workflow-preview-more">+{{ workflowTemplateNodeCount(template) - 4 }}</span>
                    </div>
                  </div>
                  <div class="workflow-template-card__body">
                    <strong :title="template.name">{{ template.name }}</strong>
                    <p>{{ template.summary || '专业画布工作流' }}</p>
                  </div>
                  <div class="workflow-template-card__meta"><span>{{ workflowTemplateNodeCount(template) }} 个节点</span><i></i><span>{{ workflowTemplateUpdatedAt(template) }}</span></div>
                  <button type="button" class="workflow-template-card__use" @click="useWorkflowTemplate(template)"><PlayIcon aria-hidden="true" />使用工作流</button>
                </article>
              </div>
            </div>
          </div>
        </Transition>
        <Transition name="workflow-library-pop">
          <div v-if="workflowTemplateDeleteTarget" class="workflow-template-delete-modal" role="alertdialog" aria-modal="true" aria-labelledby="workflow-delete-title" @pointerdown.self="closeWorkflowTemplateDeleteModal" @keydown.esc="closeWorkflowTemplateDeleteModal">
            <div class="workflow-template-delete-modal__panel" @pointerdown.stop @click.stop>
              <span class="workflow-template-delete-modal__icon"><TrashIcon /></span>
              <div><h3 id="workflow-delete-title">删除工作流</h3><p>“{{ workflowTemplateDeleteTarget.name }}”删除后无法恢复。</p></div>
              <footer><button type="button" :disabled="workflowTemplateDeleting" @click="closeWorkflowTemplateDeleteModal">取消</button><button type="button" class="danger" :disabled="workflowTemplateDeleting" @click="confirmDeleteWorkflowTemplate">{{ workflowTemplateDeleting ? '删除中' : '删除' }}</button></footer>
            </div>
          </div>
        </Transition>
        <Transition name="workflow-save-pop">
          <div v-if="showWorkflowSaveModal" class="workflow-save-modal" role="dialog" aria-modal="true" aria-labelledby="workflow-save-title" @pointerdown.self="closeWorkflowSaveModal">
            <form class="workflow-save-modal__panel" :aria-busy="workflowSaving" @pointerdown.stop @click.stop @submit.prevent="confirmSaveGroupAsWorkflow">
              <header class="workflow-save-modal__header">
                <div>
                  <div class="workflow-save-modal__title"><h3 id="workflow-save-title">保存为工作流</h3><span>{{ workflowSaveNodeCount }} 个节点</span></div>
                  <p>节点参数与连接关系将一并保存</p>
                </div>
                <button type="button" aria-label="关闭" :disabled="workflowSaving" @click="closeWorkflowSaveModal"><XMarkIcon /></button>
              </header>
              <div class="workflow-save-modal__body">
                <label class="workflow-save-name">
                  <span>名称</span>
                  <div>
                    <input ref="workflowSaveNameInput" v-model="workflowSaveName" type="text" maxlength="80" placeholder="输入工作流名称" @keydown.esc.prevent="closeWorkflowSaveModal">
                    <small>{{ workflowSaveName.trim().length }}/80</small>
                  </div>
                </label>
                <fieldset v-if="isWorkflowAdmin" class="workflow-save-scope">
                  <legend>保存到</legend>
                  <div>
                    <button type="button" :class="{ active: workflowSaveScope === 'user' }" :aria-pressed="workflowSaveScope === 'user'" @click="workflowSaveScope = 'user'">
                      <i aria-hidden="true"><span></span></i><span><strong>我的工作流</strong><small>仅自己可见</small></span>
                    </button>
                    <button type="button" :class="{ active: workflowSaveScope === 'system' }" :aria-pressed="workflowSaveScope === 'system'" @click="workflowSaveScope = 'system'">
                      <i aria-hidden="true"><span></span></i><span><strong>系统工作流</strong><small>所有用户可用</small></span>
                    </button>
                  </div>
                </fieldset>
                <div v-else class="workflow-save-destination"><span>保存到</span><div><i aria-hidden="true"><span></span></i><strong>我的工作流</strong><small>仅自己可见</small></div></div>
              </div>
              <footer class="workflow-save-modal__actions">
                <button type="button" class="secondary" :disabled="workflowSaving" @click="closeWorkflowSaveModal">取消</button>
                <button type="submit" class="primary" :disabled="workflowSaving || !workflowSaveName.trim()"><span v-if="workflowSaving" class="workflow-save-spinner" aria-hidden="true"></span>{{ workflowSaving ? '保存中' : '保存' }}</button>
              </footer>
            </form>
          </div>
        </Transition>
        </Teleport>
        <button type="button" class="toolbar-history" title="历史记录" aria-label="历史记录" @click.stop="openHistoryModal">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M4 12a8 8 0 1 0 2.35-5.66L4 8.69" />
            <path d="M4 4v4.69h4.69" />
            <path d="M12 8v4l2.5 1.5" />
          </svg>
        </button>
        <span class="toolbar-separator" aria-hidden="true"></span>
        <button type="button" class="toolbar-action" :class="{ active: showShortcutModal }" title="快捷键" aria-label="快捷键" @click.stop="openShortcutModal">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <rect x="3.5" y="6" width="17" height="12" rx="2.5" />
            <path d="M7 10h.01M10 10h.01M13 10h.01M16 10h.01" />
            <path d="M7 14h10" />
          </svg>
        </button>
        <button type="button" class="toolbar-action toolbar-help" title="帮助" aria-label="帮助" @click.stop="openWorkflowHelp">
          <QuestionMarkCircleIcon aria-hidden="true" />
        </button>
      </div>

      <div v-if="showShortcutModal" class="shortcut-modal" @pointerdown.self="closeShortcutModal">
        <div class="shortcut-modal__panel" @pointerdown.stop @click.stop>
          <button type="button" class="shortcut-modal__close" aria-label="关闭快捷键" @click="closeShortcutModal">×</button>
          <section class="shortcut-section">
            <h3>创建</h3>
            <dl>
              <div><dt>新增节点</dt><dd><kbd>+</kbd></dd></div>
              <div><dt>新增 3D 文件</dt><dd>底部 <kbd>+</kbd> 选择 3D 文件</dd></div>
              <div><dt>新增图片节点</dt><dd>底部 <kbd>+</kbd> 选择图片</dd></div>
              <div><dt>新增文本工具</dt><dd>底部 <kbd>+</kbd> 选择文本工具</dd></div>
              <div><dt>创建连线</dt><dd>从节点右侧圆点拖到目标节点</dd></div>
              <div><dt>空白处创建下游</dt><dd>从节点右侧圆点拖到空白处</dd></div>
              <div><dt>工作流模板</dt><dd>底部模板按钮</dd></div>
              <div><dt>风格库</dt><dd>底部 <kbd>+</kbd> 选择风格库</dd></div>
            </dl>
          </section>
          <section class="shortcut-section">
            <h3>画布</h3>
            <dl>
              <div><dt>移动画布</dt><dd>鼠标左键拖动画布空白处</dd></div>
              <div><dt>框选节点</dt><dd><kbd>Space</kbd> + 鼠标左键拖动</dd></div>
              <div><dt>缩放画布</dt><dd>鼠标滚轮</dd></div>
              <div><dt>整理画布</dt><dd>底部整理按钮</dd></div>
              <div><dt>历史记录</dt><dd>底部历史按钮</dd></div>
            </dl>
          </section>
          <section class="shortcut-section">
            <h3>编辑</h3>
            <dl>
              <div><dt>复制选中节点</dt><dd><kbd>Ctrl</kbd><span>+</span><kbd>C</kbd></dd></div>
              <div><dt>粘贴到鼠标位置</dt><dd><kbd>Ctrl</kbd><span>+</span><kbd>V</kbd></dd></div>
              <div><dt>删除选中内容</dt><dd><kbd>Delete</kbd></dd></div>
              <div><dt>撤销</dt><dd><kbd>Ctrl</kbd><span>+</span><kbd>Z</kbd></dd></div>
              <div><dt>组合节点</dt><dd>框选多个节点后点组合节点</dd></div>
              <div><dt>解除组合</dt><dd>选中组合框后点解除组合</dd></div>
              <div><dt>编辑图片</dt><dd>选中图片后点上方画笔</dd></div>
              <div><dt>预览图片</dt><dd>点击图片上方预览按钮</dd></div>
            </dl>
          </section>
          <section class="shortcut-section">
            <h3>3D</h3>
            <dl>
              <div><dt>上传模型</dt><dd>拖入 stp、step、obj、fbx 文件</dd></div>
              <div><dt>控制角度</dt><dd>点击 3D 节点右上角预览/控制角度</dd></div>
              <div><dt>旋转模型</dt><dd>控制弹窗内拖动模型</dd></div>
              <div><dt>校正模型</dt><dd>右侧校正模型调节 X/Y/Z</dd></div>
              <div><dt>裁切控制图</dt><dd>右侧开启裁切并调节高度</dd></div>
              <div><dt>生成场景图</dt><dd>选择风格、景别、比例和清晰度后生成</dd></div>
            </dl>
          </section>
        </div>
      </div>

      <div v-if="showHistoryModal" class="history-modal" @pointerdown.self="closeHistoryModal">
        <div class="history-modal__panel">
          <header class="history-modal__header">
            <h3>历史记录</h3>
            <div class="history-modal__tools">
              <button type="button" @click="historyZoom = Math.max(60, historyZoom - 10)">−</button>
              <span>{{ historyZoom }}%</span>
              <button type="button" @click="historyZoom = Math.min(160, historyZoom + 10)">＋</button>
              <button type="button" class="history-modal__close" @click="closeHistoryModal">×</button>
            </div>
          </header>
          <div class="history-modal__body">
            <div v-if="!groupedHistoryAssets.length" class="history-empty">
              暂无历史记录
            </div>
            <section v-for="group in groupedHistoryAssets" :key="group.date" class="history-group">
              <h4>{{ group.date }}</h4>
              <div class="history-grid" :style="{ '--history-thumb-scale': historyZoom / 100 }">
                <button
                  v-for="asset in group.items"
                  :key="asset.url"
                  type="button"
                  @click.stop
                >
                  <video
                    v-if="asset.type === 'video'"
                    :src="resolveImageUrl(asset.url)"
                    muted
                    playsinline
                    preload="metadata"
                    class="history-card__video"
                  ></video>
                  <img v-else v-lazy-src="thumbnailImage(asset.url, 360)" alt="历史记录">
                  <span class="history-card__shade" aria-hidden="true"></span>
                  <span class="history-card__delete" @click.stop="hideHistoryAsset(asset.url)" title="移除">×</span>
                  <span class="history-card__actions">
                    <em @click.stop="openPreview(asset.url, '历史记录', asset.type)">查看</em>
                    <em v-if="asset.type !== 'video'" @click.stop="useHistoryAsset(asset.url)">使用</em>
                  </span>
                </button>
              </div>
            </section>
          </div>
        </div>
      </div>

      <div v-if="showCanvasStyleLibrary" class="style-library-modal" @pointerdown.self="closeCanvasStyleLibrary">
        <div class="style-library-panel">
          <header class="style-library-header">
            <div>
              <span>风格库</span>
              <h3>{{ canvasStyleLibraryTitle }}</h3>
              <small v-if="canvasStyleLibraryMode === 'model-style'">选择一张参考图后，会绑定到当前 3D 场景流程</small>
            </div>
            <div class="style-library-scene-switch">
              <button
                type="button"
                :class="{ active: canvasStyleLibraryScene === 'kitchen' }"
                @click="switchCanvasStyleLibraryScene('kitchen')"
              >
                厨房
              </button>
              <button
                type="button"
                :class="{ active: canvasStyleLibraryScene === 'bathroom' }"
                @click="switchCanvasStyleLibraryScene('bathroom')"
              >
                浴室
              </button>
            </div>
            <button type="button" @click="closeCanvasStyleLibrary">×</button>
          </header>
          <div class="style-library-body">
            <aside class="style-library-tags">
              <button
                v-for="tag in canvasStyleLibraryTagsWithRefs"
                :key="tag.id"
                type="button"
                :class="{ active: canvasStyleLibraryTagId === tag.id }"
                @click="canvasStyleLibraryTagId = tag.id"
              >
                <strong>{{ tag.name }}</strong>
                <span>{{ tag.references?.length || 0 }}</span>
              </button>
              <div v-if="canvasStyleLibraryLoading" class="style-library-empty">加载中...</div>
              <div v-else-if="!canvasStyleLibraryTagsWithRefs.length" class="style-library-empty">暂无风格</div>
            </aside>
            <section class="style-library-refs">
              <button
                v-for="ref in canvasStyleLibraryRefs"
                :key="ref.id"
                type="button"
                class="style-library-card"
                @click="useCanvasStyleReference(ref)"
              >
                <img v-lazy-src="thumbnailImage(styleReferenceImage(ref), 360)" alt="风格参考">
              </button>
              <div v-if="!canvasStyleLibraryLoading && canvasStyleLibrarySelectedTag && !canvasStyleLibraryRefs.length" class="style-library-empty">该风格暂无参考图</div>
              <div v-if="!canvasStyleLibraryLoading && !canvasStyleLibrarySelectedTag" class="style-library-empty">请选择左侧风格</div>
            </section>
          </div>
        </div>
      </div>

      <div v-if="showCanvasMaterialLibrary" class="style-library-modal" @pointerdown.self="closeCanvasMaterialLibrary">
        <div class="style-library-panel material-library-panel">
          <header class="style-library-header">
            <div>
              <span>材质库</span>
              <h3>产品材质</h3>
              <small>选择一个材质后，会在画布创建只读材质参考节点</small>
            </div>
            <button type="button" @click="closeCanvasMaterialLibrary">×</button>
          </header>
          <div class="material-library-body">
            <div v-if="canvasMaterialLibraryLoading" class="style-library-empty">加载中...</div>
            <div v-else-if="!modelSceneMaterialOptions.length" class="style-library-empty">暂无材质</div>
            <section v-else class="material-library-grid">
              <button
                v-for="material in modelSceneMaterialOptions"
                :key="material.value"
                type="button"
                class="material-library-card"
                @click="useCanvasMaterialReference(material)"
              >
                <span class="material-library-card__image">
                  <img v-if="material.imageUrl" v-lazy-src="thumbnailImage(material.imageUrl, 360)" :alt="material.label">
                  <em v-else>{{ material.label.charAt(0) }}</em>
                </span>
                <strong>{{ material.label }}</strong>
              </button>
            </section>
          </div>
        </div>
      </div>

      <div v-if="batchMaterialPickerNode" class="style-library-modal" @pointerdown.self="closeBatchMaterialLibrary">
        <div class="style-library-panel material-library-panel batch-material-picker">
          <header class="style-library-header">
            <div>
              <span>批量换色</span>
              <h3>选择目标材质</h3>
              <small>可多选，换色节点会按选择顺序生成对应图片</small>
            </div>
            <button type="button" @click="closeBatchMaterialLibrary">×</button>
          </header>
          <div class="material-library-body">
            <div v-if="canvasMaterialLibraryLoading" class="style-library-empty">加载中...</div>
            <div v-else-if="!modelSceneMaterialOptions.length" class="style-library-empty">暂无材质</div>
            <section v-else class="material-library-grid">
              <button
                v-for="material in modelSceneMaterialOptions"
                :key="material.value"
                type="button"
                class="material-library-card"
                :class="{ selected: isBatchMaterialSelected(batchMaterialPickerNode, material.value) }"
                @click="toggleBatchRecolorMaterial(batchMaterialPickerNode, material)"
              >
                <span class="material-library-card__image">
                  <img v-if="material.imageUrl" v-lazy-src="thumbnailImage(material.imageUrl, 360)" :alt="material.label">
                  <em v-else>{{ material.label.charAt(0) }}</em>
                </span>
                <strong>{{ material.label }}</strong>
              </button>
            </section>
          </div>
          <footer class="batch-material-picker__footer">
            <span>已选 {{ batchRecolorNodeState(batchMaterialPickerNode).materials.length }} 个材质</span>
            <button type="button" @click="closeBatchMaterialLibrary">完成</button>
          </footer>
        </div>
      </div>

      <div v-if="error" class="floating-error" role="alert">
        <span>{{ error }}</span>
        <button type="button" aria-label="关闭错误提示" title="关闭" @click="error = ''">×</button>
      </div>

      <div v-if="modelPreviewOpen" class="model-modal" @click.self="closeModelPreview">
        <div class="model-modal__content">
          <div class="model-modal__header">
            <div>
              <strong>{{ fileName || '3D 模型预览' }}</strong>
              <span>视角 X {{ formatAngle(currentAngle.x) }} / Y {{ formatAngle(currentAngle.y) }}</span>
            </div>
            <button type="button" class="image-modal__close" @click="closeModelPreview" aria-label="关闭">×</button>
          </div>
          <div class="model-modal__body">
            <div ref="modelViewerModalRef" class="model-modal__viewer"></div>
            <div v-if="modelNodeState(activeModelNodeId).loading" class="model-modal-loading">
              <span class="loading-ring-sm"></span>
              <strong>{{ modelNodeState(activeModelNodeId).loadingMessage || '模型加载中...' }}</strong>
              <p>正在解析 3D 文件并生成预览，复杂模型可能需要更久。</p>
            </div>
            <div v-else-if="modelNodeState(activeModelNodeId).error" class="model-modal-loading model-modal-loading--error">
              <strong>模型预览失败</strong>
              <p>{{ modelNodeState(activeModelNodeId).error }}</p>
              <button type="button" @click="openModelUpload(activeModelNodeId)">重新上传</button>
            </div>
            <div v-else-if="!isModelNodeVisiblyLoaded(activeModelNodeId)" class="model-modal-loading">
              <strong>模型还没有加载完成</strong>
              <p>请先上传 3D 文件，或等待模型解析完成后再控制角度。</p>
            </div>
            <div class="model-modal-hint">
              左键旋转 · 滚轮缩放 · Ctrl+左键平移
            </div>
            <div class="model-orientation-panel" @pointerdown.stop @click.stop>
              <div class="model-orientation-panel__scroll">
                <div class="model-orientation-section">
                  <div class="model-orientation-panel__head">
                    <strong>校正模型</strong>
                    <span>只旋转产品，不旋转地面</span>
                  </div>
                  <label v-for="axis in modelCorrectionAxes" :key="axis.key" class="model-orientation-row">
                    <span>{{ axis.label }}</span>
                    <button type="button" @click="nudgeModelCorrection(axis.key, -90)">-90°</button>
                    <input
                      type="range"
                      min="-180"
                      max="180"
                      step="1"
                      :value="activeModelCorrection[axis.key]"
                      @input="setModelCorrectionAxis(axis.key, $event.target.value, { persist: false })"
                      @change="setModelCorrectionAxis(axis.key, $event.target.value)"
                    >
                    <button type="button" @click="nudgeModelCorrection(axis.key, 90)">+90°</button>
                    <em>{{ Math.round(activeModelCorrection[axis.key]) }}°</em>
                  </label>
                  <div class="model-orientation-actions">
                    <button type="button" @click="regroundModel">重新落地</button>
                    <button type="button" @click="resetModelCorrection">重置校正</button>
                  </div>
                </div>
                <div class="model-cut-panel">
                  <div class="model-cut-panel__head">
                    <strong>控制图裁切</strong>
                    <span>裁掉低于高度线的部分</span>
                  </div>
                  <label class="model-cut-toggle">
                    <input
                      type="checkbox"
                      :checked="activeModelCut.enabled"
                      @change="setModelCutEnabled($event.target.checked)"
                    >
                    <span>启用裁切</span>
                  </label>
                  <label class="model-orientation-row model-cut-row">
                    <span>高度</span>
                    <button type="button" @click="nudgeModelCut(-MODEL_CUT_NUDGE_STEP)">-</button>
                    <input
                      type="range"
                      min="0"
                      :max="MODEL_CUT_MAX_HEIGHT"
                      :step="MODEL_CUT_NUDGE_STEP"
                      :disabled="!activeModelCut.enabled"
                      :value="activeModelCut.height"
                      @input="setModelCutHeight($event.target.value, { persist: false })"
                      @change="setModelCutHeight($event.target.value)"
                    >
                    <button type="button" @click="nudgeModelCut(MODEL_CUT_NUDGE_STEP)">+</button>
                    <em>{{ Math.round(activeModelCut.height * 100) }}%</em>
                  </label>
                </div>
                <div class="model-export-panel">
                  <div class="model-cut-panel__head">
                    <strong>控制图用途</strong>
                    <span>影响产品在控制图里的大小</span>
                  </div>
                  <div class="model-segment">
                    <button
                      type="button"
                      :class="{ active: modelNodeState(activeModelNodeId).controlMode !== 'scene' }"
                      @click="setModelControlMode('product')"
                    >
                      白底图
                    </button>
                    <button
                      type="button"
                      :class="{ active: modelNodeState(activeModelNodeId).controlMode === 'scene' }"
                      @click="setModelControlMode('scene')"
                    >
                      场景图（只支持水龙头）
                    </button>
                  </div>
                  <span class="model-field-label">产品材质</span>
                  <div class="model-material-options">
                    <button
                      v-for="material in modelSceneMaterialOptions"
                      :key="material.value"
                      type="button"
                      :class="{ active: modelNodeState(activeModelNodeId).sceneMaterial === material.value }"
                      @click="setModelMaterial(material.value)"
                    >
                      <i>
                        <img v-if="material.imageUrl" v-lazy-src="thumbnailImage(material.imageUrl, 96)" :alt="material.label">
                        <span v-else>{{ material.label.charAt(0) }}</span>
                      </i>
                      <strong>{{ material.label }}</strong>
                    </button>
                  </div>
                  <div v-if="modelNodeState(activeModelNodeId).controlMode === 'scene'" class="model-shot-options">
                    <span class="model-field-label">场景类型</span>
                    <div class="model-segment">
                      <button
                        type="button"
                        :class="{ active: modelNodeState(activeModelNodeId).sceneType === 'bathroom' }"
                        @click="setModelSceneType('bathroom')"
                      >
                        浴室场景
                      </button>
                      <button
                        type="button"
                        :class="{ active: modelNodeState(activeModelNodeId).sceneType === 'kitchen' }"
                        @click="setModelSceneType('kitchen')"
                      >
                        厨房场景
                      </button>
                    </div>
                    <span class="model-field-label">风格库</span>
                    <button type="button" class="model-style-picker" @click="openModelStyleLibrary(activeModelNodeId)">
                      <img
                        v-if="modelNodeState(activeModelNodeId).sceneStyleRef?.url"
                        v-lazy-src="thumbnailImage(modelNodeState(activeModelNodeId).sceneStyleRef.url, 240)"
                        alt="已选风格"
                      >
                      <strong>{{ modelNodeState(activeModelNodeId).sceneStyleRef?.tagName || '选择风格库' }}</strong>
                      <small>{{ modelNodeState(activeModelNodeId).sceneStyleRef?.url ? '点击更换风格参考' : '从风格库选择一张参考图' }}</small>
                    </button>
                    <span class="model-field-label">想生成的图片景别</span>
                    <div class="model-segment">
                      <button
                        v-for="shot in modelSceneShotOptions"
                        :key="shot.value"
                        type="button"
                        :class="{ active: modelNodeState(activeModelNodeId).sceneShot === shot.value }"
                        @click="setModelSceneShot(shot.value)"
                      >
                        {{ shot.label }}
                      </button>
                    </div>
                    <span class="model-field-label">图片比例</span>
                    <div class="model-aspect-options">
                      <button
                        v-for="ratio in modelSceneAspectRatioOptions()"
                        :key="ratio"
                        type="button"
                        :class="{ active: modelNodeState(activeModelNodeId).sceneAspectRatio === ratio }"
                        @click="setModelSceneAspectRatio(ratio)"
                      >
                        <span class="generate-aspect-icon" :class="aspectIconClass(ratio)"></span>
                        <strong>{{ ratio }}</strong>
                      </button>
                    </div>
                  </div>
                  <span class="model-field-label">清晰度</span>
                  <div class="model-segment">
                    <button
                      v-for="resolution in generateResolutionOptions"
                      :key="resolution"
                      type="button"
                      :class="{ active: modelNodeState(activeModelNodeId).sceneResolution === resolution }"
                      @click="setModelSceneResolution(resolution)"
                    >
                      {{ resolution }}
                    </button>
                  </div>
                </div>
              </div>
              <button
                type="button"
                class="preview-generate-btn model-modal__generate"
                :disabled="generating"
                @click.stop="generateModelWorkflow(activeModelNodeId)"
              >
                {{ modelGenerateButtonText(activeModelNodeId) }}
              </button>
              <button
                v-if="modelNodeState(activeModelNodeId).controlMode !== 'scene'"
                type="button"
                class="preview-generate-btn model-modal__generate model-modal__generate--secondary"
                :disabled="generating"
                @click.stop="generateModel360WhiteBg(activeModelNodeId)"
              >
                生成360°白底图
              </button>
            </div>
            <div class="view-cube" aria-label="3D 方位选择">
              <div class="view-cube__stage">
                <div
                  class="view-cube__body"
                  :style="{ transform: `rotateX(${-viewCubeRotation.x}deg) rotateY(${-viewCubeRotation.y}deg)` }"
                  @pointerdown.stop.prevent="startViewCubeDrag"
                  @pointermove.stop.prevent="dragViewCube"
                  @pointerup.stop.prevent="endViewCubeDrag"
                  @pointercancel.stop.prevent="endViewCubeDrag"
                >
                  <div
                    v-for="item in orientationViews"
                    :key="item.value"
                    class="view-cube__face"
                    :class="`view-cube__face--${item.value}`"
                  >
                    <button
                      v-for="cell in viewCubeCells"
                      :key="`${item.value}-${cell.row}-${cell.col}`"
                      type="button"
                      class="view-cube__cell"
                      :class="{ 'view-cube__cell--center': cell.row === 0 && cell.col === 0 }"
                      :data-view="item.value"
                      :data-row="cell.row"
                      :data-col="cell.col"
                      :title="`切换到${item.label}${cell.label}视角`"
                      @click.stop.prevent
                    >
                      <span v-if="cell.row === 0 && cell.col === 0">{{ item.label }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="previewImage.url" class="image-modal" @click.self="closePreview" @wheel.prevent="handlePreviewWheel">
        <button type="button" class="image-modal__close" @click="closePreview" aria-label="关闭">×</button>
        <video
          v-if="previewImage.type === 'video'"
          ref="previewVideoRef"
          :src="previewImage.url"
          class="image-modal__img"
          controls
          autoplay
          muted
          playsinline
        ></video>
        <img
          v-else
          :src="previewImage.url"
          :alt="previewImage.title"
          class="image-modal__img"
          :style="{ transform: `scale(${previewZoom})` }"
        >
        <div v-if="previewImage.type !== 'video'" class="image-modal__toolbar" @click.stop>
          <button type="button" @click="setPreviewZoom(previewZoom - 0.15)">−</button>
          <span>{{ Math.round(previewZoom * 100) }}%</span>
          <button type="button" @click="setPreviewZoom(previewZoom + 0.15)">+</button>
          <button type="button" @click="setPreviewZoom(1)">重置</button>
        </div>
      </div>
      <AssetPickerModal
        :visible="assetPickerVisible"
        :title="assetPickerTitle"
        :type-filter="assetPickerTypes"
        :source-types="assetPickerSourceTypes"
        @close="assetPickerVisible = false"
        @select="handleAssetPickerSelect"
      />
    </div>
  </MainLayout>
</template>

<script setup>
import { ref, shallowRef, computed, watch, onMounted, onUnmounted, nextTick, markRaw, toRaw } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  AdjustmentsHorizontalIcon,
  ArrowDownTrayIcon,
  ArrowLongRightIcon,
  ArrowPathIcon,
  ArrowUturnLeftIcon,
  ArrowUpTrayIcon,
  ArrowsPointingOutIcon,
  DocumentTextIcon,
  LightBulbIcon,
  MinusIcon,
  PencilSquareIcon,
  QuestionMarkCircleIcon,
  Squares2X2Icon,
  SunIcon,
  TrashIcon,
  ViewfinderCircleIcon,
  XMarkIcon
} from '@heroicons/vue/24/outline'
import { PaintBrushIcon, PlayIcon } from '@heroicons/vue/24/solid'
import JSZip from 'jszip'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import MainLayout from '@/components/layout/MainLayout.vue'
import AssetPickerModal from '@/components/assets/AssetPickerModal.vue'
import request, { downloadImage, fetchImageAsDataUrl, requestDownloadMetadata } from '@/utils/request'
import { sanitizeWorkflowTemplateState } from '@/utils/workflowTemplate'
import { visibleMeshPoints, centerCameraOnPoints } from '@/utils/modelExportFraming'
import { IMAGE_EDIT_BRUSH_DEFAULTS, drawSmoothBrushStroke } from '@/utils/imageEditBrush'
import { videoAspectRatioOptions as ARK_VIDEO_ASPECT_RATIOS, videoDurationOptions as ARK_VIDEO_DURATIONS, videoModelOptions as ARK_VIDEO_MODELS, videoResolutionsForModel as arkVideoResolutions } from '@/utils/videoGeneration'

const router = useRouter()
const authStore = useAuthStore()
const isWorkflowAdmin = computed(() => (authStore.user?.roles || []).some(role => role?.name === 'admin' || role === 'admin'))
const lazyImageObserver = typeof window !== 'undefined' && 'IntersectionObserver' in window
  ? new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return
        const img = entry.target
        const src = img.dataset.lazySrc
        if (src && img.getAttribute('src') !== src) {
          img.src = src
        }
        lazyImageObserver.unobserve(img)
      })
    }, {
      rootMargin: '360px'
    })
  : null

const vLazySrc = {
  mounted(el, binding) {
    setLazyImageSource(el, binding.value)
  },
  updated(el, binding) {
    if (binding.value !== binding.oldValue) {
      setLazyImageSource(el, binding.value)
    }
  },
  unmounted(el) {
    lazyImageObserver?.unobserve(el)
  }
}

// State
const viewerContainer = ref(null)
const modelViewerModalRef = ref(null)
const sceneStageRef = ref(null)
const fileInput = ref(null)
const sceneFileInput = ref(null)
const generateFileInput = ref(null)
const generateEditCanvasRef = ref(null)
const workflowCanvasRef = ref(null)
const workflowStageRef = ref(null)
const draftLinkPathRef = ref(null)
const generateUploadTargetNode = ref('')
const modelUploadTargetNode = ref('upload')
const assetPickerVisible = ref(false)
const assetPickerMode = ref('model')
const assetPickerTargetNode = ref('')
const loading = ref(false)
const loadingMessage = ref('加载中...')
const modelLoaded = ref(false)
const fileName = ref('')
const isDragging = ref(false)
const generating = ref(false)
const renderTaskMessage = ref('')
const error = ref('')
const resultImageUrl = ref('')
const resultUrls = ref({})
const canvasPanActive = ref(false)
const currentAngle = ref({ x: 0, y: 0, z: 0 })
const activeModelCorrection = ref({ x: 0, y: 0, z: 0 })
const activeModelCut = ref({ enabled: false, height: 0 })
const stpFileUrl = ref('')
const meshData = shallowRef(null)
const viewCubeRotation = ref({ x: -24, y: 36 })
const modelPreviewOpen = ref(false)
const previewImage = ref({ url: '', title: '', type: 'image' })
const previewVideoRef = ref(null)
const previewZoom = ref(1)
const activeModelNodeId = ref('upload')
const workspaceMode = ref('model')
const activeWorkflowNode = ref('upload')
const showNodeMenu = ref(false)
const showWorkflowTemplateMenu = ref(false)
const showWorkflowTemplateModal = ref(false)
const workflowTemplates = ref([])
const workflowTemplatesLoading = ref(false)
const workflowTemplateScope = ref('system')
const workflowTemplateSearch = ref('')
const workflowTemplateSearchInput = ref(null)
const workflowTemplateDeleteTarget = ref(null)
const workflowTemplateDeleting = ref(false)
const showWorkflowSaveModal = ref(false)
const workflowSaveGroupId = ref('')
const workflowSaveName = ref('')
const workflowSaveScope = ref('user')
const workflowSaving = ref(false)
const workflowSaveStatus = ref('')
const workflowSaveError = ref('')
const workflowSaveNameInput = ref(null)
const workflowSaveNodeCount = computed(() => {
  const group = groupBoxes.value.find(item => item.id === workflowSaveGroupId.value)
  return (group?.nodes || []).filter(nodeExists).length
})
const systemWorkflowTemplates = computed(() => workflowTemplates.value.filter(item => item.scope === 'system'))
const userWorkflowTemplates = computed(() => workflowTemplates.value.filter(item => item.scope === 'user'))
const visibleWorkflowTemplates = computed(() => {
  const list = workflowTemplateScope.value === 'system' ? systemWorkflowTemplates.value : userWorkflowTemplates.value
  const keyword = workflowTemplateSearch.value.toLowerCase()
  if (!keyword) return list
  return list.filter(item => `${item.name || ''} ${item.summary || ''}`.toLowerCase().includes(keyword))
})
const showShortcutModal = ref(false)
const newProjectName = ref('')
const showProjectNameModal = ref(false)
const deleteProjectTarget = ref(null)
const showHistoryModal = ref(false)
const historyZoom = ref(100)
const hiddenHistoryAssetUrls = ref(new Set())
const showStyleLibraryMenu = ref(false)
const showCanvasStyleLibrary = ref(false)
const showCanvasMaterialLibrary = ref(false)
const canvasMaterialLibraryLoading = ref(false)
const canvasStyleLibraryScene = ref('kitchen')
const canvasStyleLibraryMode = ref('insert')
const canvasStyleLibraryTargetNode = ref('')
const canvasStyleLibraryLoading = ref(false)
const canvasStyleLibraryTags = ref([])
const canvasStyleLibraryTagId = ref(null)
const projectNameDraft = ref('')
const projectNameModalInput = ref(null)
const renamingProjectId = ref('')
const renameProjectName = ref('')
const activeProjectId = ref('')
const projects = ref([])
const workflowNodes = ref({
  upload: false,
  output: false,
  scene: false,
  generate: false,
  final: false,
  style: false,
  text: false
})
const workflowDynamicNodes = ref([])
const workflowLlmNodeStates = ref({})
const workflowLinks = ref([])
const workflowHistoryAssets = ref([])
const activeWorkflowLink = ref('')
const pendingLinkFrom = ref('')
const draggingLink = ref(null)
const hoveredLinkTarget = ref('')
const linkCreateMenu = ref({ open: false, from: '', x: 0, y: 0, screenX: 0, screenY: 0 })
const workflowUndoStack = ref([])
const WORKFLOW_UNDO_LIMIT = 50
const selectionBox = ref({ active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 })
const selectedNodeIds = ref([])
const batchDownloadSelectMode = ref(false)
const batchDownloadState = ref({ loading: false, completed: 0, total: 0 })
const groupBoxes = ref([])
const workflowGroupRunStates = ref({})
const activeGroupId = ref('')
const workflowClipboard = ref(null)
let lastCanvasPointer = null
const canvasReady = ref(false)
const canvasView = ref({
  x: 0,
  y: 0,
  scale: 1
})
const viewportCheckTick = ref(0)
const nodePositions = ref({
  upload: { x: 120, y: 112, width: 420, height: 360 },
  output: { x: 1168, y: 112, width: 430, height: 320 },
  scene: { x: 120, y: 560, width: 272, height: 238 },
  generate: { x: 528, y: 518, width: 520, height: 282 },
  final: { x: 608, y: 560, width: 380, height: 302 },
  text: { x: 1110, y: 560, width: 460, height: 300 }
})
const sceneImage = ref({ url: '', name: '', width: 0, height: 0 })
const placement = ref({
  x: 50,
  y: 58,
  scale: 0.42,
  rotation: 0,
  opacity: 0.78
})
const drawingModels = ref([])
let modelPollTimer = null
const chatModels = ref([])
const modelSceneMaterialOptions = ref([])
const batchMaterialPickerNode = ref('')
const workflowGenerateModel = ref('')
const workflowGeneratePrompt = ref('')
const workflowGenerateCount = ref(1)
const workflowGenerateAspectRatio = ref('1:1')
const workflowGenerateResolution = ref('2K')
const workflowGenerateMenu = ref('')
const workflowGenerating = ref(false)
const workflowGenerationError = ref('')
const workflowGenerationProgress = ref(0)
const workflowGenerationCancelled = ref(false)
const workflowGenerationResult = ref({ url: '', images: [], taskIds: [], taskId: '', status: '' })
const showGenerationResults = ref(false)
const workflowGenerateNodeStates = ref({})
const workflowTextNodeStates = ref({})
const workflowModelNodeStates = ref({})
const workflowOutputNodeStates = ref({})
const workflowBatchRecolorNodeStates = ref({})
const workflowBatchTranslateNodeStates = ref({})
const workflowVideoNodeStates = ref({})
const imageSizeCache = ref({})
const originalImageSizeProbes = new Map()
const DEFAULT_GENERATE_EDIT_BRUSH_SIZE = IMAGE_EDIT_BRUSH_DEFAULTS.size
const GENERATE_EDIT_MASK_ALPHA = IMAGE_EDIT_BRUSH_DEFAULTS.opacity
const generateImageEditState = ref({
  node: '',
  tool: 'brush',
  color: IMAGE_EDIT_BRUSH_DEFAULTS.color,
  size: DEFAULT_GENERATE_EDIT_BRUSH_SIZE,
  drawing: false,
  startX: 0,
  startY: 0,
  lastX: 0,
  lastY: 0,
  undoStack: [],
  redoStack: []
})
const generateCountOptions = [1, 2, 3, 4]
const generateAspectRatioOptions = ['1:1', '1:2', '2:1', '9:16', '16:9', '3:4', '4:3', '3:2', '2:3', '5:4', '4:5', '21:9', '9:21', '1:4', '1:8', '4:1', '8:1']
const GPT_IMAGE_ASPECT_RATIO_OPTIONS = ['1:1', '1:2', '2:1', '9:16', '16:9', '3:4', '4:3', '3:2', '2:3', '5:4', '4:5', '21:9', '9:21']
const GEMINI_FLASH_ASPECT_RATIO_OPTIONS = ['1:1', '2:3', '3:2', '3:4', '4:3', '4:5', '5:4', '9:16', '16:9', '1:4', '1:8', '4:1', '8:1', '21:9']
const GEMINI_PRO_ASPECT_RATIO_OPTIONS = ['1:1', '2:3', '3:2', '3:4', '4:3', '4:5', '5:4', '9:16', '16:9', '21:9']
const generateResolutionOptions = ['1K', '2K', '4K']
const videoModelOptions = ARK_VIDEO_MODELS
const videoResolutionOptions = ['480p', '720p', '1080p']
const videoDurationOptions = ARK_VIDEO_DURATIONS
const videoAspectRatioOptions = ARK_VIDEO_ASPECT_RATIOS
const WORKFLOW_HELP_URL = 'https://eqjictxoun9.feishu.cn/wiki/Sn0Swcm9IiS8aLkJGQmc0JbznPb?from=from_copylink'
const GENERATED_RESULT_NODE_GAP_X = 92
const GENERATED_RESULT_NODE_GAP_Y = 64
const MODEL_FILE_EXTENSIONS = ['stp', 'step', 'obj', 'fbx']
const MODEL_PREVIEW_BASE_COLOR = '#303236'
const assetPickerTitle = computed(() => {
  if (assetPickerMode.value === 'model') return '从资产库选择 3D 模型'
  if (assetPickerMode.value === 'node') return '从资产库添加节点'
  return '从资产库选择参考图'
})
const assetPickerTypes = computed(() => {
  if (assetPickerMode.value === 'model') return ['模型文件', 'STP模型']
  if (assetPickerMode.value === 'node') return ['模型文件', 'STP模型', '图片', '产品图', '场景图', '背景', '风格参考图', '色块', '参考方案图']
  return ['图片', '产品图', '场景图', '背景', '风格参考图', '色块', '参考方案图']
})
const assetPickerSourceTypes = computed(() => {
  if (assetPickerMode.value === 'model') return ['asset', 'product_spu_model', 'product_sku_model']
  if (assetPickerMode.value === 'node') return ['asset', 'product_spu_model', 'product_sku_model', 'scene_style_reference', 'product_color_family', 'reference_gallery_asset', 'product_spu_attachment']
  return ['asset', 'scene_style_reference', 'product_color_family', 'reference_gallery_asset', 'product_spu_attachment']
})
const modelCorrectionAxes = [
  { key: 'x', label: 'X 轴' },
  { key: 'y', label: 'Y 轴' },
  { key: 'z', label: 'Z 轴' }
]
const modelCorrectionAxisVectors = {
  x: new THREE.Vector3(1, 0, 0),
  y: new THREE.Vector3(0, 1, 0),
  z: new THREE.Vector3(0, 0, 1)
}
const modelSceneShotOptions = [
  { value: 'close', label: '近景' },
  { value: 'medium', label: '中景' },
  { value: 'wide', label: '远景' }
]
const modelSceneStyleOptions = [
  { value: 'american_modern_classic', label: '美式现代经典', prompt: '美式现代经典室内空间，深蓝或暖灰墙面、白色墙砖、木质柜体、石材台面、金属细节、精致但真实的住宅空间' },
  { value: 'modern_luxury', label: '现代轻奢', prompt: '现代轻奢室内空间，大理石台面、低饱和中性色、干净线条、金属细节、柔和高级灯光、整洁宽敞的空间' },
  { value: 'nordic_minimal', label: '北欧极简', prompt: '北欧极简室内空间，浅色木纹、白色墙面、简洁柜体、自然采光、干净柔和、空间通透' },
  { value: 'japanese_natural', label: '日式自然', prompt: '日式自然室内空间，温润木质、浅米色石材、简洁留白、柔和自然光、安静克制的空间氛围' },
  { value: 'industrial', label: '工业风', prompt: '工业风室内空间，深灰墙面、黑色金属、混凝土或石材质感、线条硬朗、低调冷峻的室内光线' }
]
const batchTranslateLanguageOptions = [
  { value: 'en', label: '英语' },
  { value: 'de', label: '德语' },
  { value: 'fr', label: '法语' },
  { value: 'it', label: '意大利语' },
  { value: 'es', label: '西班牙语' },
  { value: 'nl', label: '荷兰语' },
  { value: 'pl', label: '波兰语' },
  { value: 'sv', label: '瑞典语' },
  { value: 'zh', label: '简体中文' }
]
const IMAGE_TO_PROMPT_FIXED_TEXT = '请仔细分析这张图片，生成一段可直接用于 AI 作图工具重新创作类似作品的中文文生图提示词。提示词必须覆盖：主体内容、主体外观与材质、场景设定、风格参考、色调色彩、构图视角、光线氛围、重要细节补充。请尽量根据画面可见信息描述，不要编造不可见品牌、文字或人物身份；如果某些信息不确定，用概括性描述。必须输出一段完整提示词文本，即使图片较简单也不要留空。不要输出 JSON、Markdown、标题、项目符号或额外解释。'
const PRODUCT_GEOMETRY_RULES = [
  '1. 第1张结构图锁定外轮廓、部件数量/位置、连接关系、比例、截面和可见面；第2张深度图只补充厚度、遮挡和前后层级。',
  '2. 参考图中扁平直壁、平行边界、矩形或圆角矩形截面的部件必须保持平面和原厚宽比；禁止圆柱化、椭圆化、鼓面化或扩大倒角。只有参考图明确为圆形的部件才可呈圆形，高光渐变不能改变几何。',
  '3. 保持所有端面和边缘完整闭合；参考图未显示的槽、孔、缺口、接缝或部件一律不添加。不用常见水龙头款式补全或重设计。'
].join('\n')
const IMAGE_OPTIMIZATION_SUGGESTION_TEXT = `你是一名跨境电商运营负责人，专门审核商品主图、主图副图和 A+ 详情图。请只根据图片可见内容，找出最值得修改的问题，并直接告诉用户哪里改、怎么改。
先在心里完整理解这张图：它卖的是什么产品、画面主卖点是什么、图片文案想表达什么、用户看完应获得什么购买理由。不要输出这段理解过程。
再从运营角度审查：主卖点是否讲清、外语文案是否地道且有转化力、图文是否匹配、信息层级是否清楚、用户是否会误解、移动端是否能快速看懂、产品/场景/局部标注是否可信。不要凭空猜测平台、品牌、销量、价格、认证或不可见功能。
如果图片里有外语文案，必须先识别原文语言；替代文案必须使用原图同一种语言，禁止把德语、波兰语、英语等外语改成中文。中文只用于说明“哪里改、怎么改”。
只输出 3 到 5 条最重要的修改点，每行一条，格式固定为“位置/问题：怎么改”。
涉及文案时，直接给同语言替代表达，例如德语图就给德语改法；不确定表达是否地道时，写“建议语种运营复核”。
只说确实需要改的地方，不为了凑条数编问题，不输出地区判断，不解释分析过程，不写长段原因，不输出标题、Markdown、JSON、表格、总结或泛泛夸奖。`

// Three.js
let scene, camera, renderer, controls, mesh, modelRoot, animationId
let modelRenderPending = false
let sceneGroundGrid = null
let renderedModelNodeId = ''
let modelLoadSeq = 0
let webglRendererError = ''
const modelMeshLoadPromises = new Map()
let modelUploadRunSeq = 0
const modelUploadRuns = new Map()
let placementDrag = null
let canvasPan = null
let nodeDrag = null
let suppressNodeClick = false
let isRestoringProject = false
let projectSaveTimer = null
let projectSaveInFlight = false
let projectSaveQueued = false
let projectStateVersion = 0
let projectLastAppliedSaveVersion = 0
let canvasSaveTimer = null
let layoutSaveTimer = null
let viewportCheckTimer = null
let canvasFloatingScaleTimer = null
let projectDraftTimer = null
let errorDismissTimer = null
let canvasPanFrame = null
let canvasWheelFrame = null
let nodeDragFrame = null
let selectionMoveFrame = null
let groupDragFrame = null
let pendingCanvasPanEvent = null
let pendingCanvasWheel = null
let pendingNodeDragEvent = null
let pendingSelectionMoveEvent = null
let pendingGroupDragEvent = null
let selectionDrag = null
let selectionMove = null
let groupDrag = null
let textNodeResize = null
let lastReadonlyTextClick = { node: '', time: 0, x: 0, y: 0 }
let generateEditBaseSnapshot = null
let generateEditStrokePoints = []
let isSpacePressed = false
let isShiftPressed = false
let liveCanvasView = { x: 0, y: 0, scale: 1 }
let lastCanvasInteractionAt = 0
const textNodeProgressTimers = new Map()
const llmNodeProgressTimers = new Map()
const textNodeInputSyncTimers = new Map()
const llmNodeInputSyncTimers = new Map()
const ossUploadCache = new Map()
const modelViewerRefs = new Map()

watch(error, (message) => {
  if (errorDismissTimer) window.clearTimeout(errorDismissTimer)
  errorDismissTimer = null
  if (!message) return
  errorDismissTimer = window.setTimeout(() => {
    error.value = ''
    errorDismissTimer = null
  }, 6000)
})

const DEFAULT_WORKFLOW_NODES = {
  upload: false,
  output: false,
  scene: false,
  generate: false,
  final: false,
  style: false,
  text: false
}
const DEFAULT_NODE_POSITIONS = {
  upload: { x: 120, y: 112, width: 420, height: 360 },
  output: { x: 1168, y: 112, width: 430, height: 320 },
  scene: { x: 120, y: 560, width: 272, height: 238 },
  generate: { x: 528, y: 518, width: 520, height: 282 },
  final: { x: 608, y: 560, width: 380, height: 302 },
  style: { x: 980, y: 520, width: 300, height: 220 },
  text: { x: 1320, y: 520, width: 460, height: 300 },
  llm: { x: 1320, y: 520, width: 460, height: 300 },
  video: { x: 1420, y: 560, width: 520, height: 282 },
  'batch-recolor': { x: 980, y: 188, width: 400, height: 396 },
  'batch-translate': { x: 1420, y: 188, width: 420, height: 432 }
}
const DEFAULT_CANVAS_VIEW = { x: 0, y: 0, scale: 1 }

const RENDER_RESOLUTION = 2048
const CONTROL_RENDER_RESOLUTION = 2560
const PREVIEW_FOCAL_LENGTH = 135
const EXPORT_TARGET_FILL = 0.92
const EXPORT_SAFE_NDC = 0.97
const MODEL_SCENE_SHOT_FILL = {
  close: 0.48,
  medium: 0.36,
  wide: 0.26
}
const MODEL_SCENE_SHOT_OFFSET_Y = {
  close: -0.08,
  medium: -0.1,
  wide: -0.08
}
const MODEL_SCENE_COMPOSITE_PLACEMENT = {
  close: { x: 50, y: 72, scale: 0.18, heightScale: 0.24 },
  medium: { x: 50, y: 70, scale: 0.12, heightScale: 0.17 },
  wide: { x: 50, y: 70, scale: 0.12, heightScale: 0.17 }
}
const MODEL_CUT_MAX_HEIGHT = 0.98
const MODEL_CUT_NUDGE_STEP = 0.01
const MODEL_CUT_RESPONSE_POWER = 1.5
const MODEL_360_WHITE_BG_COUNT = 24
const BATCH_NODE_CONCURRENCY = 4
const VIEW_CUBE_MAX_PITCH = 82
const orientationViews = [
  { value: 'front', label: '\u524d', direction: [0, 0, 1], up: [0, 1, 0], right: [1, 0, 0] },
  { value: 'back', label: '\u540e', direction: [0, 0, -1], up: [0, 1, 0], right: [-1, 0, 0] },
  { value: 'right', label: '\u53f3', direction: [1, 0, 0], up: [0, 1, 0], right: [0, 0, -1] },
  { value: 'left', label: '\u5de6', direction: [-1, 0, 0], up: [0, 1, 0], right: [0, 0, 1] },
  { value: 'top', label: '\u9876', direction: [0, 1, 0], up: [0, 0, -1], right: [1, 0, 0] },
  { value: 'bottom', label: '\u5e95', direction: [0, -1, 0], up: [0, 0, 1], right: [1, 0, 0] }
]
const viewCubeCells = [
  { row: -1, col: -1, label: '\u5de6\u4e0a' },
  { row: -1, col: 0, label: '\u4e0a' },
  { row: -1, col: 1, label: '\u53f3\u4e0a' },
  { row: 0, col: -1, label: '\u5de6' },
  { row: 0, col: 0, label: '\u6b63' },
  { row: 0, col: 1, label: '\u53f3' },
  { row: 1, col: -1, label: '\u5de6\u4e0b' },
  { row: 1, col: 0, label: '\u4e0b' },
  { row: 1, col: 1, label: '\u53f3\u4e0b' }
]
let viewCubeDrag = null

function clampViewCubePitch(value) {
  return THREE.MathUtils.clamp(value, -VIEW_CUBE_MAX_PITCH, VIEW_CUBE_MAX_PITCH)
}

function restoreModelControls() {
  if (controls) {
    controls.enabled = modelPreviewOpen.value
    syncControlsState()
  }
}

function formatAngle(rad) {
  return Math.round((rad * 180 / Math.PI) % 360) + '°'
}

function currentViewerContainer() {
  const modalRef = Array.isArray(modelViewerModalRef.value) ? modelViewerModalRef.value[0] : modelViewerModalRef.value
  return modelPreviewOpen.value ? modalRef : modelViewerRefs.get(activeModelNodeId.value) || viewerContainer.value
}

function viewerContainerSize(container) {
  if (!container) return { width: 0, height: 0 }
  const rect = container.getBoundingClientRect?.()
  return {
    width: Math.round(container.clientWidth || rect?.width || 0),
    height: Math.round(container.clientHeight || rect?.height || 0)
  }
}

function isViewerContainerReady(container) {
  const size = viewerContainerSize(container)
  return size.width > 2 && size.height > 2
}

async function waitForViewerContainerReady(container, attempts = 10) {
  for (let i = 0; i < attempts; i++) {
    if (isViewerContainerReady(container)) return true
    await nextTick()
    await new Promise(resolve => requestAnimationFrame(resolve))
  }
  return isViewerContainerReady(container)
}

function setModelViewerRef(node, el) {
  if (el) modelViewerRefs.set(node, el)
  else modelViewerRefs.delete(node)
}

function hasRenderedModelMesh() {
  return !!(scene?.getObjectByName?.('stp-model-root')?.getObjectByName?.('stp-model'))
}

function isModelNodeVisiblyLoaded(node = activeModelNodeId.value || 'upload') {
  return !!node && renderedModelNodeId === node && hasRenderedModelMesh()
}

function markRenderedModelLoaded(node = activeModelNodeId.value || 'upload') {
  if (!isModelNodeVisiblyLoaded(node)) return
  const state = ensureModelNodeState(node)
  if (state.modelLoaded && !state.loading && !state.error) return
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [node]: {
      ...state,
      modelLoaded: true,
      loading: false,
      error: ''
    }
  }
}

function clearRenderedModelScene(options = {}) {
  if (options.cancelLoad !== false) modelLoadSeq += 1
  if (scene) {
    const oldRoot = scene.getObjectByName('stp-model-root')
    if (oldRoot) {
      disposeObject3D(oldRoot)
      scene.remove(oldRoot)
    }
    const oldMesh = scene.getObjectByName('stp-model')
    if (oldMesh) {
      disposeObject3D(oldMesh)
      scene.remove(oldMesh)
    }
  }
  mesh = null
  modelRoot = null
  modelLoaded.value = false
  renderedModelNodeId = ''
  if (options.render && renderer && scene && camera) {
    renderer.render(scene, camera)
  }
}

function activateModelNode(node) {
  const nextNode = node || 'upload'
  const previousNode = activeModelNodeId.value
  if (!isRestoringProject && previousNode && previousNode !== nextNode) {
    persistActiveModelPreviewState(previousNode)
  }
  if (activeModelNodeId.value !== nextNode && renderedModelNodeId && renderedModelNodeId !== nextNode) {
    clearRenderedModelScene({ render: true, cancelLoad: false })
  }
  activeModelNodeId.value = nextNode
  const state = ensureModelNodeState(activeModelNodeId.value)
  fileName.value = state.fileName
  stpFileUrl.value = state.stpFileUrl
  meshData.value = state.meshData
  modelLoaded.value = !!state.modelLoaded
  activeModelCorrection.value = normalizeModelCorrection(state.correction)
  activeModelCut.value = normalizeModelCut(state.cut)
}

async function selectModelNode(node) {
  if (suppressNodeClick) {
    suppressNodeClick = false
    return
  }
  const targetNode = node || 'upload'
  activateModelNode(targetNode)
  setActiveWorkflowNode(targetNode)
}

function moveRendererToContainer(container) {
  if (!container || !renderer || !camera) return false
  if (renderer.domElement.parentNode !== container) {
    renderer.domElement.parentNode?.removeChild(renderer.domElement)
    container.appendChild(renderer.domElement)
  }
  setupModelControls(modelPreviewOpen.value)
  if (!onResize()) return false
  renderer.render(scene, camera)
  return true
}

function setupModelControls(enabled) {
  if (!camera || !renderer || !scene) return
  const previousTarget = controls?.target?.clone?.() || getMeshFrame()?.center || new THREE.Vector3(0, 0, 0)
  if (controls) {
    controls.dispose()
    controls = null
  }

  camera.up.set(0, 1, 0)
  controls = new OrbitControls(camera, renderer.domElement)
  controls.rotateSpeed = 0.82
  controls.zoomSpeed = 0.9
  controls.panSpeed = 0.65
  controls.enableRotate = true
  controls.enableZoom = true
  controls.enablePan = true
  controls.enableDamping = true
  controls.dampingFactor = 0.08
  controls.screenSpacePanning = true
  controls.minPolarAngle = THREE.MathUtils.degToRad(0.5)
  controls.maxPolarAngle = THREE.MathUtils.degToRad(179.5)
  syncModelControlMouseButtons()
  controls.minDistance = 1
  controls.maxDistance = 20
  controls.target.copy(previousTarget)
  controls.enabled = enabled
  controls.addEventListener('end', () => {
    updateAngleFromCamera()
    updateViewCubeFromCamera()
    saveActiveModelCameraState({ persist: true })
  })
  controls.addEventListener('change', requestModelRender)
  syncControlsState()
}

function requestModelRender() {
  if (modelRenderPending || !renderer || !scene || !camera) return
  modelRenderPending = true
  animationId = window.requestAnimationFrame(() => {
    modelRenderPending = false
    camera?.up.set(0, 1, 0)
    const changed = controls?.update?.() || false
    if (changed && controls && camera && !viewCubeDrag) {
      updateAngleFromCamera()
      updateViewCubeFromCamera()
    }
    if (renderer && scene && camera) renderer.render(scene, camera)
    if (changed && controls?.enableDamping) requestModelRender()
  })
}

function syncModelControlMouseButtons() {
  if (!controls) return
  controls.mouseButtons = {
    LEFT: isShiftPressed ? THREE.MOUSE.PAN : THREE.MOUSE.ROTATE,
    MIDDLE: THREE.MOUSE.DOLLY,
    RIGHT: THREE.MOUSE.PAN
  }
}

function ensureSceneGroundGrid(size = 12, divisions = 48) {
  if (!scene) return
  if (sceneGroundGrid) {
    scene.remove(sceneGroundGrid)
    sceneGroundGrid.geometry?.dispose?.()
    sceneGroundGrid.material?.dispose?.()
  }
  sceneGroundGrid = new THREE.GridHelper(size, divisions, 0x686c73, 0x383b40)
  sceneGroundGrid.name = 'model-ground-grid'
  sceneGroundGrid.position.y = currentModelGroundY()
  sceneGroundGrid.material.transparent = true
  sceneGroundGrid.material.opacity = modelPreviewOpen.value ? 0.46 : 0.24
  scene.add(sceneGroundGrid)
}

function syncModelViewerBackground() {
  if (!scene || !renderer) return
  if (modelPreviewOpen.value) {
    scene.background = new THREE.Color(0x8a8a8a)
    renderer.setClearColor(0x8a8a8a, 1)
    renderer.domElement.style.background = '#8a8a8a'
  } else {
    scene.background = null
    renderer.setClearColor(0x000000, 0)
    renderer.domElement.style.background = 'transparent'
  }
  if (sceneGroundGrid?.material) {
    sceneGroundGrid.material.opacity = modelPreviewOpen.value ? 0.46 : 0.24
  }
}

function currentModelGroundY() {
  return 0
}

function ensureModelRoot() {
  if (!scene) return null
  if (modelRoot && modelRoot.parent === scene) return modelRoot
  modelRoot = new THREE.Group()
  modelRoot.name = 'stp-model-root'
  scene.add(modelRoot)
  return modelRoot
}

function disposeObject3D(object) {
  if (!object) return
  object.traverse?.(child => {
    child.geometry?.dispose?.()
    if (Array.isArray(child.material)) child.material.forEach(material => material?.dispose?.())
    else child.material?.dispose?.()
  })
}

function modelCorrectionQuaternion(correction) {
  const quaternion = new THREE.Quaternion()
  for (const axis of ['x', 'y', 'z']) {
    const angle = THREE.MathUtils.degToRad(Number(correction[axis]) || 0)
    if (!angle) continue
    const axisQuaternion = new THREE.Quaternion().setFromAxisAngle(modelCorrectionAxisVectors[axis], angle)
    quaternion.premultiply(axisQuaternion)
  }
  return quaternion.normalize()
}

function applyModelCorrection({ refit = false, persist = true, reground = true } = {}) {
  if (!modelRoot) return
  const correction = normalizeModelCorrection(activeModelCorrection.value)
  modelRoot.quaternion.copy(modelCorrectionQuaternion(correction))
  if (reground) {
    regroundModel({ refit, persist: false })
  } else {
    modelRoot.updateMatrixWorld(true)
    mesh?.updateMatrixWorld?.(true)
  }
  if (persist) {
    const node = activeModelNodeId.value || 'upload'
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [node]: {
        ...ensureModelNodeState(node),
        correction
      }
    }
    saveActiveProject({ persist: false })
  }
  renderer?.render?.(scene, camera)
}

function setModelCorrectionAxis(axis, value, options = {}) {
  if (!['x', 'y', 'z'].includes(axis)) return
  const nextValue = THREE.MathUtils.clamp(Number(value) || 0, -180, 180)
  activeModelCorrection.value = {
    ...activeModelCorrection.value,
    [axis]: nextValue
  }
  applyModelCorrection(options)
}

function nudgeModelCorrection(axis, delta) {
  const current = Number(activeModelCorrection.value[axis]) || 0
  let next = current + delta
  while (next > 180) next -= 360
  while (next < -180) next += 360
  setModelCorrectionAxis(axis, next)
}

function resetModelCorrection() {
  activeModelCorrection.value = { x: 0, y: 0, z: 0 }
  applyModelCorrection()
}

function regroundModel({ refit = false, persist = true, refreshGrid = true, preserveCamera = !refit } = {}) {
  if (!mesh || !modelRoot) return 0
  const previousPosition = modelRoot.position.clone()
  modelRoot.updateMatrixWorld(true)
  const box = new THREE.Box3().setFromObject(modelRoot)
  if (!box.isEmpty()) {
    const center = new THREE.Vector3()
    box.getCenter(center)
    modelRoot.position.x -= center.x
    modelRoot.position.y -= modelGroundReferenceY(box)
    modelRoot.position.z -= center.z
    modelRoot.updateMatrixWorld(true)
    if (refreshGrid) updateGroundGridForModel()
    else updateGroundGridHeight()
    if (preserveCamera) translateModelCameraY(modelRoot.position.y - previousPosition.y)
    if (refit) frameMeshToView(modelRoot)
  }
  if (persist) saveActiveProject({ persist: false })
  return modelRoot.position.y - previousPosition.y
}

function modelGroundReferenceY(box) {
  const cut = normalizeModelCut(activeModelCut.value)
  if (!cut.enabled) return box.min.y
  return box.min.y + (box.max.y - box.min.y) * normalizedModelCutHeightRatio(cut)
}

function updateGroundGridForModel() {
  if (!modelRoot) return
  modelRoot.updateMatrixWorld(true)
  const groundedBox = new THREE.Box3().setFromObject(modelRoot)
  if (groundedBox.isEmpty()) return
  const groundedSize = new THREE.Vector3()
  groundedBox.getSize(groundedSize)
  const gridSize = Math.max(8, Math.max(groundedSize.x, groundedSize.z) * 3.2, groundedSize.y * 2.2)
  ensureSceneGroundGrid(gridSize, 56)
  updateGroundGridHeight()
}

function updateGroundGridHeight() {
  if (sceneGroundGrid) sceneGroundGrid.position.y = currentModelGroundY()
}

function getModelWorldBox() {
  const target = modelRoot || mesh
  if (!target) return null
  target.updateMatrixWorld(true)
  const box = new THREE.Box3().setFromObject(target)
  return box.isEmpty() ? null : box
}

function currentModelCutWorldY() {
  const box = getModelWorldBox()
  if (!box) return 0
  return box.min.y + (box.max.y - box.min.y) * normalizedModelCutHeightRatio(activeModelCut.value)
}

function normalizedModelCutHeightRatio(cut) {
  const height = normalizeModelCut(cut).height
  return Math.pow(height, MODEL_CUT_RESPONSE_POWER)
}

function currentModelCutPlane() {
  const cut = normalizeModelCut(activeModelCut.value)
  if (!cut.enabled) return null
  return new THREE.Plane(new THREE.Vector3(0, 1, 0), -currentModelCutWorldY())
}

function applyModelCutToMaterial(material) {
  if (!material) return
  const plane = currentModelCutPlane()
  const materials = Array.isArray(material) ? material : [material]
  materials.forEach(item => {
    item.clippingPlanes = plane ? [plane] : []
    item.clipIntersection = false
    item.needsUpdate = true
  })
}

function applyModelCut({ persist = false } = {}) {
  if (renderer) renderer.localClippingEnabled = true
  regroundModel({ refit: false, persist: false, refreshGrid: false, preserveCamera: true })
  mesh?.traverse?.(child => applyModelCutToMaterial(child.material))
  updateGroundGridHeight()
  if (persist) {
    const node = activeModelNodeId.value || 'upload'
    persistActiveModelPreviewState(node)
    saveActiveProject({ persist: false })
  }
  renderer?.render?.(scene, camera)
}

function setModelCutEnabled(enabled) {
  activeModelCut.value = {
    ...normalizeModelCut(activeModelCut.value),
    enabled
  }
  applyModelCut({ persist: true })
}

function setModelCutHeight(value, options = {}) {
  activeModelCut.value = {
    ...normalizeModelCut(activeModelCut.value),
    height: THREE.MathUtils.clamp(Number(value) || 0, 0, MODEL_CUT_MAX_HEIGHT)
  }
  applyModelCut({ persist: options.persist !== false })
}

function nudgeModelCut(delta) {
  const cut = normalizeModelCut(activeModelCut.value)
  setModelCutHeight(cut.height + delta)
}

function updateActiveModelExportOptions(patch, node = activeModelNodeId.value || 'upload', options = {}) {
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [node]: normalizeModelNodeState({
      ...ensureModelNodeState(node),
      ...patch
    })
  }
  if (options.persist !== false) saveActiveProject({ persist: false })
}

function setModelControlMode(mode) {
  const controlMode = normalizeModelControlMode(mode)
  updateActiveModelExportOptions({ controlMode })
}

function setModelSceneType(type) {
  const sceneType = normalizeModelSceneType(type)
  removeModelStyleReferenceNode(activeModelNodeId.value || 'upload')
  updateActiveModelExportOptions({
    sceneType,
    sceneStyleRef: null,
    sceneStyleNode: ''
  })
}

function setModelSceneShot(shot) {
  updateActiveModelExportOptions({ sceneShot: normalizeModelSceneShot(shot), controlMode: 'scene' })
}

function setModelSceneStyle(style) {
  updateActiveModelExportOptions({ sceneStyle: normalizeModelSceneStyle(style), controlMode: 'scene' })
}

function setModelMaterial(material) {
  const node = activeModelNodeId.value || 'upload'
  removeModelMaterialReferenceNode(node)
  updateActiveModelExportOptions({
    sceneMaterial: normalizeModelSceneMaterial(material),
    sceneMaterialNode: ''
  }, node)
  if (isCurrentPreviewRenderingNode(node)) requestModelRender()
}

function setModelSceneAspectRatio(ratio) {
  updateActiveModelExportOptions({
    sceneAspectRatio: normalizeModelSceneAspectRatio(ratio),
    controlMode: 'scene'
  })
}

function setModelSceneResolution(resolution) {
  updateActiveModelExportOptions({
    sceneResolution: generateResolutionOptions.includes(resolution) ? resolution : '2K'
  })
}

function normalizeProductColorFamily(item = {}) {
  const id = item.id ?? item.value ?? item.name
  const label = String(item.name || item.label || '').trim()
  if (!id || !label) return null
  const imageUrl = resolveImageUrl(item.image_url || item.imageUrl || item.url || '')
  return {
    value: String(id),
    label,
    imageUrl,
    prompt: `${label}材质/色系，参考后台材质色块图的颜色、表面质感、反射强度和金属纹理`
  }
}

async function loadModelSceneMaterials() {
  try {
    const res = await request.get('/product-color-families')
    modelSceneMaterialOptions.value = (res.data || [])
      .map(normalizeProductColorFamily)
      .filter(Boolean)
  } catch {
    modelSceneMaterialOptions.value = []
  }
}

async function ensureModelSceneMaterialsLoaded() {
  if (modelSceneMaterialOptions.value.length) return
  canvasMaterialLibraryLoading.value = true
  try {
    await loadModelSceneMaterials()
  } finally {
    canvasMaterialLibraryLoading.value = false
  }
}

function isCurrentPreviewRenderingNode(targetNode) {
  return modelPreviewOpen.value
    && activeModelNodeId.value === targetNode
    && renderedModelNodeId === targetNode
    && hasRenderedModelMesh()
}

function persistActiveModelPreviewState(node = activeModelNodeId.value || 'upload') {
  if (!node) return
  syncControlsState()
  camera?.updateMatrixWorld?.(true)
  const cameraState = normalizeModelCamera(currentModelCameraState())
  const currentState = ensureModelNodeState(node)
  const nextState = normalizeModelNodeState({
    ...currentState,
    correction: normalizeModelCorrection(activeModelCorrection.value),
    cut: normalizeModelCut(activeModelCut.value),
    camera: cameraState || currentState.camera
  })
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [node]: nextState
  }
  renderer?.render?.(scene, camera)
}

async function prepareRenderedModelForExport(targetNode) {
  if (isCurrentPreviewRenderingNode(targetNode)) {
    persistActiveModelPreviewState(targetNode)
    return
  }
  await ensureRenderedModelNode(targetNode)
  persistActiveModelPreviewState(targetNode)
}

async function ensureRenderedModelNode(node) {
  const targetNode = node || activeModelNodeId.value || 'upload'
  activateModelNode(targetNode)
  await nextTick()
  const container = currentViewerContainer()
  if (!renderer || !scene || !camera) {
    initThreeJS()
    await nextTick()
  } else if (renderer.domElement?.parentNode !== container) {
    moveRendererToContainer(container)
    await nextTick()
  } else {
    onResize()
  }
  const state = ensureModelNodeState(targetNode)
  if (state.meshData && (renderedModelNodeId !== targetNode || !hasRenderedModelMesh())) {
    await loadStpModel(state.meshData, targetNode)
  } else if (!state.meshData && state.stpFileUrl) {
    await reloadMeshPreviewFromUrl(state.stpFileUrl, targetNode)
  }
  if (renderedModelNodeId !== targetNode || !hasRenderedModelMesh()) {
    throw new Error('目标 3D 节点模型还没有加载完成')
  }
}

async function openModelPreview(node = activeModelNodeId.value) {
  const targetNode = node || 'upload'
  const needsSwitchLoading = renderedModelNodeId && renderedModelNodeId !== targetNode
  if (needsSwitchLoading) {
    clearRenderedModelScene({ render: true, cancelLoad: false })
  }
  activateModelNode(targetNode)
  const state = ensureModelNodeState(targetNode)
  if (state.meshData && meshData.value !== state.meshData) {
    meshData.value = state.meshData
  }
  if (state.meshData && (needsSwitchLoading || renderedModelNodeId !== targetNode || !hasRenderedModelMesh())) {
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...state,
        loading: true,
        loadingMessage: '正在切换模型...',
        error: ''
      }
    }
  }
  modelPreviewOpen.value = true
  await nextTick()
  if (state.meshData || state.stpFileUrl) {
    try {
      await ensureRenderedModelNode(targetNode)
    } catch (e) {
      const message = modelPreviewFailureMessage(e)
      workflowModelNodeStates.value = {
        ...workflowModelNodeStates.value,
        [targetNode]: {
          ...ensureModelNodeState(targetNode),
          loading: false,
          error: message
        }
      }
    }
  }
  restoreModelControls()
  syncModelViewerBackground()
  if (ensureModelNodeState(targetNode).modelLoaded) {
    await refreshModelViewerFrame({ refit: !ensureModelNodeState(targetNode).camera })
  }
  onResize()
}

async function closeModelPreview() {
  persistActiveModelPreviewState(activeModelNodeId.value)
  saveActiveProject({ persist: false })
  modelPreviewOpen.value = false
  await nextTick()
  const container = currentViewerContainer()
  await waitForViewerContainerReady(container)
  moveRendererToContainer(container)
  restoreModelControls()
  syncModelViewerBackground()
}

function initThreeJS() {
  const container = currentViewerContainer()
  if (!container) return false
  if (scene && camera && renderer) {
    return renderer.domElement?.parentNode === container
      ? onResize()
      : moveRendererToContainer(container)
  }
  const size = viewerContainerSize(container)
  if (size.width <= 2 || size.height <= 2) return false

  if (animationId) {
    cancelAnimationFrame(animationId)
    animationId = null
    modelRenderPending = false
  }
  if (controls) {
    controls.dispose()
    controls = null
  }
  if (renderer) {
    renderer.dispose()
    renderer.forceContextLoss()
    renderer.domElement?.parentNode?.removeChild(renderer.domElement)
    renderer = null
  }

  scene = new THREE.Scene()
  sceneGroundGrid = null
  mesh = null
  modelRoot = null
  renderedModelNodeId = ''
  scene.background = modelPreviewOpen.value ? new THREE.Color(0x484b50) : null

  const w = size.width
  const h = size.height
  camera = new THREE.PerspectiveCamera(45, w / h, 0.1, 1000)
  camera.setFocalLength(PREVIEW_FOCAL_LENGTH)
  camera.position.set(0, 2.2, 6)

  try {
    renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true, powerPreference: 'high-performance' })
  } catch (primaryError) {
    console.warn('[STP] High-performance WebGL context unavailable, retrying with reduced settings:', primaryError)
    try {
      renderer = new THREE.WebGLRenderer({ antialias: false, alpha: true, powerPreference: 'default' })
    } catch (fallbackError) {
      webglRendererError = '当前浏览器无法创建 WebGL 预览。请开启浏览器硬件加速后重启浏览器，并关闭占用 3D 图形资源的其他页面后重试。'
      scene = null
      camera = null
      renderer = null
      error.value = webglRendererError
      console.error('[STP] WebGL renderer initialization failed:', fallbackError)
      return false
    }
  }
  webglRendererError = ''
  renderer.setSize(w, h)
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.25))
  renderer.setClearColor(modelPreviewOpen.value ? 0x484b50 : 0x000000, modelPreviewOpen.value ? 1 : 0)
  renderer.outputColorSpace = THREE.SRGBColorSpace
  renderer.localClippingEnabled = true
  renderer.domElement.style.background = modelPreviewOpen.value ? '#484b50' : 'transparent'
  container.appendChild(renderer.domElement)

  setupModelControls(modelPreviewOpen.value)

  // 娣诲姞鍏夋簮锛岀‘淇濇ā鍨嬩粠鍚勪釜瑙掑害鍙
  const ambient = new THREE.AmbientLight(0xffffff, 0.75)
  scene.add(ambient)
  const hemisphere = new THREE.HemisphereLight(0xffffff, 0x444444, 0.5)
  scene.add(hemisphere)
  const dir = new THREE.DirectionalLight(0xffffff, 0.8)
  dir.position.set(5, 5, 5)
  scene.add(dir)
  const dir2 = new THREE.DirectionalLight(0xffffff, 0.5)
  dir2.position.set(-5, 3, -5)
  scene.add(dir2)
  const dir3 = new THREE.DirectionalLight(0xffffff, 0.3)
  dir3.position.set(0, -5, 0)
  scene.add(dir3)
  ensureSceneGroundGrid()
  syncModelViewerBackground()
  renderer.render(scene, camera)

  requestModelRender()

  // 绐楀彛璋冩暣
  window.removeEventListener('resize', onResize)
  window.addEventListener('resize', onResize)
  return true
}

function updateAngleFromCamera() {
  if (!camera) return
  const target = controls?.target || new THREE.Vector3(0, 0, 0)
  const pos = camera.position.clone().sub(target)
  const r = Math.sqrt(pos.x * pos.x + pos.y * pos.y + pos.z * pos.z)
  if (r <= 1e-6) return
  const x = Math.asin(pos.y / r)
  const y = Math.atan2(pos.x, pos.z)
  const current = currentAngle.value
  if (Math.abs(current.x - x) < 0.0005 && Math.abs(current.y - y) < 0.0005) return
  currentAngle.value = { x, y, z: 0 }
}

function updateViewCubeFromCamera() {
  if (!camera || !controls) return

  const direction = camera.position.clone().sub(controls.target)
  if (direction.lengthSq() <= 1e-8) return
  direction.normalize()

  const x = clampViewCubePitch(THREE.MathUtils.radToDeg(Math.asin(direction.y)))
  const y = THREE.MathUtils.radToDeg(Math.atan2(direction.x, direction.z))
  const current = viewCubeRotation.value
  if (Math.abs(current.x - x) < 0.03 && Math.abs(current.y - y) < 0.03) return
  viewCubeRotation.value = { x, y }
}

function onResize() {
  const container = currentViewerContainer()
  if (!container || !camera || !renderer) return false
  const { width: w, height: h } = viewerContainerSize(container)
  if (w <= 2 || h <= 2) return false
  camera.aspect = w / h
  camera.updateProjectionMatrix()
  renderer.setSize(w, h)
  controls?.handleResize?.()
  return true
}

async function refreshModelViewerFrame({ refit = false } = {}) {
  await nextTick()
  await new Promise(resolve => requestAnimationFrame(resolve))
  await new Promise(resolve => requestAnimationFrame(resolve))
  const container = currentViewerContainer()
  if (!await waitForViewerContainerReady(container)) return false
  if (!onResize()) return false
  syncModelViewerBackground()
  ensureSceneGroundGrid()
  let framed = true
  if (refit && (modelRoot || mesh)) {
    framed = frameMeshToView(modelRoot || mesh)
  } else {
    syncControlsState()
  }
  renderer?.render?.(scene, camera)
  return framed
}

function syncControlsState() {
  if (!controls || !camera) return
  controls.handleResize?.()
  camera.up.set(0, 1, 0)
  controls.update()
}

function translateModelCameraY(deltaY) {
  if (!camera || !controls || !Number.isFinite(deltaY) || Math.abs(deltaY) <= 1e-8) return
  camera.position.y += deltaY
  controls.target.y += deltaY
  camera.up.set(0, 1, 0)
  camera.lookAt(controls.target)
  camera.updateProjectionMatrix()
  syncControlsState()
  updateAngleFromCamera()
  updateViewCubeFromCamera()
}

function currentModelCameraState() {
  if (!camera || !controls) return null
  return {
    position: camera.position.toArray(),
    target: controls.target.toArray(),
    zoom: camera.zoom || 1
  }
}

function currentModelViewGuide() {
  if (!camera || !controls) return null
  const offset = camera.position.clone().sub(controls.target)
  if (offset.lengthSq() <= 1e-8) return null
  const horizontal = new THREE.Vector3(offset.x, 0, offset.z)
  const yaw = THREE.MathUtils.radToDeg(Math.atan2(offset.x, offset.z))
  const pitch = THREE.MathUtils.radToDeg(Math.atan2(offset.y, Math.max(horizontal.length(), 1e-6)))
  const absYaw = Math.abs(yaw)
  const side = absYaw < 35 ? '正面' : (yaw > 0 ? '右前方' : '左前方')
  const height = pitch >= 62
    ? '接近顶视高机位俯视'
    : (pitch >= 45
        ? '强俯视'
        : (pitch >= 28
            ? '明显俯视'
            : (pitch > 12 ? '轻微俯视' : (pitch < -8 ? '低机位仰视' : '平视'))))
  const yawText = yaw > 0
    ? '目标取景参考：相机尽量看到第1张参考图中产品当前可见的侧面，但产品仍按真实水槽/台盆安装方向摆放'
    : (yaw < 0 ? '目标取景参考：相机尽量看到第1张参考图中产品当前可见的侧面，但产品仍按真实水槽/台盆安装方向摆放' : '目标取景参考：相机尽量看到第1张参考图中的正向可见面，但产品仍按真实水槽/台盆安装方向摆放')
  const pitchText = pitch > 0
    ? `相机从上方俯看约 ${Math.round(Math.abs(pitch))}°`
    : (pitch < 0 ? `相机从下方仰看约 ${Math.round(Math.abs(pitch))}°` : '相机平视产品')
  return {
    yaw: Math.round(yaw),
    pitch: Math.round(pitch),
    side,
    height,
    pitchLevel: height,
    text: `${side}${height}视角`,
    detail: `${yawText}，${pitchText}`
  }
}

function saveActiveModelCameraState({ persist = false } = {}) {
  const cameraState = normalizeModelCamera(currentModelCameraState())
  const node = activeModelNodeId.value || 'upload'
  if (!cameraState || !node) return
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [node]: {
      ...ensureModelNodeState(node),
      camera: cameraState
    }
  }
  if (persist) saveActiveProject({ persist: false })
}

function restoreModelCameraState(cameraState) {
  const normalized = normalizeModelCamera(cameraState)
  if (!normalized || !camera || !controls) return false
  if (!normalized.position.every(Number.isFinite) || !normalized.target.every(Number.isFinite)) return false
  camera.position.fromArray(normalized.position)
  camera.zoom = normalized.zoom || 1
  camera.up.set(0, 1, 0)
  controls.target.fromArray(normalized.target)
  if (camera.position.distanceTo(controls.target) <= 1e-5) return false
  camera.lookAt(controls.target)
  camera.updateProjectionMatrix()
  syncControlsState()
  updateAngleFromCamera()
  updateViewCubeFromCamera()
  return true
}

function getMeshFrame() {
  const target = modelRoot || mesh
  if (!target) return null

  const box = cameraBoxForTarget(target)
  if (!box) return null

  const center = new THREE.Vector3()
  const size = new THREE.Vector3()
  box.getCenter(center)
  box.getSize(size)
  const maxSize = Math.max(size.x, size.y, size.z)
  if (maxSize <= 0) return null

  return { center, size, maxSize }
}

function isUsableModelBox(box) {
  if (!box || box.isEmpty()) return false
  const size = new THREE.Vector3()
  box.getSize(size)
  return Math.max(size.x, size.y, size.z) > 1e-6
}

function fullModelBoxForTarget(targetMesh) {
  if (!targetMesh) return null
  targetMesh.updateMatrixWorld?.(true)
  const box = new THREE.Box3().setFromObject(targetMesh)
  return isUsableModelBox(box) ? box : null
}

function cameraBoxForTarget(targetMesh, { preferVisible = true } = {}) {
  if (!targetMesh) return null
  if (preferVisible && (targetMesh === modelRoot || targetMesh === mesh)) {
    const visibleBox = getVisibleModelWorldBox()
    if (isUsableModelBox(visibleBox)) return visibleBox
  }
  return fullModelBoxForTarget(targetMesh)
}

function isModelFrameVisible(targetMesh = modelRoot || mesh) {
  if (!targetMesh || !camera) return false
  const box = cameraBoxForTarget(targetMesh)
  if (!box) return false

  const sphere = new THREE.Sphere()
  box.getBoundingSphere(sphere)
  if (!Number.isFinite(sphere.radius) || sphere.radius <= 0) return false

  camera.updateMatrixWorld(true)
  camera.updateProjectionMatrix()
  const frustum = new THREE.Frustum()
  const matrix = new THREE.Matrix4().multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
  frustum.setFromProjectionMatrix(matrix)

  const distance = camera.position.distanceTo(sphere.center)
  return frustum.intersectsSphere(sphere) && distance > sphere.radius * 0.35 && distance < sphere.radius * 120
}

function getViewDistance(frame, extraScale = 1.18) {
  if (!camera || !controls || !frame) return 3

  camera.setFocalLength(PREVIEW_FOCAL_LENGTH)
  camera.updateProjectionMatrix()
  const fitDistance = frame.maxSize / (2 * Math.tan(THREE.MathUtils.degToRad(camera.fov) / 2))
  const currentDistance = camera.position.distanceTo(controls.target)
  return Math.max(fitDistance * extraScale, currentDistance, 3)
}

function setCameraByDirection(direction, distanceScale = 1.18) {
  if (!camera || !controls) return

  const frame = getMeshFrame()
  if (!frame) return

  const viewDirection = direction.clone().normalize()
  const distance = getViewDistance(frame, distanceScale)
  camera.position.copy(frame.center).add(viewDirection.multiplyScalar(distance))
  camera.up.set(0, 1, 0)
  camera.near = Math.max(distance / 100, 0.01)
  camera.far = distance * 100
  camera.lookAt(frame.center)
  camera.updateProjectionMatrix()

  controls.target.copy(frame.center)
  syncControlsState()
  updateAngleFromCamera()
  updateViewCubeFromCamera()
  saveActiveModelCameraState()
}

function setCameraFromViewCubeRotation(rotation) {
  const yaw = THREE.MathUtils.degToRad(rotation.y)
  const pitch = THREE.MathUtils.degToRad(clampViewCubePitch(rotation.x))
  const direction = new THREE.Vector3(
    Math.sin(yaw) * Math.cos(pitch),
    Math.sin(pitch),
    Math.cos(yaw) * Math.cos(pitch)
  )
  setCameraByDirection(direction)
}

function rotateCameraAroundTarget(deltaYaw, deltaPitch) {
  if (!camera || !controls) return
  const target = controls.target.clone()
  const offset = camera.position.clone().sub(target)
  if (offset.lengthSq() <= 1e-8) return
  const worldUp = new THREE.Vector3(0, 1, 0)
  const yawQuat = new THREE.Quaternion().setFromAxisAngle(worldUp, deltaYaw)
  const right = new THREE.Vector3().crossVectors(worldUp, offset)
  if (right.lengthSq() <= 1e-8) right.set(1, 0, 0)
  else right.normalize()
  const pitchQuat = new THREE.Quaternion().setFromAxisAngle(right, deltaPitch)
  offset.applyQuaternion(yawQuat).applyQuaternion(pitchQuat)
  const spherical = new THREE.Spherical().setFromVector3(offset)
  spherical.phi = THREE.MathUtils.clamp(
    spherical.phi,
    THREE.MathUtils.degToRad(0.5),
    THREE.MathUtils.degToRad(179.5)
  )
  offset.setFromSpherical(spherical)
  camera.position.copy(target).add(offset)
  camera.up.copy(worldUp)
  camera.lookAt(target)
  camera.updateProjectionMatrix()
  controls.target.copy(target)
  syncControlsState()
  updateAngleFromCamera()
  updateViewCubeFromCamera()
  saveActiveModelCameraState()
}

function frameMeshToView(targetMesh, { persistCamera = false } = {}) {
  if (!targetMesh || !camera || !controls) return false

  const box = cameraBoxForTarget(targetMesh)
  if (!box) return false

  const center = new THREE.Vector3()
  const size = new THREE.Vector3()
  box.getCenter(center)
  box.getSize(size)

  const maxSize = Math.max(size.x, size.y, size.z)
  camera.setFocalLength(PREVIEW_FOCAL_LENGTH)
  camera.updateProjectionMatrix()
  const fitDistance = maxSize / (2 * Math.tan(THREE.MathUtils.degToRad(camera.fov) / 2))
  const distance = Math.max(fitDistance * 1.15, 3)

  const direction = new THREE.Vector3(0, 0.55, 1).normalize()
  camera.position.copy(center).add(direction.multiplyScalar(distance))
  camera.near = Math.max(distance / 100, 0.01)
  camera.far = distance * 100
  camera.updateProjectionMatrix()

  controls.target.copy(new THREE.Vector3(center.x, Math.max(size.y * 0.28, center.y * 0.45), center.z))
  controls.minDistance = Math.max(distance * 0.2, 0.5)
  controls.maxDistance = distance * 8
  syncControlsState()
  updateAngleFromCamera()
  updateViewCubeFromCamera()
  if (persistCamera) saveActiveModelCameraState()
  return true
}

function setFixedView(view, cell = { row: 0, col: 0 }) {
  if (!(modelRoot || mesh) || !camera || !controls) return

  const config = orientationViews.find(item => item.value === view)
  if (!config) return

  if (view === 'top' || view === 'bottom') {
    const pitch = THREE.MathUtils.degToRad(view === 'top' ? 89.5 : -89.5)
    const yaw = viewCubeRotation.value.y
    const yawRad = THREE.MathUtils.degToRad(yaw)
    const forward = new THREE.Vector3(Math.sin(yawRad), 0, Math.cos(yawRad)).normalize()
    const right = new THREE.Vector3(Math.cos(yawRad), 0, -Math.sin(yawRad)).normalize()
    const viewDirection = new THREE.Vector3(
      Math.sin(yawRad) * Math.cos(pitch),
      Math.sin(pitch),
      Math.cos(yawRad) * Math.cos(pitch)
    )
      .add(right.multiplyScalar(cell.col * 0.6))
      .add(forward.multiplyScalar(-cell.row * 0.6))
      .normalize()
    setCameraByDirection(viewDirection)
    return
  }

  const direction = new THREE.Vector3(...config.direction).normalize()
  const right = new THREE.Vector3(...config.right).normalize()
  const up = new THREE.Vector3(...config.up).normalize()
  const viewDirection = direction
    .add(right.multiplyScalar(cell.col * 0.42))
    .add(up.clone().multiplyScalar(-cell.row * 0.42))
    .normalize()
  setCameraByDirection(viewDirection)
}

function startViewCubeDrag(event) {
  const cellEl = event.target.closest?.('.view-cube__cell')
  viewCubeDrag = {
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    baseX: viewCubeRotation.value.x,
    baseY: viewCubeRotation.value.y,
    moved: false,
    view: cellEl?.dataset?.view || '',
    row: Number(cellEl?.dataset?.row || 0),
    col: Number(cellEl?.dataset?.col || 0)
  }
  event.currentTarget.setPointerCapture?.(event.pointerId)
  window.addEventListener('pointerup', endViewCubeDrag, { once: true })
  window.addEventListener('pointercancel', cancelViewCubeDrag, { once: true })
}

function dragViewCube(event) {
  if (!viewCubeDrag || viewCubeDrag.pointerId !== event.pointerId) return

  const dx = event.clientX - viewCubeDrag.startX
  const dy = event.clientY - viewCubeDrag.startY
  if (Math.abs(dx) + Math.abs(dy) > 4) {
    viewCubeDrag.moved = true
  }
  if (!viewCubeDrag.moved) {
    return
  }
  const nextDx = dx - (viewCubeDrag.lastDx || 0)
  const nextDy = dy - (viewCubeDrag.lastDy || 0)
  viewCubeDrag.lastDx = dx
  viewCubeDrag.lastDy = dy
  rotateCameraAroundTarget(
    THREE.MathUtils.degToRad(nextDx * 0.42),
    THREE.MathUtils.degToRad(nextDy * 0.42)
  )
}

function endViewCubeDrag(event) {
  if (!viewCubeDrag || viewCubeDrag.pointerId !== event.pointerId) return
  const dragState = viewCubeDrag
  event.currentTarget?.releasePointerCapture?.(event.pointerId)
  window.removeEventListener('pointerup', endViewCubeDrag)
  window.removeEventListener('pointercancel', cancelViewCubeDrag)
  viewCubeDrag = null
  if (!dragState.moved && dragState.view) {
    setFixedView(dragState.view, { row: dragState.row, col: dragState.col })
  }
  restoreModelControls()
}

function cancelViewCubeDrag(event) {
  if (viewCubeDrag && event?.pointerId && viewCubeDrag.pointerId !== event.pointerId) return
  window.removeEventListener('pointerup', endViewCubeDrag)
  window.removeEventListener('pointercancel', cancelViewCubeDrag)
  viewCubeDrag = null
  restoreModelControls()
}

async function loadStpModel(meshData, node = activeModelNodeId.value) {
  const targetNode = node || 'upload'
  const loadSeq = ++modelLoadSeq
  activateModelNode(targetNode)
  if (!meshData) {
    console.error('[STP] loadStpModel: meshData is missing')
    throw new Error('模型网格数据为空')
  }

  try {
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...ensureModelNodeState(targetNode),
        meshData,
        loading: true,
        loadingMessage: '渲染模型...',
        error: ''
      }
    }
    await nextTick()

    const container = currentViewerContainer()
    if (!container) {
      console.error('[STP] FATAL: viewerContainer is null')
      throw new Error('模型预览容器不存在')
    }
    const containerReady = await waitForViewerContainerReady(container)
    if (!containerReady) {
      throw new Error('模型预览容器尺寸未准备好')
    }

    if (!renderer || !scene || !camera) {
      if (!initThreeJS()) throw new Error(webglRendererError || '模型预览初始化失败')
    } else if (renderer.domElement?.parentNode !== container) {
      if (!moveRendererToContainer(container)) throw new Error('模型预览容器尺寸未准备好')
    } else if (!onResize()) {
      throw new Error('模型预览容器尺寸未准备好')
    }
    if (!scene || !renderer || !camera) throw new Error('模型预览初始化失败')

    // 解析顶点数据 - 支持两种格式：
    // 1. 扁平数组: [x1, y1, z1, x2, y2, z2, ...]
    // 2. 嵌套数组: [[x1,y1,z1], [x2,y2,z2], ...]
    const rawVerts = meshData.vertices
    let positions
    if (rawVerts.length > 0 && typeof rawVerts[0] === 'number') {
      positions = rawVerts
    } else {
      positions = new Float32Array(rawVerts.length * 3)
      for (let i = 0; i < rawVerts.length; i += 1) {
        const offset = i * 3
        positions[offset] = rawVerts[i][0]
        positions[offset + 1] = rawVerts[i][1]
        positions[offset + 2] = rawVerts[i][2]
      }
    }

    // 解析索引数据 - 支持两种格式：
    // 1. 扁平数组: [i1, j1, k1, i2, j2, k2, ...]
    // 2. 嵌套数组: [[i1,j1,k1], [i2,j2,k2], ...]
    const rawTris = meshData.triangles
    let indices
    if (rawTris.length > 0 && typeof rawTris[0] === 'number') {
      indices = rawTris
    } else {
      indices = new Uint32Array(rawTris.length * 3)
      for (let i = 0; i < rawTris.length; i += 1) {
        const offset = i * 3
        indices[offset] = rawTris[i][0]
        indices[offset + 1] = rawTris[i][1]
        indices[offset + 2] = rawTris[i][2]
      }
    }

    // 计算顶点数（根据格式）
    const vertCount = rawVerts.length > 0 && typeof rawVerts[0] === 'number'
      ? rawVerts.length / 3
      : rawVerts.length

    // 验证索引是否越界
    let maxIndex = -Infinity
    let minIndex = Infinity
    for (const idx of indices) {
      if (idx > maxIndex) maxIndex = idx
      if (idx < minIndex) minIndex = idx
    }
    if (maxIndex >= vertCount) {
      console.warn('[STP] WARNING: indices out of bounds! maxIndex:', maxIndex, 'vertCount:', vertCount)
    }
    if (minIndex < 0) {
      console.warn('[STP] WARNING: negative indices found!')
    }

    const isValidModelIndex = idx => Number.isInteger(idx) && idx >= 0 && idx < vertCount
    const safeIndices = []
    for (let i = 0; i < indices.length; i += 3) {
      const a = indices[i]
      const b = indices[i + 1]
      const c = indices[i + 2]
      if (isValidModelIndex(a) && isValidModelIndex(b) && isValidModelIndex(c)) safeIndices.push(a, b, c)
    }
    if (!safeIndices.length) {
      throw new Error('模型网格索引无效，无法创建 3D 预览')
    }

    const sourceGeometry = new THREE.BufferGeometry()
    sourceGeometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3))
    sourceGeometry.setIndex(safeIndices)
    sourceGeometry.computeVertexNormals()
    const sourceNormals = sourceGeometry.getAttribute('normal')
    const partKeys = []

    const expandedPositions = new Float32Array(safeIndices.length * 3)
    const expandedNormals = new Float32Array(safeIndices.length * 3)
    let expandedOffset = 0
    let expandedFaceIndex = 0
    for (let i = 0; i < indices.length; i += 3) {
      const a = indices[i]
      const b = indices[i + 1]
      const c = indices[i + 2]
      if (!isValidModelIndex(a) || !isValidModelIndex(b) || !isValidModelIndex(c)) continue
      for (let vertex = 0; vertex < 3; vertex += 1) {
        const idx = vertex === 0 ? a : (vertex === 1 ? b : c)
        const base = idx * 3
        expandedPositions[expandedOffset] = positions[base]
        expandedPositions[expandedOffset + 1] = positions[base + 1]
        expandedPositions[expandedOffset + 2] = positions[base + 2]
        expandedNormals[expandedOffset] = sourceNormals.getX(idx)
        expandedNormals[expandedOffset + 1] = sourceNormals.getY(idx)
        expandedNormals[expandedOffset + 2] = sourceNormals.getZ(idx)
        expandedOffset += 3
      }
      expandedFaceIndex += 1
    }
    sourceGeometry.dispose()
    if (!expandedOffset || !expandedFaceIndex) {
      throw new Error('模型网格索引无效，无法创建 3D 预览')
    }

    // 创建 BufferGeometry
    const geometry = new THREE.BufferGeometry()
    geometry.setAttribute('position', new THREE.Float32BufferAttribute(expandedPositions, 3))
    geometry.setAttribute('normal', new THREE.Float32BufferAttribute(expandedNormals, 3))

    const material = new THREE.MeshStandardMaterial({
      color: MODEL_PREVIEW_BASE_COLOR,
      metalness: 0.15,
      roughness: 0.45,
      side: THREE.DoubleSide,
      vertexColors: false
    })
    mesh = new THREE.Mesh(geometry, material)
    mesh.userData.partKeys = partKeys
    mesh.userData.partKeysReady = partKeys.length > 0
    mesh.userData.partKeySource = { meshPayload: meshData, indices, positions, vertCount }
    applyModelCutToMaterial(mesh.material)

    // 鐠嬪啳鐦敍姘剁崣鐠囦礁鍤戞担鏇氱秼
    geometry.computeBoundingBox()

    // 鏍规嵁鍖呭洿鐩掑眳涓苟缂╂斁
    const bb = meshData.bounding_box
    if (bb && bb.center && bb.size) {
      geometry.translate(-bb.center[0], -bb.center[1], -bb.center[2])
      geometry.computeBoundingBox()
      const maxSize = Math.max(bb.size[0], bb.size[1], bb.size[2])
      if (maxSize > 0) {
        mesh.scale.setScalar(2 / maxSize)
      }
    }
    const oldRoot = scene.getObjectByName('stp-model-root')
    if (oldRoot) {
      disposeObject3D(oldRoot)
      scene.remove(oldRoot)
    }
    modelRoot = null
    const root = ensureModelRoot()
    mesh.name = 'stp-model'
    root.add(mesh)
    if (loadSeq !== modelLoadSeq || activeModelNodeId.value !== targetNode) {
      root.remove(mesh)
      disposeObject3D(mesh)
      mesh = null
      return
    }
    activeModelCorrection.value = normalizeModelCorrection(ensureModelNodeState(targetNode).correction)
    activeModelCut.value = normalizeModelCut(ensureModelNodeState(targetNode).cut)
    applyModelCorrection({ refit: false, persist: false })
    applyModelCut()

    // 寮哄埗绔嬪嵆娓叉煋涓€甯э紝纭繚妯″瀷绔嬪嵆鍙
    if (renderer && camera) {
      renderer.render(scene, camera)
    }

    const stateBeforeCamera = ensureModelNodeState(targetNode)
    const savedCameraState = stateBeforeCamera.camera || null
    const restoredCamera = restoreModelCameraState(savedCameraState)
    if (!restoredCamera || !isModelFrameVisible(root)) {
      const framed = frameMeshToView(root)
      if (!framed) throw new Error('模型已解析，但无法计算可预览范围')
    }
    if (!isModelFrameVisible(root)) {
      const framed = frameMeshToView(root)
      if (!framed || !isModelFrameVisible(root)) {
        throw new Error('模型已解析，但当前相机无法看到模型')
      }
    }

    modelLoaded.value = true
    renderedModelNodeId = targetNode
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...ensureModelNodeState(targetNode),
        meshData,
        correction: normalizeModelCorrection(activeModelCorrection.value),
        modelLoaded: true,
        loading: false,
        error: ''
      }
    }
    markRenderedModelLoaded(targetNode)
    await nextTick()
    const visibleContainer = currentViewerContainer()
    if (!visibleContainer) throw new Error('模型预览容器不存在')
    const visibleReady = await waitForViewerContainerReady(visibleContainer)
    if (!visibleReady) throw new Error('模型预览容器尺寸未准备好')
    if (renderer?.domElement?.parentNode !== visibleContainer) {
      if (!moveRendererToContainer(visibleContainer)) throw new Error('模型预览容器尺寸未准备好')
    } else if (!onResize()) {
      throw new Error('模型预览容器尺寸未准备好')
    }
    const refreshed = await refreshModelViewerFrame({ refit: !isModelFrameVisible(root) })
    if (!refreshed) throw new Error('模型预览刷新失败')
    if (!isModelFrameVisible(root)) {
      throw new Error('模型已解析，但预览区仍无法看到模型')
    }
  } catch (e) {
    const message = `模型渲染失败：${e.message || '无法创建 3D 预览'}`
    error.value = message
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...ensureModelNodeState(targetNode),
        loading: false,
        modelLoaded: false,
        error: message
      }
    }
    console.error('[STP] Mesh render error:', e)
    throw new Error(message)
  }
}

function onDragOver(e) {
  isDragging.value = true
}

function onDragLeave() {
  isDragging.value = false
}

function onDrop(e) {
  isDragging.value = false
  const file = e.dataTransfer.files[0]
  if (file) handleFile(file)
}

function onFileSelect(e) {
  const file = e.target.files[0]
  if (file) handleFile(file, modelUploadTargetNode.value)
  e.target.value = ''
}

function openModelUpload(node) {
  modelUploadTargetNode.value = node || 'upload'
  activateModelNode(modelUploadTargetNode.value)
  fileInput.value?.click()
}

function openAssetPicker(mode, node) {
  assetPickerMode.value = mode
  assetPickerTargetNode.value = mode === 'node'
    ? ''
    : (node || (mode === 'model' ? activeModelNodeId.value : activeGenerateNodeId()))
  showStyleLibraryMenu.value = false
  showNodeMenu.value = false
  if (mode === 'model') {
    activateModelNode(assetPickerTargetNode.value || 'upload')
  } else if (assetPickerTargetNode.value) {
    setActiveWorkflowNode(assetPickerTargetNode.value)
  }
  assetPickerVisible.value = true
}

async function handleAssetPickerSelect(asset) {
  if (!asset?.fileUrl) return
  if (assetPickerMode.value === 'node') {
    await createWorkflowNodeFromAsset(asset)
  } else if (assetPickerMode.value === 'model') {
    await useModelAsset(asset)
  } else {
    useImageAsset(asset)
  }
}

function assetFileExtension(asset = {}) {
  return String(asset.fileName || asset.name || asset.fileUrl || '')
    .split('?')[0]
    .split('#')[0]
    .split('.')
    .pop()
    ?.toLowerCase() || ''
}

function isModelAsset(asset = {}) {
  const ext = assetFileExtension(asset)
  return MODEL_FILE_EXTENSIONS.includes(ext) ||
    ['模型文件', 'STP模型'].includes(asset.type) ||
    asset.fileType === 'model'
}

function isImageAsset(asset = {}) {
  const ext = assetFileExtension(asset)
  return ['png', 'jpg', 'jpeg', 'webp', 'gif', 'bmp', 'avif', 'svg'].includes(ext) ||
    ['图片', '产品图', '场景图', '背景', '风格参考图', '色块', '参考方案图'].includes(asset.type) ||
    asset.fileType === 'image'
}

async function createWorkflowNodeFromAsset(asset) {
  if (isModelAsset(asset)) {
    const node = addWorkflowNode('upload', {
      position: defaultVisibleNodePosition('upload')
    })
    if (!node) return
    assetPickerTargetNode.value = node
    await useModelAsset(asset)
    return
  }
  if (isImageAsset(asset)) {
    createImageNodeFromAsset(asset)
    return
  }
  error.value = '该资产暂不支持创建画布节点'
}

async function useModelAsset(asset) {
  const targetNode = assetPickerTargetNode.value || activeModelNodeId.value || 'upload'
  const ext = String(asset.fileName || asset.fileUrl || '').split('?')[0].split('.').pop()?.toLowerCase()
  const supportedModelAsset = MODEL_FILE_EXTENSIONS.includes(ext) || asset.fileType === 'model' || ['模型文件', 'STP模型'].includes(asset.type)
  if (!supportedModelAsset) {
    error.value = '请选择 STP、STEP、OBJ 或 FBX 模型文件'
    return
  }
  const modelName = asset.fileName || asset.name || `模型文件.${ext}`
  error.value = ''
  assetPickerVisible.value = false
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [targetNode]: {
      ...ensureModelNodeState(targetNode),
      fileName: modelName,
      stpFileUrl: asset.fileUrl,
      meshData: null,
      modelLoaded: false,
      loading: false,
      loadingMessage: '点击控制角度加载模型',
      error: ''
    }
  }
  stpFileUrl.value = asset.fileUrl
  fileName.value = modelName
  syncDynamicUploadNodeFileState(targetNode, { fileName: modelName, stpFileUrl: asset.fileUrl })
  saveWorkflowStructureState({ persist: true })
}

function useImageAsset(asset) {
  const targetNode = assetPickerTargetNode.value || activeGenerateNodeId()
  if (!targetNode) return
  const state = ensureGenerateNodeState(targetNode)
  const replaceInput = isWorkflowImageInputNode(targetNode)
  const upload = {
    key: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    name: asset.name || asset.fileName || '资产图片',
    url: asset.fileUrl
  }
  if (replaceInput) clearDownstreamWorkflowRunResults(targetNode)
  updateGenerateNode(targetNode, {
    uploads: replaceInput ? [upload] : [...(state.uploads || []), upload],
    ...(replaceInput ? { uploadOnly: true, result: emptyGenerationResult(), editRef: null, editPreview: null } : {}),
    showResults: false,
    error: ''
  })
  setActiveWorkflowNode(targetNode)
  assetPickerVisible.value = false
  error.value = ''
}

function createImageNodeFromAsset(asset) {
  const imageUrl = resolveImageUrl(asset.fileUrl)
  if (!imageUrl) return
  const createdAt = new Date().toISOString()
  const node = addWorkflowNode('generate', {
    position: defaultVisibleNodePosition('generate'),
    state: {
      result: {
        url: imageUrl,
        images: [imageUrl],
        imageCreatedAt: [createdAt],
        imageSources: ['imported'],
        batchTitle: asset.name || asset.fileName || '资产图片',
        createdAt,
        status: 'imported'
      },
      uploads: [],
      showResults: false,
      error: ''
    }
  })
  if (!node) return
  assetPickerVisible.value = false
  error.value = ''
}

function handleFile(file, node = activeModelNodeId.value) {
  const ext = file.name.split('.').pop().toLowerCase()
  if (!MODEL_FILE_EXTENSIONS.includes(ext)) {
    error.value = '仅支持 .stp、.step、.obj 和 .fbx 文件'
    return
  }
  error.value = ''
  uploadAndLoad(file, node)
}

function modelPreviewFailureMessage(err) {
  const status = err?.response?.status
  const backendMessage = err?.response?.data?.message || err?.response?.data?.error || err?.message || ''
  if (backendMessage.includes('模型预览加载已取消')) return ''
  if (status === 413) return '3D 文件过大，服务器拒绝上传或解析，请压缩模型后重试。'
  if (status === 504 || err?.code === 'ECONNABORTED') return '3D 模型解析耗时过长，预览服务超时，请稍后重试或简化模型。'
  if (backendMessage.includes("No module named 'trimesh'") || backendMessage.includes('No module named "trimesh"')) {
    return 'FBX 预览依赖未安装：服务器缺少 trimesh，请安装 stp-renderer 依赖后重启 3D 解析服务。'
  }
  if (backendMessage.includes('assimp-capable runtime') || backendMessage.toLowerCase().includes('assimp')) {
    return 'FBX 预览依赖不完整：服务器需要 trimesh 和 assimp/pyassimp 支持，请补齐后重启 3D 解析服务。'
  }
  if (status >= 500) return `3D 模型解析服务异常${backendMessage ? `：${backendMessage}` : ''}`
  if (backendMessage.includes('没有解析出可预览的网格数据')) return backendMessage
  if (backendMessage.includes('任务已完成，但没有返回可渲染的网格数据')) return backendMessage
  if (backendMessage.startsWith('模型渲染失败')) return backendMessage
  return `模型预览解析失败${backendMessage ? `：${backendMessage}` : '，可能是文件结构过复杂、格式不完整或当前解析器暂不支持。'}`
}

function clearModelMeshLoadPromisesForNode(node) {
  if (!node) return
  const prefix = `${node}::`
  Array.from(modelMeshLoadPromises.keys()).forEach((key) => {
    if (key.startsWith(prefix)) modelMeshLoadPromises.delete(key)
  })
}

function beginModelUploadRun(node) {
  const token = `${Date.now()}-${++modelUploadRunSeq}`
  modelUploadRuns.set(node, token)
  return token
}

function isCurrentModelUploadRun(node, token) {
  return !!node && !!token && modelUploadRuns.get(node) === token
}

function cancelModelUploadRun(node) {
  if (!node) return
  modelUploadRuns.delete(node)
}

function assertCurrentModelUploadRun(node, token) {
  if (!isCurrentModelUploadRun(node, token)) {
    throw new Error('模型预览加载已取消')
  }
}

function isModelFile(file) {
  const ext = file?.name?.split('.').pop()?.toLowerCase()
  return MODEL_FILE_EXTENSIONS.includes(ext)
}

function imageFileToGenerateUpload(file) {
  if (!file?.type?.startsWith('image/')) {
    error.value = '请上传图片文件'
    return null
  }
  // 50MB 限制（与 AI 图像生成的参考图限制一致）
  const MAX_UPLOAD_SIZE = 50 * 1024 * 1024
  if (file.size > MAX_UPLOAD_SIZE) {
    error.value = `文件过大（${(file.size / 1024 / 1024).toFixed(1)}MB），请上传 50MB 以内的图片`
    return null
  }
  return {
    key: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    name: file.name || '参考图',
    url: URL.createObjectURL(file)
  }
}

function appendGenerateUpload(node, file) {
  const upload = imageFileToGenerateUpload(file)
  if (!upload) return false
  const state = ensureGenerateNodeState(node)
  const replaceInput = isWorkflowImageInputNode(node)
  const replacedBlobUrls = []
  if (replaceInput) {
    state.uploads.forEach((item) => {
      if (/^blob:/i.test(item?.url || '')) replacedBlobUrls.push(item.url)
    })
    clearDownstreamWorkflowRunResults(node)
  }
  updateGenerateNode(node, {
    uploads: replaceInput ? [upload] : [...state.uploads, upload],
    ...(replaceInput ? { uploadOnly: true, result: emptyGenerationResult(), editRef: null, editPreview: null } : {}),
    showResults: false
  })
  nextTick(() => replacedBlobUrls.forEach(url => URL.revokeObjectURL(url)))
  setActiveWorkflowNode(node)
  return true
}

function workflowDownstreamNodes(node) {
  const downstream = []
  const visited = new Set([node])
  const queue = [node]
  while (queue.length) {
    const source = queue.shift()
    workflowLinks.value
      .filter(link => link.from === source)
      .forEach((link) => {
        if (visited.has(link.to)) return
        visited.add(link.to)
        downstream.push(link.to)
        queue.push(link.to)
      })
  }
  return downstream
}

function clearDownstreamWorkflowRunResults(node) {
  const downstream = workflowDownstreamNodes(node)
  if (!downstream.length) return

  const generateStates = { ...workflowGenerateNodeStates.value }
  const llmStates = { ...workflowLlmNodeStates.value }
  const videoStates = { ...workflowVideoNodeStates.value }
  const recolorStates = { ...workflowBatchRecolorNodeStates.value }
  const translateStates = { ...workflowBatchTranslateNodeStates.value }

  downstream.forEach((target) => {
    const type = nodeType(target)
    if (type === 'generate' && generateStates[target] && !generateStates[target].uploadOnly) {
      generateStates[target] = {
        ...generateStates[target],
        generating: false,
        cancelled: false,
        progress: 0,
        runId: '',
        taskProgress: [],
        error: '',
        result: emptyGenerationResult(),
        editRef: null,
        editPreview: null,
        showResults: false
      }
    } else if (type === 'llm' && llmStates[target]) {
      llmStates[target] = {
        ...llmStates[target],
        loading: false,
        progress: 0,
        output: '',
        error: '',
        runId: ''
      }
    } else if (type === 'video' && videoStates[target]) {
      videoStates[target] = {
        ...videoStates[target],
        loading: false,
        progress: 0,
        error: '',
        videoUrl: '',
        videoWidth: 0,
        videoHeight: 0,
        taskId: '',
        status: '',
        rawResponse: null
      }
    } else if (type === 'batch-recolor' && recolorStates[target]) {
      recolorStates[target] = {
        ...recolorStates[target],
        tasks: [],
        loading: false,
        progress: 0,
        resultNode: '',
        error: '',
        runId: ''
      }
    } else if (type === 'batch-translate' && translateStates[target]) {
      translateStates[target] = {
        ...translateStates[target],
        tasks: [],
        loading: false,
        progress: 0,
        resultNode: '',
        error: '',
        runId: ''
      }
    }
  })

  workflowGenerateNodeStates.value = generateStates
  workflowLlmNodeStates.value = llmStates
  workflowVideoNodeStates.value = videoStates
  workflowBatchRecolorNodeStates.value = recolorStates
  workflowBatchTranslateNodeStates.value = translateStates
  invalidatePausedWorkflowRunsForNodes([node, ...downstream], { structural: true })
}

function openGenerateUpload(node) {
  generateUploadTargetNode.value = node
  generateFileInput.value?.click()
}

function onGenerateFileSelect(e) {
  const file = e.target.files?.[0]
  if (file && generateUploadTargetNode.value) {
    appendGenerateUpload(generateUploadTargetNode.value, file)
  }
  e.target.value = ''
}

async function createUploadNodeAndHandleFile(file, point) {
  const node = addWorkflowNode('upload', {
    position: {
      x: point.x - 210,
      y: point.y - 180
    }
  })
  const targetNode = node || 'upload'
  activateModelNode(targetNode)
  await nextTick()
  if (!renderer || renderer.domElement?.parentNode !== currentViewerContainer()) {
    initThreeJS()
    await nextTick()
  }
  handleFile(file, targetNode)
}

function onWorkflowCanvasDrop(e) {
  const file = e.dataTransfer.files?.[0]
  if (!file) return

  if (isModelFile(file)) {
    const point = clientPointToCanvas(e)
    createUploadNodeAndHandleFile(file, point)
    return
  }

  if (!file.type?.startsWith('image/')) return
  const upload = imageFileToGenerateUpload(file)
  if (!upload) return

  const point = clientPointToCanvas(e)
  addWorkflowNode('generate', {
    position: {
      x: point.x - 150,
      y: point.y - 150
    },
    state: {
      uploads: [upload]
    }
  })
}

function clonePlain(value) {
  return JSON.parse(JSON.stringify(value))
}

function cloneModelNodeStates(states = workflowModelNodeStates.value, { includeMeshData = true } = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    const state = states[key] || {}
    map[key] = {
      ...state,
      meshData: includeMeshData ? state.meshData : null
    }
    return map
  }, {})
}

function emptyGenerationResult() {
  return { url: '', images: [], imageCreatedAt: [], imageSources: [], taskIds: [], taskId: '', status: '' }
}

function defaultGenerateNodeState(overrides = {}) {
  return {
    prompt: '',
    model: defaultDrawingModelName(),
    count: 1,
    aspectRatio: '1:1',
    resolution: '2K',
    menu: '',
    generating: false,
    error: '',
    progress: 0,
    cancelled: false,
    runId: '',
    taskProgress: [],
    uploadOnly: false,
    result: emptyGenerationResult(),
    uploads: [],
    editRef: null,
    editPreview: null,
    showResults: false,
    generatedFromNode: '',
    sceneControlOutputNode: '',
    sceneWorkflowNode: false,
    modelWorkflowImageType: '',
    imageEditDisabled: false,
    batchTitle: '',
    inlineControlImages: { white: '', depth: '' },
    ...overrides
  }
}

function normalizeInlineControlImages(images = {}) {
  return {
    white: images?.white || '',
    depth: images?.depth || ''
  }
}

function normalizeGenerateNodeState(state = {}) {
  const modelName = normalizeGenerateModelName(state.model)
  const normalized = defaultGenerateNodeState({
    ...state,
    model: modelName,
    count: Number(state.count) || 1,
    aspectRatio: generateAspectRatioOptions.includes(state.aspectRatio) ? state.aspectRatio : '1:1',
    resolution: state.resolution || '2K',
    progress: Number(state.progress) || 0,
    result: normalizeGenerationResult(state.result || state.workflowGenerationResult || state),
    uploads: Array.isArray(state.uploads) ? state.uploads.filter(item => item?.url) : [],
    editRef: state.editRef?.url ? { ...state.editRef } : null,
    editPreview: state.editPreview?.url ? { ...state.editPreview } : null,
    showResults: !!state.showResults,
    generating: !!state.generating,
    cancelled: !!state.cancelled,
    runId: state.runId || '',
    taskProgress: Array.isArray(state.taskProgress) ? state.taskProgress : [],
    uploadOnly: !!state.uploadOnly,
    generatedFromNode: state.generatedFromNode || '',
    sceneControlOutputNode: state.sceneControlOutputNode || '',
    sceneWorkflowNode: !!state.sceneWorkflowNode,
    modelWorkflowImageType: state.modelWorkflowImageType || '',
    imageEditDisabled: !!state.imageEditDisabled,
    batchTitle: state.batchTitle || '',
    inlineControlImages: normalizeInlineControlImages(state.inlineControlImages),
    error: state.error || ''
  })
  return normalized
}

function normalizeGenerateModelName(model = '') {
  if (!drawingModels.value.length) return model || defaultDrawingModelName()
  if (model && drawingModels.value.some(item => item.name === model)) return model
  return defaultDrawingModelName()
}

function normalizeVideoDuration(value, model = '') {
  const duration = Number(value) || 5
  if (model === 'seedance-2-5' && duration >= 4 && duration <= 30) return duration
  return Math.min(30, Math.max(4, duration))
}

function defaultVideoNodeState(overrides = {}) {
  return {
    prompt: '',
    model: 'seedance-2-5',
    resolution: '720p',
    aspectRatio: '16:9',
    duration: 5,
    cameraFixed: false,
    generateAudio: true,
    inputMode: 'reference',
    firstFrame: '',
    lastFrame: '',
    referenceVideos: [],
    referenceAudios: [],
    seed: -1,
    menu: '',
    loading: false,
    progress: 0,
    error: '',
    videoUrl: '',
    videoWidth: 0,
    videoHeight: 0,
    taskId: '',
    status: '',
    rawResponse: null,
    ...overrides
  }
}

function normalizeVideoNodeState(state = {}, options = {}) {
  const model = videoModelOptions.some(item => item.value === state.model) ? state.model : 'seedance-2-5'
  const resolutionOptions = videoResolutionOptions
  const resolution = resolutionOptions.includes(state.resolution) ? state.resolution : '720p'
  const rawAspectRatio = ({
    film_horizontal_21_9: '21:9',
    widescreen_16_9: '16:9',
    classic_4_3: '4:3',
    square_1_1: '1:1',
    traditional_3_4: '3:4',
    social_story_9_16: '9:16',
    film_vertical_9_2: '9:16',
    film_vertical_9_21: '9:16'
  })[state.aspectRatio] || state.aspectRatio
  const aspectRatio = videoAspectRatioOptions.some(item => item.value === rawAspectRatio)
    ? rawAspectRatio
    : '16:9'
  return defaultVideoNodeState({
    ...state,
    model,
    resolution,
    aspectRatio,
    duration: normalizeVideoDuration(state.duration, model),
    cameraFixed: !!state.cameraFixed,
    generateAudio: state.generateAudio !== false,
    inputMode: state.inputMode === 'frames' ? 'frames' : 'reference',
    firstFrame: state.firstFrame || '',
    lastFrame: state.lastFrame || '',
    referenceVideos: Array.isArray(state.referenceVideos) ? state.referenceVideos.slice(0, 3) : [],
    referenceAudios: Array.isArray(state.referenceAudios) ? state.referenceAudios.slice(0, 3) : [],
    seed: Number.isFinite(Number(state.seed)) ? Number(state.seed) : -1,
    loading: options.resetRuntime ? false : !!state.loading,
    progress: options.resetRuntime ? 0 : (Number(state.progress) || 0),
    error: options.resetRuntime ? '' : (state.error || ''),
    videoUrl: state.videoUrl || state.video_url || '',
    videoWidth: Number(state.videoWidth || state.video_width) || 0,
    videoHeight: Number(state.videoHeight || state.video_height) || 0,
    taskId: state.taskId || state.task_id || '',
    status: state.status || '',
    menu: state.menu || '',
    rawResponse: state.rawResponse || null
  })
}

function normalizeVideoNodeStates(states = {}, options = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    map[key] = normalizeVideoNodeState(states[key], options)
    return map
  }, {})
}

function defaultTextNodeState(overrides = {}) {
  return {
    title: '',
    prompt: '',
    result: '',
    uploads: [],
    menu: '',
    loading: false,
    progress: 0,
    error: '',
    imagePromptMode: false,
    imagePromptFlowCreated: false,
    analysisLabel: '',
    resultEditing: false,
    inputTokens: 0,
    outputTokens: 0,
    ...overrides
  }
}

function normalizeTextNodeState(state = {}) {
  return defaultTextNodeState({
    ...state,
    title: state.title || '',
    prompt: state.prompt || '',
    result: state.result || '',
    uploads: Array.isArray(state.uploads) ? state.uploads.filter(item => item?.url) : [],
    menu: state.menu || '',
    loading: !!state.loading,
    progress: Number(state.progress) || 0,
    error: state.error || '',
    imagePromptMode: false,
    imagePromptFlowCreated: false,
    analysisLabel: state.analysisLabel || state.analysis_label || '',
    resultEditing: false,
    inputTokens: Number(state.inputTokens || state.input_tokens) || 0,
    outputTokens: Number(state.outputTokens || state.output_tokens) || 0
  })
}

function normalizeTextNodeStates(states = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    map[key] = normalizeTextNodeState(states[key])
    return map
  }, {})
}

function textNodeStatesForPersistence(states = {}) {
  const normalized = normalizeTextNodeStates(states)
  return Object.keys(normalized).reduce((map, key) => {
    map[key] = {
      ...normalized[key],
      resultEditing: false
    }
    return map
  }, {})
}

function mergePersistedTextNodeStates(persistedStates = {}, currentStates = workflowTextNodeStates.value) {
  const normalizedPersisted = normalizeTextNodeStates(persistedStates)
  const merged = { ...normalizedPersisted }
  Object.keys(currentStates || {}).forEach((key) => {
    if (!currentStates[key]?.resultEditing) return
    merged[key] = {
      ...(normalizedPersisted[key] || normalizeTextNodeState(currentStates[key])),
      resultEditing: true
    }
  })
  return merged
}

function normalizeBatchMaterial(material = {}) {
  const value = String(material.value ?? material.id ?? material.name ?? '').trim()
  const label = String(material.label || material.name || value).trim()
  if (!value || !label) return null
  return {
    value,
    label,
    imageUrl: resolveImageUrl(material.imageUrl || material.image_url || material.url || ''),
    prompt: material.prompt || `${label}材质/色系，参考后台材质色块图的颜色、表面质感、反射强度和金属纹理`
  }
}

function shouldUseBatchMaterialReference(material = {}) {
  return String(material.value ?? material.id ?? '').trim() !== '1'
}

function defaultBatchRecolorNodeState(overrides = {}) {
  return {
    target: '',
    aspectRatio: '1:1',
    resolution: '2K',
    menu: '',
    materials: [],
    tasks: [],
    loading: false,
    progress: 0,
    resultNode: '',
    error: '',
    runId: '',
    ...overrides
  }
}

function normalizeBatchRecolorTask(task = {}, index = 0) {
  return {
    index: Number(task.index ?? index) || 0,
    materialId: String(task.materialId || task.material_id || task.value || '').trim(),
    materialName: task.materialName || task.material_name || task.label || '',
    materialImageUrl: resolveImageUrl(task.materialImageUrl || task.material_image_url || ''),
    sourceImageUrl: resolveImageUrl(task.sourceImageUrl || task.source_image_url || ''),
    sourceNode: task.sourceNode || task.source_node || '',
    sourceName: task.sourceName || task.source_name || '',
    sourceIndex: Number(task.sourceIndex ?? task.source_index ?? 0) || 0,
    imageUrl: resolveImageUrl(task.imageUrl || task.image_url || ''),
    taskId: task.taskId || task.task_id || '',
    status: task.status || 'pending',
    progress: Number(task.progress) || 0,
    error: task.error || ''
  }
}

function normalizeBatchRecolorNodeState(state = {}) {
  return defaultBatchRecolorNodeState({
    ...state,
    target: state.target || '',
    aspectRatio: generateAspectRatioOptions.includes(state.aspectRatio) ? state.aspectRatio : '1:1',
    resolution: generateResolutionOptions.includes(state.resolution) ? state.resolution : '2K',
    menu: state.menu || '',
    materials: (Array.isArray(state.materials) ? state.materials : []).map(normalizeBatchMaterial).filter(Boolean),
    tasks: (Array.isArray(state.tasks) ? state.tasks : []).map(normalizeBatchRecolorTask),
    loading: !!state.loading,
    progress: Number(state.progress) || 0,
    resultNode: state.resultNode || '',
    error: state.error || '',
    runId: state.runId || ''
  })
}

function normalizeBatchRecolorNodeStates(states = {}, options = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    const state = normalizeBatchRecolorNodeState(states[key])
    map[key] = options.resetRuntime ? resetBatchRecolorRuntimeState(state) : state
    return map
  }, {})
}

function defaultBatchTranslateNodeState(overrides = {}) {
  return {
    targetLanguage: 'en',
    targetLanguages: ['en'],
    aspectRatio: '1:1',
    resolution: '2K',
    menu: '',
    tasks: [],
    loading: false,
    progress: 0,
    resultNode: '',
    error: '',
    runId: '',
    ...overrides
  }
}

function resetBatchRecolorRuntimeState(state = {}) {
  return normalizeBatchRecolorNodeState({
    ...state,
    loading: false,
    runId: '',
    progress: state.loading ? 0 : state.progress,
    tasks: (state.tasks || []).map((task, index) => ({
      ...task,
      status: task.status === 'running' ? 'pending' : task.status,
      progress: task.status === 'running' ? 0 : task.progress
    }))
  })
}

function resetBatchTranslateRuntimeState(state = {}) {
  return normalizeBatchTranslateNodeState({
    ...state,
    loading: false,
    runId: '',
    progress: state.loading ? 0 : state.progress,
    tasks: (state.tasks || []).map((task, index) => ({
      ...task,
      status: task.status === 'running' ? 'pending' : task.status,
      progress: task.status === 'running' ? 0 : task.progress
    }))
  })
}

function normalizeBatchTranslateTask(task = {}, index = 0) {
  return {
    index: Number(task.index ?? index) || 0,
    materialId: String(task.materialId || task.material_id || '').trim(),
    materialName: task.materialName || task.material_name || '',
    sourceImageUrl: resolveImageUrl(task.sourceImageUrl || task.source_image_url || task.recoloredImageUrl || task.recolored_image_url || ''),
    imageUrl: resolveImageUrl(task.imageUrl || task.image_url || task.translatedImageUrl || task.translated_image_url || ''),
    targetLanguage: task.targetLanguage || task.target_language || '',
    targetLanguageName: task.targetLanguageName || task.target_language_name || '',
    status: task.status || 'pending',
    progress: Number(task.progress) || 0,
    error: task.error || ''
  }
}

function normalizeBatchTranslateNodeState(state = {}) {
  const legacyLanguage = batchTranslateLanguageOptions.some(item => item.value === state.targetLanguage)
    ? state.targetLanguage
    : 'en'
  const selectedLanguages = uniqueValues([
    ...(Array.isArray(state.targetLanguages) ? state.targetLanguages : []),
    legacyLanguage
  ]).filter(value => batchTranslateLanguageOptions.some(item => item.value === value))
  const targetLanguages = selectedLanguages.length ? selectedLanguages : ['en']
  return defaultBatchTranslateNodeState({
    ...state,
    targetLanguage: targetLanguages[0],
    targetLanguages,
    aspectRatio: generateAspectRatioOptions.includes(state.aspectRatio) ? state.aspectRatio : '1:1',
    resolution: generateResolutionOptions.includes(state.resolution) ? state.resolution : '2K',
    menu: state.menu || '',
    tasks: (Array.isArray(state.tasks) ? state.tasks : []).map(normalizeBatchTranslateTask),
    loading: !!state.loading,
    progress: Number(state.progress) || 0,
    resultNode: state.resultNode || '',
    error: state.error || '',
    runId: state.runId || ''
  })
}

function normalizeBatchTranslateNodeStates(states = {}, options = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    const state = normalizeBatchTranslateNodeState(states[key])
    map[key] = options.resetRuntime ? resetBatchTranslateRuntimeState(state) : state
    return map
  }, {})
}

function defaultModelNodeState(overrides = {}) {
  return {
    fileName: '',
    stpFileUrl: '',
    meshData: null,
    correction: { x: 0, y: 0, z: 0 },
    cut: { enabled: false, height: 0 },
    camera: null,
    controlMode: 'product',
    sceneType: 'bathroom',
    sceneShot: 'wide',
    sceneStyle: 'american_modern_classic',
    sceneMaterial: '',
    sceneMaterialNode: '',
    sceneStyleRef: null,
    sceneStyleNode: '',
    sceneAspectRatio: '1:1',
    sceneResolution: '2K',
    modelLoaded: false,
    loading: false,
    loadingMessage: '加载中...',
    error: '',
    ...overrides
  }
}

function defaultOutputNodeState(overrides = {}) {
  return {
    resultUrls: {},
    sourceNode: '',
    ...overrides
  }
}

function normalizeOutputNodeState(state = {}) {
  return defaultOutputNodeState({
    ...state,
    resultUrls: { ...(state.resultUrls || {}) },
    sourceNode: state.sourceNode || '',
    controlMode: normalizeModelControlMode(state.controlMode),
    sceneType: normalizeModelSceneType(state.sceneType),
    sceneShot: normalizeModelSceneShot(state.sceneShot),
    sceneStyle: normalizeModelSceneStyle(state.sceneStyle),
    sceneMaterial: normalizeModelSceneMaterial(state.sceneMaterial),
    sceneStyleRef: normalizeModelSceneStyleRef(state.sceneStyleRef),
    viewGuide: state.viewGuide || null
  })
}

function normalizeOutputNodeStates(states = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    map[key] = normalizeOutputNodeState(states[key])
    return map
  }, {})
}

function normalizeModelCorrection(correction = {}) {
  return {
    x: Number(correction.x) || 0,
    y: Number(correction.y) || 0,
    z: Number(correction.z) || 0
  }
}

function normalizeModelCamera(cameraState = null) {
  if (!cameraState || !Array.isArray(cameraState.position) || !Array.isArray(cameraState.target)) return null
  const position = cameraState.position.slice(0, 3).map(value => Number(value) || 0)
  const target = cameraState.target.slice(0, 3).map(value => Number(value) || 0)
  if (position.length < 3 || target.length < 3) return null
  return {
    position,
    target,
    zoom: Number(cameraState.zoom) || 1
  }
}

function normalizeModelCut(cut = {}) {
  return {
    enabled: !!cut.enabled,
    height: THREE.MathUtils.clamp(Number(cut.height) || 0, 0, MODEL_CUT_MAX_HEIGHT)
  }
}

function normalizeModelControlMode(mode) {
  return mode === 'scene' ? 'scene' : 'product'
}

function normalizeModelSceneShot(shot) {
  return ['close', 'medium', 'wide'].includes(shot) ? shot : 'wide'
}

function normalizeModelSceneType(type) {
  return type === 'kitchen' ? 'kitchen' : 'bathroom'
}

function normalizeModelSceneStyle(style) {
  return modelSceneStyleOptions.some(item => item.value === style) ? style : 'american_modern_classic'
}

function normalizeModelSceneMaterial(material) {
  const value = String(material || '').trim()
  if (!value) return ''
  const options = modelSceneMaterialOptions.value || []
  if (!options.length) return value
  const matched = options.find(item => item.value === value || item.label === value)
  return matched?.value || ''
}

function selectedModelSceneMaterial(material) {
  const value = String(material || '').trim()
  if (!value) return null
  const options = modelSceneMaterialOptions.value || []
  return options.find(item => item.value === value || item.label === value) || null
}

function normalizeModelSceneStyleRef(ref = null) {
  const url = styleReferenceImage(ref || {})
  if (!url) return null
  return {
    url,
    tagName: ref.tagName || ref.tag_name || ref.name || '',
    sceneType: normalizeModelSceneType(ref.sceneType),
    prompt: String(ref.prompt || ref.colorPrompt || ref.palettePrompt || '').trim(),
    promptImageUrl: ref.promptImageUrl || ref.prompt_image_url || ''
  }
}

function normalizeModelNodeState(state = {}) {
  const hasMeshData = !!(state.meshData?.vertices?.length)
  return defaultModelNodeState({
    ...state,
    fileName: state.fileName || '',
    stpFileUrl: state.stpFileUrl || '',
    meshData: state.meshData ? markRaw(state.meshData) : null,
    correction: normalizeModelCorrection(state.correction),
    cut: normalizeModelCut(state.cut),
    camera: normalizeModelCamera(state.camera),
    controlMode: normalizeModelControlMode(state.controlMode),
    sceneType: normalizeModelSceneType(state.sceneType),
    sceneShot: normalizeModelSceneShot(state.sceneShot),
    sceneStyle: normalizeModelSceneStyle(state.sceneStyle),
    sceneMaterial: normalizeModelSceneMaterial(state.sceneMaterial),
    sceneMaterialNode: state.sceneMaterialNode || '',
    sceneStyleRef: normalizeModelSceneStyleRef(state.sceneStyleRef),
    sceneStyleNode: state.sceneStyleNode || '',
    sceneAspectRatio: normalizeModelSceneAspectRatio(state.sceneAspectRatio),
    sceneResolution: generateResolutionOptions.includes(state.sceneResolution) ? state.sceneResolution : '2K',
    modelLoaded: !!(state.modelLoaded && hasMeshData),
    loading: false,
    loadingMessage: state.loadingMessage || '加载中...',
    error: state.error || ''
  })
}

function normalizeModelNodeStates(states = {}) {
  return Object.keys(states || {}).reduce((map, key) => {
    map[key] = normalizeModelNodeState(states[key])
    return map
  }, {})
}

function uniqueUrls(urls = []) {
  return [...new Set(urls.filter(Boolean).map(resolveImageUrl))]
}

function uniqueValues(values = []) {
  return [...new Set(values.filter(Boolean))]
}

function normalizeGenerationResult(result = {}) {
  const images = uniqueUrls([
    ...(Array.isArray(result.images) ? result.images : []),
    result.url
  ])
  const imageCreatedAt = Array.isArray(result.imageCreatedAt)
    ? result.imageCreatedAt
    : (Array.isArray(result.image_created_at) ? result.image_created_at : [])
  const imageSources = Array.isArray(result.imageSources)
    ? result.imageSources
    : (Array.isArray(result.image_sources) ? result.image_sources : [])
  return {
    ...emptyGenerationResult(),
    ...result,
    url: images[0] || resolveImageUrl(result.url || ''),
    images,
    createdAt: result.createdAt || result.created_at || '',
    imageCreatedAt,
    imageSources,
    batchItems: Array.isArray(result.batchItems) ? result.batchItems : [],
    batchTitle: result.batchTitle || '',
    taskIds: Array.isArray(result.taskIds)
      ? result.taskIds
      : (result.taskId ? [result.taskId] : [])
  }
}

function normalizeHistoryAssets(assets = []) {
  const seen = new Set()
  return (Array.isArray(assets) ? assets : [])
    .map((asset) => {
      const url = resolveImageUrl(asset?.url || asset?.image_url || '')
      const createdAt = asset?.createdAt || asset?.created_at || ''
      if (!url || !createdAt || seen.has(url)) return null
      const date = new Date(createdAt)
      if (Number.isNaN(date.getTime())) return null
      seen.add(url)
      return {
        url,
        createdAt: date.toISOString(),
        node: asset?.node || '',
        taskId: asset?.taskId || asset?.task_id || '',
        type: asset?.type === 'video' || isVideoAssetUrl(url) ? 'video' : 'image'
      }
    })
    .filter(Boolean)
}

function historyAssetsFromGenerateStates(states = {}) {
  const items = []
  Object.keys(states || {}).forEach((node) => {
    const result = normalizeGenerationResult(states[node]?.result || {})
    result.images.forEach((url, index) => {
      const source = result.imageSources[index]
      if (source && source !== 'generated') return
      if (result.status === 'imported') return
      if (result.status === 'edited' && index === 0 && !source) return
      const createdAt = result.imageCreatedAt[index] || result.createdAt || ''
      items.push({
        url,
        createdAt,
        node,
        taskId: result.taskIds[index] || result.taskId || ''
      })
    })
  })
  return normalizeHistoryAssets(items)
}

function getDefaultProjectName() {
  return `工作流项目 ${projects.value.length + 1}`
}

function createEmptyProject(name) {
  return {
    id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    name: name || getDefaultProjectName(),
    workflowStateRevision: 0,
    fileName: '',
    stpFileUrl: '',
    meshData: null,
    modelLoaded: false,
    resultImageUrl: '',
    resultUrls: {},
    workflowNodes: clonePlain(DEFAULT_WORKFLOW_NODES),
    workflowDynamicNodes: [],
    workflowLinks: [],
    workflowHistoryAssets: [],
    workflowGroups: [],
    nodePositions: clonePlain(DEFAULT_NODE_POSITIONS),
    canvasView: { ...DEFAULT_CANVAS_VIEW },
    activeWorkflowNode: 'upload',
    sceneImage: { url: '', name: '', width: 0, height: 0 },
    placement: { x: 50, y: 58, scale: 0.42, rotation: 0, opacity: 0.78 },
    workflowGeneratePrompt: '',
    workflowGenerateModel: workflowGenerateModel.value,
    workflowGenerateNodeStates: {},
    workflowTextNodeStates: {},
    workflowLlmNodeStates: {},
    workflowModelNodeStates: {},
    workflowOutputNodeStates: {},
    workflowBatchRecolorNodeStates: {},
    workflowBatchTranslateNodeStates: {},
    workflowVideoNodeStates: {},
    workflowGenerationResult: emptyGenerationResult()
  }
}

function projectMetaForResponse(project = {}) {
  return {
    createdAt: project.created_at || project.createdAt || '',
    updatedAt: project.updated_at || project.updatedAt || ''
  }
}

function normalizeProject(project = {}) {
  const base = createEmptyProject(project.name)
  const normalizedGenerateStates = normalizeGenerateNodeStates(project.workflowGenerateNodeStates || {})
  const normalizedWorkflowGenerationResult = normalizeGenerationResult(project.workflowGenerationResult)
  const normalizedDynamicNodes = normalizeDynamicNodes(project.workflowDynamicNodes || [])
  if (project.workflowNodes?.generate && normalizedWorkflowGenerationResult.images.length) {
    normalizedGenerateStates.generate = normalizeGenerateNodeState({
      result: normalizedWorkflowGenerationResult,
      ...normalizedGenerateStates.generate
    })
  }
  const workflowHistoryAssets = normalizeHistoryAssets(project.workflowHistoryAssets || project.workflow_history_assets || [])
  const migratedHistoryAssets = workflowHistoryAssets.length ? workflowHistoryAssets : historyAssetsFromGenerateStates(normalizedGenerateStates)
  return {
    ...base,
    ...project,
    ...projectMetaForResponse(project),
    id: project.id || project.project_id || base.id,
    name: project.name || base.name,
    workflowStateRevision: Number(project.workflowStateRevision) || 0,
    meshData: project.meshData || null,
    modelLoaded: !!project.modelLoaded,
    resultUrls: { ...(project.resultUrls || {}) },
    workflowNodes: normalizeWorkflowNodes(project.workflowNodes || {}),
    workflowDynamicNodes: normalizedDynamicNodes,
    workflowLinks: normalizeWorkflowLinks(project.workflowLinks || []),
    workflowHistoryAssets: migratedHistoryAssets,
    workflowGroups: Array.isArray(project.workflowGroups) ? clonePlain(project.workflowGroups) : [],
    nodePositions: clonePlain(project.nodePositions || DEFAULT_NODE_POSITIONS),
    canvasView: { ...(project.canvasView || DEFAULT_CANVAS_VIEW) },
    sceneImage: { ...(project.sceneImage || { url: '', name: '', width: 0, height: 0 }) },
    placement: { ...(project.placement || base.placement) },
    workflowGenerateNodeStates: normalizedGenerateStates,
    workflowTextNodeStates: normalizeTextNodeStates(project.workflowTextNodeStates || {}),
    workflowLlmNodeStates: normalizeLlmNodeStates(project.workflowLlmNodeStates || {}, { resetRuntime: true }),
    workflowModelNodeStates: modelStatesWithDynamicUploadFallback(project.workflowModelNodeStates || {}, normalizedDynamicNodes),
    workflowOutputNodeStates: normalizeOutputNodeStates(project.workflowOutputNodeStates || {}),
    workflowBatchRecolorNodeStates: normalizeBatchRecolorNodeStates(project.workflowBatchRecolorNodeStates || {}, { resetRuntime: true }),
    workflowBatchTranslateNodeStates: normalizeBatchTranslateNodeStates(project.workflowBatchTranslateNodeStates || {}, { resetRuntime: true }),
    workflowVideoNodeStates: normalizeVideoNodeStates(project.workflowVideoNodeStates || {}, { resetRuntime: true }),
    workflowGenerationResult: normalizedWorkflowGenerationResult
  }
}

function normalizeGenerateNodeStates(states = {}) {
  return Object.keys(states).reduce((map, key) => {
    map[key] = normalizeGenerateNodeState(states[key])
    return map
  }, {})
}

function mergePersistedGenerateNodeStates(persistedStates = {}, currentStates = workflowGenerateNodeStates.value) {
  const normalizedPersisted = normalizeGenerateNodeStates(persistedStates)
  const merged = { ...normalizedPersisted }

  Object.keys(currentStates || {}).forEach((key) => {
    const current = currentStates[key]
    if (!current?.generating) return
    const persisted = normalizedPersisted[key] || normalizeGenerateNodeState(current)
    merged[key] = {
      ...persisted,
      generating: true,
      cancelled: current.cancelled,
      progress: current.progress,
      runId: current.runId,
      taskProgress: Array.isArray(current.taskProgress) ? [...current.taskProgress] : [],
      error: current.error,
      showResults: false,
      result: normalizeGenerationResult({
        ...persisted.result,
        ...current.result,
        images: uniqueUrls([
          ...(persisted.result?.images || []),
          ...(current.result?.images || [])
        ])
      })
    }
  })

  return merged
}

function normalizeWorkflowNodes(nodes = {}) {
  const source = clonePlain(nodes || {})
  const normalized = Object.keys(DEFAULT_WORKFLOW_NODES).reduce((map, key) => {
    map[key] = !!source[key]
    return map
  }, {})
  normalized.upload = !!(source.upload || source.preview)
  return normalized
}

function normalizeWorkflowLinks(links = []) {
  const normalized = links
    .map(link => ({
      from: link.from === 'preview' ? 'upload' : link.from,
      to: link.to === 'preview' ? 'upload' : link.to
    }))
    .filter(link => link.from && link.to && link.from !== link.to)

  return normalized.filter((link, index) => (
    normalized.findIndex(item => item.from === link.from && item.to === link.to) === index
  ))
}

function workflowLinksWithValidVideoInputs(links = []) {
  const videoInputs = new Set()
  return normalizeWorkflowLinks(links).filter((link) => {
    if (nodeType(link.to) !== 'video') return true
    const sourceType = nodeType(link.from)
    if (['text', 'llm'].includes(sourceType)) return true
    if (sourceType !== 'generate') return false
    if (videoInputs.has(link.to)) return false
    videoInputs.add(link.to)
    return true
  })
}

function normalizeDynamicNodes(nodes = []) {
  return Array.isArray(nodes)
    ? nodes
      .filter(item => item?.id && item?.type)
      .map(item => ({
        id: item.id,
        type: item.type,
        ...(item.type === 'upload' && item.fileName ? { fileName: item.fileName } : {}),
        ...(item.type === 'upload' && item.stpFileUrl ? { stpFileUrl: item.stpFileUrl } : {})
      }))
    : []
}

function syncDynamicUploadNodeFileState(node, patch = {}) {
  if (!node || node === 'upload') return
  if (nodeType(node) !== 'upload') return
  workflowDynamicNodes.value = workflowDynamicNodes.value.map(item => (
    item.id === node
      ? {
          ...item,
          ...(patch.fileName ? { fileName: patch.fileName } : {}),
          ...(patch.stpFileUrl ? { stpFileUrl: patch.stpFileUrl } : {})
        }
      : item
  ))
}

function ensureDynamicOutputNodesFromState() {
  const existing = new Set(workflowDynamicNodes.value.map(item => item.id))
  const additions = []
  Object.keys(workflowOutputNodeStates.value || {}).forEach((node) => {
    if (node === 'output' || existing.has(node)) return
    additions.push({ id: node, type: 'output' })
    existing.add(node)
  })
  workflowLinks.value.forEach((link) => {
    if (link.to === 'output' || existing.has(link.to)) return
    if (workflowOutputNodeStates.value[link.to]) {
      additions.push({ id: link.to, type: 'output' })
      existing.add(link.to)
    }
  })
  if (additions.length) {
    workflowDynamicNodes.value = [...workflowDynamicNodes.value, ...additions]
  }
}

function isLocalProjectImageUrl(url) {
  return typeof url === 'string' && (url.startsWith('data:') || url.startsWith('blob:'))
}

function dataUrlInfo(dataUrl, fallbackMimeType = 'image/png') {
  const match = /^data:([^;]+);base64,(.*)$/i.exec(dataUrl || '')
  return {
    mimeType: match?.[1] || fallbackMimeType,
    data: match?.[2] || ''
  }
}

async function uploadImageUrlToOSS(url, fallbackMimeType = 'image/png') {
  if (!isLocalProjectImageUrl(url)) return url || ''
  if (ossUploadCache.has(url)) return ossUploadCache.get(url)

  let data = ''
  let mimeType = fallbackMimeType
  if (url.startsWith('data:')) {
    const info = dataUrlInfo(url, fallbackMimeType)
    data = info.data
    mimeType = info.mimeType
  } else {
    const res = await fetch(url)
    const blob = await res.blob()
    mimeType = blob.type || fallbackMimeType
    data = await new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result || '').split(',')[1] || '')
      reader.onerror = reject
      reader.readAsDataURL(blob)
    })
  }

  if (!data) return ''
  const uploadRes = await request.post('/upload/image', { data, mime_type: mimeType })
  const ossUrl = uploadRes.data?.url || ''
  ossUploadCache.set(url, ossUrl)
  return ossUrl
}

async function ensureProjectImagesOnOSS(project) {
  const next = clonePlain(project)
  next.resultImageUrl = await uploadImageUrlToOSS(next.resultImageUrl)
  if (next.resultUrls) {
    next.resultUrls.white = await uploadImageUrlToOSS(next.resultUrls.white)
    next.resultUrls.depth = await uploadImageUrlToOSS(next.resultUrls.depth)
    next.resultUrls.productRgba = await uploadImageUrlToOSS(next.resultUrls.productRgba)
  }
  if (next.sceneImage?.url) {
    next.sceneImage.url = await uploadImageUrlToOSS(next.sceneImage.url)
  }
  next.workflowGenerationResult = await ensureGenerationResultImagesOnOSS(next.workflowGenerationResult)
  next.workflowGenerateNodeStates = await ensureGenerateNodeStatesImagesOnOSS(next.workflowGenerateNodeStates || {})
  next.workflowHistoryAssets = await ensureHistoryAssetsOnOSS(next.workflowHistoryAssets || [])
  next.workflowTextNodeStates = await ensureTextNodeStatesImagesOnOSS(next.workflowTextNodeStates || {})
  next.workflowOutputNodeStates = await ensureOutputNodeStatesImagesOnOSS(next.workflowOutputNodeStates || {})
  next.workflowVideoNodeStates = await ensureVideoNodeStatesOnOSS(next.workflowVideoNodeStates || {})
  return next
}

async function ensureGenerationResultImagesOnOSS(result = {}) {
  const normalized = normalizeGenerationResult(result)
  const images = []
  for (const image of normalized.images || []) {
    const ossUrl = await uploadImageUrlToOSS(image)
    if (ossUrl) images.push(ossUrl)
  }
  return {
    ...normalized,
    images,
    url: images[0] || ''
  }
}

async function ensureGenerateNodeStatesImagesOnOSS(states = {}) {
  const next = {}
  for (const key of Object.keys(states)) {
    const state = normalizeGenerateNodeState(states[key])
    const uploads = []
    for (const item of state.uploads || []) {
      const ossUrl = await uploadImageUrlToOSS(item.url)
      if (ossUrl) uploads.push({ ...item, url: ossUrl })
    }
    next[key] = {
      ...state,
      uploads,
      editRef: state.editRef?.url
        ? { ...state.editRef, url: await uploadImageUrlToOSS(state.editRef.url) }
        : null,
      editPreview: state.editPreview?.url
        ? { ...state.editPreview, url: await uploadImageUrlToOSS(state.editPreview.url) }
        : null,
      result: await ensureGenerationResultImagesOnOSS(state.result)
    }
  }
  return next
}

async function ensureHistoryAssetsOnOSS(assets = []) {
  const uploaded = []
  for (const asset of normalizeHistoryAssets(assets)) {
    const ossUrl = await uploadImageUrlToOSS(asset.url, asset.type === 'video' ? 'video/mp4' : 'image/png')
    if (ossUrl) uploaded.push({ ...asset, url: ossUrl })
  }
  return normalizeHistoryAssets(uploaded)
}

async function ensureVideoNodeStatesOnOSS(states = {}) {
  const next = {}
  for (const key of Object.keys(states || {})) {
    const state = normalizeVideoNodeState(states[key])
    next[key] = {
      ...state,
      videoUrl: await uploadImageUrlToOSS(state.videoUrl, 'video/mp4'),
      firstFrame: await uploadImageUrlToOSS(state.firstFrame),
      lastFrame: await uploadImageUrlToOSS(state.lastFrame),
      referenceVideos: await Promise.all((state.referenceVideos || []).map(url => uploadImageUrlToOSS(url, 'video/mp4'))),
      referenceAudios: await Promise.all((state.referenceAudios || []).map(url => uploadImageUrlToOSS(url, 'audio/mpeg')))
    }
  }
  return next
}

async function ensureTextNodeStatesImagesOnOSS(states = {}) {
  const next = {}
  for (const key of Object.keys(states || {})) {
    const state = normalizeTextNodeState(states[key])
    const uploads = []
    for (const item of state.uploads || []) {
      const ossUrl = await uploadImageUrlToOSS(item.url)
      if (ossUrl) uploads.push({ ...item, url: ossUrl })
    }
    next[key] = {
      ...state,
      uploads
    }
  }
  return next
}

async function ensureOutputNodeStatesImagesOnOSS(states = {}) {
  const next = {}
  for (const key of Object.keys(states || {})) {
    const state = normalizeOutputNodeState(states[key])
    next[key] = normalizeOutputNodeState({
      ...state,
      resultUrls: {
        white: await uploadImageUrlToOSS(state.resultUrls?.white),
        depth: await uploadImageUrlToOSS(state.resultUrls?.depth),
        productRgba: await uploadImageUrlToOSS(state.resultUrls?.productRgba)
      }
    })
  }
  return next
}

function projectStateForDatabase(project) {
  const state = {
    ...clonePlain({
      ...project,
      meshData: null,
      workflowTextNodeStates: textNodeStatesForPersistence(project.workflowTextNodeStates || {}),
      workflowModelNodeStates: cloneModelNodeStates(project.workflowModelNodeStates || {}, { includeMeshData: false })
    })
  }
  return state
}

function nextWorkflowStateRevision(project) {
  return (Number(project?.workflowStateRevision) || 0) + 1
}

function projectDraftKey(projectId = activeProjectId.value) {
  return projectId ? `professional-canvas-draft:${projectId}` : ''
}

function saveProjectDraft(project = projects.value.find(item => item.id === activeProjectId.value)) {
  if (!project?.id) return
  try {
    localStorage.setItem(projectDraftKey(project.id), JSON.stringify({
      savedAt: Date.now(),
      state: projectStateForDatabase(project)
    }))
  } catch (e) {
    console.warn('[Workflow] save local draft failed:', e)
  }
}

function scheduleProjectDraft(project = projects.value.find(item => item.id === activeProjectId.value)) {
  if (!project?.id) return
  if (projectDraftTimer) window.clearTimeout(projectDraftTimer)
  projectDraftTimer = window.setTimeout(() => {
    projectDraftTimer = null
    saveProjectDraft(project)
  }, 240)
}

function flushProjectDraft(project = projects.value.find(item => item.id === activeProjectId.value)) {
  if (projectDraftTimer) {
    window.clearTimeout(projectDraftTimer)
    projectDraftTimer = null
  }
  if (project) saveProjectDraft(project)
}

function clearProjectDraft(projectId = activeProjectId.value) {
  if (projectDraftTimer && projectId === activeProjectId.value) {
    window.clearTimeout(projectDraftTimer)
    projectDraftTimer = null
  }
  const key = projectDraftKey(projectId)
  if (!key) return
  try {
    localStorage.removeItem(key)
  } catch {}
}

function mergeProjectDraft(project) {
  if (!project?.id) return project
  try {
    const raw = localStorage.getItem(projectDraftKey(project.id))
    if (!raw) return project
    const draft = JSON.parse(raw)
    const draftState = draft?.state
    if (!draftState) return project
    const draftRevision = Number(draftState.workflowStateRevision) || 0
    const projectRevision = Number(project.workflowStateRevision) || 0
    if (draftRevision || projectRevision) {
      if (draftRevision >= projectRevision) {
        return normalizeProject({
          ...project,
          ...draftState,
          id: project.id,
          name: project.name
        })
      }
      clearProjectDraft(project.id)
      return project
    }
    const projectTime = new Date(project.updatedAt || project.updated_at || 0).getTime()
    if (projectTime && draft.savedAt && draft.savedAt < projectTime) {
      clearProjectDraft(project.id)
      return project
    }
    return normalizeProject({
      ...project,
      ...draftState,
      id: project.id,
      name: project.name
    })
  } catch (e) {
    console.warn('[Workflow] restore local draft failed:', e)
    return project
  }
}

function workflowProjectFromResponse(item) {
  let state = {}
  try {
    state = typeof item?.state === 'string'
      ? JSON.parse(item.state || '{}')
      : (item?.state || {})
  } catch (e) {
    console.warn('[Workflow] parse project state failed:', e)
  }
  return normalizeProject({
    ...state,
    id: item.project_id || state.id,
    name: item.name || state.name
  })
}

function nodeExists(node) {
  if (workflowNodes.value[node]) return true
  return workflowDynamicNodes.value.some(item => item.id === node)
}

function nodeType(node) {
  return workflowDynamicNodes.value.find(item => item.id === node)?.type || node
}

function generateWorkflowNodes() {
  const nodes = workflowDynamicNodes.value.filter(item => item.type === 'generate')
  if (workflowNodes.value.generate) nodes.unshift({ id: 'generate', type: 'generate' })
  return nodes.map(item => ({
    ...item,
    ...ensureGenerateNodeState(item.id)
  }))
}

function modelWorkflowNodes() {
  const nodes = workflowDynamicNodes.value.filter(item => item.type === 'upload')
  if (workflowNodes.value.upload) nodes.unshift({ id: 'upload', type: 'upload' })
  return nodes
}

function nonGenerateDynamicNodes() {
  return workflowDynamicNodes.value.filter(item => !['generate', 'upload'].includes(item.type))
}

function dynamicNodeTitle(node) {
  if (node.type === 'generate' && isMaterialReferenceNode(node.id)) return '材质参考'
  if (node.type === 'generate' && isStyleReferenceNode(node.id)) return '风格参考'
  if (node.type === 'text') return ensureTextNodeState(node.id).title || '文本'
  const titles = {
    upload: '3D 文件 / 预览节点',
    output: '控制图输出节点',
    scene: '场景图节点',
    final: '最终场景输出节点',
    style: '风格库节点',
    llm: '大模型节点',
    video: '视频节点',
    'batch-recolor': '批量换色节点',
    'batch-translate': '批量翻译节点'
  }
  return titles[node.type] || '工作流节点'
}

function dynamicNodeIcon(node) {
  const icons = {
    upload: '3D',
    output: 'OUT',
    scene: 'IMG',
    final: 'GEN',
    style: 'ST',
    video: 'VID',
    text: 'TXT',
    llm: 'AI',
    'batch-recolor': 'CLR',
    'batch-translate': 'TR'
  }
  return icons[node.type] || 'NODE'
}

function dynamicNodeDescription(node) {
  const descriptions = {
    upload: '这是一个额外的 3D 输入节点，可用于扩展后续流程。',
    output: '连接 3D 节点后可作为白底结构图和深度图的输出占位。',
    scene: '上传或连接场景图后，可作为生图参考。',
    final: '连接上游节点后输出最终结果。',
    style: '收集风格参考和材质方向，用于后续生图节点。',
    video: '使用固定 Seedance 2.5 视频模型生成视频。',
    text: '用户可直接输入文本，并连接到图片节点作为提示词。',
    llm: '理解上游图片和文字，生成可继续连接的文本结果。',
    'batch-recolor': '输入换色对象并选择多个材质，完成后自动生成换色图节点。',
    'batch-translate': '接收多张图片，按图片翻译规则批量输出翻译图。'
  }
  return descriptions[node.type] || '可连接到其他节点继续处理。'
}

function activeGenerateNodeId() {
  return nodeType(activeWorkflowNode.value) === 'generate' ? activeWorkflowNode.value : 'generate'
}

function ensureGenerateNodeState(node = activeGenerateNodeId()) {
  if (!workflowGenerateNodeStates.value[node]) {
    workflowGenerateNodeStates.value = {
      ...workflowGenerateNodeStates.value,
      [node]: defaultGenerateNodeState()
    }
  }
  return workflowGenerateNodeStates.value[node]
}

function generateNodeState(node = activeGenerateNodeId()) {
  return ensureGenerateNodeState(node)
}

function ensureTextNodeState(node) {
  if (!workflowTextNodeStates.value[node]) {
    workflowTextNodeStates.value = {
      ...workflowTextNodeStates.value,
      [node]: defaultTextNodeState()
    }
  }
  return workflowTextNodeStates.value[node]
}

function ensureModelNodeState(node = activeModelNodeId.value || 'upload') {
  if (!workflowModelNodeStates.value[node]) {
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [node]: defaultModelNodeState()
    }
  }
  return workflowModelNodeStates.value[node]
}

function modelNodeState(node) {
  const state = ensureModelNodeState(node)
  if (isModelNodeVisiblyLoaded(node) && !state.modelLoaded) {
    return {
      ...state,
      modelLoaded: true,
      loading: false,
      error: ''
    }
  }
  return state
}

function modelNodeViewerVisible(node) {
  const state = modelNodeState(node)
  return activeModelNodeId.value === node && (state.modelLoaded || (state.loading && !!state.meshData))
}

function ensureOutputNodeState(node = 'output') {
  if (!workflowOutputNodeStates.value[node]) {
    workflowOutputNodeStates.value = {
      ...workflowOutputNodeStates.value,
      [node]: defaultOutputNodeState()
    }
  }
  return workflowOutputNodeStates.value[node]
}

function outputNodeResultUrls(node = 'output') {
  if (node === 'output') return resultUrls.value || {}
  return ensureOutputNodeState(node).resultUrls || {}
}

function outputNodeStateFor(node = 'output') {
  return ensureOutputNodeState(node)
}

function outputNodeDone(node = 'output') {
  const urls = outputNodeResultUrls(node)
  return !!(urls.white || urls.depth)
}

function textNodeState(node) {
  return ensureTextNodeState(node)
}

function textNodeOutputText(node) {
  const state = ensureTextNodeState(node)
  return (state.result || state.prompt || '').trim()
}

function textNodeTextValue(node) {
  const state = ensureTextNodeState(node)
  return state.prompt || state.result || ''
}

function syncNodeStateMap(stateKey, states, options = {}) {
  if (options.transient) return
  if (options.persist !== false) {
    saveActiveProject()
    return
  }
  saveProjectPatch({ [stateKey]: states }, { persist: false })
}

function scheduleTextNodeInputSync(node) {
  window.clearTimeout(textNodeInputSyncTimers.get(node))
  textNodeInputSyncTimers.set(node, window.setTimeout(() => {
    textNodeInputSyncTimers.delete(node)
    markTemplateWorkflowGroupChangedForNode(node)
    workflowTextNodeStates.value = { ...workflowTextNodeStates.value }
    syncNodeStateMap('workflowTextNodeStates', workflowTextNodeStates.value, { persist: false })
  }, 240))
}

function flushTextNodeInputSync(node) {
  if (!textNodeInputSyncTimers.has(node)) return
  window.clearTimeout(textNodeInputSyncTimers.get(node))
  textNodeInputSyncTimers.delete(node)
  markTemplateWorkflowGroupChangedForNode(node)
  workflowTextNodeStates.value = { ...workflowTextNodeStates.value }
  syncNodeStateMap('workflowTextNodeStates', workflowTextNodeStates.value, { persist: false })
}

function scheduleLlmNodeInputSync(node) {
  window.clearTimeout(llmNodeInputSyncTimers.get(node))
  llmNodeInputSyncTimers.set(node, window.setTimeout(() => {
    llmNodeInputSyncTimers.delete(node)
    markTemplateWorkflowGroupChangedForNode(node)
    workflowLlmNodeStates.value = { ...workflowLlmNodeStates.value }
    syncNodeStateMap('workflowLlmNodeStates', workflowLlmNodeStates.value, { persist: false })
  }, 240))
}

function flushLlmNodeInputSync(node) {
  if (!llmNodeInputSyncTimers.has(node)) return
  window.clearTimeout(llmNodeInputSyncTimers.get(node))
  llmNodeInputSyncTimers.delete(node)
  markTemplateWorkflowGroupChangedForNode(node)
  workflowLlmNodeStates.value = { ...workflowLlmNodeStates.value }
  syncNodeStateMap('workflowLlmNodeStates', workflowLlmNodeStates.value, { persist: false })
}

function updateTextNode(node, patch, options = {}) {
  if (Object.keys(patch || {}).some(key => ['prompt', 'result', 'uploads', 'title'].includes(key))) {
    markTemplateWorkflowGroupChangedForNode(node)
  }
  const current = ensureTextNodeState(node)
  Object.assign(current, patch)
  syncNodeStateMap('workflowTextNodeStates', workflowTextNodeStates.value, options)
}

function textNodeInputElement(node) {
  return Array.from(document.querySelectorAll('textarea.text-node-input'))
    .find(input => input.dataset.textNode === node) || null
}

function editingTextNodeIds() {
  return Object.entries(workflowTextNodeStates.value || {})
    .filter(([node, state]) => nodeExists(node) && state?.resultEditing)
    .map(([node]) => node)
}

function closeEditingTextNodesExcept(keepNode = '', options = {}) {
  let changed = false
  const nextStates = { ...workflowTextNodeStates.value }
  for (const node of editingTextNodeIds()) {
    if (node === keepNode) continue
    flushTextNodeInputSync(node)
    nextStates[node] = {
      ...nextStates[node],
      resultEditing: false
    }
    changed = true
  }
  if (!changed) return
  workflowTextNodeStates.value = nextStates
  syncNodeStateMap('workflowTextNodeStates', workflowTextNodeStates.value, options)
}

function closeTextNodeEditOnOutside(e) {
  const editingNodes = editingTextNodeIds()
  const editingLlmNodes = Object.entries(workflowLlmNodeStates.value || {})
    .filter(([node, state]) => nodeExists(node) && state?.editing)
    .map(([node]) => node)
  if (!editingNodes.length && !editingLlmNodes.length) return
  const target = e.target
  const insideEditingTextNode = editingNodes.some(node => {
    const card = textNodeInputElement(node)?.closest?.('.node-text-workflow')
    return card?.contains?.(target)
  })
  const insideEditingLlmNode = editingLlmNodes.some(node => llmNodeInputElement(node)?.closest?.('.node-llm-workflow')?.contains?.(target))
  if (insideEditingTextNode || insideEditingLlmNode) return
  closeEditingTextNodesExcept('')
  editingLlmNodes.forEach(node => disableLlmNodeEdit(node))
}

function normalizeLlmNodeStates(states = {}, { resetRuntime = false } = {}) {
  return Object.keys(states).reduce((map, key) => {
    map[key] = defaultLlmNodeState({
      ...states[key],
      ...(resetRuntime ? { loading: false, progress: 0, analysisLabel: '', runId: '', editing: false } : {})
    })
    return map
  }, {})
}

function enableTextNodeEdit(node, e) {
  updateTextNode(node, { resultEditing: true }, { persist: false })
  closeEditingTextNodesExcept(node, { persist: false })
  nextTick(() => {
    const target = textNodeInputElement(node) || e?.target
    if (target?.focus) {
      target.focus()
      target.setSelectionRange?.(target.value.length, target.value.length)
    }
  })
}

function handleTextNodeInputKeydown(node, e) {
  if (!ensureTextNodeState(node).resultEditing) return
  if (e.key === 'Escape') {
    e.preventDefault()
    disableTextNodeEdit(node)
  }
  e.stopPropagation()
}

function handleTextNodeInputKeyup(node, e) {
  if (!ensureTextNodeState(node).resultEditing) return
  e.stopPropagation()
}

function disableTextNodeEdit(node, options = {}) {
  if (!ensureTextNodeState(node).resultEditing) return
  flushTextNodeInputSync(node)
  updateTextNode(node, { resultEditing: false }, options)
}

function startTextNodeProgress(node) {
  window.clearInterval(textNodeProgressTimers.get(node))
  textNodeProgressTimers.set(node, window.setInterval(() => {
    const state = ensureTextNodeState(node)
    if (!state.loading) {
      window.clearInterval(textNodeProgressTimers.get(node))
      textNodeProgressTimers.delete(node)
      return
    }
    const nextProgress = Math.min(92, (Number(state.progress) || 0) + 3)
    updateTextNode(node, { progress: nextProgress }, { persist: false, transient: true })
  }, 800))
}

function stopTextNodeProgress(node) {
  window.clearInterval(textNodeProgressTimers.get(node))
  textNodeProgressTimers.delete(node)
}

function startLlmNodeProgress(node) {
  stopLlmNodeProgress(node)
  llmNodeProgressTimers.set(node, window.setInterval(() => {
    const state = workflowLlmNodeStates.value[node]
    if (!nodeExists(node) || !state?.loading) {
      stopLlmNodeProgress(node)
      return
    }
    state.progress = Math.min(92, (Number(state.progress) || 0) + 3)
  }, 800))
}

function stopLlmNodeProgress(node) {
  window.clearInterval(llmNodeProgressTimers.get(node))
  llmNodeProgressTimers.delete(node)
}

function startTextNodeResize(e, node) {
  const pos = nodePositions.value[node]
  if (!pos) return
  markCanvasInteraction()
  textNodeResize = {
    node,
    pointerId: e.pointerId,
    startX: e.clientX,
    startY: e.clientY,
    width: pos.width,
    height: pos.height
  }
  e.currentTarget.setPointerCapture?.(e.pointerId)
  window.addEventListener('pointermove', dragTextNodeResize)
  window.addEventListener('pointerup', endTextNodeResize)
  window.addEventListener('pointercancel', endTextNodeResize)
}

function dragTextNodeResize(e) {
  if (!textNodeResize || textNodeResize.pointerId !== e.pointerId) return
  markCanvasInteraction()
  const pos = nodePositions.value[textNodeResize.node]
  if (!pos) return
  const scale = liveCanvasView.scale || 1
  pos.width = Math.max(320, Math.round(textNodeResize.width + (e.clientX - textNodeResize.startX) / scale))
  pos.height = Math.max(220, Math.round(textNodeResize.height + (e.clientY - textNodeResize.startY) / scale))
}

function endTextNodeResize(e) {
  if (!textNodeResize || textNodeResize.pointerId !== e.pointerId) return
  window.removeEventListener('pointermove', dragTextNodeResize)
  window.removeEventListener('pointerup', endTextNodeResize)
  window.removeEventListener('pointercancel', endTextNodeResize)
  textNodeResize = null
  bumpViewportCheck()
  scheduleNodeLayoutStateSave()
}

async function imageUrlToPromptPayload(url) {
  const payload = await imageUrlToRef(url)
  if (!payload?.data) throw new Error('未获取到参考图')
  return {
    image_data: payload.data,
    mime_type: payload.mime_type || 'image/png'
  }
}

const SCENE_STYLE_COLOR_PROMPT_INSTRUCTION = `请只分析图片中的配色和硬装材质，不要生成任何额外提示词、解释、约束或总结。
如果画面里有水龙头、卫浴五金、厨房五金、产品主体、品牌文字、人物、具体商品或醒目可替换物体，请忽略它们，不要写进结果。
不要描述或继承任何产品造型、产品角度、产品材质、安装孔位、出水方向、镜头焦段、相机角度、构图裁切、空间布局、窗户、镜子、灯具、软装、装饰陈设或画面中原产品的位置。
只按下面字段输出，字段缺失或看不清就写“无”：
主色：
辅助色：
点缀色：
冷暖：
明暗倾向：
背景墙面材质质感：
地面材质质感：
台面材质质感：
柜子材质质感：`

async function generateSceneStyleColorPrompt(styleRef = null) {
  const url = styleRef?.url ? resolveImageUrl(styleRef.url) : ''
  if (!url) return ''
  const payload = await imageUrlToPromptPayload(url)
  const res = await request.post('/image-to-prompt/generate', {
    ...payload,
    instruction: SCENE_STYLE_COLOR_PROMPT_INSTRUCTION
  })
  return (res.data?.prompt || '').trim()
}

async function ensureModelSceneStyleColorPrompt(modelNode) {
  const state = ensureModelNodeState(modelNode)
  const styleRef = normalizeModelSceneStyleRef(state.sceneStyleRef)
  const styleUrl = styleRef?.url ? resolveImageUrl(styleRef.url) : ''
  if (!styleUrl) return styleRef
  const analyzedUrl = styleRef.promptImageUrl ? resolveImageUrl(styleRef.promptImageUrl) : ''
  if (styleRef.prompt && analyzedUrl === styleUrl) {
    return styleRef
  }
  renderTaskMessage.value = '分析配色材质...'
  const prompt = await generateSceneStyleColorPrompt(styleRef)
  const nextRef = {
    ...styleRef,
    prompt,
    promptImageUrl: styleUrl
  }
  updateActiveModelExportOptions({ sceneStyleRef: nextRef }, modelNode)
  return nextRef
}

async function createImageAnalysisTextNode(imageNode, mode = 'prompt') {
  const imageUrl = primaryGenerationImage(imageNode)
  if (!imageUrl) return
  const sourceRect = computedNodeRect(imageNode) || nodePositions.value[imageNode] || DEFAULT_NODE_POSITIONS.generate
  const textBase = DEFAULT_NODE_POSITIONS.text
  const position = findNonOverlappingPosition({
    ...textBase,
    x: Math.round(sourceRect.x + (sourceRect.width || DEFAULT_NODE_POSITIONS.generate.width) + 92),
    y: Math.round(sourceRect.y + ((sourceRect.height || DEFAULT_NODE_POSITIONS.generate.height) - textBase.height) / 2)
  }, { exclude: [imageNode], stepY: textBase.height + 48, gap: 56 })
  const isSuggestion = mode === 'suggestion'
  const analysisLabel = isSuggestion ? '生成图片优化建议' : '反推图片提示词'
  pushWorkflowUndoSnapshot()
  const textNode = addWorkflowNode('text', {
    position,
    groupId: workflowGroupIdForNode(imageNode),
    state: {
      title: isSuggestion ? '图片优化建议节点' : '图片提示词节点',
      prompt: '',
      result: '',
      loading: true,
      progress: 0,
      error: '',
      analysisLabel
    },
    persist: false,
    undo: false
  })
  if (!textNode) return
  connectWorkflowNodes(imageNode, textNode, { persist: false, undo: false })
  activeWorkflowNode.value = textNode
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  saveWorkflowStructureState({ persist: false })
  startTextNodeProgress(textNode)
  try {
    const payload = await imageUrlToPromptPayload(imageUrl)
    const res = await request.post('/image-to-prompt/generate', {
      ...payload,
      instruction: isSuggestion ? IMAGE_OPTIMIZATION_SUGGESTION_TEXT : IMAGE_TO_PROMPT_FIXED_TEXT
    })
    if (!nodeExists(textNode)) return
    stopTextNodeProgress(textNode)
    updateTextNode(textNode, {
      prompt: (res.data?.prompt || '').trim(),
      result: '',
      loading: false,
      progress: 100,
      error: '',
      imagePromptMode: false,
      imagePromptFlowCreated: false,
      analysisLabel: '',
      inputTokens: res.data?.input_tokens || 0,
      outputTokens: res.data?.output_tokens || 0
    })
  } catch (e) {
    if (!nodeExists(textNode)) return
    stopTextNodeProgress(textNode)
    updateTextNode(textNode, {
      loading: false,
      progress: 0,
      analysisLabel: '',
      error: e?.response?.data?.error || e?.message || (isSuggestion ? '图片优化建议生成失败' : '图片反推提示词失败')
    })
  }
}

function createDepthPlaceholderNode(imageNode) {
  const imageUrl = primaryGenerationImage(imageNode)
  if (!imageUrl) return ''
  const sourceRect = computedNodeRect(imageNode) || nodePositions.value[imageNode] || DEFAULT_NODE_POSITIONS.generate
  const base = generateResultNodeSizeForImage(imageUrl, DEFAULT_NODE_POSITIONS.generate)
  const position = findNonOverlappingPosition({
    ...base,
    x: Math.round(sourceRect.x + (sourceRect.width || DEFAULT_NODE_POSITIONS.generate.width) + GENERATED_RESULT_NODE_GAP_X),
    y: Math.round(sourceRect.y + Math.max(0, ((sourceRect.height || base.height) - base.height) / 2))
  }, {
    exclude: [imageNode],
    stepY: GENERATED_RESULT_NODE_GAP_Y + 120
  })
  const sourceState = ensureGenerateNodeState(imageNode)
  const node = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(imageNode),
    position,
    state: {
      prompt: sourceState.prompt || '',
      model: sourceState.model || defaultDrawingModelName(),
      count: 1,
      aspectRatio: sourceState.aspectRatio || '1:1',
      resolution: sourceState.resolution || '2K',
      generatedFromNode: imageNode,
      generating: true,
      progress: 4,
      result: {
        url: '',
        images: [],
        imageCreatedAt: [],
        imageSources: ['depth-of-field'],
        taskIds: [],
        taskId: '',
        createdAt: new Date().toISOString(),
        status: 'running',
        batchTitle: '景深图'
      },
      uploads: [],
      showResults: false
    }
  })
  if (!node) return ''
  connectWorkflowNodes(imageNode, node, { undo: false, persist: false })
  return node
}

function activeDepthNodeRunState(node, runId) {
  const state = generateNodeStateIfExists(node)
  if (!state || state.runId !== runId || state.cancelled) return null
  return state
}

function depthQualityForNode(node) {
  const resolution = ensureGenerateNodeState(node).resolution
  return resolution === '4K' ? '4K' : '2K'
}

async function pollDepthTaskForNode(node, taskId, runId) {
  for (let i = 0; i < 240; i += 1) {
    const state = activeDepthNodeRunState(node, runId)
    if (!state) return ''
    const res = await request.get(`/depth/tasks/${taskId}`)
    const data = res.data || {}
    const item = Array.isArray(data.images) ? data.images[0] || {} : {}
    const status = item.status || data.status || 'running'
    const imageUrl = resolveImageUrl(item.image_url || '')
    updateGenerateNode(node, {
      progress: Math.min(96, Math.max(state.progress || 0, 12 + i * 3)),
      result: {
        ...state.result,
        taskId,
        taskIds: [taskId],
        status
      }
    }, { persist: false })
    if (imageUrl) return imageUrl
    if (status === 'error' || data.status === 'failed') {
      throw new Error(item.error || data.error || '景深图生成失败')
    }
    await new Promise(resolve => window.setTimeout(resolve, 2000))
  }
  throw new Error('景深图生成超时，请稍后查看任务结果')
}

async function createDepthOfFieldImageNode(imageNode) {
  const imageUrl = primaryGenerationImage(imageNode)
  if (!imageUrl || ensureGenerateNodeState(imageNode).generating) return
  pushWorkflowUndoSnapshot()
  const depthNode = createDepthPlaceholderNode(imageNode)
  if (!depthNode) return
  const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`
  updateGenerateNode(depthNode, { runId, cancelled: false, progress: 4 }, { persist: false })
  activeWorkflowNode.value = depthNode
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  saveWorkflowStructureState({ persist: false })
  try {
    const payload = await imageUrlToPromptPayload(imageUrl)
    const sourceState = ensureGenerateNodeState(imageNode)
    const res = await request.post('/depth/tasks', {
      images: [{ data: payload.image_data, mime_type: payload.mime_type || 'image/png' }],
      aspect_ratio: sourceState.aspectRatio || '1:1',
      quality: depthQualityForNode(imageNode)
    })
    const taskId = res.data?.task_id || ''
    if (!taskId) throw new Error('未获取到景深图任务ID')
    const resultUrl = await pollDepthTaskForNode(depthNode, taskId, runId)
    if (!resultUrl || !activeDepthNodeRunState(depthNode, runId)) return
    const createdAt = new Date().toISOString()
    updateGenerateNode(depthNode, {
      generating: false,
      runId: '',
      progress: 100,
      error: '',
      result: {
        url: resultUrl,
        images: [resultUrl],
        imageCreatedAt: [createdAt],
        imageSources: ['depth-of-field'],
        taskIds: [taskId],
        taskId,
        createdAt,
        status: 'completed',
        batchTitle: '景深图'
      },
      uploads: [],
      showResults: false
    })
    appendWorkflowHistoryAssets([{ url: resultUrl, createdAt, node: depthNode, taskId }])
  } catch (e) {
    if (!nodeExists(depthNode)) return
    updateGenerateNode(depthNode, {
      generating: false,
      runId: '',
      progress: 100,
      error: e?.response?.data?.error || e?.message || '景深图生成失败',
      result: {
        ...ensureGenerateNodeState(depthNode).result,
        status: 'failed'
      }
    })
  }
}

function llmNodeState(node) {
  if (!workflowLlmNodeStates.value[node]) workflowLlmNodeStates.value[node] = defaultLlmNodeState()
  return workflowLlmNodeStates.value[node]
}

function workflowTextOutput(node) {
  const type = nodeType(node)
  if (type === 'text') return textNodeOutputText(node)
  if (type === 'llm') {
    const state = llmNodeState(node)
    return String(state.output || state.prompt || '').trim()
  }
  return ''
}

function linkedWorkflowTextOutputs(node) {
  return [...new Set(workflowLinks.value
    .filter(link => link.to === node && ['text', 'llm'].includes(nodeType(link.from)))
    .map(link => workflowTextOutput(link.from))
    .filter(Boolean))]
}

function effectiveLlmPrompt(node) {
  const state = llmNodeState(node)
  return [...linkedWorkflowTextOutputs(node), state.prompt.trim()].filter(Boolean).join('\n')
}

const MAX_LLM_IMAGE_INPUTS = 8

function linkedLlmImageUrls(node) {
  return generationReferenceImages(node)
    .filter(item => item.type !== 'text' && item.url)
    .map(item => item.url)
}

function defaultChatModelName() {
  return chatModels.value.find(model => model.is_default)?.name || chatModels.value[0]?.name || ''
}

function effectiveChatModelName(node) {
  const configured = llmNodeState(node).model
  if (configured && (!chatModels.value.length || chatModels.value.some(model => model.name === configured))) return configured
  return defaultChatModelName()
}

function defaultLlmNodeState(state = {}) {
  return {
    prompt: '',
    output: '',
    error: '',
    loading: false,
    progress: 0,
    analysisLabel: '',
    runId: '',
    editing: false,
    ...state,
    model: state.model || defaultChatModelName()
  }
}

function onLlmNodeInput(node, value) {
  const state = toRaw(llmNodeState(node))
  state.prompt = value
  state.error = ''
  state.output = ''
  scheduleLlmNodeInputSync(node)
}

function llmNodeInputElement(node) {
  return document.querySelector(`textarea.llm-node-textarea[data-llm-node="${node}"]`)
}

function enableLlmNodeEdit(node, event) {
  const state = llmNodeState(node)
  state.editing = true
  closeEditingTextNodesExcept('', { persist: false })
  saveProjectPatch({ workflowLlmNodeStates: clonePlain(workflowLlmNodeStates.value) }, { persist: false })
  nextTick(() => {
    const input = llmNodeInputElement(node) || event?.target
    input?.focus?.()
    input?.setSelectionRange?.(input.value.length, input.value.length)
  })
}

function disableLlmNodeEdit(node, options = {}) {
  const state = llmNodeState(node)
  if (!state.editing) return
  flushLlmNodeInputSync(node)
  state.editing = false
  saveProjectPatch({ workflowLlmNodeStates: clonePlain(workflowLlmNodeStates.value) }, { persist: options.persist !== false })
}

function handleLlmNodeKeydown(node, event) {
  if (!llmNodeState(node).editing) return
  event.stopPropagation()
  if (event.key === 'Escape') {
    event.preventDefault()
    disableLlmNodeEdit(node)
  }
}

function convertLlmNodeToText(node, content) {
  workflowDynamicNodes.value = workflowDynamicNodes.value.map(item => (
    item.id === node ? { ...item, type: 'text' } : item
  ))
  workflowTextNodeStates.value = {
    ...workflowTextNodeStates.value,
    [node]: defaultTextNodeState({ prompt: content })
  }
  const { [node]: removedState, ...nextStates } = workflowLlmNodeStates.value
  workflowLlmNodeStates.value = nextStates
}

function migrateCompletedLlmNodesToText(nodes = null) {
  const included = nodes ? new Set(nodes) : null
  workflowDynamicNodes.value
    .filter(item => item.type === 'llm' && (!included || included.has(item.id)))
    .forEach((item) => {
      const content = String(workflowLlmNodeStates.value[item.id]?.output || '').trim()
      if (content) convertLlmNodeToText(item.id, content)
    })
}

async function runLlmNode(node) {
  const state = llmNodeState(node)
  const prompt = effectiveLlmPrompt(node)
  if (!prompt || state.loading) return
  const projectId = activeProjectId.value
  const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`
  const modelName = effectiveChatModelName(node)
  const imageUrls = linkedLlmImageUrls(node)
  if (imageUrls.length > MAX_LLM_IMAGE_INPUTS) {
    state.error = `大模型节点最多支持 ${MAX_LLM_IMAGE_INPUTS} 张图片`
    return
  }
  disableLlmNodeEdit(node, { persist: false })
  state.loading = true
  state.progress = 0
  state.analysisLabel = imageUrls.length > 1 ? `正在分析 ${imageUrls.length} 张图片` : (imageUrls.length ? '正在分析图片' : '正在生成文本')
  state.error = ''
  state.runId = runId
  state.model = modelName
  startLlmNodeProgress(node)
  let completedContent = ''
  try {
    const res = imageUrls.length
      ? await request.post('/image-to-prompt/generate', {
          images: await Promise.all(imageUrls.map(imageUrlToPromptPayload)),
          instruction: prompt
        })
      : await request.post('/chat', { model: modelName, messages: [{ role: 'user', content: prompt }], stream: false })
    if (activeProjectId.value !== projectId || workflowLlmNodeStates.value[node]?.runId !== runId) return
    const content = String((imageUrls.length ? res.data?.prompt : res.data?.content) || '').trim()
    if (!content) throw new Error('模型没有返回内容')
    state.model = res.data?.model || modelName
    state.output = content
    state.prompt = content
    state.progress = 100
    completedContent = content
  } catch (e) {
    if (activeProjectId.value === projectId && workflowLlmNodeStates.value[node]?.runId === runId) {
      state.progress = 0
      state.error = e?.response?.data?.error || e?.message || '大模型运行失败'
    }
  } finally {
    stopLlmNodeProgress(node)
    if (activeProjectId.value === projectId && workflowLlmNodeStates.value[node]?.runId === runId) {
      state.loading = false
      state.analysisLabel = ''
      state.runId = ''
      if (completedContent) convertLlmNodeToText(node, completedContent)
      saveWorkflowStructureState()
    }
  }
}

function createLightShadowPlaceholderNode(imageNode) {
  const imageUrl = primaryGenerationImage(imageNode)
  if (!imageUrl) return ''
  const sourceRect = computedNodeRect(imageNode) || nodePositions.value[imageNode] || DEFAULT_NODE_POSITIONS.generate
  const base = generateResultNodeSizeForImage(imageUrl, DEFAULT_NODE_POSITIONS.generate)
  const position = findNonOverlappingPosition({
    ...base,
    x: Math.round(sourceRect.x + (sourceRect.width || DEFAULT_NODE_POSITIONS.generate.width) + GENERATED_RESULT_NODE_GAP_X),
    y: Math.round(sourceRect.y + Math.max(0, ((sourceRect.height || base.height) - base.height) / 2))
  }, {
    exclude: [imageNode],
    stepY: GENERATED_RESULT_NODE_GAP_Y + 120
  })
  const sourceState = ensureGenerateNodeState(imageNode)
  const node = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(imageNode),
    position,
    state: {
      prompt: sourceState.prompt || '',
      model: sourceState.model || defaultDrawingModelName(),
      count: 1,
      aspectRatio: sourceState.aspectRatio || '1:1',
      resolution: sourceState.resolution || '2K',
      generatedFromNode: imageNode,
      generating: true,
      progress: 4,
      result: {
        url: '',
        images: [],
        imageCreatedAt: [],
        imageSources: ['light-shadow'],
        taskIds: [],
        taskId: '',
        createdAt: new Date().toISOString(),
        status: 'running',
        batchTitle: '光影优化'
      },
      uploads: [],
      showResults: false
    }
  })
  if (!node) return ''
  connectWorkflowNodes(imageNode, node, { undo: false, persist: false })
  return node
}

function activeLightShadowNodeRunState(node, runId) {
  const state = generateNodeStateIfExists(node)
  if (!state || state.runId !== runId || state.cancelled) return null
  return state
}

async function pollLightShadowTaskForNode(node, taskId, runId) {
  for (let i = 0; i < 240; i += 1) {
    const state = activeLightShadowNodeRunState(node, runId)
    if (!state) return ''
    const res = await request.get(`/light-shadow/tasks/${taskId}`)
    const data = res.data || {}
    const item = Array.isArray(data.images) ? data.images[0] || {} : {}
    const status = item.status || data.status || 'running'
    const imageUrl = resolveImageUrl(item.image_url || '')
    updateGenerateNode(node, {
      progress: Math.min(96, Math.max(state.progress || 0, 12 + i * 3)),
      result: {
        ...state.result,
        taskId,
        taskIds: [taskId],
        status
      }
    }, { persist: false })
    if (imageUrl) return imageUrl
    if (status === 'error' || data.status === 'failed') {
      throw new Error(item.error || data.error || '光影优化失败')
    }
    await new Promise(resolve => window.setTimeout(resolve, 2000))
  }
  throw new Error('光影优化超时，请稍后查看任务结果')
}

async function createLightShadowImageNode(imageNode) {
  const imageUrl = primaryGenerationImage(imageNode)
  if (!imageUrl || ensureGenerateNodeState(imageNode).generating) return
  pushWorkflowUndoSnapshot()
  const lightNode = createLightShadowPlaceholderNode(imageNode)
  if (!lightNode) return
  const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`
  updateGenerateNode(lightNode, { runId, cancelled: false, progress: 4 }, { persist: false })
  activeWorkflowNode.value = lightNode
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  saveWorkflowStructureState({ persist: false })
  try {
    const payload = await imageUrlToPromptPayload(imageUrl)
    const sourceState = ensureGenerateNodeState(imageNode)
    const res = await request.post('/light-shadow/tasks', {
      images: [{ data: payload.image_data, mime_type: payload.mime_type || 'image/png' }],
      aspect_ratio: sourceState.aspectRatio || '1:1',
      quality: depthQualityForNode(imageNode)
    })
    const taskId = res.data?.task_id || ''
    if (!taskId) throw new Error('未获取到光影优化任务ID')
    const resultUrl = await pollLightShadowTaskForNode(lightNode, taskId, runId)
    if (!resultUrl || !activeLightShadowNodeRunState(lightNode, runId)) return
    const createdAt = new Date().toISOString()
    updateGenerateNode(lightNode, {
      generating: false,
      runId: '',
      progress: 100,
      error: '',
      result: {
        url: resultUrl,
        images: [resultUrl],
        imageCreatedAt: [createdAt],
        imageSources: ['light-shadow'],
        taskIds: [taskId],
        taskId,
        createdAt,
        status: 'completed',
        batchTitle: '光影优化'
      },
      uploads: [],
      showResults: false
    })
    appendWorkflowHistoryAssets([{ url: resultUrl, createdAt, node: lightNode, taskId }])
  } catch (e) {
    if (!nodeExists(lightNode)) return
    updateGenerateNode(lightNode, {
      generating: false,
      runId: '',
      progress: 100,
      error: e?.response?.data?.error || e?.message || '光影优化失败',
      result: {
        ...ensureGenerateNodeState(lightNode).result,
        status: 'failed'
      }
    })
  }
}

function updateGenerateNode(node, patch, options = {}) {
  markTemplateWorkflowGroupChangedForNode(node)
  const current = ensureGenerateNodeState(node)
  Object.assign(current, patch)
  syncNodeStateMap('workflowGenerateNodeStates', workflowGenerateNodeStates.value, options)
}

function ensureBatchRecolorNodeState(node) {
  if (!workflowBatchRecolorNodeStates.value[node]) {
    workflowBatchRecolorNodeStates.value = {
      ...workflowBatchRecolorNodeStates.value,
      [node]: defaultBatchRecolorNodeState()
    }
  }
  return workflowBatchRecolorNodeStates.value[node]
}

function batchRecolorNodeState(node) {
  return ensureBatchRecolorNodeState(node)
}

function updateBatchRecolorNode(node, patch, options = {}) {
  markTemplateWorkflowGroupChangedForNode(node)
  const current = ensureBatchRecolorNodeState(node)
  Object.assign(current, normalizeBatchRecolorNodeState({ ...current, ...patch }))
  syncNodeStateMap('workflowBatchRecolorNodeStates', workflowBatchRecolorNodeStates.value, options)
}

function ensureBatchTranslateNodeState(node) {
  if (!workflowBatchTranslateNodeStates.value[node]) {
    workflowBatchTranslateNodeStates.value = {
      ...workflowBatchTranslateNodeStates.value,
      [node]: defaultBatchTranslateNodeState()
    }
  }
  return workflowBatchTranslateNodeStates.value[node]
}

function batchTranslateNodeState(node) {
  return ensureBatchTranslateNodeState(node)
}

function updateBatchTranslateNode(node, patch, options = {}) {
  markTemplateWorkflowGroupChangedForNode(node)
  const current = ensureBatchTranslateNodeState(node)
  Object.assign(current, normalizeBatchTranslateNodeState({ ...current, ...patch }))
  syncNodeStateMap('workflowBatchTranslateNodeStates', workflowBatchTranslateNodeStates.value, options)
}

function ensureVideoNodeState(node) {
  if (!workflowVideoNodeStates.value[node]) {
    workflowVideoNodeStates.value = {
      ...workflowVideoNodeStates.value,
      [node]: defaultVideoNodeState()
    }
  }
  return workflowVideoNodeStates.value[node]
}

function videoNodeState(node) {
  return ensureVideoNodeState(node)
}

function updateVideoNode(node, patch, options = {}) {
  markTemplateWorkflowGroupChangedForNode(node)
  const current = ensureVideoNodeState(node)
  Object.assign(current, normalizeVideoNodeState({ ...current, ...patch }))
  syncNodeStateMap('workflowVideoNodeStates', workflowVideoNodeStates.value, options)
}

function rememberVideoNodeSize(node, event) {
  const video = event?.target
  const width = Number(video?.videoWidth) || 0
  const height = Number(video?.videoHeight) || 0
  if (!node || !width || !height) return
  const state = ensureVideoNodeState(node)
  if (state.videoWidth === width && state.videoHeight === height) return
  updateVideoNode(node, { videoWidth: width, videoHeight: height }, { persist: false })
  resizeGroupsForNode(node)
  scheduleNodeLayoutStateSave()
}

function updateTextNodeText(node, text) {
  Object.assign(toRaw(ensureTextNodeState(node)), {
    prompt: text,
    result: '',
    error: '',
    imagePromptMode: false,
    imagePromptFlowCreated: false,
    resultEditing: true
  })
  scheduleTextNodeInputSync(node)
}

function activeBatchRecolorRunState(node, runId) {
  if (!nodeExists(node)) return null
  const state = workflowBatchRecolorNodeStates.value[node]
  return state?.runId === runId ? state : null
}

function activeBatchTranslateRunState(node, runId) {
  if (!nodeExists(node)) return null
  const state = workflowBatchTranslateNodeStates.value[node]
  return state?.runId === runId ? state : null
}

function generateNodeStateIfExists(node) {
  if (!node || !nodeExists(node) || nodeType(node) !== 'generate') return null
  return ensureGenerateNodeState(node)
}

function activeGenerateRunState(node, runId) {
  const state = generateNodeStateIfExists(node)
  if (!state) return null
  return state.runId === runId ? state : null
}

async function runLimitedBatch(items, limit, worker) {
  const queue = items.map((item, index) => ({ item, index }))
  const workers = Array.from({ length: Math.min(Math.max(1, limit), queue.length) }, async () => {
    while (queue.length) {
      const next = queue.shift()
      if (!next) return
      await worker(next.item, next.index)
    }
  })
  await Promise.all(workers)
}

function batchFailedCount(tasks = []) {
  return tasks.filter(task => task.status === 'failed' || task.status === 'error').length
}

function batchCompletedCount(tasks = []) {
  return tasks.filter(task => task.status === 'completed' && task.imageUrl).length
}

function batchOverallProgress(tasks = []) {
  if (!tasks.length) return 0
  const total = tasks.reduce((sum, task) => {
    if (task.status === 'completed' || task.status === 'failed' || task.status === 'error') return sum + 100
    return sum + Math.min(98, Math.max(0, Number(task.progress) || 0))
  }, 0)
  return Math.floor(total / tasks.length)
}

function batchRecolorProgressText(node) {
  const state = ensureBatchRecolorNodeState(node)
  if (!state.tasks.length) return `${state.materials.length} 个材质`
  const failed = batchFailedCount(state.tasks)
  return failed ? `${batchCompletedCount(state.tasks)}/${state.tasks.length} · 失败 ${failed}` : `${batchCompletedCount(state.tasks)}/${state.tasks.length}`
}

function batchTranslateProgressText(node) {
  const state = ensureBatchTranslateNodeState(node)
  if (!state.tasks.length) return '等待输入'
  const failed = batchFailedCount(state.tasks)
  return failed ? `${batchCompletedCount(state.tasks)}/${state.tasks.length} · 失败 ${failed}` : `${batchCompletedCount(state.tasks)}/${state.tasks.length}`
}

function batchRecolorQualityLabel(node) {
  const state = ensureBatchRecolorNodeState(node)
  return `${state.aspectRatio} · ${state.resolution}`
}

function batchTranslateQualityLabel(node) {
  const state = ensureBatchTranslateNodeState(node)
  return `${state.aspectRatio} · ${state.resolution}`
}

function toggleBatchRecolorMenu(node, menu) {
  const state = ensureBatchRecolorNodeState(node)
  updateBatchRecolorNode(node, { menu: state.menu === menu ? '' : menu }, { persist: false })
}

function toggleBatchTranslateMenu(node, menu) {
  const state = ensureBatchTranslateNodeState(node)
  updateBatchTranslateNode(node, { menu: state.menu === menu ? '' : menu }, { persist: false })
}

function isBatchTranslateLanguageSelected(node, language) {
  return ensureBatchTranslateNodeState(node).targetLanguages.includes(language)
}

function toggleBatchTranslateLanguage(node, language) {
  if (!batchTranslateLanguageOptions.some(item => item.value === language)) return
  const state = ensureBatchTranslateNodeState(node)
  const exists = state.targetLanguages.includes(language)
  const nextLanguages = exists
    ? state.targetLanguages.filter(item => item !== language)
    : [...state.targetLanguages, language]
  if (!nextLanguages.length) return
  updateBatchTranslateNode(node, {
    targetLanguage: nextLanguages[0],
    targetLanguages: nextLanguages,
    tasks: [],
    error: ''
  })
}

function batchTaskStatusText(task = {}) {
  if (task.status === 'completed') return '完成'
  if (task.status === 'running') return '进行中'
  if (task.status === 'failed' || task.status === 'error') return '失败'
  return '等待'
}

function batchRecolorInputText(node) {
  const state = ensureBatchRecolorNodeState(node)
  const sourceCount = batchSourceImages(node).length
  const sourceText = sourceCount ? `已连接 ${sourceCount} 张图片` : '未连接图片'
  const materialText = state.materials.length ? `${state.materials.length} 个材质` : '未选材质'
  return `${sourceText} · ${materialText}`
}

function batchRecolorTaskMeta(task = {}) {
  if (task.imageUrl) return '已生成'
  if (task.error) return task.error
  return '等待换色'
}

function batchLanguageLabel(value) {
  return batchTranslateLanguageOptions.find(item => item.value === value)?.label || value || '翻译'
}

function openBatchMaterialLibrary(node) {
  if (ensureBatchRecolorNodeState(node).loading) return
  batchMaterialPickerNode.value = node
  ensureBatchRecolorNodeState(node)
  ensureModelSceneMaterialsLoaded()
}

function closeBatchMaterialLibrary() {
  batchMaterialPickerNode.value = ''
}

function isBatchMaterialSelected(node, materialValue) {
  return ensureBatchRecolorNodeState(node).materials.some(item => item.value === materialValue)
}

function toggleBatchRecolorMaterial(node, material) {
  const normalized = normalizeBatchMaterial(material)
  if (!normalized) return
  const state = ensureBatchRecolorNodeState(node)
  if (state.loading) return
  const exists = state.materials.some(item => item.value === normalized.value)
  updateBatchRecolorNode(node, {
    materials: exists
      ? state.materials.filter(item => item.value !== normalized.value)
      : [...state.materials, normalized],
    tasks: [],
    error: ''
  })
}

function removeBatchRecolorMaterial(node, materialValue) {
  const state = ensureBatchRecolorNodeState(node)
  if (state.loading) return
  updateBatchRecolorNode(node, {
    materials: state.materials.filter(item => item.value !== materialValue),
    tasks: [],
    error: ''
  })
}

function canRunBatchRecolor(node) {
  const state = ensureBatchRecolorNodeState(node)
  return !!state.target.trim() && state.materials.length > 0 && batchSourceImages(node).length > 0 && !state.loading
}

function canRunBatchTranslate(node) {
  const state = ensureBatchTranslateNodeState(node)
  return batchTranslateInputItems(node).length > 0 && !state.loading
}

function linkedInputGenerateNodes(node) {
  return workflowLinks.value
    .filter(link => link.to === node && nodeType(link.from) === 'generate')
    .map(link => link.from)
}

function batchSourceImages(node) {
  const items = []
  linkedInputGenerateNodes(node).forEach((sourceNode) => {
    batchResultItemsFromGenerateNode(sourceNode).forEach((item) => {
      items.push({
        sourceNode,
        sourceIndex: items.length,
        sourceName: item.materialName || generateNodeTitle(sourceNode),
        imageUrl: item.imageUrl
      })
    })
  })
  return items.filter(item => item.imageUrl)
}

function batchSourceImage(node) {
  return batchSourceImages(node)[0]?.imageUrl || ''
}

function batchResultItemsFromGenerateNode(node) {
  if (!node || nodeType(node) !== 'generate') return []
  const state = ensureGenerateNodeState(node)
  const result = normalizeGenerationResult(state.result)
  const batchItems = Array.isArray(result.batchItems) ? result.batchItems : []
  const images = result.images.length ? result.images : generationUploadImages(node)
  return images.map((url, index) => ({
    index,
    materialId: batchItems[index]?.materialId || '',
    materialName: batchItems[index]?.materialName || (result.images.length ? '' : `图片${index + 1}`),
    materialImageUrl: batchItems[index]?.materialImageUrl || '',
    sourceImageUrl: batchItems[index]?.sourceImageUrl || resolveImageUrl(url),
    imageUrl: resolveImageUrl(url)
  }))
}

function batchTranslateInputItems(node) {
  const linked = linkedInputGenerateNodes(node)
  return linked.flatMap(batchResultItemsFromGenerateNode)
}

function buildBatchRecolorPrompt(target, material) {
  return [
    `只修改图片中的【${target}】。`,
    `将【${target}】改成“${material.label}”材质/颜色。`,
    material.prompt || '',
    '必须保持其他所有内容完全不变，包括产品结构、角度、比例、背景、文字、排版、光影、构图和图片尺寸。',
    '不要新增文字，不要删除文字，不要改变非换色对象，不要重画产品其他部件。'
  ].filter(Boolean).join('\n')
}

async function createImageGenerationJob(payload, runId, index, progressUpdater) {
  const res = await request.post('/image/generate-async', {
    ...payload,
    request_nonce: `${runId}-${index}`
  })
  const taskId = res.data?.task_id
  if (taskId) {
    for (let i = 0; i < 600; i += 1) {
      if (progressUpdater?.(Math.min(92, i * 3), taskId) === false) throw new Error('任务已取消')
      const statusRes = await request.get(`/image/generate-async/${taskId}`)
      const data = statusRes.data || {}
      const imageUrl = extractTaskImage(data)
      if (data.status === 'completed' || data.status === 'done' || imageUrl) {
        if (!imageUrl) throw new Error('未获取到生成图片')
        if (progressUpdater?.(100, taskId) === false) throw new Error('任务已取消')
        return { imageUrl, taskId }
      }
      if (data.status === 'failed' || data.status === 'error') {
        throw new Error(data.error || '生图失败')
      }
      await new Promise(resolve => window.setTimeout(resolve, 1500))
    }
    throw new Error('生图超时，请稍后查看任务结果')
  }
  const imageUrl = extractTaskImage(res.data || {})
  if (!imageUrl) throw new Error('未获取到生成图片')
  if (progressUpdater?.(100, '') === false) throw new Error('任务已取消')
  return { imageUrl, taskId: '' }
}

function createBatchResultImageNode(parentNode, images, { type, title, batchItems = [], taskIds = [] } = {}) {
  const finalImages = images.filter(Boolean).map(resolveImageUrl)
  if (!finalImages.length) return ''
  const parentPos = nodePositions.value[parentNode] || defaultNewNodePosition('generate')
  const resultNode = addWorkflowNode('generate', {
    groupId: workflowGroupIdForNode(parentNode),
    position: {
      x: parentPos.x + (parentPos.width || DEFAULT_NODE_POSITIONS.generate.width) + 100,
      y: parentPos.y
    },
    state: {
      result: {
        url: finalImages[0],
        images: finalImages,
        imageCreatedAt: finalImages.map(() => new Date().toISOString()),
        imageSources: finalImages.map(() => type),
        taskIds,
        taskId: taskIds[0] || '',
        createdAt: new Date().toISOString(),
        status: 'completed',
        batchItems,
        batchTitle: title
      },
      uploads: [],
      showResults: false
    },
    persist: false,
    undo: false
  })
  return resultNode
}

function createBatchResultImageNodes(parentNode, items, { type, title, taskIds = [], titleForItem = null } = {}) {
  const finalItems = (items || []).filter(item => item?.imageUrl)
  if (!finalItems.length) return []
  const now = new Date().toISOString()
  const parentRect = computedNodeRect(parentNode) || nodePositions.value[parentNode] || DEFAULT_NODE_POSITIONS.generate
  const nodeSizes = finalItems.map(item => generateResultNodeSizeForImage(item.imageUrl, DEFAULT_NODE_POSITIONS.generate))
  const maxWidth = Math.max(...nodeSizes.map(size => size.width), DEFAULT_NODE_POSITIONS.generate.width)
  const totalHeight = nodeSizes.reduce((sum, size) => sum + size.height, 0) + Math.max(0, nodeSizes.length - 1) * GENERATED_RESULT_NODE_GAP_Y
  const resultX = Math.round(parentRect.x + parentRect.width + GENERATED_RESULT_NODE_GAP_X)
  let currentY = parentRect.y + ((parentRect.height || DEFAULT_NODE_POSITIONS.generate.height) - totalHeight) / 2
  return finalItems.map((item, index) => {
    const size = nodeSizes[index]
    const taskId = taskIds[index] || taskIds.slice(-1)[0] || ''
    const itemTitle = typeof titleForItem === 'function' ? titleForItem(item, index) : title
    const nodeId = addWorkflowNode('generate', {
      undo: false,
      persist: false,
      groupId: workflowGroupIdForNode(parentNode),
      position: {
        x: Math.round(resultX + (maxWidth - size.width) / 2),
        y: Math.round(currentY)
      },
      state: {
        result: {
          url: item.imageUrl,
          images: [item.imageUrl],
          imageCreatedAt: [now],
          imageSources: [type],
          taskIds: taskId ? [taskId] : [],
          taskId,
          createdAt: now,
          status: 'completed',
          batchItems: [item],
          batchTitle: itemTitle || title
        },
        uploads: [],
        showResults: false
      }
    })
    currentY += size.height + GENERATED_RESULT_NODE_GAP_Y
    return nodeId
  }).filter(Boolean)
}

function createBatchPlaceholderImageNodes(parentNode, items, { type, titleForItem = null } = {}) {
  const finalItems = (items || []).filter(Boolean)
  if (!finalItems.length) return []
  const parentRect = computedNodeRect(parentNode) || nodePositions.value[parentNode] || DEFAULT_NODE_POSITIONS.generate
  const base = DEFAULT_NODE_POSITIONS.generate
  const placeholderHeight = base.height + 28
  const totalHeight = finalItems.length * placeholderHeight + Math.max(0, finalItems.length - 1) * GENERATED_RESULT_NODE_GAP_Y
  const resultX = Math.round(parentRect.x + parentRect.width + GENERATED_RESULT_NODE_GAP_X)
  let currentY = parentRect.y + ((parentRect.height || placeholderHeight) - totalHeight) / 2
  return finalItems.map((item, index) => {
    const itemTitle = typeof titleForItem === 'function' ? titleForItem(item, index) : ''
    const nodeId = addWorkflowNode('generate', {
      undo: false,
      persist: false,
      groupId: workflowGroupIdForNode(parentNode),
      position: {
        x: resultX,
        y: Math.round(currentY)
      },
      state: {
        generatedFromNode: parentNode,
        generating: true,
        progress: 0,
        result: {
          url: '',
          images: [],
          imageCreatedAt: [],
          imageSources: [type],
          taskIds: [],
          taskId: '',
          createdAt: new Date().toISOString(),
          status: 'running',
          batchItems: [item],
          batchTitle: itemTitle
        },
        uploads: [],
        showResults: false
      }
    })
    currentY += placeholderHeight + GENERATED_RESULT_NODE_GAP_Y
    return nodeId
  }).filter(Boolean)
}

function isBatchResultNode(node, type) {
  if (nodeType(node) !== 'generate') return false
  return normalizeGenerationResult(ensureGenerateNodeState(node).result).imageSources?.some(source => source === type)
}

function batchDownstreamTargets(parentNode, type) {
  const targets = []
  workflowLinks.value
    .filter(link => link.from === parentNode)
    .forEach((link) => {
      if (isBatchResultNode(link.to, type)) {
        workflowLinks.value
          .filter(childLink => childLink.from === link.to)
          .forEach(childLink => targets.push(childLink.to))
        return
      }
      targets.push(link.to)
    })
  return uniqueValues(targets)
}

function removeBatchResultChildren(parentNode, type) {
  const children = workflowLinks.value
    .filter(link => link.from === parentNode && isBatchResultNode(link.to, type))
    .map(link => link.to)
  children.forEach(child => removeWorkflowNode(child, { undo: false, persist: false }))
}

function insertBatchResultBetween(parentNode, resultNode) {
  if (!resultNode) return
  const outgoing = workflowLinks.value.filter(link => link.from === parentNode)
  workflowLinks.value = workflowLinks.value.filter(link => link.from !== parentNode)
  connectWorkflowNodes(parentNode, resultNode, { undo: false, persist: false })
  outgoing.forEach(link => connectWorkflowNodes(resultNode, link.to, { undo: false, persist: false }))
}

async function runBatchRecolorNode(node) {
  const state = ensureBatchRecolorNodeState(node)
  const sourceImages = batchSourceImages(node)
  const target = state.target.trim()
  if (!sourceImages.length || !target || !state.materials.length || state.loading) return
  const materials = [...state.materials]
  const recolorItems = sourceImages.flatMap(source => materials.map(material => ({ source, material })))
  const modelName = defaultDrawingModelName()
  if (!modelName) {
    updateBatchRecolorNode(node, { error: '暂无可用生图模型' })
    return
  }
  const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`
  const initialTasks = recolorItems.map(({ source, material }, index) => normalizeBatchRecolorTask({
    index,
    materialId: material.value,
    materialName: material.label,
    materialImageUrl: material.imageUrl,
    sourceImageUrl: source.imageUrl,
    sourceNode: source.sourceNode,
    sourceName: source.sourceName,
    sourceIndex: source.sourceIndex,
    status: 'pending'
  }, index))
  updateBatchRecolorNode(node, {
    loading: true,
    progress: 0,
    tasks: initialTasks,
    resultNode: '',
    error: '',
    runId
  })
  const downstreamTargets = batchDownstreamTargets(node, 'batch-recolor')
  removeBatchResultChildren(node, 'batch-recolor')
  workflowLinks.value = workflowLinks.value.filter(link => link.from !== node)
  const placeholderNodes = createBatchPlaceholderImageNodes(node, initialTasks, {
    type: 'batch-recolor',
    titleForItem: item => sourceImages.length > 1
      ? `${item.materialName || '换色图'} ${Number(item.sourceIndex || 0) + 1}`
      : (item.materialName || '换色图')
  })
  placeholderNodes.forEach((resultNode) => {
    connectWorkflowNodes(node, resultNode, { undo: false, persist: false })
    downstreamTargets.forEach(target => connectWorkflowNodes(resultNode, target, { undo: false, persist: false }))
  })
  saveWorkflowStructureState()
  try {
    const completed = []
    const taskIds = []
    const sourceRefCache = new Map()
    await runLimitedBatch(recolorItems, BATCH_NODE_CONCURRENCY, async ({ source, material }, index) => {
      if (!activeBatchRecolorRunState(node, runId)) return
      const refImages = []
      const markTask = (patch) => {
        const currentState = activeBatchRecolorRunState(node, runId)
        if (!currentState) return false
        const safePatch = Object.fromEntries(Object.entries(patch).filter(([, value]) => value !== undefined))
        const nextTasks = [...currentState.tasks]
        nextTasks[index] = normalizeBatchRecolorTask({
          ...nextTasks[index],
          ...safePatch
        }, index)
        updateBatchRecolorNode(node, {
          tasks: nextTasks,
          progress: batchOverallProgress(nextTasks)
        }, { persist: false, transient: true })
        return true
      }
      try {
        if (!markTask({ status: 'running', progress: 5, error: '' })) return
        const placeholderNode = placeholderNodes[index]
        const placeholderState = generateNodeStateIfExists(placeholderNode)
        if (placeholderState) {
          updateGenerateNode(placeholderNode, {
            generating: true,
            progress: 5,
            result: {
              ...placeholderState.result,
              status: 'running'
            }
          }, { persist: false })
          relayoutGeneratedResultNodes(node, { persist: false })
        }
        let sourceRef = sourceRefCache.get(source.imageUrl)
        if (!sourceRefCache.has(source.imageUrl)) {
          sourceRef = await imageUrlToGenerationRef(source.imageUrl)
          if (!activeBatchRecolorRunState(node, runId)) return
          sourceRefCache.set(source.imageUrl, sourceRef || null)
        }
        if (!sourceRef) throw new Error('原图参考加载失败')
        refImages.push(sourceRef)
        if (shouldUseBatchMaterialReference(material)) {
          if (!material.imageUrl) throw new Error('缺少材质参考图')
          const materialRef = await imageUrlToGenerationRef(material.imageUrl)
          if (!activeBatchRecolorRunState(node, runId)) return
          if (!materialRef) throw new Error('材质参考图加载失败')
          refImages.push(materialRef)
        }
        const result = await createImageGenerationJob({
          prompt: buildBatchRecolorPrompt(target, material),
          model: modelName,
          aspect_ratio: state.aspectRatio,
          resolution: state.resolution,
          count: 1,
          num_images: 1,
          ref_images: refImages
        }, runId, index, (progress, taskId) => {
          if (!markTask({ taskId: taskId || undefined, status: 'running', progress: Math.min(98, progress) })) return false
          const current = generateNodeStateIfExists(placeholderNode)
          if (current) {
            updateGenerateNode(placeholderNode, {
              generating: true,
              progress: Math.min(98, progress),
              result: {
                ...current.result,
                taskId: taskId || current.result.taskId,
                taskIds: taskId ? [taskId] : current.result.taskIds,
                status: 'running'
              }
            }, { persist: false })
          }
          return true
        })
        if (!activeBatchRecolorRunState(node, runId)) return
        const imageUrl = resolveImageUrl(result.imageUrl)
        taskIds[index] = result.taskId || ''
        completed[index] = {
          index,
          materialId: material.value,
          materialName: material.label,
          materialImageUrl: material.imageUrl,
          sourceImageUrl: source.imageUrl,
          sourceNode: source.sourceNode,
          sourceName: source.sourceName,
          sourceIndex: source.sourceIndex,
          imageUrl
        }
        if (!markTask({ imageUrl, taskId: result.taskId || '', status: 'completed', progress: 100 })) return
        const completedPlaceholderState = generateNodeStateIfExists(placeholderNode)
        if (completedPlaceholderState) {
          updateGenerateNode(placeholderNode, {
            generating: false,
            progress: 100,
            error: '',
            result: {
              ...completedPlaceholderState.result,
              url: imageUrl,
              images: [imageUrl],
              imageCreatedAt: [new Date().toISOString()],
              imageSources: ['batch-recolor'],
              taskIds: result.taskId ? [result.taskId] : [],
              taskId: result.taskId || '',
              status: 'completed',
              batchItems: [completed[index]],
              batchTitle: sourceImages.length > 1 ? `${material.label || '换色图'} ${Number(source.sourceIndex || 0) + 1}` : (material.label || '换色图')
            }
          }, { persist: false })
          relayoutGeneratedResultNodes(node, { persist: false })
        }
      } catch (e) {
        const message = e?.response?.data?.error || e?.message || '换色失败'
        if (!markTask({ status: 'failed', progress: 100, error: message })) return
        const current = generateNodeStateIfExists(placeholderNodes[index])
        if (current) {
          updateGenerateNode(placeholderNodes[index], {
            generating: false,
            progress: 100,
            error: message,
            result: {
              ...current.result,
              status: current.result.images?.length ? 'partial' : 'failed'
            }
          }, { persist: false })
        }
      }
    })
    const finalState = activeBatchRecolorRunState(node, runId)
    if (!finalState) return
    const finalTasks = finalState.tasks
    const failed = batchFailedCount(finalTasks)
    updateBatchRecolorNode(node, {
      loading: false,
      progress: 100,
      resultNode: placeholderNodes[0] || '',
      tasks: finalTasks,
      error: failed ? `有 ${failed} 个换色任务失败` : ''
    })
    appendWorkflowHistoryAssets(completed.filter(Boolean).map((item, index) => ({ url: item.imageUrl, createdAt: new Date().toISOString(), node, taskId: taskIds[item.index ?? index] || '' })))
    saveWorkflowStructureState()
  } catch (e) {
    const currentBatchState = activeBatchRecolorRunState(node, runId)
    if (!currentBatchState) return
    placeholderNodes.forEach((resultNode, index) => {
      const current = generateNodeStateIfExists(resultNode)
      if (!current) return
      if (!current.generating) return
      const message = e?.response?.data?.error || e?.message || '批量换色失败'
      updateGenerateNode(resultNode, {
        generating: false,
        progress: 100,
        error: index === 0 ? message : current.error,
        result: {
          ...current.result,
          status: current.result.images?.length ? 'partial' : 'failed'
        }
      }, { persist: false })
    })
    const failedTasks = currentBatchState.tasks.map((task, index) => normalizeBatchRecolorTask({
      ...task,
      status: task.status === 'completed' ? task.status : 'failed',
      progress: 100,
      error: task.status === 'completed' ? task.error : (e?.response?.data?.error || e?.message || '批量换色失败')
    }, index))
    updateBatchRecolorNode(node, {
      loading: false,
      progress: 100,
      tasks: failedTasks,
      error: e?.response?.data?.error || e?.message || '批量换色失败'
    })
  }
}

async function runBatchTranslateNode(node) {
  const state = ensureBatchTranslateNodeState(node)
  const inputItems = batchTranslateInputItems(node)
  const targetLanguages = state.targetLanguages.length ? [...state.targetLanguages] : [state.targetLanguage || 'en']
  const translateItems = inputItems.flatMap((item, inputIndex) => targetLanguages.map(language => ({
    ...item,
    inputIndex,
    targetLanguage: language,
    targetLanguageName: batchLanguageLabel(language)
  })))
  if (!translateItems.length || state.loading) return
  const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`
  const initialTasks = translateItems.map((item, index) => normalizeBatchTranslateTask({
    index,
    materialId: item.materialId,
    materialName: item.materialName,
    sourceImageUrl: item.imageUrl,
    targetLanguage: item.targetLanguage,
    targetLanguageName: item.targetLanguageName,
    status: 'pending'
  }, index))
  updateBatchTranslateNode(node, {
    loading: true,
    progress: 0,
    tasks: initialTasks,
    resultNode: '',
    error: '',
    runId
  })
  removeBatchResultChildren(node, 'batch-translate')
  workflowLinks.value = workflowLinks.value.filter(link => link.from !== node || !isBatchResultNode(link.to, 'batch-translate'))
  const placeholderNodes = createBatchPlaceholderImageNodes(node, initialTasks, {
    type: 'batch-translate',
    titleForItem: item => inputItems.length > 1
      ? `${item.targetLanguageName || batchLanguageLabel(item.targetLanguage)} ${Number(item.index || 0) + 1}`
      : (item.targetLanguageName || batchLanguageLabel(item.targetLanguage))
  })
  placeholderNodes.forEach(resultNode => connectWorkflowNodes(node, resultNode, { undo: false, persist: false }))
  saveWorkflowStructureState()
  try {
    const completed = []
    await runLimitedBatch(translateItems, BATCH_NODE_CONCURRENCY, async (item, index) => {
      if (!activeBatchTranslateRunState(node, runId)) return
      const placeholderNode = placeholderNodes[index]
      const markTask = (patch) => {
        const currentState = activeBatchTranslateRunState(node, runId)
        if (!currentState) return false
        const safePatch = Object.fromEntries(Object.entries(patch).filter(([, value]) => value !== undefined))
        const nextTasks = [...currentState.tasks]
        nextTasks[index] = normalizeBatchTranslateTask({
          ...nextTasks[index],
          ...safePatch
        }, index)
        updateBatchTranslateNode(node, {
          tasks: nextTasks,
          progress: batchOverallProgress(nextTasks)
        }, { persist: false, transient: true })
        return true
      }
      const updatePlaceholderProgress = (progress) => {
        const current = generateNodeStateIfExists(placeholderNode)
        if (!current) return false
        updateGenerateNode(placeholderNode, {
          generating: true,
          progress,
          result: {
            ...current.result,
            status: 'running'
          }
        }, { persist: false, transient: true })
        return true
      }
      let progressTimer = null
      const stopSmoothProgress = () => {
        if (!progressTimer) return
        window.clearInterval(progressTimer)
        progressTimer = null
      }
      const startSmoothProgress = () => {
        stopSmoothProgress()
        progressTimer = window.setInterval(() => {
          const currentState = activeBatchTranslateRunState(node, runId)
          if (!currentState) {
            stopSmoothProgress()
            return
          }
          const task = currentState.tasks[index]
          if (!currentState.loading || task?.status !== 'running') {
            stopSmoothProgress()
            return
          }
          const placeholderState = generateNodeStateIfExists(placeholderNode)
          const currentProgress = Math.max(Number(task.progress) || 0, Number(placeholderState?.progress) || 0)
          const step = currentProgress < 36 ? 3 : (currentProgress < 72 ? 2 : 1)
          const nextProgress = Math.min(92, currentProgress + step)
          if (!markTask({ progress: nextProgress })) {
            stopSmoothProgress()
            return
          }
          updatePlaceholderProgress(nextProgress)
        }, 600)
      }
      try {
        if (!markTask({ status: 'running', progress: 5, error: '' })) return
        updatePlaceholderProgress(5)
        startSmoothProgress()
        const imagePayload = await imageUrlToRef(item.imageUrl)
        if (!activeBatchTranslateRunState(node, runId)) {
          stopSmoothProgress()
          return
        }
        if (!imagePayload?.data) throw new Error('未获取到翻译图片')
        const res = await request.post('/image-translate', {
          image_data: imagePayload.data,
          mime_type: imagePayload.mime_type || 'image/png',
          target_language: item.targetLanguage,
          aspect_ratio: state.aspectRatio,
          resolution: state.resolution
        })
        if (!activeBatchTranslateRunState(node, runId)) {
          stopSmoothProgress()
          return
        }
        const translatedUrl = resolveImageUrl(res.data?.image_url || res.data?.url || res.data?.resultImageUrl || '')
        if (!translatedUrl) throw new Error('图片翻译未返回结果')
        stopSmoothProgress()
        completed[index] = {
          index,
          materialId: item.materialId,
          materialName: item.materialName,
          sourceImageUrl: item.imageUrl,
          targetLanguage: item.targetLanguage,
          targetLanguageName: item.targetLanguageName,
          imageUrl: translatedUrl
        }
        if (!markTask({ imageUrl: translatedUrl, status: 'completed', progress: 100 })) return
        const completedPlaceholderState = generateNodeStateIfExists(placeholderNode)
        if (completedPlaceholderState) {
          updateGenerateNode(placeholderNode, {
            generating: false,
            progress: 100,
            error: '',
            result: {
              ...completedPlaceholderState.result,
              url: translatedUrl,
              images: [translatedUrl],
              imageCreatedAt: [new Date().toISOString()],
              imageSources: ['batch-translate'],
              taskIds: [],
              taskId: '',
              status: 'completed',
              batchItems: [completed[index]],
              batchTitle: inputItems.length > 1 ? `${item.targetLanguageName || batchLanguageLabel(item.targetLanguage)} ${Number(item.inputIndex || 0) + 1}` : (item.targetLanguageName || batchLanguageLabel(item.targetLanguage))
            }
          }, { persist: false })
          relayoutGeneratedResultNodes(node, { persist: false })
        }
      } catch (e) {
        stopSmoothProgress()
        const message = e?.response?.data?.error || e?.message || '翻译失败'
        if (!markTask({ status: 'failed', progress: 100, error: message })) return
        const current = generateNodeStateIfExists(placeholderNode)
        if (current) {
          updateGenerateNode(placeholderNode, {
            generating: false,
            progress: 100,
            error: message,
            result: {
              ...current.result,
              status: current.result.images?.length ? 'partial' : 'failed'
            }
          }, { persist: false })
          relayoutGeneratedResultNodes(node, { persist: false })
        }
      }
    })
    const finalState = activeBatchTranslateRunState(node, runId)
    if (!finalState) return
    const finalTasks = finalState.tasks
    const failed = batchFailedCount(finalTasks)
    updateBatchTranslateNode(node, {
      loading: false,
      progress: 100,
      resultNode: placeholderNodes[0] || '',
      tasks: finalTasks,
      error: failed ? `有 ${failed} 个翻译任务失败` : ''
    })
    appendWorkflowHistoryAssets(completed.filter(Boolean).map(item => ({ url: item.imageUrl, createdAt: new Date().toISOString(), node, taskId: '' })))
    saveWorkflowStructureState()
  } catch (e) {
    const currentBatchState = activeBatchTranslateRunState(node, runId)
    if (!currentBatchState) return
    placeholderNodes.forEach((resultNode, index) => {
      const current = generateNodeStateIfExists(resultNode)
      if (!current) return
      if (!current.generating) return
      const message = e?.response?.data?.error || e?.message || '批量翻译失败'
      updateGenerateNode(resultNode, {
        generating: false,
        progress: 100,
        error: index === 0 ? message : current.error,
        result: {
          ...current.result,
          status: current.result.images?.length ? 'partial' : 'failed'
        }
      }, { persist: false })
    })
    const failedTasks = currentBatchState.tasks.map((task, index) => normalizeBatchTranslateTask({
      ...task,
      status: task.status === 'completed' ? task.status : 'failed',
      progress: 100,
      error: task.status === 'completed' ? task.error : (e?.response?.data?.error || e?.message || '批量翻译失败')
    }, index))
    updateBatchTranslateNode(node, {
      loading: false,
      progress: 100,
      tasks: failedTasks,
      error: e?.response?.data?.error || e?.message || '批量翻译失败'
    })
  }
}

function workflowSnapshot() {
  return clonePlain({
    workflowNodes: normalizeWorkflowNodes(workflowNodes.value),
    workflowDynamicNodes: workflowDynamicNodes.value,
    workflowLinks: normalizeWorkflowLinks(workflowLinks.value),
    workflowHistoryAssets: workflowHistoryAssets.value,
    workflowGroups: groupBoxes.value,
    nodePositions: nodePositions.value,
    canvasView: liveCanvasView,
    activeWorkflowNode: activeWorkflowNode.value,
    activeWorkflowLink: activeWorkflowLink.value,
    resultImageUrl: resultImageUrl.value,
    resultUrls: resultUrls.value,
    sceneImage: sceneImage.value,
    placement: placement.value,
    workflowGenerateNodeStates: workflowGenerateNodeStates.value,
    workflowTextNodeStates: workflowTextNodeStates.value,
    workflowLlmNodeStates: workflowLlmNodeStates.value,
    workflowModelNodeStates: cloneModelNodeStates(workflowModelNodeStates.value, { includeMeshData: false }),
    workflowOutputNodeStates: workflowOutputNodeStates.value,
    workflowBatchRecolorNodeStates: workflowBatchRecolorNodeStates.value,
    workflowBatchTranslateNodeStates: workflowBatchTranslateNodeStates.value,
    workflowVideoNodeStates: workflowVideoNodeStates.value
  })
}

function nodePositionSnapshot() {
  return {
    snapshotType: 'node-positions',
    nodePositions: clonePlain(nodePositions.value),
    activeWorkflowNode: activeWorkflowNode.value,
    activeWorkflowLink: activeWorkflowLink.value
  }
}

function workflowSnapshotSignature(snapshot = {}) {
  const nodes = [
    ...Object.keys(snapshot.workflowNodes || {}).filter(key => snapshot.workflowNodes[key]),
    ...(snapshot.workflowDynamicNodes || []).map(item => item.id)
  ].sort()
  const links = normalizeWorkflowLinks(snapshot.workflowLinks || [])
    .map(link => `${link.from}->${link.to}`)
    .sort()
  const positions = nodes.map((node) => {
    const pos = snapshot.nodePositions?.[node] || {}
    return `${node}:${Math.round(pos.x || 0)},${Math.round(pos.y || 0)},${Math.round(pos.width || 0)},${Math.round(pos.height || 0)}`
  })
  return [
    nodes.join('|'),
    links.join('|'),
    positions.join('|'),
    snapshot.activeWorkflowNode || '',
    snapshot.activeWorkflowLink || ''
  ].join('::')
}

function pushWorkflowUndoSnapshot() {
  if (isRestoringProject || !activeProjectId.value) return
  const snapshot = workflowSnapshot()
  const previous = workflowUndoStack.value[workflowUndoStack.value.length - 1]
  if (previous && workflowSnapshotSignature(previous) === workflowSnapshotSignature(snapshot)) return
  workflowUndoStack.value = [...workflowUndoStack.value, snapshot].slice(-WORKFLOW_UNDO_LIMIT)
}

function restoreWorkflowSnapshot(snapshot) {
  if (!snapshot) return
  if (snapshot.snapshotType === 'node-positions') {
    isRestoringProject = true
    nodePositions.value = clonePlain(snapshot.nodePositions || nodePositions.value)
    activeWorkflowNode.value = snapshot.activeWorkflowNode || ''
    activeWorkflowLink.value = snapshot.activeWorkflowLink || ''
    isRestoringProject = false
    bumpViewportCheck()
    saveActiveProject()
    return
  }
  isRestoringProject = true
  workflowNodes.value = normalizeWorkflowNodes(snapshot.workflowNodes || DEFAULT_WORKFLOW_NODES)
  workflowDynamicNodes.value = Array.isArray(snapshot.workflowDynamicNodes) ? clonePlain(snapshot.workflowDynamicNodes) : []
  workflowLinks.value = normalizeWorkflowLinks(snapshot.workflowLinks || [])
  workflowHistoryAssets.value = normalizeHistoryAssets(snapshot.workflowHistoryAssets || [])
  groupBoxes.value = Array.isArray(snapshot.workflowGroups) ? clonePlain(snapshot.workflowGroups) : []
  nodePositions.value = clonePlain(snapshot.nodePositions || DEFAULT_NODE_POSITIONS)
  canvasView.value = { ...(snapshot.canvasView || DEFAULT_CANVAS_VIEW) }
  applyCanvasTransform(canvasView.value)
  activeWorkflowNode.value = snapshot.activeWorkflowNode || ''
  activeWorkflowLink.value = snapshot.activeWorkflowLink || ''
  pendingLinkFrom.value = ''
  draggingLink.value = null
  resultImageUrl.value = snapshot.resultImageUrl || ''
  resultUrls.value = { ...(snapshot.resultUrls || {}) }
  sceneImage.value = { ...(snapshot.sceneImage || { url: '', name: '', width: 0, height: 0 }) }
  placement.value = { ...(snapshot.placement || placement.value) }
  workflowGenerateNodeStates.value = clonePlain(snapshot.workflowGenerateNodeStates || {})
  workflowTextNodeStates.value = normalizeTextNodeStates(snapshot.workflowTextNodeStates || {})
  workflowLlmNodeStates.value = clonePlain(snapshot.workflowLlmNodeStates || {})
  workflowModelNodeStates.value = normalizeModelNodeStates({
    ...workflowModelNodeStates.value,
    ...(snapshot.workflowModelNodeStates || {})
  })
  workflowOutputNodeStates.value = normalizeOutputNodeStates(snapshot.workflowOutputNodeStates || {})
  workflowBatchRecolorNodeStates.value = normalizeBatchRecolorNodeStates(snapshot.workflowBatchRecolorNodeStates || {}, { resetRuntime: true })
  workflowBatchTranslateNodeStates.value = normalizeBatchTranslateNodeStates(snapshot.workflowBatchTranslateNodeStates || {}, { resetRuntime: true })
  workflowVideoNodeStates.value = normalizeVideoNodeStates(snapshot.workflowVideoNodeStates || {}, { resetRuntime: true })
  closeAllGenerateMenus()
  isRestoringProject = false
  bumpViewportCheck()
  saveActiveProject()
}

function undoWorkflowChange() {
  if (!workflowUndoStack.value.length) return
  const snapshot = workflowUndoStack.value[workflowUndoStack.value.length - 1]
  workflowUndoStack.value = workflowUndoStack.value.slice(0, -1)
  restoreWorkflowSnapshot(snapshot)
}

function setGenerateNodeMenu(node, menu) {
  updateGenerateNode(node, { menu }, { persist: false })
}

function saveActiveProject({ persist = true, draft = persist } = {}) {
  if (!activeProjectId.value) return
  const project = projects.value.find(item => item.id === activeProjectId.value)
  if (!project) return
  projectStateVersion += 1
  const workflowStateRevision = nextWorkflowStateRevision(project)

  Object.assign(project, {
    workflowStateRevision,
    fileName: fileName.value,
    stpFileUrl: stpFileUrl.value,
    meshData: meshData.value,
    modelLoaded: modelLoaded.value,
    resultImageUrl: resultImageUrl.value,
    resultUrls: { ...resultUrls.value },
    workflowNodes: normalizeWorkflowNodes(workflowNodes.value),
    workflowDynamicNodes: workflowDynamicNodes.value.map(item => ({ ...item })),
    workflowLinks: normalizeWorkflowLinks(workflowLinks.value),
    workflowHistoryAssets: normalizeHistoryAssets(workflowHistoryAssets.value),
    workflowGroups: clonePlain(groupBoxes.value),
    nodePositions: clonePlain(nodePositions.value),
    canvasView: { ...liveCanvasView },
    activeWorkflowNode: activeWorkflowNode.value,
    sceneImage: { ...sceneImage.value },
    placement: { ...placement.value },
    workflowGeneratePrompt: workflowGeneratePrompt.value,
    workflowGenerateModel: workflowGenerateModel.value,
    workflowGenerateCount: workflowGenerateCount.value,
    workflowGenerateAspectRatio: workflowGenerateAspectRatio.value,
    workflowGenerateResolution: workflowGenerateResolution.value,
    workflowGenerateNodeStates: clonePlain(workflowGenerateNodeStates.value),
    workflowTextNodeStates: clonePlain(workflowTextNodeStates.value),
    workflowLlmNodeStates: clonePlain(workflowLlmNodeStates.value),
    workflowModelNodeStates: cloneModelNodeStates(workflowModelNodeStates.value),
    workflowOutputNodeStates: clonePlain(workflowOutputNodeStates.value),
    workflowBatchRecolorNodeStates: clonePlain(workflowBatchRecolorNodeStates.value),
    workflowBatchTranslateNodeStates: clonePlain(workflowBatchTranslateNodeStates.value),
    workflowVideoNodeStates: clonePlain(workflowVideoNodeStates.value),
    workflowGenerationResult: { ...workflowGenerationResult.value }
  })

  if (persist && !isRestoringProject) {
    scheduleProjectDraft(project)
    scheduleProjectSave()
  } else if (draft && !isRestoringProject) {
    scheduleProjectDraft(project)
  }
}

function saveProjectPatch(patch, { persist = true, draft = persist } = {}) {
  if (!activeProjectId.value) return
  const project = projects.value.find(item => item.id === activeProjectId.value)
  if (!project) return
  projectStateVersion += 1
  Object.assign(project, {
    ...patch,
    workflowStateRevision: nextWorkflowStateRevision(project)
  })
  if (draft && !isRestoringProject) scheduleProjectDraft(project)
  if (persist && !isRestoringProject) scheduleProjectSave()
}

function saveCanvasViewState({ persist = true } = {}) {
  saveProjectPatch({ canvasView: { ...liveCanvasView } }, { persist })
}

function saveNodeLayoutState({ includeGroups = false, persist = true } = {}) {
  const patch = {
    nodePositions: clonePlain(nodePositions.value),
    activeWorkflowNode: activeWorkflowNode.value,
    activeWorkflowLink: activeWorkflowLink.value
  }
  if (includeGroups) patch.workflowGroups = clonePlain(groupBoxes.value)
  saveProjectPatch(patch, { persist })
}

function saveWorkflowStructureState({ persist = true } = {}) {
  if (!isRestoringProject) markTemplateWorkflowGroupsChanged()
  saveProjectPatch({
    workflowNodes: normalizeWorkflowNodes(workflowNodes.value),
    workflowDynamicNodes: workflowDynamicNodes.value.map(item => ({ ...item })),
    workflowLinks: normalizeWorkflowLinks(workflowLinks.value),
    workflowGroups: clonePlain(groupBoxes.value),
    nodePositions: clonePlain(nodePositions.value),
    activeWorkflowNode: activeWorkflowNode.value,
    activeWorkflowLink: activeWorkflowLink.value,
    workflowGenerateNodeStates: clonePlain(workflowGenerateNodeStates.value),
    workflowTextNodeStates: clonePlain(workflowTextNodeStates.value),
    workflowLlmNodeStates: clonePlain(workflowLlmNodeStates.value),
    workflowModelNodeStates: cloneModelNodeStates(workflowModelNodeStates.value),
    workflowOutputNodeStates: clonePlain(workflowOutputNodeStates.value),
    workflowBatchRecolorNodeStates: clonePlain(workflowBatchRecolorNodeStates.value),
    workflowBatchTranslateNodeStates: clonePlain(workflowBatchTranslateNodeStates.value),
    workflowVideoNodeStates: clonePlain(workflowVideoNodeStates.value)
  }, { persist })
}

function markCanvasInteraction() {
  lastCanvasInteractionAt = Date.now()
}

function scheduleNodeLayoutStateSave(options = {}) {
  if (layoutSaveTimer) window.clearTimeout(layoutSaveTimer)
  layoutSaveTimer = window.setTimeout(() => {
    layoutSaveTimer = null
    saveNodeLayoutState({ ...options, persist: false })
    scheduleProjectSave(2400)
  }, 900)
}

function scheduleProjectSave(delay = 1600) {
  if (!activeProjectId.value) return
  if (projectSaveTimer) window.clearTimeout(projectSaveTimer)
  projectSaveTimer = window.setTimeout(() => {
    persistActiveProject()
  }, delay)
}

async function persistActiveProject({ force = false } = {}) {
  if (!activeProjectId.value) return
  const quietFor = Date.now() - lastCanvasInteractionAt
  if (!force && quietFor < 1200) {
    scheduleProjectSave(1200 - quietFor + 800)
    return
  }
  if (projectSaveInFlight) {
    projectSaveQueued = true
    return
  }

  if (projectSaveTimer) {
    window.clearTimeout(projectSaveTimer)
    projectSaveTimer = null
  }

  const project = projects.value.find(item => item.id === activeProjectId.value)
  if (!project) return
  const saveVersion = projectStateVersion
  const projectSnapshot = clonePlain(projectStateForDatabase(project))

  projectSaveInFlight = true
  try {
    const ossProject = await ensureProjectImagesOnOSS(projectSnapshot)
    if (projectStateVersion !== saveVersion) {
      projectSaveQueued = true
      return
    }
    const dbState = projectStateForDatabase(ossProject)
    await request.put(`/workflow-projects/${encodeURIComponent(project.id)}`, {
      name: project.name,
      state: dbState
    })

    if (saveVersion < projectLastAppliedSaveVersion) return
    projectLastAppliedSaveVersion = saveVersion
    const hasNewerLocalState = projectStateVersion !== saveVersion

    if (!hasNewerLocalState) {
      clearProjectDraft(project.id)
      Object.assign(project, {
        resultImageUrl: ossProject.resultImageUrl,
        resultUrls: ossProject.resultUrls,
        sceneImage: ossProject.sceneImage,
        workflowGenerateNodeStates: mergePersistedGenerateNodeStates(ossProject.workflowGenerateNodeStates, project.workflowGenerateNodeStates),
        workflowHistoryAssets: normalizeHistoryAssets(ossProject.workflowHistoryAssets || project.workflowHistoryAssets || []),
        workflowTextNodeStates: mergePersistedTextNodeStates(ossProject.workflowTextNodeStates || project.workflowTextNodeStates || {}, project.workflowTextNodeStates),
        workflowModelNodeStates: normalizeModelNodeStates(ossProject.workflowModelNodeStates || project.workflowModelNodeStates || {}),
        workflowOutputNodeStates: normalizeOutputNodeStates(ossProject.workflowOutputNodeStates || project.workflowOutputNodeStates || {}),
        workflowBatchRecolorNodeStates: normalizeBatchRecolorNodeStates(ossProject.workflowBatchRecolorNodeStates || project.workflowBatchRecolorNodeStates || {}),
        workflowBatchTranslateNodeStates: normalizeBatchTranslateNodeStates(ossProject.workflowBatchTranslateNodeStates || project.workflowBatchTranslateNodeStates || {}),
        workflowVideoNodeStates: normalizeVideoNodeStates(ossProject.workflowVideoNodeStates || project.workflowVideoNodeStates || {}),
        workflowGenerationResult: ossProject.workflowGenerationResult
      })
    }
    if (!hasNewerLocalState && project.id === activeProjectId.value) {
      resultImageUrl.value = ossProject.resultImageUrl || ''
      resultUrls.value = { ...(ossProject.resultUrls || {}) }
      sceneImage.value = { ...(ossProject.sceneImage || sceneImage.value) }
      workflowGenerateNodeStates.value = mergePersistedGenerateNodeStates(ossProject.workflowGenerateNodeStates, workflowGenerateNodeStates.value)
      workflowHistoryAssets.value = normalizeHistoryAssets(ossProject.workflowHistoryAssets || workflowHistoryAssets.value)
      workflowTextNodeStates.value = mergePersistedTextNodeStates(ossProject.workflowTextNodeStates || workflowTextNodeStates.value, workflowTextNodeStates.value)
      workflowModelNodeStates.value = normalizeModelNodeStates(ossProject.workflowModelNodeStates || workflowModelNodeStates.value)
      workflowOutputNodeStates.value = normalizeOutputNodeStates(ossProject.workflowOutputNodeStates || workflowOutputNodeStates.value)
      workflowBatchRecolorNodeStates.value = normalizeBatchRecolorNodeStates(ossProject.workflowBatchRecolorNodeStates || workflowBatchRecolorNodeStates.value)
      workflowBatchTranslateNodeStates.value = normalizeBatchTranslateNodeStates(ossProject.workflowBatchTranslateNodeStates || workflowBatchTranslateNodeStates.value)
      workflowVideoNodeStates.value = normalizeVideoNodeStates(ossProject.workflowVideoNodeStates || workflowVideoNodeStates.value)
      workflowGenerationResult.value = normalizeGenerationResult(ossProject.workflowGenerationResult)
    }
  } catch (e) {
    console.warn('[Workflow] save project failed:', e)
  } finally {
    projectSaveInFlight = false
    if (projectSaveQueued) {
      projectSaveQueued = false
      scheduleProjectSave()
    }
  }
}

function flushProjectSaveBeforeUnload() {
  const project = projects.value.find(item => item.id === activeProjectId.value)
  flushProjectDraft(project)
}

function clearCurrentModel() {
  clearRenderedModelScene()
}

async function restoreProject(project) {
  project = mergeProjectDraft(project)
  projectStateVersion = Number(project.workflowStateRevision) || 0
  projectLastAppliedSaveVersion = projectStateVersion
  isRestoringProject = true
  canvasReady.value = false
  try {
    clearCurrentModel()
    fileName.value = project.fileName || ''
    stpFileUrl.value = project.stpFileUrl || ''
    meshData.value = project.meshData || null
    resultImageUrl.value = project.resultImageUrl || ''
    resultUrls.value = { ...(project.resultUrls || {}) }
    workflowOutputNodeStates.value = normalizeOutputNodeStates(project.workflowOutputNodeStates || {})
    workflowNodes.value = normalizeWorkflowNodes(project.workflowNodes || DEFAULT_WORKFLOW_NODES)
    const normalizedDynamicNodes = normalizeDynamicNodes(project.workflowDynamicNodes || [])
    if (stpFileUrl.value || meshData.value) {
      workflowNodes.value = {
        ...workflowNodes.value,
        upload: true
      }
    }
    workflowModelNodeStates.value = modelStatesWithDynamicUploadFallback({
      ...(project.workflowModelNodeStates || {}),
      ...((stpFileUrl.value || meshData.value) && !(project.workflowModelNodeStates || {}).upload
        ? {
            upload: {
              fileName: fileName.value,
              stpFileUrl: stpFileUrl.value,
              meshData: meshData.value,
              modelLoaded: !!meshData.value
            }
          }
        : {})
    }, normalizedDynamicNodes)
    if (resultUrls.value.white || resultUrls.value.depth) {
      workflowNodes.value = {
        ...workflowNodes.value,
        output: true
      }
    }
    workflowDynamicNodes.value = normalizedDynamicNodes
    workflowLinks.value = workflowLinksWithValidVideoInputs(project.workflowLinks || [])
    ensureDynamicOutputNodesFromState()
    workflowHistoryAssets.value = normalizeHistoryAssets(project.workflowHistoryAssets || [])
    groupBoxes.value = Array.isArray(project.workflowGroups) ? clonePlain(project.workflowGroups) : []
    activeWorkflowLink.value = ''
    pendingLinkFrom.value = ''
    draggingLink.value = null
    closeLinkCreateMenu()
    nodePositions.value = clonePlain(project.nodePositions || DEFAULT_NODE_POSITIONS)
    canvasView.value = { ...(project.canvasView || DEFAULT_CANVAS_VIEW) }
    applyCanvasTransform(canvasView.value)
    activeWorkflowNode.value = project.activeWorkflowNode || 'upload'
    sceneImage.value = { ...(project.sceneImage || { url: '', name: '', width: 0, height: 0 }) }
    placement.value = { ...(project.placement || { x: 50, y: 58, scale: 0.42, rotation: 0, opacity: 0.78 }) }
    workflowGeneratePrompt.value = project.workflowGeneratePrompt || ''
    workflowGenerateModel.value = project.workflowGenerateModel || workflowGenerateModel.value
    workflowGenerateCount.value = Number(project.workflowGenerateCount) || 1
    workflowGenerateAspectRatio.value = generateAspectRatioOptions.includes(project.workflowGenerateAspectRatio)
      ? project.workflowGenerateAspectRatio
      : '1:1'
    workflowGenerateResolution.value = project.workflowGenerateResolution || '2K'
    workflowGenerationResult.value = normalizeGenerationResult(project.workflowGenerationResult)
    workflowTextNodeStates.value = normalizeTextNodeStates(project.workflowTextNodeStates || {})
    workflowLlmNodeStates.value = normalizeLlmNodeStates(project.workflowLlmNodeStates || {}, { resetRuntime: true })
    migrateCompletedLlmNodesToText()
    workflowBatchRecolorNodeStates.value = normalizeBatchRecolorNodeStates(project.workflowBatchRecolorNodeStates || {}, { resetRuntime: true })
    workflowBatchTranslateNodeStates.value = normalizeBatchTranslateNodeStates(project.workflowBatchTranslateNodeStates || {}, { resetRuntime: true })
    workflowVideoNodeStates.value = normalizeVideoNodeStates(project.workflowVideoNodeStates || {}, { resetRuntime: true })
    const savedGenerateStates = project.workflowGenerateNodeStates || {}
    workflowGenerateNodeStates.value = Object.keys(savedGenerateStates).reduce((map, key) => {
      map[key] = normalizeGenerateNodeState(savedGenerateStates[key])
      return map
    }, {})
    if (workflowNodes.value.generate) {
      workflowGenerateNodeStates.value.generate = normalizeGenerateNodeState({
        prompt: project.workflowGeneratePrompt || workflowGeneratePrompt.value,
        model: project.workflowGenerateModel || workflowGenerateModel.value,
        count: project.workflowGenerateCount || workflowGenerateCount.value,
        aspectRatio: project.workflowGenerateAspectRatio || workflowGenerateAspectRatio.value,
        resolution: project.workflowGenerateResolution || workflowGenerateResolution.value,
        result: project.workflowGenerationResult || workflowGenerationResult.value,
        ...workflowGenerateNodeStates.value.generate
      })
    }
    workflowGenerationProgress.value = 0
    workflowGenerationCancelled.value = false
    workflowGenerationError.value = ''
    showGenerationResults.value = false
    error.value = ''
    showNodeMenu.value = false

    await nextTick()
    applyCanvasTransform(canvasView.value)
    canvasView.value = { ...liveCanvasView }
    canvasReady.value = true
    bumpViewportCheck()
    const preferredModelNode = activeWorkflowNode.value && nodeType(activeWorkflowNode.value) === 'upload'
      ? activeWorkflowNode.value
      : (modelWorkflowNodes().find(item => ensureModelNodeState(item.id).meshData || ensureModelNodeState(item.id).stpFileUrl)?.id || 'upload')
    activateModelNode(preferredModelNode)
    if (modelWorkflowNodes().length && (!renderer || !renderer.domElement?.parentNode)) {
      initThreeJS()
    } else if (modelWorkflowNodes().length) {
      onResize()
    }
    resumeWorkflowGenerationTasks()
  } finally {
    isRestoringProject = false
    canvasReady.value = true
  }
}

async function reloadMeshPreviewFromUrl(url, node = activeModelNodeId.value) {
  const targetNode = node || 'upload'
  activateModelNode(targetNode)
  const loadKey = `${targetNode}::${url}`
  const existingPromise = modelMeshLoadPromises.get(loadKey)
  if (existingPromise) return existingPromise
  const loadPromise = doReloadMeshPreviewFromUrl(url, targetNode, loadKey)
  modelMeshLoadPromises.set(loadKey, loadPromise)
  loadPromise.finally(() => {
    if (modelMeshLoadPromises.get(loadKey) === loadPromise) {
      modelMeshLoadPromises.delete(loadKey)
    }
  })
  return loadPromise
}

async function doReloadMeshPreviewFromUrl(url, targetNode, loadKey) {
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [targetNode]: {
      ...ensureModelNodeState(targetNode),
      stpFileUrl: url,
      loading: true,
      loadingMessage: '加载模型...',
      error: ''
    }
  }
  try {
    const meshPayload = await requestMeshPreviewData(url, targetNode)
    if (modelMeshLoadPromises.get(loadKey) !== undefined && ensureModelNodeState(targetNode).stpFileUrl !== url) {
      throw new Error('模型预览加载已取消')
    }
    meshData.value = meshPayload
    await loadStpModel(meshPayload, targetNode)
    saveActiveProject({ persist: false })
    return meshPayload
  } catch (e) {
    console.warn('[Workflow] reload mesh preview failed:', e)
    const message = modelPreviewFailureMessage(e)
    if (!message || ensureModelNodeState(targetNode).stpFileUrl !== url || ensureModelNodeState(targetNode).modelLoaded) {
      return
    }
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...ensureModelNodeState(targetNode),
        loading: false,
        modelLoaded: false,
        error: message
      }
    }
    throw new Error(message)
  } finally {
    loading.value = false
  }
}

async function requestMeshPreviewData(url, node = activeModelNodeId.value) {
  const submitRes = await request.get('/stp/mesh-preview', {
    params: {
      stp_url: url,
      async: 1
    }
  })
  const submitData = submitRes.data?.data || {}
  const directMesh = normalizeMeshPreviewPayload(submitData)
  if (directMesh) return directMesh
  const taskId = submitData.task_id || submitRes.data?.task_id
  if (!taskId) throw new Error('后端没有返回 3D 预览任务 ID')
  updateModelLoadingMessage(node, renderTaskStatusMessage(submitData, '模型解析排队中...'))

  const maxAttempts = 180
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    await new Promise(resolve => window.setTimeout(resolve, 1000))
    const currentState = ensureModelNodeState(node)
    if (currentState.stpFileUrl && currentState.stpFileUrl !== url) throw new Error('模型预览加载已取消')
    const statusRes = await request.get(`/stp/mesh-preview/${taskId}`)
    const data = statusRes.data?.data || {}
    updateModelLoadingMessage(node, renderTaskStatusMessage(data, '模型解析中...'))
    const meshPayload = normalizeMeshPreviewPayload(data)
    if ((data.status === 'completed' || data.status === 'done') && meshPayload) {
      return meshPayload
    }
    if ((data.status === 'completed' || data.status === 'done') && !meshPayload) {
      throw new Error('模型预览任务已完成，但没有返回可渲染的网格数据')
    }
    if (data.status === 'failed' || data.status === 'error') {
      throw new Error(data.error_msg || data.error || '模型预览解析失败')
    }
  }
  throw new Error('模型预览解析超时，请稍后重试或简化模型')
}

function normalizeMeshPreviewPayload(payload) {
  const meshPayload = payload?.mesh_data || payload?.meshData || payload?.data || payload
  if (!meshPayload?.vertices?.length || !meshPayload?.triangles?.length) return null
  return markRaw(meshPayload)
}

function taskQueuePosition(data = {}) {
  return Number(data.queue_position ?? data.queuePosition ?? 0) || 0
}

function taskQueueTotal(data = {}) {
  return Number(data.queue_total ?? data.queueTotal ?? 0) || 0
}

function renderTaskStatusMessage(data = {}, fallback = '等待渲染...') {
  const status = String(data.status || '').toLowerCase()
  const position = taskQueuePosition(data)
  const total = taskQueueTotal(data)
  if (status === 'pending' || status === 'queued') {
    if (position > 0 && total > 0) return `排队中，第 ${position}/${total} 位`
    if (position > 0) return `排队中，第 ${position} 位`
    return '排队中，等待渲染资源'
  }
  if (status === 'processing' || status === 'running') return '渲染中...'
  return fallback
}

function updateModelLoadingMessage(node, message) {
  const targetNode = node || activeModelNodeId.value
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [targetNode]: {
      ...ensureModelNodeState(targetNode),
      loading: true,
      loadingMessage: message,
      error: ''
    }
  }
  if (targetNode === activeModelNodeId.value) {
    loadingMessage.value = message
  }
}

function defaultModelGenerateText(node = activeModelNodeId.value) {
  return modelNodeState(node).controlMode === 'scene' ? '生成场景图' : '生成白底图'
}

function modelGenerateButtonText(node = activeModelNodeId.value) {
  return generating.value ? (renderTaskMessage.value || '生成中...') : defaultModelGenerateText(node)
}

async function createProject() {
  saveActiveProject()
  const project = createEmptyProject(newProjectName.value)
  try {
    const dbState = await ensureProjectImagesOnOSS(projectStateForDatabase(project))
    const res = await request.post('/workflow-projects', {
      project_id: project.id,
      name: project.name,
      state: dbState
    })
    const savedProject = workflowProjectFromResponse(res.data?.data || {})
    projects.value = [savedProject, ...projects.value]
    newProjectName.value = ''
    activeProjectId.value = savedProject.id
    await restoreProject(savedProject)
  } catch (e) {
    error.value = '创建项目失败：' + (e.response?.data?.error || e.message || e)
  }
}

function openCreateProjectModal() {
  projectNameDraft.value = getDefaultProjectName()
  showProjectNameModal.value = true
  nextTick(() => {
    projectNameModalInput.value?.focus?.()
    projectNameModalInput.value?.select?.()
  })
}

function closeProjectNameModal() {
  showProjectNameModal.value = false
  projectNameDraft.value = ''
}

function closeDeleteProjectModal() {
  deleteProjectTarget.value = null
}

function openHistoryModal() {
  showHistoryModal.value = true
  showNodeMenu.value = false
}

function closeHistoryModal() {
  showHistoryModal.value = false
}

async function confirmCreateProject() {
  const name = projectNameDraft.value.trim()
  if (!name) {
    error.value = '请输入项目名称'
    return
  }
  newProjectName.value = name
  await createProject()
  closeProjectNameModal()
}

function startRenameProject(project) {
  renamingProjectId.value = project.id
  renameProjectName.value = project.name
}

function cancelRenameProject() {
  renamingProjectId.value = ''
  renameProjectName.value = ''
}

async function confirmRenameProject(projectId) {
  if (renamingProjectId.value !== projectId) return
  const project = projects.value.find(item => item.id === projectId)
  const nextName = renameProjectName.value.trim()
  cancelRenameProject()
  if (!project || !nextName || nextName === project.name) return

  const previousName = project.name
  project.name = nextName
  try {
    const state = projectStateForDatabase(project)
    await request.put(`/workflow-projects/${encodeURIComponent(project.id)}`, {
      name: project.name,
      state
    })
  } catch (e) {
    project.name = previousName
    error.value = '重命名项目失败：' + (e.response?.data?.error || e.message || e)
  }
}

function deleteProject(projectId) {
  const project = projects.value.find(item => item.id === projectId)
  if (!project) return
  deleteProjectTarget.value = project
}

async function confirmDeleteProject() {
  const project = deleteProjectTarget.value
  if (!project) return
  const projectId = project.id
  closeDeleteProjectModal()
  try {
    await request.delete(`/workflow-projects/${encodeURIComponent(projectId)}`)
    projects.value = projects.value.filter(item => item.id !== projectId)
    if (activeProjectId.value === projectId) {
      const nextProject = projects.value[0]
      if (nextProject) {
        activeProjectId.value = nextProject.id
        await restoreProject(nextProject)
      } else {
        activeProjectId.value = ''
        workflowNodes.value = normalizeWorkflowNodes({})
        workflowDynamicNodes.value = []
        workflowLinks.value = []
        nodePositions.value = clonePlain(DEFAULT_NODE_POSITIONS)
      }
    }
  } catch (e) {
    error.value = '删除项目失败：' + (e.response?.data?.error || e.message || e)
  }
}

async function selectProject(projectId) {
  if (projectId === activeProjectId.value) return
  saveActiveProject()
  let project = projects.value.find(item => item.id === projectId)
  if (!project) return
  try {
    const res = await request.get(`/workflow-projects/${encodeURIComponent(projectId)}`)
    project = workflowProjectFromResponse(res.data?.data || {})
    projects.value = projects.value.map(item => item.id === project.id ? project : item)
  } catch (e) {
    console.warn('[Workflow] load project detail failed:', e)
  }
  activeProjectId.value = project.id
  await restoreProject(project)
  nextTick(bumpViewportCheck)
}

async function loadWorkflowProjects() {
  try {
    const res = await request.get('/workflow-projects')
    const list = Array.isArray(res.data?.data) ? res.data.data : []
    projects.value = list.map(workflowProjectFromResponse)
    if (projects.value.length > 0) {
      activeProjectId.value = projects.value[0].id
      await restoreProject(projects.value[0])
      nextTick(bumpViewportCheck)
    } else {
      activeProjectId.value = ''
    }
  } catch (e) {
    console.warn('[Workflow] load projects failed:', e)
    projects.value = []
    activeProjectId.value = ''
  }
}

function setWorkspaceMode(mode) {
  workspaceMode.value = mode
  nextTick(() => {
    if (mode === 'model') onResize()
  })
}

function setActiveWorkflowNode(node) {
  if (generateImageEditState.value.node && generateImageEditState.value.node !== node) {
    stopGenerateImageEdit()
  }
  closeEditingTextNodesExcept(node)
  closeLinkCreateMenu()
  activeWorkflowNode.value = node
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  closeAllGenerateMenus()
}

function activateGenerateNodeEditor(node) {
  updateGenerateNode(node, { showResults: false })
  setActiveWorkflowNode(node)
}

function editableWorkflowImageNode(sourceNode, imageUrl, sourceType = '') {
  const resolved = resolveImageUrl(imageUrl)
  if (!resolved) return ''
  const source = String(sourceType || '')
  return Object.entries(workflowGenerateNodeStates.value || {})
    .find(([node, state]) => {
      if (!nodeExists(node)) return false
      const result = normalizeGenerationResult(state.result)
      const images = result.images.map(resolveImageUrl)
      if (!images.includes(resolved)) return false
      if (source && !result.imageSources.includes(source)) return false
      if (sourceNode && state.importedFromNode && state.importedFromNode !== sourceNode) return false
      return true
    })?.[0] || ''
}

function openWorkflowImageAsEditableNode(sourceNode, imageUrl, title = '图片', sourceType = 'imported') {
  const resolved = resolveImageUrl(imageUrl)
  if (!resolved) return ''
  const existingNode = editableWorkflowImageNode(sourceNode, resolved, sourceType)
  if (existingNode) {
    activateGenerateNodeEditor(existingNode)
    return existingNode
  }

  const sourceRect = computedNodeRect(sourceNode) || nodePositions.value[sourceNode] || defaultNewNodePosition('generate')
  const base = generateResultNodeSizeForImage(resolved, DEFAULT_NODE_POSITIONS.generate)
  const position = findNonOverlappingPosition({
    ...base,
    x: Math.round(sourceRect.x + (sourceRect.width || DEFAULT_NODE_POSITIONS.generate.width) + GENERATED_RESULT_NODE_GAP_X),
    y: Math.round(sourceRect.y + Math.max(0, ((sourceRect.height || base.height) - base.height) / 2))
  }, {
    exclude: [sourceNode],
    stepY: GENERATED_RESULT_NODE_GAP_Y + 120
  })
  const now = new Date().toISOString()
  pushWorkflowUndoSnapshot()
  const imageNode = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(sourceNode),
    position,
    state: {
      importedFromNode: sourceNode,
      result: {
        url: resolved,
        images: [resolved],
        imageCreatedAt: [now],
        imageSources: [sourceType || 'imported'],
        taskIds: [],
        taskId: '',
        createdAt: now,
        status: 'imported',
        batchTitle: title
      },
      uploads: [],
      showResults: false
    }
  })
  if (!imageNode) return ''
  if (sourceNode && nodeExists(sourceNode)) {
    connectWorkflowNodes(sourceNode, imageNode, { undo: false, persist: false })
  }
  activateGenerateNodeEditor(imageNode)
  saveActiveProject()
  return imageNode
}

function defaultDrawingModelName() {
  if (workflowGenerateModel.value && drawingModels.value.some(model => model.name === workflowGenerateModel.value)) {
    return workflowGenerateModel.value
  }
  return drawingModels.value.find(model => model.is_default)?.name || drawingModels.value[0]?.name || workflowGenerateModel.value || ''
}

function effectiveDrawingModelName(node = activeGenerateNodeId()) {
  const state = ensureGenerateNodeState(node)
  if (state.model && drawingModels.value.some(model => model.name === state.model)) return state.model
  return defaultDrawingModelName()
}

function selectedDrawingModel(node = activeGenerateNodeId()) {
  const modelName = effectiveDrawingModelName(node)
  return drawingModels.value.find(model => model.name === modelName) || null
}

function generateAspectRatioOptionsForModel(model = '') {
  const normalized = String(model || '').toLowerCase()
  if (normalized.includes('gemini-3.1-flash-image')) return GEMINI_FLASH_ASPECT_RATIO_OPTIONS
  if (normalized.includes('gemini-3-pro-image')) return GEMINI_PRO_ASPECT_RATIO_OPTIONS
  if (normalized.includes('gpt-image-2')) return GPT_IMAGE_ASPECT_RATIO_OPTIONS
  return generateAspectRatioOptions
}

function generateAspectRatioOptionsForNode(node = activeGenerateNodeId()) {
  return generateAspectRatioOptionsForModel(effectiveDrawingModelName(node))
}

function modelSceneAspectRatioOptions() {
  return generateAspectRatioOptionsForModel(workflowGenerateModel.value)
}

function normalizeModelSceneAspectRatio(ratio) {
  const options = modelSceneAspectRatioOptions()
  return options.includes(ratio) ? ratio : options[0] || '1:1'
}

function supportedGenerateAspectRatio(node, ratio) {
  const options = generateAspectRatioOptionsForNode(node)
  return options.includes(ratio) ? ratio : options[0] || '1:1'
}

function workflowGenerateControlLabel(node = activeGenerateNodeId()) {
  const state = ensureGenerateNodeState(node)
  return `${state.aspectRatio} · ${state.resolution}`
}

function aspectIconClass(ratio) {
  const normalized = ratio.replace(':', '-')
  return `generate-aspect-icon--${normalized}`
}

function closeAllGenerateMenus() {
  workflowGenerateMenu.value = ''
  Object.keys(workflowGenerateNodeStates.value).forEach((key) => {
    if (workflowGenerateNodeStates.value[key]?.menu || workflowGenerateNodeStates.value[key]?.showResults) {
      workflowGenerateNodeStates.value[key] = {
        ...workflowGenerateNodeStates.value[key],
        menu: '',
        showResults: false
      }
    }
  })
  Object.keys(workflowVideoNodeStates.value).forEach((key) => {
    if (workflowVideoNodeStates.value[key]?.menu) {
      workflowVideoNodeStates.value[key] = {
        ...workflowVideoNodeStates.value[key],
        menu: ''
      }
    }
  })
}

function closeGenerateResults(exceptNode = '') {
  Object.keys(workflowGenerateNodeStates.value).forEach((key) => {
    if (key === exceptNode || !workflowGenerateNodeStates.value[key]?.showResults) return
    workflowGenerateNodeStates.value[key] = {
      ...workflowGenerateNodeStates.value[key],
      showResults: false
    }
  })
}

function toggleWorkflowGenerateMenu(node, menu) {
  const state = ensureGenerateNodeState(node)
  updateGenerateNode(node, { menu: state.menu === menu ? '' : menu }, { persist: false })
}

function selectWorkflowGenerateOption(node, type, value) {
  const current = ensureGenerateNodeState(node)
  const patch = {}
  if (type === 'model') {
    patch.model = value
    const options = generateAspectRatioOptionsForModel(value)
    if (!options.includes(current.aspectRatio)) patch.aspectRatio = options[0] || '1:1'
  }
  if (type === 'aspect') patch.aspectRatio = value
  if (type === 'resolution') patch.resolution = value
  if (type === 'count') patch.count = Number(value) || 1
  if (type === 'model' || type === 'count') patch.menu = ''
  updateGenerateNode(node, patch)
}

function toggleVideoNodeMenu(node, menu) {
  const state = ensureVideoNodeState(node)
  updateVideoNode(node, { menu: state.menu === menu ? '' : menu }, { persist: false })
}

function videoResolutionOptionsForNode(node) {
  return arkVideoResolutions(ensureVideoNodeState(node).model)
}

function selectVideoNodeModel(node, model) {
  const state = ensureVideoNodeState(node)
  const resolutions = arkVideoResolutions(model)
  updateVideoNode(node, {
    model,
    resolution: resolutions.includes(state.resolution) ? state.resolution : resolutions[resolutions.length - 1],
    menu: ''
  })
}

function applyCanvasTransform(view = liveCanvasView, options = {}) {
  const syncFloatingScale = options.syncFloatingScale !== false
  liveCanvasView = {
    x: Number(view.x) || 0,
    y: Number(view.y) || 0,
    scale: Number(view.scale) || 1
  }
  if (workflowStageRef.value) {
    workflowStageRef.value.style.transform = `translate3d(${liveCanvasView.x}px, ${liveCanvasView.y}px, 0) scale(${liveCanvasView.scale})`
    if (syncFloatingScale) {
      workflowStageRef.value.style.setProperty('--canvas-editor-scale', `${canvasFloatingPanelScale(liveCanvasView.scale)}`)
    }
  }
}

function scheduleCanvasFloatingScaleSync() {
  if (canvasFloatingScaleTimer) window.clearTimeout(canvasFloatingScaleTimer)
  canvasFloatingScaleTimer = window.setTimeout(() => {
    canvasFloatingScaleTimer = null
    if (!workflowStageRef.value) return
    workflowStageRef.value.style.setProperty('--canvas-editor-scale', `${canvasFloatingPanelScale(liveCanvasView.scale)}`)
  }, 100)
}

function canvasFloatingPanelScale(scale = 1) {
  const normalized = THREE.MathUtils.clamp(Number(scale) || 1, 0.08, 3)
  return Number(THREE.MathUtils.clamp(Math.pow(normalized, -1.1), 0.22, 5.5).toFixed(4))
}

function syncCanvasViewFromLive({ persist = true, viewport = true } = {}) {
  canvasView.value = { ...liveCanvasView }
  if (viewport) bumpViewportCheck()
  if (persist) saveCanvasViewState()
}

function bumpViewportCheck() {
  viewportCheckTick.value += 1
}

function scheduleViewportCheck(delay = 120) {
  if (viewportCheckTimer) window.clearTimeout(viewportCheckTimer)
  viewportCheckTimer = window.setTimeout(() => {
    viewportCheckTimer = null
    bumpViewportCheck()
  }, delay)
}

function scheduleCanvasStateSave() {
  if (canvasSaveTimer) window.clearTimeout(canvasSaveTimer)
  canvasSaveTimer = window.setTimeout(() => {
    canvasSaveTimer = null
    saveCanvasViewState({ persist: false })
    scheduleProjectSave(2400)
  }, 1200)
}

function nodeStyle(node) {
  const pos = computedNodeRect(node)
  if (!pos) return {}

  return {
    left: `${pos.x}px`,
    top: `${pos.y}px`,
    width: `${pos.width}px`,
    minHeight: `${pos.height}px`,
    height: `${pos.height}px`
  }
}

function nodeMinVisualSize(node) {
  const type = nodeType(node)
  if (type === 'batch-recolor') return { width: 400, height: 396 }
  if (type === 'batch-translate') return { width: 420, height: 432 }
  if (type === 'video') return { width: 300, height: 220 }
  return null
}

function aspectRatioValue(ratio) {
  const [width, height] = String(ratio || '1:1').split(':').map(Number)
  return width > 0 && height > 0 ? width / height : 1
}

function cacheOriginalImageSize(resolved, nextSize) {
  const current = imageSizeCache.value[resolved]
  if (current?.width === nextSize.width && current?.height === nextSize.height) return
  imageSizeCache.value = {
    ...imageSizeCache.value,
    [resolved]: nextSize
  }
  relayoutGeneratedResultNodesForImage(resolved)
}

function probeOriginalImageSize(resolved) {
  if (!resolved || originalImageSizeProbes.has(resolved)) return
  originalImageSizeProbes.set(resolved, true)
  const probe = new Image()
  probe.onload = () => {
    originalImageSizeProbes.delete(resolved)
    if (!probe.naturalWidth || !probe.naturalHeight) return
    cacheOriginalImageSize(resolved, {
      width: probe.naturalWidth,
      height: probe.naturalHeight
    })
  }
  probe.onerror = () => {
    originalImageSizeProbes.delete(resolved)
  }
  probe.src = resolved
}

function comparableImageUrl(url) {
  const resolved = resolveImageUrl(url)
  if (!resolved) return ''
  try {
    const parsed = new URL(resolved, window.location.origin)
    if (parsed.origin === window.location.origin) return `${parsed.pathname}${parsed.search}`
    return parsed.href
  } catch {
    return resolved
  }
}

function rememberImageSize(url, event) {
  const resolved = resolveImageUrl(url)
  const img = event?.target
  if (!resolved) return
  const loadedSrc = resolveImageUrl(img?.currentSrc || img?.src || '')
  if (loadedSrc && comparableImageUrl(loadedSrc) !== comparableImageUrl(resolved)) {
    probeOriginalImageSize(resolved)
    return
  }
  if (!img?.naturalWidth || !img?.naturalHeight) return
  cacheOriginalImageSize(resolved, {
    width: img.naturalWidth,
    height: img.naturalHeight
  })
}

function primaryGenerationImageRatio(node) {
  const image = primaryGenerationImage(node)
  const size = imageSizeCache.value[resolveImageUrl(image)]
  if (size?.width > 0 && size?.height > 0) return size.width / size.height
  return aspectRatioValue(ensureGenerateNodeState(node).aspectRatio)
}

function generationImageRatio(url) {
  const size = imageSizeCache.value[resolveImageUrl(url)]
  if (size?.width > 0 && size?.height > 0) return size.width / size.height
  return 16 / 9
}

function generateNodeImageSizeText(node) {
  const image = primaryGenerationImage(node)
  const size = imageSizeCache.value[resolveImageUrl(image)]
  if (!size?.width || !size?.height) return ''
  return `${size.width} × ${size.height}`
}

function generateNodeVisualSize(node) {
  const base = nodePositions.value[node] || DEFAULT_NODE_POSITIONS.generate
  if (nodeType(node) !== 'generate' || ensureGenerateNodeState(node).generating || !generationImages(node).length) {
    return { width: base.width, height: base.height }
  }

  const ratio = primaryGenerationImageRatio(node)
  const minWidth = 360
  const minHeight = 220
  const maxWidth = 720
  const maxHeight = 620
  let width = Math.max(minWidth, Math.min(maxWidth, base.width || 520))
  let height = width / ratio
  if (height > maxHeight) {
    height = maxHeight
    width = height * ratio
  }
  if (height < minHeight) {
    height = minHeight
    width = height * ratio
  }
  if (width > maxWidth) {
    width = maxWidth
    height = width / ratio
  }
  if (width < minWidth) {
    width = minWidth
    height = width / ratio
  }
  return {
    width: Math.round(width),
    height: Math.round(height)
  }
}

function generateResultNodeSizeForImage(url, base = DEFAULT_NODE_POSITIONS.generate) {
  const ratio = generationImageRatio(url)
  const minWidth = 360
  const minHeight = 220
  const maxWidth = 720
  const maxHeight = 620
  let width = Math.max(minWidth, Math.min(maxWidth, base.width || 520))
  let height = width / ratio
  if (height > maxHeight) {
    height = maxHeight
    width = height * ratio
  }
  if (height < minHeight) {
    height = minHeight
    width = height * ratio
  }
  if (width > maxWidth) {
    width = maxWidth
    height = width / ratio
  }
  if (width < minWidth) {
    width = minWidth
    height = width / ratio
  }
  return {
    width: Math.round(width),
    height: Math.round(height)
  }
}

function videoNodeVisualSize(node) {
  const base = nodePositions.value[node] || DEFAULT_NODE_POSITIONS.video
  const state = ensureVideoNodeState(node)
  if (!state.videoUrl || !state.videoWidth || !state.videoHeight) {
    return { width: base.width, height: base.height }
  }

  const ratio = state.videoWidth / state.videoHeight
  const minWidth = 300
  const minHeight = 220
  const maxWidth = 720
  const maxHeight = 620
  let width = Math.max(minWidth, Math.min(maxWidth, base.width || DEFAULT_NODE_POSITIONS.video.width))
  let height = width / ratio
  if (height > maxHeight) {
    height = maxHeight
    width = height * ratio
  }
  if (height < minHeight) {
    height = minHeight
    width = height * ratio
  }
  if (width > maxWidth) {
    width = maxWidth
    height = width / ratio
  }
  if (width < minWidth) {
    width = minWidth
    height = width / ratio
  }
  return {
    width: Math.round(width),
    height: Math.round(height)
  }
}

function computedNodeRect(node) {
  const pos = nodePositions.value[node]
  if (!pos) return null
  const minSize = nodeMinVisualSize(node)
  if (nodeType(node) === 'video') {
    return {
      ...pos,
      ...videoNodeVisualSize(node)
    }
  }
  if (minSize) {
    return {
      ...pos,
      width: Math.max(pos.width || 0, minSize.width),
      height: Math.max(pos.height || 0, minSize.height)
    }
  }
  if (nodeType(node) !== 'generate') return pos
  return {
    ...pos,
    ...generateNodeVisualSize(node)
  }
}

function defaultNewNodePosition(node) {
  const base = nodePositions.value[node] || DEFAULT_NODE_POSITIONS[node] || DEFAULT_NODE_POSITIONS.generate
  const canvas = workflowCanvasRef.value
  const scale = liveCanvasView.scale || 1
  if (!canvas) return { ...base }
  const centerX = (canvas.clientWidth / 2 - liveCanvasView.x) / scale
  const centerY = (canvas.clientHeight / 2 - liveCanvasView.y) / scale
  return {
    ...base,
    x: Math.round(centerX - (base.width || 0) / 2),
    y: Math.round(centerY - (base.height || 0) / 2)
  }
}

function rectsIntersect(a, b, gap = 56) {
  if (!a || !b) return false
  return a.x < b.x + b.width + gap &&
    a.x + a.width + gap > b.x &&
    a.y < b.y + b.height + gap &&
    a.y + a.height + gap > b.y
}

function existingNodeRects(exclude = []) {
  const excluded = new Set(exclude)
  return workflowNodeIds()
    .filter(id => !excluded.has(id))
    .map(id => computedNodeRect(id))
    .filter(Boolean)
}

function findNonOverlappingPosition(base, options = {}) {
  const width = Number(base.width) || DEFAULT_NODE_POSITIONS.generate.width
  const height = Number(base.height) || DEFAULT_NODE_POSITIONS.generate.height
  const startX = Math.round(Number(base.x) || 0)
  const startY = Math.round(Number(base.y) || 0)
  const stepY = Number(options.stepY) || 420
  const maxRows = Number(options.maxRows) || 80
  const gap = Number(options.gap) || 56
  const rects = existingNodeRects(options.exclude || [])
  for (let i = 0; i < maxRows; i += 1) {
    const direction = i % 2 === 0 ? 1 : -1
    const distance = Math.ceil(i / 2) * stepY
    const candidate = {
      x: startX,
      y: Math.round(startY + direction * distance),
      width,
      height
    }
    if (!rects.some(rect => rectsIntersect(candidate, rect, gap))) {
      return {
        ...base,
        x: candidate.x,
        y: candidate.y,
        width,
        height
      }
    }
  }
  return {
    ...base,
    x: startX,
    y: Math.round(startY + maxRows * stepY),
    width,
    height
  }
}

function setWorkflowNodePosition(node, position) {
  if (!node || !position) return
  nodePositions.value = {
    ...nodePositions.value,
    [node]: {
      ...(nodePositions.value[node] || {}),
      ...position,
      x: Math.round(position.x),
      y: Math.round(position.y)
    }
  }
}

function placeOutputNodeNearUpload(node = activeModelNodeId.value || 'upload') {
  const upload = nodePositions.value[node] || DEFAULT_NODE_POSITIONS.upload
  const output = nodePositions.value.output || DEFAULT_NODE_POSITIONS.output
  nodePositions.value = {
    ...nodePositions.value,
    output: {
      ...output,
      x: Math.round(upload.x + upload.width + 48),
      y: Math.round(upload.y + Math.max(0, (upload.height - output.height) / 2))
    }
  }
}

function outputNodeForModelNode(modelNode) {
  const existing = workflowLinks.value.find(link => link.from === modelNode && nodeType(link.to) === 'output')?.to
  if (existing) return existing
  if (!workflowNodes.value.output) return 'output'
  if (!workflowLinks.value.some(link => nodeType(link.from) === 'upload' && link.to === 'output')) return 'output'
  return createWorkflowNodeId('output')
}

function ensureOutputNodeForModel(modelNode, options = {}) {
  const outputNode = options.forceNew ? createWorkflowNodeId('output') : outputNodeForModelNode(modelNode)
  const upload = nodePositions.value[modelNode] || DEFAULT_NODE_POSITIONS.upload
  const base = nodePositions.value[outputNode] || DEFAULT_NODE_POSITIONS.output
  const offsetY = Number(options.offsetY) || 0
  if (outputNode === 'output') {
    workflowNodes.value = {
      ...workflowNodes.value,
      output: true
    }
  } else if (!workflowDynamicNodes.value.some(item => item.id === outputNode)) {
    workflowDynamicNodes.value = [
      ...workflowDynamicNodes.value,
      { id: outputNode, type: 'output' }
    ]
  }
  const targetPosition = findNonOverlappingPosition({
    ...base,
    x: Math.round(upload.x + upload.width + 96),
    y: Math.round(upload.y + Math.max(0, (upload.height - base.height) / 2) + offsetY),
    width: base.width || DEFAULT_NODE_POSITIONS.output.width,
    height: base.height || DEFAULT_NODE_POSITIONS.output.height
  }, {
    exclude: [outputNode, modelNode],
    stepY: 460
  })
  setWorkflowNodePosition(outputNode, targetPosition)
  return outputNode
}

function productWhiteTaskCountForModel(modelNode) {
  return Object.entries(workflowOutputNodeStates.value || {})
    .filter(([node, state]) => nodeExists(node) && state?.sourceNode === modelNode && state?.controlMode !== 'scene')
    .length
}

function removeModelStyleReferenceNode(modelNode) {
  const state = ensureModelNodeState(modelNode)
  const styleNode = state.sceneStyleNode
  if (!styleNode || !nodeExists(styleNode)) return
  removeWorkflowNode(styleNode, { undo: false, persist: false })
}

function removeModelMaterialReferenceNode(modelNode) {
  const state = ensureModelNodeState(modelNode)
  const materialNode = state.sceneMaterialNode
  if (!materialNode || !nodeExists(materialNode)) return
  removeWorkflowNode(materialNode, { undo: false, persist: false })
}

function sceneCameraInstruction(control = {}) {
  const pitch = Number(control.viewGuide?.pitch) || 0
  const pitchText = pitch > 0
    ? `约 ${Math.round(pitch)}° 俯视`
    : (pitch < 0 ? `约 ${Math.round(Math.abs(pitch))}° 仰视` : '接近平视')
  const cameraText = control.viewGuide?.camera || control.viewGuide?.detail || '第1张参考图的观察方向'
  return `${cameraText}，${pitchText}。只移动相机，不旋转产品或水槽；产品安装和出水方向优先。`
}

function sceneCoordinateInstruction(sceneType = 'bathroom') {
  const roomName = sceneType === 'kitchen' ? '厨房' : '浴室'
  const sinkName = sceneType === 'kitchen' ? '水槽' : '台盆'
  return `先建立固定${roomName}空间，再将产品装入${sinkName}后沿或侧后沿孔位，最后只移动相机；台面、${sinkName}和柜体不随产品旋转。`
}

function sceneInstallationInstruction(sceneType = 'bathroom') {
  return sceneType === 'kitchen'
    ? '底座贴合厨房水槽后沿或侧后沿的实心台面孔位，主体垂直，出水口朝水槽内。'
    : '底座贴合浴室台盆后沿或侧后沿的实心台面孔位，主体垂直，出水口朝台盆内。'
}

function sceneLayoutInstruction(sceneType = 'bathroom') {
  if (sceneType === 'kitchen') {
    return '真实住宅厨房：水槽嵌入连续台面，柜体、后挡板和周边空间关系合理。'
  }
  return '真实住宅浴室：台盆嵌入洗手台，柜体、镜子、墙面和周边空间关系合理。'
}

function sceneShotScaleInstruction(shot = 'wide') {
  const normalized = normalizeModelSceneShot(shot)
  if (normalized === 'close') {
    return '近景，突出产品与盆/槽的安装区域。'
  }
  if (normalized === 'medium') {
    return '中景，保留产品、台面、盆/槽和周边柜体关系。'
  }
  return '远景，展示完整空间，产品仍清晰可识别。'
}

function scenePhysicalScaleInstruction(sceneType = 'bathroom') {
  return sceneType === 'kitchen'
    ? '保持真实厨房五金尺度，产品明显小于水槽和台面。'
    : '保持真实浴室五金尺度，产品明显小于台盆和台面。'
}

function sceneGeometryStabilityInstruction(sceneType = 'bathroom') {
  const sinkName = sceneType === 'kitchen' ? '水槽' : '台盆'
  return `台面、${sinkName}、柜体和墙面保持统一透视与清晰直线；${sinkName}形状规则，产品不倾倒、悬浮或斜插。`
}

function buildCommercialScenePrompt(control = {}) {
  const sceneType = normalizeModelSceneType(control.sceneType)
  const sceneName = sceneType === 'kitchen' ? '厨房' : '浴室'
  const material = selectedModelSceneMaterial(control.sceneMaterial)
  const styleColorPrompt = String(control.sceneStyleRef?.prompt || '').trim()
  const hasStyleReference = !!styleColorPrompt
  const materialRefLabel = material ? '第3张参考图' : ''
  const styleText = hasStyleReference
    ? `仅将以下文字用于环境配色和硬装材质，不复制布局或其中产品：${styleColorPrompt}`
    : '环境使用干净、真实、低饱和的住宅配色。'
  const materialText = material
    ? `${materialRefLabel}只提供产品本体的“${material.label}”颜色、金属质感和反射，不控制几何或场景。`
    : '产品使用干净、真实的金属质感，并匹配场景光照。'
  const viewText = sceneCameraInstruction(control)
  const installationRules = sceneInstallationInstruction(sceneType)
  const layoutRules = sceneLayoutInstruction(sceneType)
  const shotScaleRules = sceneShotScaleInstruction(control.sceneShot)
  const physicalScaleRules = scenePhysicalScaleInstruction(sceneType)
  const geometryRules = sceneGeometryStabilityInstruction(sceneType)
  const coordinateRules = sceneCoordinateInstruction(sceneType)
  const referenceText = material
    ? `- 图1：唯一产品造型来源\n- 图2：仅提供深度、厚度和遮挡\n- 图3：仅提供“${material.label}”材质`
    : '- 图1：唯一产品造型来源\n- 图2：仅提供深度、厚度和遮挡'
  const sceneLines = [
    coordinateRules,
    `${installationRules} ${physicalScaleRules}`,
    `${shotScaleRules} ${viewText}`,
    `${layoutRules} ${styleText}`,
    materialText,
    '产品必须作为一个连续、完整的真实制造件安装在场景中；禁止拆分、拼接、补全或改变任何部件边界。'
  ].filter(Boolean).map(text => `- ${text}`).join('\n')
  return [
    `## 任务\n生成真实可信的${sceneName}产品场景图，只出现一套完整目标产品。优先级：产品几何 > 安装与尺度 > 场景透视 > 材质与风格。`,
    `## 参考图\n${referenceText}`,
    `## 产品几何（最高权重）\n${PRODUCT_GEOMETRY_RULES}`,
    `## 场景与拍摄\n${sceneLines}`,
    `## 输出检查\n产品结构、比例、部件连接和可见面与图1一致；场景透视统一，底座与台面真实接触，遮挡关系和接触阴影自然。产品在画面中保持真实五金尺度，不得因为景别放大或拉伸。不要重复产品、额外部件、镜中产品、悬浮、倾倒、巨大化、文字或 logo。${geometryRules}`
  ].join('\n\n')
}

function buildProductWhiteBgPrompt(control = {}) {
  const material = selectedModelSceneMaterial(control.sceneMaterial)
  const productSource = '唯一产品造型、视角、尺寸比例和部件连接关系来源'
  const referenceText = material
    ? `- 图1：${productSource}\n- 图2：仅提供深度、厚度和遮挡，不是遮罩\n- 图3：仅提供“${material.label}”材质`
    : `- 图1：${productSource}\n- 图2：仅提供深度、厚度和遮挡，不是遮罩`
  const materialText = material
    ? `图3只控制产品本体的颜色、粗糙度和反射；不得影响几何、背景或部件边界。`
    : '产品使用真实、干净的金属质感，必须有清晰的软箱高光、边缘高光、细腻的拉丝/电镀微表面和自然的明暗过渡，避免灰模、塑料感和死白。'
  const viewText = control.viewGuide?.camera
    ? `保持图1机位：${control.viewGuide.camera}。`
    : '保持图1的机位、朝向和透视。'
  const appearanceLines = [
    viewText,
    materialText,
    '保持产品原始颜色与材质分区，不把深度图、材质参考图或背景颜色误当作产品颜色。',
    '严格沿用图1产品轮廓的画面位置和占比：产品可见外轮廓的包围框中心必须位于图片正中心，左右留白相等、上下留白相等；不要因阴影或材质变化移动、放大、缩小或重新取景。',
    '使用商业棚拍的两侧柔光和顶部轮廓光表现真实材质；背景仍为纯白，允许极轻微、紧贴产品底部的自然接触阴影以增强立体感。'
  ].filter(Boolean).map(text => `- ${text}`).join('\n')
  return [
    '## 任务\n生成高分辨率商业产品白底主图，只展示一套完整产品。优先级：产品几何 > 完整性 > 机位 > 材质 > 构图与白底。',
    `## 参考图\n${referenceText}`,
    `## 产品几何（最高权重）\n${PRODUCT_GEOMETRY_RULES}`,
    `## 外观\n${appearanceLines}`,
    '## 输出检查\n产品完整、居中、不裁切，四周留白均衡；背景为均匀纯白 #FFFFFF，不出现地面、道具、文字、logo、水流、额外部件、镜像或材质参考图残留。不得拉伸、压扁、弯曲、圆柱化、缩放失真或改变任一部件的相对比例；保留真实金属高光、边缘反射、微表面纹理和细腻阴影，不能输出灰模或无质感的纯色塑料效果。'
  ].join('\n\n')
}

function sceneWorkflowPromptFromControl(control = {}) {
  return buildCommercialScenePrompt(control)
}

function productWhiteBgPromptFromControl(control = {}) {
  return buildProductWhiteBgPrompt(control)
}

function ensureMaterialReferenceNode(modelNode, outputNode, options = {}) {
  const state = ensureModelNodeState(modelNode)
  const material = selectedModelSceneMaterial(state.sceneMaterial)
  const materialUrl = material?.imageUrl ? resolveImageUrl(material.imageUrl) : ''
  if (!materialUrl) return ''
  const existingNode = !options.forceNew && state.sceneMaterialNode && nodeExists(state.sceneMaterialNode) ? state.sceneMaterialNode : ''
  if (existingNode) return existingNode

  const outputPos = nodePositions.value[outputNode] || DEFAULT_NODE_POSITIONS.output
  const materialBase = DEFAULT_NODE_POSITIONS.generate
  const position = findNonOverlappingPosition({
    ...materialBase,
    x: Math.round(outputPos.x + outputPos.width + 120),
    y: Math.round(outputPos.y - materialBase.height - 96),
    width: materialBase.width,
    height: materialBase.height
  }, {
    exclude: [outputNode],
    stepY: 360
  })
  const materialNode = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(outputNode),
    position,
    state: {
      uploadOnly: true,
      showResults: false,
      result: {
        url: materialUrl,
        images: [materialUrl],
        imageCreatedAt: [new Date().toISOString()],
        imageSources: ['material-reference'],
        createdAt: new Date().toISOString(),
        status: 'imported'
      },
      uploads: []
    }
  })
  if (!materialNode) return ''
  if (options.remember !== false) updateActiveModelExportOptions({ sceneMaterialNode: materialNode }, modelNode)
  return materialNode
}

function isReusableSceneGenerateNode(node, outputNode) {
  if (!nodeExists(node) || nodeType(node) !== 'generate') return false
  const state = ensureGenerateNodeState(node)
  if (state.generatedFromNode || state.uploadOnly || state.generating) return false
  if (generationResultImages(node).length || generationUploadImages(node).length) return false
  return state.sceneControlOutputNode === outputNode
}

function ensureScenePresetGenerateNode(modelNode, outputNode, options = {}) {
  const existing = options.forceNew
    ? ''
    : (Object.entries(workflowGenerateNodeStates.value || {})
        .find(([node]) => isReusableSceneGenerateNode(node, outputNode))?.[0]
        || workflowLinks.value.find(link => link.from === outputNode && isReusableSceneGenerateNode(link.to, outputNode))?.to)
  const modelState = ensureModelNodeState(modelNode)
  const outputState = outputNodeStateFor(outputNode)
  const styleNode = ''
  const materialNode = options.materialNode ?? ensureMaterialReferenceNode(modelNode, outputNode, {
    forceNew: !!options.forceNew,
    remember: options.forceNew ? false : undefined
  })
  const sceneAspectRatio = normalizeModelSceneAspectRatio(modelState.sceneAspectRatio)
  const sceneResolution = generateResolutionOptions.includes(modelState.sceneResolution) ? modelState.sceneResolution : '2K'
  const prompt = sceneWorkflowPromptFromControl({
    sceneType: modelState.sceneType,
    sceneShot: modelState.sceneShot,
    sceneStyle: modelState.sceneStyle,
    sceneMaterial: modelState.sceneMaterial,
    sceneStyleRef: modelState.sceneStyleRef,
    viewGuide: outputState.viewGuide
  })
  if (existing) {
    updateGenerateNode(existing, {
      prompt,
      count: 4,
      aspectRatio: sceneAspectRatio,
      resolution: sceneResolution,
      sceneControlOutputNode: outputNode,
      sceneWorkflowNode: true,
      modelWorkflowImageType: 'scene',
      imageEditDisabled: true
    }, { persist: false })
    connectWorkflowNodes(outputNode, existing, { undo: false, persist: false })
    if (styleNode) connectWorkflowNodes(styleNode, existing, { undo: false, persist: false })
    if (materialNode) connectWorkflowNodes(materialNode, existing, { undo: false, persist: false })
    return existing
  }

  const outputPos = nodePositions.value[outputNode] || DEFAULT_NODE_POSITIONS.output
  const generateBase = DEFAULT_NODE_POSITIONS.generate
  const generatePosition = findNonOverlappingPosition({
    ...generateBase,
    x: Math.round(outputPos.x + outputPos.width + 120),
    y: Math.round(outputPos.y + Math.max(0, (outputPos.height - generateBase.height) / 2)),
    width: generateBase.width,
    height: generateBase.height
  }, {
    exclude: [outputNode],
    stepY: 420
  })
  const generateNode = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(outputNode),
    position: generatePosition,
    state: {
      prompt,
      count: 4,
      aspectRatio: sceneAspectRatio,
      resolution: sceneResolution,
      showResults: false,
      sceneControlOutputNode: outputNode,
      sceneWorkflowNode: true,
      modelWorkflowImageType: 'scene',
      imageEditDisabled: true
    }
  })
  if (!generateNode) return ''

  connectWorkflowNodes(outputNode, generateNode, { undo: false, persist: false })
  if (styleNode) connectWorkflowNodes(styleNode, generateNode, { undo: false, persist: false })
  if (materialNode) connectWorkflowNodes(materialNode, generateNode, { undo: false, persist: false })
  return generateNode
}

function createProductWhiteBgGenerateNode(modelNode, outputNode, options = {}) {
  const modelState = ensureModelNodeState(modelNode)
  const outputState = outputNodeStateFor(outputNode)
  const materialNode = options.materialNode ?? ensureMaterialReferenceNode(modelNode, outputNode, {
    forceNew: true,
    remember: false
  })
  const outputPos = nodePositions.value[outputNode] || DEFAULT_NODE_POSITIONS.output
  const generateBase = DEFAULT_NODE_POSITIONS.generate
  const prompt = productWhiteBgPromptFromControl({
    sceneMaterial: modelState.sceneMaterial,
    viewGuide: outputState.viewGuide
  })
  const productResolution = generateResolutionOptions.includes(modelState.sceneResolution) ? modelState.sceneResolution : '2K'
  const generateNode = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(outputNode),
    position: {
      x: Math.round(outputPos.x + outputPos.width + 88),
      y: Math.round(outputPos.y + Math.max(0, (outputPos.height - generateBase.height) / 2))
    },
    state: {
      prompt,
      count: 1,
      aspectRatio: normalizeModelSceneAspectRatio('1:1'),
      resolution: productResolution,
      sceneControlOutputNode: outputNode,
      modelWorkflowImageType: 'product-white',
      imageEditDisabled: true,
      showResults: false,
      batchTitle: options.batchTitle || ''
    }
  })
  if (!generateNode) return ''

  connectWorkflowNodes(outputNode, generateNode, { undo: false, persist: false })
  if (materialNode) connectWorkflowNodes(materialNode, generateNode, { undo: false, persist: false })
  return generateNode
}

function productWhiteGenerateCountForModel(modelNode) {
  return workflowLinks.value
    .filter(link => link.from === modelNode && nodeType(link.to) === 'generate')
    .filter(link => ensureGenerateNodeState(link.to).modelWorkflowImageType === 'product-white')
    .length
}

function createDirectProductWhiteBgGenerateNode(modelNode, controlImages, options = {}) {
  const modelState = ensureModelNodeState(modelNode)
  const modelRect = computedNodeRect(modelNode) || nodePositions.value[modelNode] || DEFAULT_NODE_POSITIONS.upload
  const generateBase = DEFAULT_NODE_POSITIONS.generate
  const offsetY = Number(options.offsetY) || 0
  const prompt = productWhiteBgPromptFromControl({
    sceneMaterial: modelState.sceneMaterial,
    viewGuide: options.viewGuide || null
  })
  const productResolution = generateResolutionOptions.includes(modelState.sceneResolution) ? modelState.sceneResolution : '2K'
  const generatePosition = findNonOverlappingPosition({
    ...generateBase,
    x: Math.round(modelRect.x + (modelRect.width || DEFAULT_NODE_POSITIONS.upload.width) + 120),
    y: Math.round(modelRect.y + Math.max(0, ((modelRect.height || DEFAULT_NODE_POSITIONS.upload.height) - generateBase.height) / 2) + offsetY),
    width: generateBase.width,
    height: generateBase.height
  }, {
    exclude: [modelNode],
    stepY: 380
  })
  const generateNode = addWorkflowNode('generate', {
    undo: false,
    persist: false,
    groupId: workflowGroupIdForNode(modelNode),
    position: generatePosition,
    state: {
      prompt,
      count: 1,
      aspectRatio: normalizeModelSceneAspectRatio('1:1'),
      resolution: productResolution,
      inlineControlImages: normalizeInlineControlImages(controlImages),
      sceneControlOutputNode: '',
      modelWorkflowImageType: 'product-white',
      imageEditDisabled: true,
      showResults: false,
      batchTitle: options.batchTitle || ''
    }
  })
  if (!generateNode) return ''
  connectWorkflowNodes(modelNode, generateNode, { undo: false, persist: false })
  return generateNode
}

function isWorkflowEmpty() {
  return !Object.values(workflowNodes.value).some(Boolean) && workflowDynamicNodes.value.length === 0
}

function workflowNodeIds() {
  const staticIds = Object.keys(workflowNodes.value).filter(key => workflowNodes.value[key])
  const dynamicIds = workflowDynamicNodes.value.map(item => item.id)
  return [...staticIds, ...dynamicIds]
}

const visibleWorkflowNodeIds = computed(() => {
  viewportCheckTick.value
  const ids = workflowNodeIds()
  const canvas = workflowCanvasRef.value
  if (!canvas || !ids.length) return new Set(ids)
  if (canvasPanActive.value) return new Set(ids)
  const scale = liveCanvasView.scale || 1
  const padding = 480 / scale
  const left = -liveCanvasView.x / scale - padding
  const top = -liveCanvasView.y / scale - padding
  const right = left + canvas.clientWidth / scale + padding * 2
  const bottom = top + canvas.clientHeight / scale + padding * 2
  return new Set(ids.filter((id) => {
    const rect = computedNodeRect(id)
    return !!rect && rect.x + rect.width >= left && rect.x <= right && rect.y + rect.height >= top && rect.y <= bottom
  }))
})

function shouldRenderWorkflowNode(node) {
  return visibleWorkflowNodeIds.value.has(node) ||
    activeWorkflowNode.value === node ||
    activeModelNodeId.value === node ||
    generateImageEditState.value.node === node ||
    selectedNodeIds.value.includes(node) ||
    groupBoxes.value.some(group => group.id === activeGroupId.value && (group.nodes || []).includes(node)) ||
    groupBoxes.value.some(group => groupRunState(group.id).running && groupRunState(group.id).currentNode === node)
}

function visibleModelWorkflowNodes() {
  return modelWorkflowNodes().filter(node => shouldRenderWorkflowNode(node.id))
}

function visibleNonGenerateDynamicNodes() {
  return nonGenerateDynamicNodes().filter(node => shouldRenderWorkflowNode(node.id))
}

function visibleGenerateWorkflowNodes() {
  return generateWorkflowNodes().filter(node => shouldRenderWorkflowNode(node.id))
}

function workflowContentBounds() {
  const rects = workflowNodeIds()
    .map(id => computedNodeRect(id))
    .filter(Boolean)
  if (!rects.length) return null
  const minX = Math.min(...rects.map(rect => rect.x))
  const minY = Math.min(...rects.map(rect => rect.y))
  const maxX = Math.max(...rects.map(rect => rect.x + rect.width))
  const maxY = Math.max(...rects.map(rect => rect.y + rect.height))
  return {
    x: minX,
    y: minY,
    width: Math.max(1, maxX - minX),
    height: Math.max(1, maxY - minY)
  }
}

const hasNodesInViewport = computed(() => {
  viewportCheckTick.value
  const canvas = workflowCanvasRef.value
  const ids = workflowNodeIds()
  if (!canvas || !ids.length) return true
  const scale = liveCanvasView.scale || 1
  const left = -liveCanvasView.x / scale
  const top = -liveCanvasView.y / scale
  const right = left + canvas.clientWidth / scale
  const bottom = top + canvas.clientHeight / scale
  const padding = 80 / scale
  return ids.some((id) => {
    const rect = computedNodeRect(id)
    if (!rect) return false
    return rect.x + rect.width >= left - padding &&
      rect.x <= right + padding &&
      rect.y + rect.height >= top - padding &&
      rect.y <= bottom + padding
  })
})

function focusCanvasOnNodes(nodes = []) {
  const canvas = workflowCanvasRef.value
  const bounds = nodes.length ? selectedNodesBounds(nodes) : workflowContentBounds()
  if (!canvas || !bounds) return
  const padding = 160
  const fitScale = Math.min(
    (canvas.clientWidth - padding) / bounds.width,
    (canvas.clientHeight - padding) / bounds.height,
    1.1
  )
  const nextScale = THREE.MathUtils.clamp(fitScale, 0.08, 3)
  applyCanvasTransform({
    scale: nextScale,
    x: canvas.clientWidth / 2 - (bounds.x + bounds.width / 2) * nextScale,
    y: canvas.clientHeight / 2 - (bounds.y + bounds.height / 2) * nextScale
  })
  syncCanvasViewFromLive()
}

function nodeLinkPath(from, to) {
  if (!nodeExists(from) || !nodeExists(to)) return ''
  const start = computedNodeRect(from)
  const end = computedNodeRect(to)
  if (!start || !end) return ''

  return nodeLinkPathForRects(start, end)
}

function nodeLinkPathForRects(start, end) {
  const x1 = start.x + start.width + 18
  const y1 = start.y + start.height / 2
  const x2 = end.x - 18
  const y2 = end.y + end.height / 2

  const gap = Math.max(80, Math.abs(x2 - x1) * 0.45)
  return `M ${x1} ${y1} C ${x1 + gap} ${y1}, ${x2 - gap} ${y2}, ${x2} ${y2}`
}

const workflowLinkViews = computed(() => (
  workflowLinks.value
    .map(link => ({
      ...link,
      key: workflowLinkKey(link),
      path: nodeLinkPath(link.from, link.to)
    }))
    .filter(link => link.path)
))

function getNodeOutputPoint(node) {
  const pos = computedNodeRect(node)
  if (!pos) return { x: 0, y: 0 }
  return {
    x: pos.x + pos.width + 18,
    y: pos.y + pos.height / 2
  }
}

function clientPointToCanvas(e) {
  const canvas = e.currentTarget?.closest?.('.node-canvas') || workflowCanvasRef.value || e.currentTarget
  const rect = canvas.getBoundingClientRect()
  const scale = liveCanvasView.scale || 1
  return {
    x: (e.clientX - rect.left - liveCanvasView.x) / scale,
    y: (e.clientY - rect.top - liveCanvasView.y) / scale
  }
}

function clientPointToCanvasFromEvent(e) {
  const canvas = workflowCanvasRef.value || e.currentTarget
  const rect = canvas.getBoundingClientRect()
  const scale = liveCanvasView.scale || 1
  return {
    x: (e.clientX - rect.left - liveCanvasView.x) / scale,
    y: (e.clientY - rect.top - liveCanvasView.y) / scale
  }
}

function clientPointToCanvasViewport(e) {
  const canvas = workflowCanvasRef.value || e.currentTarget
  const rect = canvas.getBoundingClientRect()
  return {
    x: e.clientX - rect.left,
    y: e.clientY - rect.top
  }
}

function updateLastCanvasPointer(e) {
  if (!workflowCanvasRef.value) return
  lastCanvasPointer = clientPointToCanvasFromEvent(e)
}

function normalizeRect(rect) {
  const x = Math.min(rect.x, rect.x + rect.width)
  const y = Math.min(rect.y, rect.y + rect.height)
  return {
    x,
    y,
    width: Math.abs(rect.width),
    height: Math.abs(rect.height)
  }
}

const selectionBoxVisible = computed(() => (
  selectionBox.value.active ||
  (selectionBox.value.fixed && selectionBox.value.width > 4 && selectionBox.value.height > 4)
))

function selectionBoxStyle() {
  const rect = normalizeRect(selectionBox.value)
  return {
    left: `${rect.x}px`,
    top: `${rect.y}px`,
    width: `${rect.width}px`,
    height: `${rect.height}px`
  }
}

function rectIntersectsNode(rect, node) {
  const nodeRect = computedNodeRect(node)
  if (!nodeRect) return false
  const right = rect.x + rect.width
  const bottom = rect.y + rect.height
  const nodeRight = nodeRect.x + nodeRect.width
  const nodeBottom = nodeRect.y + nodeRect.height
  return (
    nodeRight >= rect.x &&
    nodeRect.x <= right &&
    nodeBottom >= rect.y &&
    nodeRect.y <= bottom
  )
}

function updateSelectedNodesFromBox() {
  const rect = normalizeRect(selectionBox.value)
  selectedNodeIds.value = workflowNodeIds().filter(node => rectIntersectsNode(rect, node))
}

function selectedNodesBounds(nodes = selectedNodeIds.value) {
  const rects = nodes.map(node => computedNodeRect(node)).filter(Boolean)
  if (!rects.length) return null
  const minX = Math.min(...rects.map(rect => rect.x))
  const minY = Math.min(...rects.map(rect => rect.y))
  const maxX = Math.max(...rects.map(rect => rect.x + rect.width))
  const maxY = Math.max(...rects.map(rect => rect.y + rect.height))
  return { x: minX, y: minY, width: maxX - minX, height: maxY - minY }
}

function selectionBoundsForNodes(nodes = selectedNodeIds.value) {
  const bounds = selectedNodesBounds(nodes)
  if (!bounds) return null
  const padding = 18
  return {
    x: bounds.x - padding,
    y: bounds.y - padding,
    width: bounds.width + padding * 2,
    height: bounds.height + padding * 2
  }
}

function snapSelectionBoxToNodes({ fixed = true } = {}) {
  const bounds = selectionBoundsForNodes()
  if (!bounds) return false
  selectionBox.value = {
    ...selectionBox.value,
    ...bounds,
    active: false,
    fixed
  }
  return true
}

function currentClipboardNodeIds() {
  if (selectedNodeIds.value.length) return [...new Set(selectedNodeIds.value)].filter(nodeExists)
  if (activeGroupId.value) {
    const group = groupBoxes.value.find(item => item.id === activeGroupId.value)
    return [...new Set(group?.nodes || [])].filter(nodeExists)
  }
  if (activeWorkflowNode.value && nodeExists(activeWorkflowNode.value)) return [activeWorkflowNode.value]
  return []
}

function copyWorkflowSelection() {
  const nodes = currentClipboardNodeIds()
  if (!nodes.length) return
  const bounds = selectedNodesBounds(nodes)
  if (!bounds) return
  const nodeSet = new Set(nodes)
  const activeGroup = activeGroupId.value
    ? groupBoxes.value.find(group => group.id === activeGroupId.value)
    : null
  workflowClipboard.value = {
    nodes: nodes.map(node => ({
      id: node,
      type: nodeType(node),
      position: clonePlain(nodePositions.value[node] || {}),
      generateState: nodeType(node) === 'generate' ? clonePlain(ensureGenerateNodeState(node)) : null,
      textState: nodeType(node) === 'text' ? clonePlain(ensureTextNodeState(node)) : null,
      videoState: nodeType(node) === 'video' ? clonePlain(ensureVideoNodeState(node)) : null,
      modelState: nodeType(node) === 'upload' ? clonePlain(ensureModelNodeState(node)) : null
    })),
    links: workflowLinks.value
      .filter(link => nodeSet.has(link.from) && nodeSet.has(link.to))
      .map(link => ({ ...link })),
    groups: activeGroup
      ? [{ nodes: (activeGroup.nodes || []).filter(node => nodeSet.has(node)) }]
      : groupBoxes.value
        .filter(group => (group.nodes || []).length > 1 && (group.nodes || []).every(node => nodeSet.has(node)))
        .map(group => ({ nodes: [...group.nodes] })),
    bounds
  }
}

function removeSelectedWorkflowNodes() {
  const nodes = [...new Set(selectedNodeIds.value)].filter(nodeExists)
  if (!nodes.length) return
  removeWorkflowNodes(nodes, { persist: false })
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  selectedNodeIds.value = []
  selectionBox.value = { active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 }
  bumpViewportCheck()
  saveActiveProject()
}

function pasteWorkflowSelection() {
  const clip = workflowClipboard.value
  if (!clip?.nodes?.length || !activeProjectId.value) return
  const bounds = clip.bounds
  if (!bounds) return
  const anchor = lastCanvasPointer || {
    x: bounds.x + Math.max(24, bounds.width + 36),
    y: bounds.y + 24
  }
  const offsetX = anchor.x - bounds.x
  const offsetY = anchor.y - bounds.y
  const idMap = {}
  const dynamicNodes = []
  const nextPositions = { ...nodePositions.value }
  const nextGenerateStates = { ...workflowGenerateNodeStates.value }
  const nextTextStates = { ...workflowTextNodeStates.value }
  const nextVideoStates = { ...workflowVideoNodeStates.value }
  const nextModelStates = { ...workflowModelNodeStates.value }

  pushWorkflowUndoSnapshot()
  clip.nodes.forEach((item) => {
    const nextId = createWorkflowNodeId(item.type || 'node')
    idMap[item.id] = nextId
    dynamicNodes.push({ id: nextId, type: item.type })
    nextPositions[nextId] = {
      ...(item.position || DEFAULT_NODE_POSITIONS[item.type] || DEFAULT_NODE_POSITIONS.generate),
      x: Math.round((typeof item.position?.x === 'number' ? item.position.x : bounds.x) + offsetX),
      y: Math.round((typeof item.position?.y === 'number' ? item.position.y : bounds.y) + offsetY)
    }
    if (item.type === 'generate') {
      nextGenerateStates[nextId] = normalizeGenerateNodeState({
        ...(item.generateState || {}),
        generatedFromNode: '',
        generating: false,
        cancelled: false,
        runId: '',
        taskProgress: []
      })
    }
    if (item.type === 'text') {
      nextTextStates[nextId] = normalizeTextNodeState(item.textState || {})
    }
    if (item.type === 'video') {
      nextVideoStates[nextId] = normalizeVideoNodeState(item.videoState || {}, { resetRuntime: true })
    }
    if (item.type === 'upload') {
      nextModelStates[nextId] = normalizeModelNodeState(item.modelState || {})
    }
  })

  workflowDynamicNodes.value = [
    ...workflowDynamicNodes.value,
    ...dynamicNodes
  ]
  nodePositions.value = nextPositions
  workflowGenerateNodeStates.value = nextGenerateStates
  workflowTextNodeStates.value = nextTextStates
  workflowVideoNodeStates.value = nextVideoStates
  workflowModelNodeStates.value = nextModelStates
  workflowLinks.value = normalizeWorkflowLinks([
    ...workflowLinks.value,
    ...clip.links
      .map(link => ({ from: idMap[link.from], to: idMap[link.to] }))
      .filter(link => link.from && link.to)
  ])

  const pastedNodeIds = Object.values(idMap)
  const nextGroups = clip.groups
    .map(group => (group.nodes || []).map(node => idMap[node]).filter(Boolean))
    .filter(nodes => nodes.length > 1)
  if (nextGroups.length) {
    nextGroups.forEach(nodes => createWorkflowGroup(nodes, { activate: false, persist: false }))
    activeGroupId.value = groupBoxes.value[groupBoxes.value.length - 1]?.id || ''
    activeWorkflowNode.value = ''
  } else {
    activeWorkflowNode.value = pastedNodeIds[0] || ''
    activeGroupId.value = ''
  }
  activeWorkflowLink.value = ''
  selectionBox.value = { active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 }
  selectedNodeIds.value = []
  closeAllGenerateMenus()
  bumpViewportCheck()
  saveActiveProject()
}

function moveNodesBy(nodes, dx, dy) {
  if (!nodes.length || (!dx && !dy)) return
  nodes.forEach((node) => {
    const pos = nodePositions.value[node]
    if (!pos) return
    pos.x += dx
    pos.y += dy
  })
}

function groupBoxStyle(group) {
  return {
    left: `${group.x}px`,
    top: `${group.y}px`,
    width: `${group.width}px`,
    height: `${group.height}px`
  }
}

function createWorkflowGroup(nodes, { activate = true, persist = true } = {}) {
  const groupNodes = [...new Set(nodes)].filter(nodeExists)
  if (groupNodes.length < 2) return ''
  const sortedNodes = [...groupNodes].sort()
  const existing = groupBoxes.value.find((group) => {
    const existingNodes = [...new Set(group.nodes || [])].filter(nodeExists).sort()
    return existingNodes.length === sortedNodes.length && existingNodes.every((node, index) => node === sortedNodes[index])
  })
  const bounds = selectedNodesBounds(groupNodes)
  if (!bounds) return ''
  const padding = 28
  if (existing) {
    const nextGroup = {
      ...existing,
      nodes: groupNodes,
      x: bounds.x - padding,
      y: bounds.y - padding,
      width: bounds.width + padding * 2,
      height: bounds.height + padding * 2
    }
    groupBoxes.value = groupBoxes.value.map(group => group.id === existing.id ? nextGroup : group)
    if (activate) activeGroupId.value = existing.id
    if (persist) saveActiveProject()
    return existing.id
  }
  const group = {
    id: `group-${Date.now()}-${Math.random().toString(16).slice(2, 7)}`,
    nodes: groupNodes,
    x: bounds.x - padding,
    y: bounds.y - padding,
    width: bounds.width + padding * 2,
    height: bounds.height + padding * 2
  }
  groupBoxes.value = [...groupBoxes.value, group]
  if (activate) activeGroupId.value = group.id
  if (persist) saveActiveProject()
  return group.id
}

function workflowGroupIsPristine(group) {
  return !!group?.sourceTemplateId && group.templatePristine !== false
}

function markTemplateWorkflowGroupsChanged() {
  if (!groupBoxes.value.some(workflowGroupIsPristine)) return
  groupBoxes.value = groupBoxes.value.map(group => workflowGroupIsPristine(group) && !groupRunState(group.id).running
    ? { ...group, templatePristine: false }
    : group)
}

function markTemplateWorkflowGroupChangedForNode(node) {
  if (isRestoringProject) return
  invalidatePausedWorkflowRunsForNodes([node])
  let changed = false
  const nextGroups = groupBoxes.value.map((group) => {
    if (!workflowGroupIsPristine(group) || groupRunState(group.id).running || !(group.nodes || []).includes(node)) return group
    changed = true
    return { ...group, templatePristine: false }
  })
  if (!changed) return
  groupBoxes.value = nextGroups
  saveProjectPatch({ workflowGroups: clonePlain(nextGroups) }, { persist: false })
}

const selectedDownloadImages = computed(() => {
  const images = []
  const selectedNodes = selectedNodeIds.value.length
    ? selectedNodeIds.value
    : (activeWorkflowNode.value && nodeExists(activeWorkflowNode.value) ? [activeWorkflowNode.value] : [])
  const nodes = [...new Set(selectedNodes)].filter(nodeExists)
  nodes.forEach((node) => {
    const type = nodeType(node)
    let urls = []
    if (type === 'generate') {
      urls = generationImages(node)
    } else if (type === 'output') {
      const output = ensureOutputNodeState(node).resultUrls || {}
      urls = [output.white, output.depth, output.productRgba]
    } else if (type === 'scene') {
      urls = [sceneImage.value.url]
    }
    urls.filter(Boolean).forEach((url, index) => {
      images.push({ node, url: resolveImageUrl(url), index })
    })
  })
  return images
})

const selectedDownloadImageCount = computed(() => selectedDownloadImages.value.length)

function handleBatchDownloadButton() {
  if (batchDownloadState.value.loading) return
  if (!batchDownloadSelectMode.value) {
    batchDownloadSelectMode.value = true
    selectedNodeIds.value = []
    activeWorkflowNode.value = ''
    resetGenerateImageEditState()
    return
  }
  if (!selectedDownloadImageCount.value) {
    batchDownloadSelectMode.value = false
    selectedNodeIds.value = []
    bumpViewportCheck()
    nextTick(bumpViewportCheck)
    return
  }
  downloadSelectedWorkflowImages()
}

function onWorkflowNodeClick(event, node) {
  if (batchDownloadSelectMode.value) {
    event?.stopPropagation?.()
    const current = new Set(selectedNodeIds.value)
    if (current.has(node)) current.delete(node)
    else current.add(node)
    selectedNodeIds.value = [...current]
    return
  }
  if (nodeType(node) === 'upload') {
    selectModelNode(node)
    return
  }
  setActiveWorkflowNode(node)
}

function batchDownloadSelectionOrder(node) {
  return selectedNodeIds.value.indexOf(node) + 1
}

function batchDownloadFilename(item, index, mimeType = '') {
  const nodeLabel = nodeType(item.node) === 'generate' ? generateNodeTitle(item.node) : dynamicNodeTitle({ type: nodeType(item.node) })
  const safeLabel = String(nodeLabel || '图片').replace(/[\\/:*?"<>|]/g, '_')
  const extension = mimeType.includes('png') ? 'png' : mimeType.includes('webp') ? 'webp' : 'jpg'
  return `${String(index + 1).padStart(2, '0')}-${safeLabel}-${item.index + 1}.${extension}`
}

async function downloadSelectedWorkflowImages() {
  const items = selectedDownloadImages.value
  if (!items.length || batchDownloadState.value.loading) return
  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  const token = localStorage.getItem('token')
  const downloadMeta = await requestDownloadMetadata(apiBase, token, {
    filename: `workflow-images-${new Date().toISOString().slice(0, 10)}.zip`,
  })
  if (downloadMeta === null) return
  batchDownloadState.value = { loading: true, completed: 0, total: items.length }
  const zip = new JSZip()
  let failedDownloads = 0
  for (let index = 0; index < items.length; index += 1) {
    const item = items[index]
    try {
      const image = await fetchImageAsDataUrl(item.url)
      zip.file(batchDownloadFilename(item, index, image.mimeType), image.blob)
      try {
        const state = nodeType(item.node) === 'generate' ? ensureGenerateNodeState(item.node) : null
        const result = state ? normalizeGenerationResult(state.result) : {}
        await request.post('/usage-logs/record-download', {
          image_url: item.url,
          model: state ? effectiveDrawingModelName(item.node) : 'workflow-canvas',
          prompt: state ? effectiveGenerationPrompt(item.node) : '',
          resolution: state?.resolution || '',
          aspect_ratio: state?.aspectRatio || '',
          source: 'workflow_canvas_batch',
          task_id: result.taskIds?.[item.index] || result.taskId || ''
        })
      } catch (recordError) {
        failedDownloads += 1
        console.warn('[Workflow] record batch image download failed:', recordError)
      }
    } catch (downloadError) {
      failedDownloads += 1
      console.warn('[Workflow] batch image download failed:', downloadError)
    } finally {
      batchDownloadState.value = { ...batchDownloadState.value, completed: index + 1 }
    }
  }
  try {
    const blob = await zip.generateAsync({ type: 'blob' })
    const url = URL.createObjectURL(blob)
    const archiveLabel = [downloadMeta.brand, downloadMeta.sku].filter(Boolean).join('-')
    const archiveFilename = `${archiveLabel ? `${archiveLabel}-` : ''}workflow-images-${new Date().toISOString().slice(0, 10)}.zip`
    try {
      const form = new FormData()
      form.append('file', blob, archiveFilename)
      const nasResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL || '/api'}/downloads/archive-to-nas`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: form,
      })
      if (!nasResponse.ok) throw new Error(`HTTP ${nasResponse.status}`)
    } catch (nasError) {
      console.warn('[Workflow] batch archive NAS sync failed:', nasError)
    }
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = archiveFilename
    anchor.click()
    URL.revokeObjectURL(url)
    if (failedDownloads) error.value = `${failedDownloads} 张图片下载或记录失败，已打包其余图片`
  } catch (zipError) {
    error.value = zipError?.message || '压缩包生成失败，请重试'
  } finally {
    batchDownloadState.value = { loading: false, completed: 0, total: 0 }
    batchDownloadSelectMode.value = false
    selectedNodeIds.value = []
  }
}

function groupRunState(groupId) {
  return workflowGroupRunStates.value[groupId] || { running: false, paused: false, currentNode: '', completedNodes: [], errorNode: '', error: '' }
}

function runningWorkflowGroupForNode(node) {
  return groupBoxes.value.find(group => groupRunState(group.id).running && (group.nodes || []).includes(node)) || null
}

function updateGroupRunState(groupId, patch) {
  const current = workflowGroupRunStates.value[groupId]
  if (current) {
    Object.assign(current, patch)
    return
  }
  workflowGroupRunStates.value[groupId] = { ...groupRunState(groupId), ...patch }
}

function resetWorkflowGroupRunProgress(groupId) {
  updateGroupRunState(groupId, { running: false, paused: false, currentNode: '', completedNodes: [], errorNode: '', error: '' })
}

function clearWorkflowGroupRunState(groupId) {
  if (!workflowGroupRunStates.value[groupId]) return
  const { [groupId]: removedState, ...nextStates } = workflowGroupRunStates.value
  workflowGroupRunStates.value = nextStates
}

function invalidatePausedWorkflowRunsForNodes(nodes, { structural = false } = {}) {
  const changedNodes = new Set((Array.isArray(nodes) ? nodes : [nodes]).filter(Boolean))
  groupBoxes.value.forEach((group) => {
    const state = groupRunState(group.id)
    if (state.running || !state.paused || !(group.nodes || []).some(node => changedNodes.has(node))) return
    if (structural || state.completedNodes.some(node => changedNodes.has(node))) resetWorkflowGroupRunProgress(group.id)
  })
}

function isWorkflowDerivedResultNode(node) {
  if (nodeType(node) !== 'generate') return false
  const state = ensureGenerateNodeState(node)
  const sources = normalizeGenerationResult(state.result).imageSources || []
  return !!state.generatedFromNode || sources.some(source => ['batch-recolor', 'batch-translate', 'depth-of-field', 'light-shadow'].includes(source))
}

function workflowGroupExecutionOrder(group) {
  const allNodes = [...new Set((group?.nodes || []).filter(nodeExists))]
  const included = new Set(allNodes)
  const nodes = allNodes.filter(node => !isWorkflowDerivedResultNode(node))
  const directOutgoing = Object.fromEntries(allNodes.map(node => [node, []]))
  workflowLinks.value.forEach((link) => {
    if (included.has(link.from) && included.has(link.to)) directOutgoing[link.from].push(link.to)
  })
  const incoming = Object.fromEntries(nodes.map(node => [node, 0]))
  const outgoing = Object.fromEntries(nodes.map(node => [node, new Set()]))
  nodes.forEach((source) => {
    const queue = [...(directOutgoing[source] || [])]
    const visited = new Set()
    while (queue.length) {
      const target = queue.shift()
      if (visited.has(target)) continue
      visited.add(target)
      if (isWorkflowDerivedResultNode(target)) {
        queue.push(...(directOutgoing[target] || []))
        continue
      }
      if (target !== source && outgoing[source] && !outgoing[source].has(target)) {
        outgoing[source].add(target)
        incoming[target] += 1
      }
    }
  })
  const queue = nodes.filter(node => incoming[node] === 0)
  const ordered = []
  while (queue.length) {
    const node = queue.shift()
    ordered.push(node)
    ;(outgoing[node] || []).forEach((next) => {
      incoming[next] -= 1
      if (incoming[next] === 0) queue.push(next)
    })
  }
  if (ordered.length !== nodes.length) throw new Error('工作流存在循环连接，请调整连线后再运行')
  return ordered
}

function workflowNodeExecutionLabel(node) {
  const type = nodeType(node)
  if (type === 'generate') return generateNodeTitle(node)
  if (type === 'llm') return '大模型'
  if (type === 'batch-recolor') return '批量换色'
  if (type === 'batch-translate') return '批量翻译'
  if (type === 'video') return '视频'
  if (type === 'upload') return '3D 文件'
  if (type === 'output') return '视图输出'
  if (type === 'text') return '文本'
  return '节点'
}

function workflowNodeExecutionFailure(node, message) {
  const error = new Error(`${workflowNodeExecutionLabel(node)}：${message}`)
  error.workflowNode = node
  return error
}

function recordWorkflowNodeExecutionError(node, message) {
  const type = nodeType(node)
  if (type === 'llm') llmNodeState(node).error = message
  if (type === 'generate') updateGenerateNode(node, { error: message }, { persist: false })
  if (type === 'batch-recolor') updateBatchRecolorNode(node, { error: message }, { persist: false })
  if (type === 'batch-translate') updateBatchTranslateNode(node, { error: message }, { persist: false })
  if (type === 'video') updateVideoNode(node, { error: message }, { persist: false })
}

function throwWorkflowNodeExecutionError(node, message) {
  recordWorkflowNodeExecutionError(node, message)
  throw workflowNodeExecutionFailure(node, message)
}

function assertWorkflowNodeExecutionSucceeded(node) {
  if (!nodeExists(node)) throw workflowNodeExecutionFailure(node, '运行过程中节点已被删除')
  const type = nodeType(node)
  if (type === 'llm') {
    const state = llmNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '运行未正常结束')
    if (state.error) throw workflowNodeExecutionFailure(node, state.error)
    if (!String(state.output || '').trim()) throwWorkflowNodeExecutionError(node, '模型没有返回内容')
  }
  if (type === 'generate') {
    const state = ensureGenerateNodeState(node)
    if (state.generating) throwWorkflowNodeExecutionError(node, '运行未正常结束')
    if (state.cancelled) throwWorkflowNodeExecutionError(node, '生成已取消')
    if (state.error) throw workflowNodeExecutionFailure(node, state.error)
    if (!generationResultImages(node).length) throwWorkflowNodeExecutionError(node, '没有生成图片')
  }
  if (type === 'batch-recolor') {
    const state = ensureBatchRecolorNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '运行未正常结束')
    if (state.error) throw workflowNodeExecutionFailure(node, state.error)
    if (batchFailedCount(state.tasks || [])) throwWorkflowNodeExecutionError(node, '部分换色任务失败')
  }
  if (type === 'batch-translate') {
    const state = ensureBatchTranslateNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '运行未正常结束')
    if (state.error) throw workflowNodeExecutionFailure(node, state.error)
    if (batchFailedCount(state.tasks || [])) throwWorkflowNodeExecutionError(node, '部分翻译任务失败')
  }
  if (type === 'video') {
    const state = ensureVideoNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '运行未正常结束')
    if (state.error) throw workflowNodeExecutionFailure(node, state.error)
    if (!state.videoUrl) throwWorkflowNodeExecutionError(node, '没有生成视频')
  }
}

async function executeWorkflowGroupNode(node) {
  if (!nodeExists(node)) throw workflowNodeExecutionFailure(node, '节点不存在')
  const type = nodeType(node)
  if (type === 'llm') {
    const state = llmNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '节点正在运行')
    if (!effectiveLlmPrompt(node)) throwWorkflowNodeExecutionError(node, '请输入文本或连接上游文本')
    await runLlmNode(node)
    assertWorkflowNodeExecutionSucceeded(node)
    return
  }
  if (type === 'generate') {
    const state = ensureGenerateNodeState(node)
    const prompt = effectiveGenerationPrompt(node)
    const passThrough = state.uploadOnly || state.generatedFromNode || isReferenceOnlyGenerateNode(node) || (!prompt && generationImages(node).length > 0)
    if (passThrough) return
    if (state.generating) throwWorkflowNodeExecutionError(node, '节点正在运行')
    if (!effectiveDrawingModelName(node)) throwWorkflowNodeExecutionError(node, '暂无可用生图模型')
    if (!prompt) throwWorkflowNodeExecutionError(node, '缺少提示词')
    await generateWorkflowImage(node, { undo: false, replaceCurrentNode: true, includeSelfReference: false })
    assertWorkflowNodeExecutionSucceeded(node)
    return
  }
  if (type === 'batch-recolor') {
    const state = ensureBatchRecolorNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '节点正在运行')
    if (!state.target.trim()) throwWorkflowNodeExecutionError(node, '请输入换色对象')
    if (!state.materials.length) throwWorkflowNodeExecutionError(node, '请选择材质')
    if (!batchSourceImages(node).length) throwWorkflowNodeExecutionError(node, '缺少上游图片')
    await runBatchRecolorNode(node)
    assertWorkflowNodeExecutionSucceeded(node)
    return
  }
  if (type === 'batch-translate') {
    const state = ensureBatchTranslateNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '节点正在运行')
    if (!batchTranslateInputItems(node).length) throwWorkflowNodeExecutionError(node, '缺少上游图片')
    await runBatchTranslateNode(node)
    assertWorkflowNodeExecutionSucceeded(node)
    return
  }
  if (type === 'video') {
    const state = ensureVideoNodeState(node)
    if (state.loading) throwWorkflowNodeExecutionError(node, '节点正在运行')
    if (!effectiveVideoPrompt(node)) throwWorkflowNodeExecutionError(node, '请输入视频提示词')
    await runVideoNode(node, { undo: false })
    assertWorkflowNodeExecutionSucceeded(node)
  }
}

async function runGroupWorkflow(groupId) {
  const group = groupBoxes.value.find(item => item.id === groupId)
  if (!group || groupRunState(groupId).running) return
  const previous = groupRunState(groupId)
  const groupNodes = new Set((group.nodes || []).filter(nodeExists))
  const completed = previous.paused ? previous.completedNodes.filter(node => groupNodes.has(node)) : []
  updateGroupRunState(groupId, { running: true, paused: false, currentNode: '', completedNodes: completed, errorNode: '', error: '' })
  try {
    for (const node of workflowGroupExecutionOrder(group)) {
      const state = groupRunState(groupId)
      if (state.paused) break
      if (state.completedNodes.includes(node)) continue
      updateGroupRunState(groupId, { currentNode: node })
      await executeWorkflowGroupNode(node)
      const latest = groupRunState(groupId)
      updateGroupRunState(groupId, { completedNodes: [...new Set([...latest.completedNodes, node])] })
      if (groupRunState(groupId).paused) break
    }
  } catch (e) {
    const message = e?.response?.data?.error || e?.message || '工作流运行失败'
    const failedNode = e?.workflowNode || groupRunState(groupId).currentNode || ''
    updateGroupRunState(groupId, { running: false, paused: true, currentNode: failedNode, errorNode: failedNode, error: message })
    error.value = message
  } finally {
    const state = groupRunState(groupId)
    if (state.running) {
      updateGroupRunState(groupId, state.paused
        ? { running: false, currentNode: '' }
        : { running: false, currentNode: '', completedNodes: [], errorNode: '', error: '' })
    }
  }
}

function pauseGroupWorkflow(groupId) {
  const state = groupRunState(groupId)
  if (!state.running || state.paused) return
  activeGroupId.value = groupId
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  updateGroupRunState(groupId, { paused: true })
}

const transmittingWorkflowLinkKeys = computed(() => {
  const keys = new Set()
  groupBoxes.value.forEach((group) => {
    const state = groupRunState(group.id)
    if (!state.running || !state.currentNode) return
    const nodes = new Set(group.nodes || [])
    workflowLinks.value.forEach((link) => {
      if (link.to === state.currentNode && nodes.has(link.from) && nodes.has(link.to)) {
        keys.add(workflowLinkKey(link))
      }
    })
  })
  return keys
})

function resizeGroupsForNode(node) {
  const padding = 28
  let changed = false
  const nextGroups = groupBoxes.value.map((group) => {
    if (!(group.nodes || []).includes(node)) return group
    const bounds = selectedNodesBounds(group.nodes || [])
    if (!bounds) return group
    const nextGroup = {
      ...group,
      x: bounds.x - padding,
      y: bounds.y - padding,
      width: bounds.width + padding * 2,
      height: bounds.height + padding * 2
    }
    if (
      nextGroup.x !== group.x ||
      nextGroup.y !== group.y ||
      nextGroup.width !== group.width ||
      nextGroup.height !== group.height
    ) {
      changed = true
      return nextGroup
    }
    return group
  })
  if (changed) groupBoxes.value = nextGroups
}

function draftLinkPath() {
  if (!draggingLink.value) return ''
  const start = getNodeOutputPoint(draggingLink.value.from)
  const end = draggingLink.value.to
  const gap = Math.max(80, Math.abs(end.x - start.x) * 0.45)
  return `M ${start.x} ${start.y} C ${start.x + gap} ${start.y}, ${end.x - gap} ${end.y}, ${end.x} ${end.y}`
}

function closeLinkCreateMenu() {
  linkCreateMenu.value = { open: false, from: '', x: 0, y: 0, screenX: 0, screenY: 0 }
}

function canOpenLinkCreateMenu(from) {
  return !runningWorkflowGroupForNode(from) && ['generate', 'text', 'llm', 'output'].includes(nodeType(from))
}

function openLinkCreateMenu(from, e) {
  if (!canOpenLinkCreateMenu(from)) {
    closeLinkCreateMenu()
    return
  }
  const point = clientPointToCanvasFromEvent(e)
  const screenPoint = clientPointToCanvasViewport(e)
  linkCreateMenu.value = {
    open: true,
    from,
    x: point.x,
    y: point.y,
    screenX: screenPoint.x,
    screenY: screenPoint.y
  }
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  showNodeMenu.value = false
  showStyleLibraryMenu.value = false
}

function linkCreateMenuStyle() {
  const canvas = workflowCanvasRef.value
  const menuWidth = 244
  const menuHeight = 154
  const padding = 14
  const maxX = Math.max(padding, (canvas?.clientWidth || 0) - menuWidth - padding)
  const maxY = Math.max(padding, (canvas?.clientHeight || 0) - menuHeight - padding)
  return {
    left: `${Math.round(THREE.MathUtils.clamp(linkCreateMenu.value.screenX + 12, padding, maxX))}px`,
    top: `${Math.round(THREE.MathUtils.clamp(linkCreateMenu.value.screenY + 12, padding, maxY))}px`
  }
}

function createLinkedNodeFromMenu(type) {
  const menu = linkCreateMenu.value
  if (!menu.open || !menu.from) return
  const base = DEFAULT_NODE_POSITIONS[type] || DEFAULT_NODE_POSITIONS.generate
  const sourceGroup = groupBoxes.value.find(group => (group.nodes || []).includes(menu.from))
  const nodeId = addWorkflowNode(type, {
    position: {
      x: Math.round(menu.x - (base.width || 0) / 2),
      y: Math.round(menu.y - (base.height || 0) / 2)
    },
    groupId: sourceGroup?.id || '',
    persist: false
  })
  if (nodeId) connectWorkflowNodes(menu.from, nodeId, { undo: false, persist: false })
  closeLinkCreateMenu()
  saveWorkflowStructureState()
}

function autoArrangeWorkflow() {
  const ids = workflowNodeIds()
  if (!ids.length) return
  pushWorkflowUndoSnapshot()
  closeLinkCreateMenu()
  showNodeMenu.value = false
  showStyleLibraryMenu.value = false
  showWorkflowTemplateMenu.value = false

  const idSet = new Set(ids)
  const incoming = Object.fromEntries(ids.map(id => [id, []]))
  const outgoing = Object.fromEntries(ids.map(id => [id, []]))
  workflowLinks.value.forEach((link) => {
    if (!idSet.has(link.from) || !idSet.has(link.to)) return
    incoming[link.to].push(link.from)
    outgoing[link.from].push(link.to)
  })

  const startX = 220
  const startY = 160
  const columnGap = 120
  const rowGap = 92
  const nextPositions = { ...nodePositions.value }
  const placed = new Set()
  let currentY = startY

  const nodeSort = (a, b) => (nodePositions.value[a]?.y || 0) - (nodePositions.value[b]?.y || 0)

  const nodeRectForLayout = (node) => (
    computedNodeRect(node) ||
    nodePositions.value[node] ||
    DEFAULT_NODE_POSITIONS[nodeType(node)] ||
    DEFAULT_NODE_POSITIONS.generate
  )
  const nodeHeight = (node) => nodeRectForLayout(node)?.height || DEFAULT_NODE_POSITIONS.generate.height
  const nodeWidth = (node) => nodeRectForLayout(node)?.width || DEFAULT_NODE_POSITIONS.generate.width
  const stackHeight = (nodes, gap = rowGap) => {
    const validNodes = nodes.filter(nodeExists)
    if (!validNodes.length) return 0
    return validNodes.reduce((sum, node) => sum + nodeHeight(node), 0) + Math.max(0, validNodes.length - 1) * gap
  }
  const placeNode = (node, x, y) => {
    if (!nodeExists(node)) return
    const base = nextPositions[node] || nodePositions.value[node] || DEFAULT_NODE_POSITIONS[nodeType(node)] || DEFAULT_NODE_POSITIONS.generate
    nextPositions[node] = {
      ...base,
      x: Math.round(x),
      y: Math.round(y)
    }
    placed.add(node)
  }
  const placeStack = (nodes, x, centerY, gap = rowGap) => {
    const validNodes = nodes.filter(nodeExists)
    let y = centerY - stackHeight(validNodes, gap) / 2
    validNodes.forEach((node) => {
      placeNode(node, x, y)
      y += nodeHeight(node) + gap
    })
  }
  const generationResultImage = (node) => {
    if (nodeType(node) !== 'generate') return ''
    return normalizeGenerationResult(ensureGenerateNodeState(node).result).images[0] || ''
  }
  const batchResultOrder = (node) => {
    if (nodeType(node) !== 'generate') return 0
    const item = normalizeGenerationResult(ensureGenerateNodeState(node).result).batchItems?.[0] || {}
    return Number(item.index ?? item.sourceIndex ?? 0) || 0
  }
  const batchResultNodes = (parentNode, type) => (
    (outgoing[parentNode] || [])
      .filter(target => isBatchResultNode(target, type))
      .sort((a, b) => batchResultOrder(a) - batchResultOrder(b) || nodeSort(a, b))
  )
  const linkedBatchTranslateNodes = (recolorNode, recolorResults) => uniqueValues([
    ...(outgoing[recolorNode] || []),
    ...recolorResults.flatMap(node => outgoing[node] || [])
  ]).filter(node => nodeType(node) === 'batch-translate')
  const groupTranslateResultsBySource = (translateResults, recolorResults) => {
    const imageToRecolorNode = new Map()
    const materialToRecolorNode = new Map()
    recolorResults.forEach((node) => {
      const image = generationResultImage(node)
      if (image) imageToRecolorNode.set(image, node)
      const item = normalizeGenerationResult(ensureGenerateNodeState(node).result).batchItems?.[0] || {}
      if (item.materialId) materialToRecolorNode.set(String(item.materialId), node)
    })
    const grouped = new Map(recolorResults.map(node => [node, []]))
    const unmatched = []
    translateResults.forEach((node) => {
      const item = normalizeGenerationResult(ensureGenerateNodeState(node).result).batchItems?.[0] || {}
      const source = resolveImageUrl(item.sourceImageUrl || item.source_image_url || '')
      const sourceNode = imageToRecolorNode.get(source) || materialToRecolorNode.get(String(item.materialId || ''))
      if (sourceNode) grouped.get(sourceNode).push(node)
      else unmatched.push(node)
    })
    grouped.forEach(nodes => nodes.sort((a, b) => batchResultOrder(a) - batchResultOrder(b) || nodeSort(a, b)))
    unmatched.sort((a, b) => batchResultOrder(a) - batchResultOrder(b) || nodeSort(a, b))
    return { grouped, unmatched }
  }

  function placeBatchRecolorFlow(recolorNode) {
    if (placed.has(recolorNode)) return false
    const sourceNodes = (incoming[recolorNode] || [])
      .filter(node => nodeType(node) === 'generate')
      .sort(nodeSort)
    const recolorResults = batchResultNodes(recolorNode, 'batch-recolor')
    const translateNodes = linkedBatchTranslateNodes(recolorNode, recolorResults)
      .filter(nodeExists)
      .sort(nodeSort)
    const translateResults = translateNodes.flatMap(node => batchResultNodes(node, 'batch-translate'))
    const { grouped, unmatched } = groupTranslateResultsBySource(translateResults, recolorResults)
    const laneGap = GENERATED_RESULT_NODE_GAP_Y
    const lanes = recolorResults.length
      ? recolorResults.map((node) => {
          const resultNodes = grouped.get(node) || []
          return {
            recolorResult: node,
            translateResults: resultNodes,
            height: Math.max(nodeHeight(node), stackHeight(resultNodes, laneGap), DEFAULT_NODE_POSITIONS.generate.height)
          }
        })
      : []
    unmatched.forEach((node) => {
      lanes.push({
        recolorResult: '',
        translateResults: [node],
        height: Math.max(nodeHeight(node), DEFAULT_NODE_POSITIONS.generate.height)
      })
    })

    const laneHeight = lanes.length
      ? lanes.reduce((sum, lane) => sum + lane.height, 0) + Math.max(0, lanes.length - 1) * laneGap
      : 0
    const sourceStackHeight = stackHeight(sourceNodes)
    const translateStackHeight = stackHeight(translateNodes)
    const coreHeight = Math.max(
      laneHeight,
      sourceStackHeight,
      nodeHeight(recolorNode),
      translateStackHeight,
      DEFAULT_NODE_POSITIONS.generate.height
    )
    const centerY = currentY + coreHeight / 2
    const inputX = startX
    const sourceColumnWidth = sourceNodes.length ? Math.max(...sourceNodes.map(nodeWidth)) : 0
    const recolorX = inputX + (sourceNodes.length ? sourceColumnWidth + columnGap : 0)
    const recolorResultX = recolorX + nodeWidth(recolorNode) + columnGap
    const recolorResultColumnWidth = recolorResults.length ? Math.max(...recolorResults.map(nodeWidth)) : 0
    const translateX = recolorX + nodeWidth(recolorNode) + columnGap + (recolorResults.length ? recolorResultColumnWidth + columnGap : 0)
    const translateColumnWidth = translateNodes.length ? Math.max(...translateNodes.map(nodeWidth)) : DEFAULT_NODE_POSITIONS['batch-translate'].width
    const translateResultX = translateX + translateColumnWidth + columnGap

    placeStack(sourceNodes, inputX, centerY)
    placeNode(recolorNode, recolorX, centerY - nodeHeight(recolorNode) / 2)
    placeStack(translateNodes, translateX, centerY)

    if (lanes.length) {
      let laneY = currentY
      lanes.forEach((lane) => {
        const laneCenter = laneY + lane.height / 2
        if (lane.recolorResult) placeNode(lane.recolorResult, recolorResultX, laneCenter - nodeHeight(lane.recolorResult) / 2)
        placeStack(lane.translateResults, translateResultX, laneCenter, laneGap)
        laneY += lane.height + laneGap
      })
    }

    currentY += coreHeight + 160
    return true
  }

  function placeBatchTranslateFlow(translateNode) {
    if (placed.has(translateNode)) return false
    const sourceNodes = (incoming[translateNode] || [])
      .filter(node => nodeType(node) === 'generate')
      .sort(nodeSort)
    const translateResults = batchResultNodes(translateNode, 'batch-translate')
    const sourceStackHeight = stackHeight(sourceNodes)
    const resultStackHeight = stackHeight(translateResults, GENERATED_RESULT_NODE_GAP_Y)
    const coreHeight = Math.max(sourceStackHeight, nodeHeight(translateNode), resultStackHeight, DEFAULT_NODE_POSITIONS.generate.height)
    const centerY = currentY + coreHeight / 2
    const sourceX = startX
    const sourceColumnWidth = sourceNodes.length ? Math.max(...sourceNodes.map(nodeWidth)) : 0
    const translateX = sourceX + (sourceNodes.length ? sourceColumnWidth + columnGap : 0)
    const resultX = translateX + nodeWidth(translateNode) + columnGap

    placeStack(sourceNodes.filter(node => !placed.has(node)), sourceX, centerY)
    placeNode(translateNode, translateX, centerY - nodeHeight(translateNode) / 2)
    placeStack(translateResults, resultX, centerY, GENERATED_RESULT_NODE_GAP_Y)
    currentY += coreHeight + 160
    return true
  }

  ids
    .filter(id => nodeType(id) === 'batch-recolor')
    .sort(nodeSort)
    .forEach(placeBatchRecolorFlow)
  ids
    .filter(id => nodeType(id) === 'batch-translate' && !placed.has(id))
    .sort(nodeSort)
    .forEach(placeBatchTranslateFlow)

  const starters = ids
    .filter(id => !placed.has(id))
    .filter(id => incoming[id].length === 0 && outgoing[id].length > 0)
    .sort(nodeSort)
  const linkedFallback = ids
    .filter(id => !placed.has(id))
    .filter(id => (incoming[id].length || outgoing[id].length) && !starters.includes(id))
    .sort(nodeSort)

  function placeChain(start, rowY) {
    let x = startX
    let node = start
    const seenInChain = new Set()
    while (node && !seenInChain.has(node) && idSet.has(node)) {
      seenInChain.add(node)
      if (!placed.has(node)) {
        const rect = computedNodeRect(node) || nodePositions.value[node]
        if (!rect) break
        nextPositions[node] = {
          ...nodePositions.value[node],
          x,
          y: rowY
        }
        placed.add(node)
        x += rect.width + columnGap
      } else {
        const rect = computedNodeRect(node) || nodePositions.value[node]
        x = Math.max(x, (nextPositions[node]?.x || x) + (rect?.width || 360) + columnGap)
      }
      node = (outgoing[node] || []).filter(target => !placed.has(target)).sort(nodeSort)[0] || ''
    }
    return Math.max(0, x - startX - columnGap)
  }

  function placeLinkedFlow(start) {
    if (placed.has(start)) return
    const width = placeChain(start, currentY)
    const flowNodes = ids.filter(id => placed.has(id) && (nextPositions[id]?.y === currentY))
    const flowHeight = Math.max(...flowNodes.map(id => computedNodeRect(id)?.height || nodePositions.value[id]?.height || 260), 260)
    let branchY = currentY + flowHeight + rowGap
    flowNodes.forEach((node) => {
      ;(outgoing[node] || [])
        .filter(target => !placed.has(target))
        .sort(nodeSort)
        .forEach((target) => {
          const anchor = nextPositions[node] || nodePositions.value[node]
          const rect = computedNodeRect(node) || nodePositions.value[node]
          const targetRect = computedNodeRect(target) || nodePositions.value[target]
          nextPositions[target] = {
            ...nodePositions.value[target],
            x: (anchor?.x || startX) + (rect?.width || 360) + columnGap,
            y: branchY
          }
          placed.add(target)
          placeChain(target, branchY)
          branchY += Math.max(targetRect?.height || 260, 260) + rowGap
        })
    })
    currentY = Math.max(branchY, currentY + flowHeight + rowGap)
    return width
  }

  ;[...starters, ...linkedFallback].forEach(placeLinkedFlow)
  ids
    .filter(id => !placed.has(id))
    .sort(nodeSort)
    .forEach((node) => {
      const rect = computedNodeRect(node) || nodePositions.value[node]
      if (!rect) return
      nextPositions[node] = {
        ...nodePositions.value[node],
        x: startX,
        y: currentY
      }
      placed.add(node)
      currentY += rect.height + rowGap
    })

  nodePositions.value = nextPositions
  groupBoxes.value.forEach(group => (group.nodes || []).forEach(resizeGroupsForNode))
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  bumpViewportCheck()
  nextTick(focusCanvasOnNodes)
  saveActiveProject()
}

function workflowTemplateOrigin() {
  const canvas = workflowCanvasRef.value
  const scale = liveCanvasView.scale || 1
  if (!canvas) return { x: 260, y: 180 }
  return {
    x: Math.round((canvas.clientWidth / 2 - liveCanvasView.x) / scale - 620),
    y: Math.round((canvas.clientHeight / 2 - liveCanvasView.y) / scale - 210)
  }
}

function createTemplateNode(type, position, state = {}) {
  return addWorkflowNode(type, {
    position,
    state,
    persist: false,
    undo: false
  })
}

function ensureTemplateStaticNode(type, position) {
  if (workflowNodes.value[type]) {
    nodePositions.value = {
      ...nodePositions.value,
      [type]: {
        ...(nodePositions.value[type] || DEFAULT_NODE_POSITIONS[type]),
        ...position
      }
    }
    return type
  }
  const id = addWorkflowNode(type, {
    position,
    persist: false,
    undo: false
  })
  if (id) return id
  workflowNodes.value = {
    ...workflowNodes.value,
    [type]: true
  }
  nodePositions.value = {
    ...nodePositions.value,
    [type]: {
      ...(nodePositions.value[type] || DEFAULT_NODE_POSITIONS[type]),
      ...position
    }
  }
  return type
}

function applyWorkflowTemplate(template) {
  pushWorkflowUndoSnapshot()
  showWorkflowTemplateMenu.value = false
  showNodeMenu.value = false
  showStyleLibraryMenu.value = false
  closeLinkCreateMenu()
  const origin = workflowTemplateOrigin()
  const createdNodes = []

  if (template === 'model-image') {
    const upload = ensureTemplateStaticNode('upload', { x: origin.x, y: origin.y })
    const output = ensureTemplateStaticNode('output', { x: origin.x + 540, y: origin.y + 18 })
    const generate = createTemplateNode('generate', { x: origin.x + 1040, y: origin.y + 38 })
    createdNodes.push(upload, output, generate)
    connectWorkflowNodes(upload, output, { persist: false, undo: false })
    connectWorkflowNodes(output, generate, { persist: false, undo: false })
  }

  if (template === 'image-prompt') {
    const sourceImage = createTemplateNode('generate', {
      x: origin.x,
      y: origin.y + 22
    }, {
      uploads: [],
      uploadOnly: true,
      showResults: false
    })
    const generate = createTemplateNode('generate', {
      x: origin.x + 620,
      y: origin.y + 22
    })
    createdNodes.push(sourceImage, generate)
    connectWorkflowNodes(sourceImage, generate, { persist: false, undo: false })
  }

  const validNodes = createdNodes.filter(Boolean)
  if (validNodes.length > 1) createWorkflowGroup(validNodes, { activate: false, persist: false })
  activeWorkflowNode.value = validNodes[0] || ''
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  bumpViewportCheck()
  nextTick(focusCanvasOnNodes)
  saveActiveProject()
}

function hasBatchRecolorInputLock(node, exceptTo = '') {
  if (nodeType(node) !== 'generate') return false
  return workflowLinks.value.some(link => (
    link.from === node &&
    link.to !== exceptTo &&
    nodeType(link.to) === 'batch-recolor'
  ))
}

function hasAnyGenerateDownstream(node, exceptTo = '') {
  if (nodeType(node) !== 'generate') return false
  return workflowLinks.value.some(link => link.from === node && link.to !== exceptTo)
}

function violatesBatchRecolorInputLock(from, to) {
  if (nodeType(from) !== 'generate') return false
  if (nodeType(to) === 'batch-recolor') return hasAnyGenerateDownstream(from, to)
  return hasBatchRecolorInputLock(from, to)
}

function violatesVideoInputRule(from, to) {
  if (nodeType(to) !== 'video') return false
  const sourceType = nodeType(from)
  if (['text', 'llm'].includes(sourceType)) return false
  if (sourceType !== 'generate') return true
  return workflowLinks.value.some(link => (
    link.to === to &&
    link.from !== from &&
    nodeType(link.from) === 'generate'
  ))
}

function isWorkflowLinkCandidate(from, to) {
  if (!from || !to || from === to || !nodeExists(from) || !nodeExists(to)) return false
  if (runningWorkflowGroupForNode(from) || runningWorkflowGroupForNode(to)) return false
  if (isReferenceOnlyGenerateNode(to) || isUploadOnlyGenerateNode(to)) return false
  if (violatesBatchRecolorInputLock(from, to) || violatesVideoInputRule(from, to)) return false
  return !workflowLinks.value.some(link => link.from === from && link.to === to)
}

function connectWorkflowNodes(from, to, options = {}) {
  if (!from || !to || from === to) return
  if (!nodeExists(from) || !nodeExists(to)) return
  if (options.undo !== false && (runningWorkflowGroupForNode(from) || runningWorkflowGroupForNode(to))) {
    error.value = '工作流运行中，暂时不能修改连线'
    return
  }
  if (options.undo !== false) invalidatePausedWorkflowRunsForNodes([from, to], { structural: true })
  if (isReferenceOnlyGenerateNode(to)) return
  if (isUploadOnlyGenerateNode(to)) return
  if (violatesBatchRecolorInputLock(from, to)) return
  if (violatesVideoInputRule(from, to)) return
  const exists = workflowLinks.value.some(link => link.from === from && link.to === to)
  if (exists) return
  if (options.undo !== false) pushWorkflowUndoSnapshot()
  markTemplateWorkflowGroupChangedForNode(from)
  markTemplateWorkflowGroupChangedForNode(to)
  activeWorkflowLink.value = ''
  workflowLinks.value = [
    ...workflowLinks.value,
    { from, to }
  ]
  if (options.persist !== false) saveActiveProject()
}

function linkDropTargetNode(e) {
  const targetElement = document.elementFromPoint(e.clientX, e.clientY)
  const targetPort = targetElement?.closest?.('.node-port[data-port="in"]')
  return targetPort?.dataset?.node || targetElement?.closest?.('.canvas-node')?.querySelector?.('.node-port[data-port="in"]')?.dataset?.node || ''
}

function workflowLinkKey(link) {
  return `${link.from}->${link.to}`
}

function selectWorkflowLink(link) {
  closeLinkCreateMenu()
  hoveredLinkTarget.value = ''
  activeWorkflowLink.value = workflowLinkKey(link)
  activeWorkflowNode.value = ''
  activeGroupId.value = ''
  closeAllGenerateMenus()
}

function removeWorkflowLink(linkKey) {
  const removedLink = workflowLinks.value.find(link => workflowLinkKey(link) === linkKey)
  if (removedLink && (runningWorkflowGroupForNode(removedLink.from) || runningWorkflowGroupForNode(removedLink.to))) {
    error.value = '工作流运行中，暂时不能删除连线'
    return
  }
  if (removedLink) invalidatePausedWorkflowRunsForNodes([removedLink.from, removedLink.to], { structural: true })
  pushWorkflowUndoSnapshot()
  if (removedLink) {
    markTemplateWorkflowGroupChangedForNode(removedLink.from)
    markTemplateWorkflowGroupChangedForNode(removedLink.to)
  }
  workflowLinks.value = workflowLinks.value.filter(link => workflowLinkKey(link) !== linkKey)
  if (activeWorkflowLink.value === linkKey) activeWorkflowLink.value = ''
  saveActiveProject()
}

function startLinkDrag(e, node) {
  activeWorkflowLink.value = ''
  closeLinkCreateMenu()
  hoveredLinkTarget.value = ''
  draggingLink.value = {
    from: node,
    pointerId: e.pointerId,
    to: clientPointToCanvas(e)
  }
  pendingLinkFrom.value = node
  window.addEventListener('pointermove', dragLink)
  window.addEventListener('pointerup', endLinkDrag, { once: true })
  window.addEventListener('pointercancel', cancelLinkDrag, { once: true })
}

function dragLink(e) {
  if (!draggingLink.value || draggingLink.value.pointerId !== e.pointerId) return
  toRaw(draggingLink.value).to = clientPointToCanvas(e)
  const targetNode = linkDropTargetNode(e)
  hoveredLinkTarget.value = isWorkflowLinkCandidate(draggingLink.value.from, targetNode) ? targetNode : ''
  draftLinkPathRef.value?.setAttribute('d', draftLinkPath())
}

function endLinkDrag(e) {
  if (!draggingLink.value || draggingLink.value.pointerId !== e.pointerId) return
  const source = draggingLink.value.from
  const targetNode = linkDropTargetNode(e)
  window.removeEventListener('pointermove', dragLink)
  window.removeEventListener('pointercancel', cancelLinkDrag)
  draggingLink.value = null
  pendingLinkFrom.value = ''
  hoveredLinkTarget.value = ''
  if (targetNode) {
    connectWorkflowNodes(source, targetNode)
    return
  }
  openLinkCreateMenu(source, e)
}

function cancelLinkDrag(e) {
  if (!draggingLink.value || draggingLink.value.pointerId !== e.pointerId) return
  window.removeEventListener('pointermove', dragLink)
  window.removeEventListener('pointerup', endLinkDrag)
  draggingLink.value = null
  pendingLinkFrom.value = ''
  hoveredLinkTarget.value = ''
}

function handleNodePortClick(node, portType) {
  if (portType === 'out') {
    pendingLinkFrom.value = pendingLinkFrom.value === node ? '' : node
    hoveredLinkTarget.value = ''
    return
  }

  if (!pendingLinkFrom.value) return
  if (pendingLinkFrom.value === node) {
    pendingLinkFrom.value = ''
    return
  }

  connectWorkflowNodes(pendingLinkFrom.value, node)
  pendingLinkFrom.value = ''
  hoveredLinkTarget.value = ''
}

function resetCanvasView() {
  applyCanvasTransform({ x: 0, y: 0, scale: 1 })
  syncCanvasViewFromLive()
}

function toggleNodeMenu() {
  closeLinkCreateMenu()
  showWorkflowTemplateMenu.value = false
  showNodeMenu.value = !showNodeMenu.value
  if (!showNodeMenu.value) showStyleLibraryMenu.value = false
}

function toggleWorkflowTemplateMenu() {
  showNodeMenu.value = false
  showStyleLibraryMenu.value = false
  showShortcutModal.value = false
  closeLinkCreateMenu()
  showWorkflowTemplateMenu.value = !showWorkflowTemplateMenu.value
}

function openShortcutModal() {
  showNodeMenu.value = false
  showStyleLibraryMenu.value = false
  showWorkflowTemplateMenu.value = false
  closeLinkCreateMenu()
  showShortcutModal.value = true
}

function closeShortcutModal() {
  showShortcutModal.value = false
}

function openWorkflowHelp() {
  const url = WORKFLOW_HELP_URL.trim()
  if (!url) return
  closeShortcutModal()
  window.open(url, '_blank', 'noopener,noreferrer')
}

function closeNodeMenuOnOutside(e) {
  if (!showNodeMenu.value && !showWorkflowTemplateMenu.value) return
  if (e.target.closest?.('.canvas-toolbar')) return
  showNodeMenu.value = false
  showStyleLibraryMenu.value = false
  showWorkflowTemplateMenu.value = false
}

function startCanvasPan(e) {
  updateLastCanvasPointer(e)
  if (
    e.button !== 0 ||
    draggingLink.value ||
    e.target.closest?.('.canvas-node') ||
    e.target.closest?.('.node-selection-box') ||
    e.target.closest?.('.node-group-box') ||
    e.target.closest?.('.canvas-toolbar')
  ) return
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  closeAllGenerateMenus()
  showNodeMenu.value = false
  closeLinkCreateMenu()
  pendingLinkFrom.value = ''
  if (!isSpacePressed) {
    markCanvasInteraction()
    selectionBox.value = { active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 }
    selectedNodeIds.value = []
    canvasPan = {
      pointerId: e.pointerId,
      startX: e.clientX,
      startY: e.clientY,
      originX: liveCanvasView.x,
      originY: liveCanvasView.y
    }
    canvasPanActive.value = true
    e.currentTarget.setPointerCapture?.(e.pointerId)
    return
  }
  const point = clientPointToCanvasFromEvent(e)
  selectionDrag = {
    pointerId: e.pointerId,
    startX: point.x,
    startY: point.y
  }
  selectionBox.value = {
    active: true,
    fixed: false,
    x: point.x,
    y: point.y,
    width: 0,
    height: 0,
    startX: point.x,
    startY: point.y
  }
  selectedNodeIds.value = []
  e.currentTarget.setPointerCapture?.(e.pointerId)
}

function onCanvasPointerMove(e) {
  updateLastCanvasPointer(e)
  if (draggingLink.value) {
    return
  }
  if (selectionDrag?.pointerId === e.pointerId) {
    dragSelectionBox(e)
    return
  }
  dragCanvasPan(e)
}

function onCanvasPointerUp(e) {
  if (draggingLink.value) {
    endLinkDrag(e)
    return
  }
  if (selectionDrag?.pointerId === e.pointerId) {
    endSelectionBox(e)
    return
  }
  endCanvasPan(e)
}

function onCanvasPointerCancel(e) {
  if (draggingLink.value) {
    cancelLinkDrag(e)
    return
  }
  if (selectionDrag?.pointerId === e.pointerId) {
    endSelectionBox(e)
    return
  }
  endCanvasPan(e)
}

function cancelCanvasPanFrame() {
  if (canvasPanFrame) window.cancelAnimationFrame(canvasPanFrame)
  canvasPanFrame = null
  pendingCanvasPanEvent = null
}

function dragSelectionBox(e) {
  if (!selectionDrag || selectionDrag.pointerId !== e.pointerId) return
  markCanvasInteraction()
  const point = clientPointToCanvasFromEvent(e)
  selectionBox.value = {
    ...selectionBox.value,
    width: point.x - selectionDrag.startX,
    height: point.y - selectionDrag.startY
  }
  updateSelectedNodesFromBox()
}

function endSelectionBox(e) {
  if (!selectionDrag || selectionDrag.pointerId !== e.pointerId) return
  e.currentTarget.releasePointerCapture?.(e.pointerId)
  selectionDrag = null
  if (selectedNodeIds.value.length) {
    snapSelectionBoxToNodes()
    activeWorkflowNode.value = ''
    activeWorkflowLink.value = ''
    activeGroupId.value = ''
  } else {
    selectionBox.value.fixed = false
    selectionBox.value.active = false
  }
}

function startSelectionMove(e) {
  if (!selectionBox.value.fixed || !selectedNodeIds.value.length) return
  const point = clientPointToCanvasFromEvent(e)
  selectionMove = {
    pointerId: e.pointerId,
    startX: point.x,
    startY: point.y,
    box: normalizeRect(selectionBox.value),
    positions: Object.fromEntries(selectedNodeIds.value.map(node => [node, { ...(nodePositions.value[node] || {}) }]))
  }
  e.currentTarget.setPointerCapture?.(e.pointerId)
  window.addEventListener('pointermove', dragSelectionMove)
  window.addEventListener('pointerup', endSelectionMove)
}

function dragSelectionMove(e) {
  if (!selectionMove || selectionMove.pointerId !== e.pointerId) return
  markCanvasInteraction()
  pendingSelectionMoveEvent = e
  if (selectionMoveFrame) return
  selectionMoveFrame = window.requestAnimationFrame(() => {
    selectionMoveFrame = null
    flushSelectionMove()
  })
}

function flushSelectionMove() {
  if (!selectionMove || !pendingSelectionMoveEvent) return
  const point = clientPointToCanvasFromEvent(pendingSelectionMoveEvent)
  pendingSelectionMoveEvent = null
  const dx = point.x - selectionMove.startX
  const dy = point.y - selectionMove.startY
  selectedNodeIds.value.forEach((node) => {
    const start = selectionMove.positions[node]
    const pos = nodePositions.value[node]
    if (!start || !pos) return
    pos.x = start.x + dx
    pos.y = start.y + dy
  })
  snapSelectionBoxToNodes()
}

function endSelectionMove(e) {
  if (!selectionMove || selectionMove.pointerId !== e.pointerId) return
  window.removeEventListener('pointermove', dragSelectionMove)
  window.removeEventListener('pointerup', endSelectionMove)
  if (selectionMoveFrame) {
    window.cancelAnimationFrame(selectionMoveFrame)
    selectionMoveFrame = null
  }
  flushSelectionMove()
  selectionMove = null
  bumpViewportCheck()
  scheduleNodeLayoutStateSave()
}

function groupSelectedNodes() {
  if (selectedNodeIds.value.length < 2) return
  createWorkflowGroup(selectedNodeIds.value, { persist: false })
  selectionBox.value = { active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 }
  selectedNodeIds.value = []
  saveActiveProject()
}

function renderedWorkflowNodeElements(nodes = []) {
  const included = new Set(nodes)
  return new Map(Array.from(workflowStageRef.value?.querySelectorAll?.('[data-workflow-node]') || [])
    .filter(element => included.has(element.dataset.workflowNode))
    .map(element => [element.dataset.workflowNode, element]))
}

function dragLinkSnapshots(nodes = []) {
  const included = new Set(nodes)
  const renderedLinkElements = workflowStageRef.value?.querySelectorAll?.('.node-link') || []
  return workflowLinkViews.value.map((link, index) => ({
    link,
    element: renderedLinkElements[index] || null,
    paths: Array.from(renderedLinkElements[index]?.querySelectorAll?.('path') || []),
    start: computedNodeRect(link.from),
    end: computedNodeRect(link.to)
  })).filter(item => (
    item.element && item.start && item.end && (included.has(item.link.from) || included.has(item.link.to))
  ))
}

function updateDraggedLinkPaths(linkSnapshots, movedNodes, dx, dy) {
  linkSnapshots.forEach(({ link, paths, start, end }) => {
    const movedStart = movedNodes.has(link.from) ? { ...start, x: start.x + dx, y: start.y + dy } : start
    const movedEnd = movedNodes.has(link.to) ? { ...end, x: end.x + dx, y: end.y + dy } : end
    const path = nodeLinkPathForRects(movedStart, movedEnd)
    paths.forEach(item => item.setAttribute('d', path))
  })
}

function startGroupDrag(e, groupId) {
  const group = groupBoxes.value.find(item => item.id === groupId)
  if (!group) return
  activeGroupId.value = groupId
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  selectionBox.value = { active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 }
  selectedNodeIds.value = []
  if (groupRunState(groupId).running) return
  const point = clientPointToCanvasFromEvent(e)
  const nodes = (group.nodes || []).filter(nodeExists)
  const nodeElements = renderedWorkflowNodeElements(nodes)
  nodeElements.forEach(element => element.classList.add('is-group-dragging'))
  e.currentTarget.classList.add('is-dragging')
  groupDrag = {
    pointerId: e.pointerId,
    groupId,
    startX: point.x,
    startY: point.y,
    group: { ...group },
    nodes,
    positions: Object.fromEntries(nodes.map(node => [node, { ...(nodePositions.value[node] || {}) }])),
    element: e.currentTarget,
    nodeElements,
    linkSnapshots: dragLinkSnapshots(nodes),
    movedNodes: new Set(nodes),
    dx: 0,
    dy: 0
  }
  e.currentTarget.setPointerCapture?.(e.pointerId)
  window.addEventListener('pointermove', dragGroupBox)
  window.addEventListener('pointerup', endGroupDrag)
  window.addEventListener('pointercancel', endGroupDrag)
}

function dragGroupBox(e) {
  if (!groupDrag || groupDrag.pointerId !== e.pointerId) return
  markCanvasInteraction()
  pendingGroupDragEvent = e
  if (groupDragFrame) return
  groupDragFrame = window.requestAnimationFrame(() => {
    groupDragFrame = null
    flushGroupDrag()
  })
}

function flushGroupDrag() {
  if (!groupDrag || !pendingGroupDragEvent) return
  const point = clientPointToCanvasFromEvent(pendingGroupDragEvent)
  pendingGroupDragEvent = null
  const dx = point.x - groupDrag.startX
  const dy = point.y - groupDrag.startY
  groupDrag.dx = dx
  groupDrag.dy = dy
  if (groupDrag.nodeElements.size < groupDrag.nodes.length) {
    const rendered = renderedWorkflowNodeElements(groupDrag.nodes)
    rendered.forEach((element, node) => {
      if (groupDrag.nodeElements.has(node)) return
      element.classList.add('is-group-dragging')
      groupDrag.nodeElements.set(node, element)
    })
  }
  if (groupDrag.element) groupDrag.element.style.transform = `translate3d(${dx}px, ${dy}px, 0)`
  groupDrag.nodeElements.forEach(element => {
    element.style.transform = `translate3d(${dx}px, ${dy}px, 0)`
  })
  updateDraggedLinkPaths(groupDrag.linkSnapshots, groupDrag.movedNodes, dx, dy)
}

function endGroupDrag(e) {
  if (!groupDrag || groupDrag.pointerId !== e.pointerId) return
  if (groupDrag.element?.hasPointerCapture?.(e.pointerId)) groupDrag.element.releasePointerCapture(e.pointerId)
  window.removeEventListener('pointermove', dragGroupBox)
  window.removeEventListener('pointerup', endGroupDrag)
  window.removeEventListener('pointercancel', endGroupDrag)
  if (groupDragFrame) {
    window.cancelAnimationFrame(groupDragFrame)
    groupDragFrame = null
  }
  flushGroupDrag()
  const finishedDrag = groupDrag
  const nextPositions = { ...nodePositions.value }
  finishedDrag.nodes.forEach((node) => {
    const start = finishedDrag.positions[node]
    if (!start) return
    nextPositions[node] = { ...start, x: start.x + finishedDrag.dx, y: start.y + finishedDrag.dy }
    const element = finishedDrag.nodeElements.get(node)
    if (element) {
      element.style.left = `${nextPositions[node].x}px`
      element.style.top = `${nextPositions[node].y}px`
      element.style.transform = ''
      element.classList.remove('is-group-dragging')
    }
  })
  nodePositions.value = nextPositions
  groupBoxes.value = groupBoxes.value.map(group => group.id === finishedDrag.groupId
    ? { ...group, x: finishedDrag.group.x + finishedDrag.dx, y: finishedDrag.group.y + finishedDrag.dy }
    : group)
  if (finishedDrag.element) {
    finishedDrag.element.style.left = `${finishedDrag.group.x + finishedDrag.dx}px`
    finishedDrag.element.style.top = `${finishedDrag.group.y + finishedDrag.dy}px`
    finishedDrag.element.style.transform = ''
    finishedDrag.element.classList.remove('is-dragging')
  }
  groupDrag = null
  bumpViewportCheck()
  scheduleNodeLayoutStateSave({ includeGroups: true })
}

function removeWorkflowGroup(groupId) {
  if (!groupId) return
  if (groupRunState(groupId).running) return
  pushWorkflowUndoSnapshot()
  groupBoxes.value = groupBoxes.value.filter(group => group.id !== groupId)
  clearWorkflowGroupRunState(groupId)
  if (activeGroupId.value === groupId) activeGroupId.value = ''
  saveActiveProject()
}

function pruneWorkflowGroupsAfterNodeRemoval() {
  const beforeActiveGroup = activeGroupId.value
  const beforeGroupIds = new Set(groupBoxes.value.map(group => group.id))
  groupBoxes.value = groupBoxes.value
    .map(group => ({
      ...group,
      nodes: (group.nodes || []).filter(nodeExists)
    }))
    .filter(group => group.nodes.length > 1 || groupRunState(group.id).running)
  beforeGroupIds.forEach((groupId) => {
    if (!groupBoxes.value.some(group => group.id === groupId)) clearWorkflowGroupRunState(groupId)
  })
  if (beforeActiveGroup && !groupBoxes.value.some(group => group.id === beforeActiveGroup)) {
    activeGroupId.value = ''
  }
}

function removeWorkflowGroupWithNodes(groupId) {
  const group = groupBoxes.value.find(item => item.id === groupId)
  if (!group || groupRunState(groupId).running) return
  const nodesToRemove = new Set((group.nodes || []).filter(nodeExists))
  if (!nodesToRemove.size) {
    removeWorkflowGroup(groupId)
    return
  }
  removeWorkflowNodes([...nodesToRemove], { persist: false })
  groupBoxes.value = groupBoxes.value
    .map(item => ({
      ...item,
      nodes: (item.nodes || []).filter(node => !nodesToRemove.has(node))
    }))
    .filter(item => item.id !== groupId && item.nodes.length > 1)
  clearWorkflowGroupRunState(groupId)
  activeGroupId.value = ''
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  bumpViewportCheck()
  saveActiveProject()
}

function ungroupNodes(groupId) {
  if (groupRunState(groupId).running) return
  groupBoxes.value = groupBoxes.value.filter(group => group.id !== groupId)
  clearWorkflowGroupRunState(groupId)
  if (activeGroupId.value === groupId) activeGroupId.value = ''
  saveActiveProject()
}

function dragCanvasPan(e) {
  if (!canvasPan || canvasPan.pointerId !== e.pointerId) return
  markCanvasInteraction()
  pendingCanvasPanEvent = { clientX: e.clientX, clientY: e.clientY }
  if (canvasPanFrame) return
  canvasPanFrame = window.requestAnimationFrame(() => {
    canvasPanFrame = null
    if (!canvasPan || !pendingCanvasPanEvent) return
    const event = pendingCanvasPanEvent
    pendingCanvasPanEvent = null
    applyCanvasTransform({
      ...liveCanvasView,
      x: canvasPan.originX + event.clientX - canvasPan.startX,
      y: canvasPan.originY + event.clientY - canvasPan.startY
    })
  })
}

function flushCanvasPanFrame() {
  if (!canvasPanFrame) return
  window.cancelAnimationFrame(canvasPanFrame)
  canvasPanFrame = null
  if (!canvasPan || !pendingCanvasPanEvent) return
  const event = pendingCanvasPanEvent
  pendingCanvasPanEvent = null
  applyCanvasTransform({
    ...liveCanvasView,
    x: canvasPan.originX + event.clientX - canvasPan.startX,
    y: canvasPan.originY + event.clientY - canvasPan.startY
  })
}

function endCanvasPan(e) {
  if (!canvasPan || canvasPan.pointerId !== e.pointerId) return
  e.currentTarget.releasePointerCapture?.(e.pointerId)
  flushCanvasPanFrame()
  canvasPan = null
  canvasPanActive.value = false
  canvasView.value = { ...liveCanvasView }
  bumpViewportCheck()
  scheduleCanvasStateSave()
}

async function restartGroupWorkflow(groupId) {
  if (groupRunState(groupId).running) return
  resetWorkflowGroupRunProgress(groupId)
  await runGroupWorkflow(groupId)
}

function onCanvasWheel(e) {
  if (e.target?.closest?.('.viewer-container, .model-modal__viewer')) return
  e.preventDefault()
  markCanvasInteraction()
  pendingCanvasWheel = {
    deltaY: (pendingCanvasWheel?.deltaY || 0) + e.deltaY,
    clientX: e.clientX,
    clientY: e.clientY,
    target: e.currentTarget
  }
  if (canvasWheelFrame) return
  canvasWheelFrame = window.requestAnimationFrame(() => {
    canvasWheelFrame = null
    const wheel = pendingCanvasWheel
    pendingCanvasWheel = null
    if (!wheel) return
    const rect = wheel.target.getBoundingClientRect()
    const pointerX = wheel.clientX - rect.left
    const pointerY = wheel.clientY - rect.top
    const currentScale = liveCanvasView.scale
    const steps = Math.max(1, Math.min(Math.abs(wheel.deltaY) / 100, 8))
    const direction = wheel.deltaY > 0 ? 0.9 : 1.1
    const nextScale = THREE.MathUtils.clamp(currentScale * Math.pow(direction, Math.max(1, Math.min(steps, 8))), 0.08, 3)
    if (nextScale === currentScale) return
    const worldX = (pointerX - liveCanvasView.x) / currentScale
    const worldY = (pointerY - liveCanvasView.y) / currentScale
    applyCanvasTransform({
      scale: nextScale,
      x: pointerX - worldX * nextScale,
      y: pointerY - worldY * nextScale
    }, { syncFloatingScale: false })
    scheduleCanvasFloatingScaleSync()
    scheduleViewportCheck()
    scheduleCanvasStateSave()
  })
}

function startNodeDrag(e, node) {
  if (batchDownloadSelectMode.value) {
    e.preventDefault()
    e.stopPropagation()
    return
  }
  const readonlyTextInput = e.target.closest?.('textarea.text-node-input[readonly], textarea.llm-node-textarea[readonly]')
  if (readonlyTextInput && ['text', 'llm'].includes(nodeType(node))) {
    const now = Date.now()
    const isDoubleClick = lastReadonlyTextClick.node === node &&
      now - lastReadonlyTextClick.time < 350 &&
      Math.abs(e.clientX - lastReadonlyTextClick.x) < 8 &&
      Math.abs(e.clientY - lastReadonlyTextClick.y) < 8
    lastReadonlyTextClick = {
      node,
      time: now,
      x: e.clientX,
      y: e.clientY
    }
    if (isDoubleClick) {
      e.preventDefault()
      e.stopPropagation()
      lastReadonlyTextClick = { node: '', time: 0, x: 0, y: 0 }
      if (nodeType(node) === 'llm') enableLlmNodeEdit(node, { target: readonlyTextInput })
      else enableTextNodeEdit(node, { target: readonlyTextInput })
      return
    }
  }
  const blockedTarget = readonlyTextInput
    ? null
    : e.target.closest?.('.node-port, .image-node-toolbar, .generate-edit-toolbar, .generate-edit-canvas, .generate-editor-panel, input, textarea, select, .view-cube, .scene-product-layer')
  const buttonTarget = e.target.closest?.('button')
  const canDragFromButton = buttonTarget?.classList?.contains('generate-node__image-button')
  if (e.button !== 0 || blockedTarget || (buttonTarget && !canDragFromButton)) return
  markCanvasInteraction()
  if (!readonlyTextInput) e.preventDefault()
  e.stopPropagation()
  const pos = nodePositions.value[node]
  if (!pos) return
  nodeDrag = {
    node,
    pointerId: e.pointerId,
    startX: e.clientX,
    startY: e.clientY,
    originX: pos.x,
    originY: pos.y,
    undoSnapshot: nodePositionSnapshot(),
    element: e.currentTarget,
    dx: 0,
    dy: 0,
    moved: false,
    linkSnapshots: dragLinkSnapshots([node]),
    movedNodes: new Set([node])
  }
  e.currentTarget.classList.add('is-dragging')
  suppressNodeClick = false
  if (!readonlyTextInput) e.currentTarget.setPointerCapture?.(e.pointerId)
  window.addEventListener('pointermove', dragNode)
  window.addEventListener('pointerup', endNodeDrag)
  window.addEventListener('pointercancel', endNodeDrag)
}

function dragNode(e) {
  if (!nodeDrag || nodeDrag.pointerId !== e.pointerId) return
  markCanvasInteraction()
  pendingNodeDragEvent = { clientX: e.clientX, clientY: e.clientY }
  if (nodeDragFrame) return
  nodeDragFrame = window.requestAnimationFrame(() => {
    nodeDragFrame = null
    flushNodeDragMove()
  })
}

function flushNodeDragMove() {
  if (!nodeDrag || !pendingNodeDragEvent) return
  const event = pendingNodeDragEvent
  pendingNodeDragEvent = null
  const scale = liveCanvasView.scale || 1
  const dx = (event.clientX - nodeDrag.startX) / scale
  const dy = (event.clientY - nodeDrag.startY) / scale
  if (Math.abs(event.clientX - nodeDrag.startX) > 3 || Math.abs(event.clientY - nodeDrag.startY) > 3) {
    nodeDrag.moved = true
    suppressNodeClick = true
  }
  nodeDrag.dx = dx
  nodeDrag.dy = dy
  if (nodeDrag.element) {
    nodeDrag.element.style.transform = `translate3d(${dx}px, ${dy}px, 0)`
  }
  updateDraggedLinkPaths(nodeDrag.linkSnapshots, nodeDrag.movedNodes, dx, dy)
}

function endNodeDrag(e) {
  if (!nodeDrag || nodeDrag.pointerId !== e.pointerId) return
  if (nodeDrag.element?.hasPointerCapture?.(e.pointerId)) {
    nodeDrag.element.releasePointerCapture(e.pointerId)
  }
  window.removeEventListener('pointermove', dragNode)
  window.removeEventListener('pointerup', endNodeDrag)
  window.removeEventListener('pointercancel', endNodeDrag)
  if (nodeDragFrame) {
    window.cancelAnimationFrame(nodeDragFrame)
    nodeDragFrame = null
  }
  flushNodeDragMove()
  const draggedNode = nodeDrag.node
  const didMove = nodeDrag.moved
  const undoSnapshot = nodeDrag.undoSnapshot
  const dragElement = nodeDrag.element
  if (didMove) {
    const pos = nodePositions.value[draggedNode]
    if (pos) {
      pos.x = nodeDrag.originX + nodeDrag.dx
      pos.y = nodeDrag.originY + nodeDrag.dy
      if (dragElement) {
        dragElement.style.left = `${pos.x}px`
        dragElement.style.top = `${pos.y}px`
      }
      resizeGroupsForNode(draggedNode)
    }
  }
  nodeDrag = null
  if (dragElement) {
    dragElement.style.transform = ''
    dragElement.classList.remove('is-dragging')
  }
  if (didMove) {
    if (undoSnapshot) {
      workflowUndoStack.value = [...workflowUndoStack.value, undoSnapshot].slice(-WORKFLOW_UNDO_LIMIT)
    }
    window.setTimeout(() => {
      suppressNodeClick = false
    }, 80)
    bumpViewportCheck()
    scheduleNodeLayoutStateSave({ includeGroups: true })
    return
  }
  setActiveWorkflowNode(draggedNode)
}

function workflowTemplateNodeCount(template) {
  const state = template?.state || {}
  return Object.values(state.workflowNodes || {}).filter(Boolean).length + (state.workflowDynamicNodes || []).length
}

const WORKFLOW_TEMPLATE_NODE_META = {
  upload: { label: '3D', className: 'is-model' },
  output: { label: '视图', className: 'is-output' },
  generate: { label: '图片', className: 'is-image' },
  text: { label: '文本', className: 'is-text' },
  llm: { label: '模型', className: 'is-llm' },
  video: { label: '视频', className: 'is-video' },
  'batch-recolor': { label: '换色', className: 'is-color' },
  'batch-translate': { label: '翻译', className: 'is-translate' }
}

function workflowTemplatePreviewNodes(template) {
  const state = template?.state || {}
  const nodes = [
    ...Object.entries(state.workflowNodes || {}).filter(([, enabled]) => enabled).map(([id]) => ({ id, type: id })),
    ...(state.workflowDynamicNodes || []).map(item => ({ id: item.id, type: item.type }))
  ]
  const byId = new Map(nodes.map(node => [node.id, node]))
  const incoming = new Map(nodes.map(node => [node.id, 0]))
  const outgoing = new Map(nodes.map(node => [node.id, []]))
  ;(state.workflowLinks || []).forEach((link) => {
    if (!byId.has(link.from) || !byId.has(link.to)) return
    incoming.set(link.to, (incoming.get(link.to) || 0) + 1)
    outgoing.get(link.from)?.push(link.to)
  })
  const queue = nodes.filter(node => incoming.get(node.id) === 0).map(node => node.id)
  const ordered = []
  while (queue.length) {
    const id = queue.shift()
    ordered.push(id)
    ;(outgoing.get(id) || []).forEach((next) => {
      incoming.set(next, incoming.get(next) - 1)
      if (incoming.get(next) === 0) queue.push(next)
    })
  }
  const orderedNodes = ordered.length === nodes.length ? ordered.map(id => byId.get(id)) : nodes
  return orderedNodes.slice(0, 4).map((node) => ({
    ...node,
    ...(WORKFLOW_TEMPLATE_NODE_META[node.type] || { label: '节点', className: 'is-default' })
  }))
}

function workflowTemplateUpdatedAt(template) {
  const date = new Date(template?.updated_at || template?.updatedAt || template?.created_at || '')
  if (Number.isNaN(date.getTime())) return '刚刚更新'
  return `${date.getMonth() + 1}月${date.getDate()}日更新`
}

async function loadWorkflowTemplates() {
  workflowTemplatesLoading.value = true
  try {
    const res = await request.get('/workflow-templates')
    workflowTemplates.value = Array.isArray(res.data?.data) ? res.data.data : []
  } catch (e) { error.value = e.response?.data?.error || '工作流模板加载失败' } finally { workflowTemplatesLoading.value = false }
}

async function openWorkflowTemplateModal() {
  showWorkflowTemplateMenu.value = false
  workflowTemplateSearch.value = ''
  showWorkflowTemplateModal.value = true
  nextTick(() => workflowTemplateSearchInput.value?.focus?.())
  await loadWorkflowTemplates()
}

function closeWorkflowTemplateModal() {
  showWorkflowTemplateModal.value = false
  workflowTemplateDeleteTarget.value = null
}

function workflowSnapshotForNodes(nodes = []) {
  const included = new Set(nodes.filter(nodeExists))
  const snapshot = workflowSnapshot()
  const filterStateMap = map => Object.fromEntries(Object.entries(map || {}).filter(([node]) => included.has(node)))
  return sanitizeWorkflowTemplateState({
    ...snapshot,
    workflowNodes: Object.fromEntries(Object.keys(snapshot.workflowNodes || {}).map(node => [node, included.has(node)])),
    workflowDynamicNodes: (snapshot.workflowDynamicNodes || []).filter(node => included.has(node.id)),
    workflowLinks: (snapshot.workflowLinks || []).filter(link => included.has(link.from) && included.has(link.to)),
    workflowHistoryAssets: [],
    workflowGroups: [],
    nodePositions: filterStateMap(snapshot.nodePositions),
    activeWorkflowNode: nodes[0] || '',
    activeWorkflowLink: '',
    workflowGenerateNodeStates: filterStateMap(snapshot.workflowGenerateNodeStates),
    workflowTextNodeStates: filterStateMap(snapshot.workflowTextNodeStates),
    workflowLlmNodeStates: filterStateMap(snapshot.workflowLlmNodeStates),
    workflowModelNodeStates: filterStateMap(snapshot.workflowModelNodeStates),
    workflowOutputNodeStates: filterStateMap(snapshot.workflowOutputNodeStates),
    workflowBatchRecolorNodeStates: filterStateMap(snapshot.workflowBatchRecolorNodeStates),
    workflowBatchTranslateNodeStates: filterStateMap(snapshot.workflowBatchTranslateNodeStates),
    workflowVideoNodeStates: filterStateMap(snapshot.workflowVideoNodeStates)
  })
}

async function ensureWorkflowTemplateAssetsOnOSS(value) {
  if (isLocalProjectImageUrl(value)) return uploadImageUrlToOSS(value)
  if (Array.isArray(value)) {
    const uploaded = []
    for (const item of value) uploaded.push(await ensureWorkflowTemplateAssetsOnOSS(item))
    return uploaded
  }
  if (value && typeof value === 'object') {
    const uploaded = {}
    for (const [key, item] of Object.entries(value)) uploaded[key] = await ensureWorkflowTemplateAssetsOnOSS(item)
    return uploaded
  }
  return value
}

function saveGroupAsWorkflow(groupId) {
  const group = groupBoxes.value.find(item => item.id === groupId)
  const nodes = (group?.nodes || []).filter(nodeExists)
  if (nodes.length < 2) return
  workflowSaveGroupId.value = groupId
  workflowSaveName.value = projects.value.find(item => item.id === activeProjectId.value)?.name || '我的工作流'
  workflowSaveScope.value = 'user'
  workflowSaveStatus.value = ''
  workflowSaveError.value = ''
  showWorkflowSaveModal.value = true
  nextTick(() => {
    workflowSaveNameInput.value?.focus?.()
    workflowSaveNameInput.value?.select?.()
  })
}

function closeWorkflowSaveModal() {
  if (workflowSaving.value) return
  showWorkflowSaveModal.value = false
  workflowSaveGroupId.value = ''
  workflowSaveStatus.value = ''
  workflowSaveError.value = ''
}

async function confirmSaveGroupAsWorkflow() {
  const group = groupBoxes.value.find(item => item.id === workflowSaveGroupId.value)
  const nodes = (group?.nodes || []).filter(nodeExists)
  const name = workflowSaveName.value.trim()
  if (workflowSaving.value) return
  if (!name) {
    workflowSaveError.value = '请输入工作流名称'
    nextTick(() => workflowSaveNameInput.value?.focus?.())
    return
  }
  if (nodes.length < 2) {
    workflowSaveError.value = '当前组合至少需要两个节点'
    return
  }
  workflowSaveError.value = ''
  workflowSaveStatus.value = '正在处理资源'
  workflowSaving.value = true
  try {
    const snapshot = await ensureProjectImagesOnOSS(workflowSnapshotForNodes(nodes))
    const templateState = await ensureWorkflowTemplateAssetsOnOSS(snapshot)
    workflowSaveStatus.value = '正在保存'
    await request.post('/workflow-templates', { name, scope: isWorkflowAdmin.value ? workflowSaveScope.value : 'user', summary: `${nodes.length} 个节点的专业画布工作流`, state: templateState })
    await loadWorkflowTemplates()
    workflowSaving.value = false
    closeWorkflowSaveModal()
  } catch (e) {
    workflowSaveError.value = e.response?.data?.error || '工作流保存失败，请重试'
    workflowSaveStatus.value = ''
    workflowSaving.value = false
  }
}

function deleteWorkflowTemplate(template) {
  workflowTemplateDeleteTarget.value = template
}

function closeWorkflowTemplateDeleteModal() {
  if (workflowTemplateDeleting.value) return
  workflowTemplateDeleteTarget.value = null
}

async function confirmDeleteWorkflowTemplate() {
  const template = workflowTemplateDeleteTarget.value
  if (!template || workflowTemplateDeleting.value) return
  workflowTemplateDeleting.value = true
  try { await request.delete(`/workflow-templates/${template.id}`); await loadWorkflowTemplates() } catch (e) { error.value = e.response?.data?.error || '删除失败' }
  finally {
    workflowTemplateDeleting.value = false
    workflowTemplateDeleteTarget.value = null
  }
}

function workflowTemplateSourceNodes(state = {}) {
  const staticNodes = Object.entries(state.workflowNodes || {})
    .filter(([, enabled]) => enabled)
    .map(([id]) => ({ id, type: id === 'preview' ? 'upload' : id }))
  const dynamicNodes = (state.workflowDynamicNodes || []).map(item => ({ id: item.id, type: item.type }))
  return [...staticNodes, ...dynamicNodes].filter((node, index, nodes) => node.id && nodes.findIndex(item => item.id === node.id) === index)
}

function workflowTemplateStateBounds(state, sourceNodes) {
  const rects = sourceNodes.map(({ id, type }) => {
    const position = state.nodePositions?.[id] || DEFAULT_NODE_POSITIONS[type] || DEFAULT_NODE_POSITIONS.generate
    return {
      x: Number(position.x) || 0,
      y: Number(position.y) || 0,
      width: Number(position.width) || DEFAULT_NODE_POSITIONS[type]?.width || DEFAULT_NODE_POSITIONS.generate.width,
      height: Number(position.height) || DEFAULT_NODE_POSITIONS[type]?.height || DEFAULT_NODE_POSITIONS.generate.height
    }
  })
  const minX = Math.min(...rects.map(rect => rect.x))
  const minY = Math.min(...rects.map(rect => rect.y))
  const maxX = Math.max(...rects.map(rect => rect.x + rect.width))
  const maxY = Math.max(...rects.map(rect => rect.y + rect.height))
  return { x: minX, y: minY, width: maxX - minX, height: maxY - minY }
}

function workflowTemplateInsertPosition(bounds) {
  const canvas = workflowCanvasRef.value
  const scale = liveCanvasView.scale || 1
  const centerX = canvas ? (canvas.clientWidth / 2 - liveCanvasView.x) / scale : bounds.x + bounds.width / 2
  const centerY = canvas ? (canvas.clientHeight / 2 - liveCanvasView.y) / scale : bounds.y + bounds.height / 2
  return findNonOverlappingPosition({
    x: Math.round(centerX - bounds.width / 2),
    y: Math.round(centerY - bounds.height / 2),
    width: bounds.width,
    height: bounds.height
  }, {
    stepY: Math.max(420, bounds.height + 80),
    gap: 72,
    maxRows: 120
  })
}

function remapWorkflowTemplateValue(value, idMap) {
  if (typeof value === 'string') return idMap[value] || value
  if (Array.isArray(value)) return value.map(item => remapWorkflowTemplateValue(item, idMap))
  if (!value || typeof value !== 'object') return value
  return Object.fromEntries(Object.entries(value).map(([key, item]) => [idMap[key] || key, remapWorkflowTemplateValue(item, idMap)]))
}

function remapWorkflowTemplateStateMap(sourceMap, idMap, normalizer = value => value) {
  return Object.fromEntries(Object.entries(sourceMap || {})
    .filter(([node]) => idMap[node])
    .map(([node, value]) => [idMap[node], normalizer(remapWorkflowTemplateValue(value, idMap))]))
}

function useWorkflowTemplate(template) {
  const state = clonePlain(template?.state || {})
  const sourceNodes = workflowTemplateSourceNodes(state)
  if (!sourceNodes.length) return
  const sourceBounds = workflowTemplateStateBounds(state, sourceNodes)
  const insertPosition = workflowTemplateInsertPosition(sourceBounds)
  const offsetX = insertPosition.x - sourceBounds.x
  const offsetY = insertPosition.y - sourceBounds.y
  const idMap = Object.fromEntries(sourceNodes.map(node => [node.id, createWorkflowNodeId(node.type)]))
  const importedNodes = sourceNodes.map(node => ({ id: idMap[node.id], type: node.type }))
  const importedNodeIds = importedNodes.map(node => node.id)

  pushWorkflowUndoSnapshot()
  workflowDynamicNodes.value = [...workflowDynamicNodes.value, ...importedNodes]
  nodePositions.value = {
    ...nodePositions.value,
    ...Object.fromEntries(sourceNodes.map(({ id, type }) => {
      const sourcePosition = state.nodePositions?.[id] || DEFAULT_NODE_POSITIONS[type] || DEFAULT_NODE_POSITIONS.generate
      return [idMap[id], {
        ...sourcePosition,
        x: Math.round((Number(sourcePosition.x) || 0) + offsetX),
        y: Math.round((Number(sourcePosition.y) || 0) + offsetY)
      }]
    }))
  }
  workflowLinks.value = normalizeWorkflowLinks([
    ...workflowLinks.value,
    ...(state.workflowLinks || []).map(link => ({ from: idMap[link.from], to: idMap[link.to] })).filter(link => link.from && link.to)
  ])
  const importedOutputStates = remapWorkflowTemplateStateMap(state.workflowOutputNodeStates, idMap, normalizeOutputNodeState)
  sourceNodes.filter(node => node.type === 'output' && !importedOutputStates[idMap[node.id]]).forEach((node) => {
    const sourceModelNode = (state.workflowLinks || []).find(link => link.to === node.id && idMap[link.from])?.from || ''
    importedOutputStates[idMap[node.id]] = normalizeOutputNodeState(remapWorkflowTemplateValue({
      resultUrls: node.id === 'output' ? (state.resultUrls || {}) : {},
      sourceNode: sourceModelNode
    }, idMap))
  })
  workflowGenerateNodeStates.value = {
    ...workflowGenerateNodeStates.value,
    ...remapWorkflowTemplateStateMap(state.workflowGenerateNodeStates, idMap, value => normalizeGenerateNodeState({ ...value, generating: false, cancelled: false, runId: '', progress: 0, taskProgress: [] }))
  }
  workflowTextNodeStates.value = { ...workflowTextNodeStates.value, ...remapWorkflowTemplateStateMap(state.workflowTextNodeStates, idMap, normalizeTextNodeState) }
  workflowLlmNodeStates.value = {
    ...workflowLlmNodeStates.value,
    ...remapWorkflowTemplateStateMap(state.workflowLlmNodeStates, idMap, value => defaultLlmNodeState({ ...value, loading: false, editing: false, error: '' }))
  }
  migrateCompletedLlmNodesToText(importedNodeIds)
  workflowModelNodeStates.value = { ...workflowModelNodeStates.value, ...remapWorkflowTemplateStateMap(state.workflowModelNodeStates, idMap, value => normalizeModelNodeState({ ...value, loading: false, error: '' })) }
  workflowOutputNodeStates.value = { ...workflowOutputNodeStates.value, ...importedOutputStates }
  workflowBatchRecolorNodeStates.value = { ...workflowBatchRecolorNodeStates.value, ...remapWorkflowTemplateStateMap(state.workflowBatchRecolorNodeStates, idMap, value => resetBatchRecolorRuntimeState(normalizeBatchRecolorNodeState(value))) }
  workflowBatchTranslateNodeStates.value = { ...workflowBatchTranslateNodeStates.value, ...remapWorkflowTemplateStateMap(state.workflowBatchTranslateNodeStates, idMap, value => resetBatchTranslateRuntimeState(normalizeBatchTranslateNodeState(value))) }
  workflowVideoNodeStates.value = { ...workflowVideoNodeStates.value, ...remapWorkflowTemplateStateMap(state.workflowVideoNodeStates, idMap, value => normalizeVideoNodeState(value, { resetRuntime: true })) }

  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  activeGroupId.value = createWorkflowGroup(importedNodeIds, { activate: true, persist: false })
  groupBoxes.value = groupBoxes.value.map(group => group.id === activeGroupId.value
    ? { ...group, sourceTemplateId: template.id, templatePristine: true }
    : group)
  closeAllGenerateMenus()
  bumpViewportCheck()
  saveActiveProject()
  closeWorkflowTemplateModal()
  nextTick(() => focusCanvasOnNodes(importedNodeIds))
}

function onGeneratePreviewClick(node) {
  if (batchDownloadSelectMode.value) {
    onWorkflowNodeClick(null, node)
    return
  }
  if (suppressNodeClick) {
    suppressNodeClick = false
    return
  }
  if (isModelWorkflowGeneratedImageNode(node)) {
    openGenerateNodePreview(node)
    return
  }
  activateGenerateNodeEditor(node)
}

function prepareGenerateImageNodeInteraction(node) {
  if (batchDownloadSelectMode.value) return
  if (isModelWorkflowGeneratedImageNode(node)) return
  if (!node || activeWorkflowNode.value === node || isGenerateImageEditing(node)) return
  activateGenerateNodeEditor(node)
}

function onGenerateNodeClick(e, node) {
  if (batchDownloadSelectMode.value) {
    onWorkflowNodeClick(e, node)
    return
  }
  if (suppressNodeClick) {
    suppressNodeClick = false
    return
  }
  if (e.target.closest?.([
    '.node-port',
    '.generate-upload-btn',
    '.image-node-toolbar',
    '.generate-edit-toolbar',
    '.generate-edit-canvas',
    '.generate-editor-panel',
    'input',
    'textarea',
    'select',
    'button'
  ].join(', '))) {
    return
  }
  activateGenerateNodeEditor(node)
}

function createWorkflowNodeId(type) {
  return `${type}-${Date.now()}-${Math.random().toString(16).slice(2, 7)}`
}

function workflowGroupIdForNode(node) {
  return groupBoxes.value.find(group => (group.nodes || []).includes(node))?.id || ''
}

function workflowGroupForNewNode(node, preferredGroupId = '') {
  if (preferredGroupId) {
    const preferred = groupBoxes.value.find(group => group.id === preferredGroupId)
    if (preferred) return preferred
  }
  const rect = computedNodeRect(node) || nodePositions.value[node]
  if (!rect) return null
  const centerX = rect.x + (rect.width || 0) / 2
  const centerY = rect.y + (rect.height || 0) / 2
  return [...groupBoxes.value].reverse().find(group => (
    !groupRunState(group.id).running &&
    centerX >= group.x && centerX <= group.x + group.width &&
    centerY >= group.y && centerY <= group.y + group.height
  )) || null
}

function assignNewNodeToWorkflowGroup(node, preferredGroupId = '') {
  const group = workflowGroupForNewNode(node, preferredGroupId)
  if (!group || (group.nodes || []).includes(node)) return
  if (groupRunState(group.id).paused) resetWorkflowGroupRunProgress(group.id)
  groupBoxes.value = groupBoxes.value.map(item => item.id === group.id
    ? { ...item, nodes: [...(item.nodes || []), node], templatePristine: groupRunState(item.id).running ? item.templatePristine : false }
    : item)
  resizeGroupsForNode(node)
  nextTick(() => resizeGroupsForNode(node))
}

function addWorkflowNode(node, options = {}) {
  if (node === 'preview') node = 'upload'
  if (node === 'output') return
  showStyleLibraryMenu.value = false
  const shouldPersist = options.persist !== false
  if (options.undo !== false) pushWorkflowUndoSnapshot()
  if (['style', 'text', 'llm', 'video', 'batch-recolor', 'batch-translate'].includes(node)) {
    const id = createWorkflowNodeId(node)
    const base = defaultNewNodePosition(node)
    const count = workflowDynamicNodes.value.filter(item => item.type === node).length
    workflowDynamicNodes.value = [
      ...workflowDynamicNodes.value,
      { id, type: node }
    ]
    nodePositions.value = {
      ...nodePositions.value,
      [id]: {
        ...base,
        x: options.position?.x ?? (base.x + count * 42),
        y: options.position?.y ?? (base.y + count * 42)
      }
    }
    if (node === 'text') {
      workflowTextNodeStates.value = {
        ...workflowTextNodeStates.value,
        [id]: defaultTextNodeState(options.state || {})
      }
    }
    if (node === 'llm') {
      workflowLlmNodeStates.value = { ...workflowLlmNodeStates.value, [id]: defaultLlmNodeState(options.state || {}) }
    }
    if (node === 'batch-recolor') {
      workflowBatchRecolorNodeStates.value = {
        ...workflowBatchRecolorNodeStates.value,
        [id]: defaultBatchRecolorNodeState(options.state || {})
      }
    }
    if (node === 'batch-translate') {
      workflowBatchTranslateNodeStates.value = {
        ...workflowBatchTranslateNodeStates.value,
        [id]: defaultBatchTranslateNodeState(options.state || {})
      }
    }
    if (node === 'video') {
      workflowVideoNodeStates.value = {
        ...workflowVideoNodeStates.value,
        [id]: defaultVideoNodeState(options.state || {})
      }
    }
    activeWorkflowNode.value = id
    showNodeMenu.value = false
    assignNewNodeToWorkflowGroup(id, options.groupId)
    bumpViewportCheck()
    saveWorkflowStructureState({ persist: shouldPersist })
    return id
  }
  if (node === 'generate') {
    const id = createWorkflowNodeId('generate')
    workflowDynamicNodes.value = [
      ...workflowDynamicNodes.value,
      { id, type: 'generate' }
    ]
    const base = defaultNewNodePosition('generate')
    const count = generateWorkflowNodes().length
    nodePositions.value = {
      ...nodePositions.value,
      [id]: {
        ...base,
        x: options.position?.x ?? (base.x + count * 36),
        y: options.position?.y ?? (base.y + count * 36)
      }
    }
    workflowGenerateNodeStates.value = {
      ...workflowGenerateNodeStates.value,
      [id]: defaultGenerateNodeState(options.state || {})
    }
    activeWorkflowNode.value = id
    showNodeMenu.value = false
    assignNewNodeToWorkflowGroup(id, options.groupId)
    bumpViewportCheck()
    saveWorkflowStructureState({ persist: shouldPersist })
    return id
  }

  if (workflowNodes.value[node]) {
    const id = createWorkflowNodeId(node)
    const base = defaultNewNodePosition(node)
    const count = workflowDynamicNodes.value.filter(item => item.type === node).length + 1
    workflowDynamicNodes.value = [
      ...workflowDynamicNodes.value,
      { id, type: node }
    ]
    nodePositions.value = {
      ...nodePositions.value,
      [id]: {
        ...base,
        x: options.position?.x ?? (base.x + count * 42),
        y: options.position?.y ?? (base.y + count * 42)
      }
    }
    if (node === 'upload') {
      workflowModelNodeStates.value = {
        ...workflowModelNodeStates.value,
        [id]: defaultModelNodeState(options.state || {})
      }
      activeModelNodeId.value = id
    }
    activeWorkflowNode.value = id
    showNodeMenu.value = false
    assignNewNodeToWorkflowGroup(id, options.groupId)
    bumpViewportCheck()
    saveWorkflowStructureState({ persist: shouldPersist })
    return id
  }

  workflowNodes.value = {
    ...workflowNodes.value,
    [node]: true
  }
  const base = defaultNewNodePosition(node)
  nodePositions.value = {
    ...nodePositions.value,
    [node]: {
      ...base,
      x: options.position?.x ?? base.x,
      y: options.position?.y ?? base.y
    }
  }
  activeWorkflowNode.value = node
  if (node === 'upload') {
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [node]: normalizeModelNodeState({
        fileName: fileName.value,
        stpFileUrl: stpFileUrl.value,
        meshData: meshData.value,
        modelLoaded: modelLoaded.value,
        ...(options.state || {})
      })
    }
    activeModelNodeId.value = node
  }
  showNodeMenu.value = false
  assignNewNodeToWorkflowGroup(node, options.groupId)
  bumpViewportCheck()
  saveWorkflowStructureState({ persist: shouldPersist })
  return node
}

function removeGeneratedChildNodes(parentNode) {
  const childNodes = Object.entries(workflowGenerateNodeStates.value || {})
    .filter(([, state]) => state?.generatedFromNode === parentNode)
    .map(([node]) => node)
    .filter(nodeExists)
  if (!childNodes.length) return
  childNodes.forEach((node) => removeWorkflowNode(node, { undo: false, persist: false }))
}

function cleanupReferencesToRemovedNodes(nodes = []) {
  const removed = new Set((Array.isArray(nodes) ? nodes : [nodes]).filter(Boolean))
  if (!removed.size) return

  workflowLinks.value = normalizeWorkflowLinks(workflowLinks.value)
    .filter(link => !removed.has(link.from) && !removed.has(link.to) && nodeExists(link.from) && nodeExists(link.to))

  workflowDynamicNodes.value = workflowDynamicNodes.value.filter(item => !removed.has(item.id))
  const nextPositions = { ...nodePositions.value }
  removed.forEach(node => {
    delete nextPositions[node]
  })
  nodePositions.value = nextPositions

  workflowNodes.value = normalizeWorkflowNodes(workflowNodes.value)

  groupBoxes.value = groupBoxes.value
    .map(group => ({
      ...group,
      nodes: (group.nodes || []).filter(node => !removed.has(node) && nodeExists(node))
    }))
    .filter(group => group.nodes.length > 1)

  const nextGenerateStates = { ...workflowGenerateNodeStates.value }
  Object.keys(nextGenerateStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node)) {
      delete nextGenerateStates[node]
      return
    }
    const state = nextGenerateStates[node] || {}
    nextGenerateStates[node] = {
      ...state,
      generatedFromNode: removed.has(state.generatedFromNode) ? '' : state.generatedFromNode,
      sceneControlOutputNode: removed.has(state.sceneControlOutputNode) ? '' : state.sceneControlOutputNode
    }
  })
  workflowGenerateNodeStates.value = nextGenerateStates

  const nextModelStates = { ...workflowModelNodeStates.value }
  Object.keys(nextModelStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node)) {
      delete nextModelStates[node]
      return
    }
    const state = nextModelStates[node] || {}
    nextModelStates[node] = normalizeModelNodeState({
      ...state,
      sceneStyleNode: removed.has(state.sceneStyleNode) ? '' : state.sceneStyleNode,
      sceneMaterialNode: removed.has(state.sceneMaterialNode) ? '' : state.sceneMaterialNode
    })
  })
  workflowModelNodeStates.value = nextModelStates

  const nextOutputStates = { ...workflowOutputNodeStates.value }
  Object.keys(nextOutputStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node) || removed.has(nextOutputStates[node]?.sourceNode)) {
      delete nextOutputStates[node]
    }
  })
  workflowOutputNodeStates.value = nextOutputStates

  const nextRecolorStates = { ...workflowBatchRecolorNodeStates.value }
  Object.keys(nextRecolorStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node)) {
      delete nextRecolorStates[node]
      return
    }
    if (removed.has(nextRecolorStates[node]?.resultNode)) {
      nextRecolorStates[node] = { ...nextRecolorStates[node], resultNode: '' }
    }
  })
  workflowBatchRecolorNodeStates.value = nextRecolorStates

  const nextTranslateStates = { ...workflowBatchTranslateNodeStates.value }
  Object.keys(nextTranslateStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node)) {
      delete nextTranslateStates[node]
      return
    }
    if (removed.has(nextTranslateStates[node]?.resultNode)) {
      nextTranslateStates[node] = { ...nextTranslateStates[node], resultNode: '' }
    }
  })
  workflowBatchTranslateNodeStates.value = nextTranslateStates

  const nextVideoStates = { ...workflowVideoNodeStates.value }
  Object.keys(nextVideoStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node)) delete nextVideoStates[node]
  })
  workflowVideoNodeStates.value = nextVideoStates

  const nextTextStates = { ...workflowTextNodeStates.value }
  Object.keys(nextTextStates).forEach(node => {
    if (removed.has(node) || !nodeExists(node)) delete nextTextStates[node]
  })
  workflowTextNodeStates.value = nextTextStates

  if (removed.has(activeWorkflowNode.value) || (activeWorkflowNode.value && !nodeExists(activeWorkflowNode.value))) activeWorkflowNode.value = ''
  if (activeWorkflowLink.value && !workflowLinks.value.some(link => workflowLinkKey(link) === activeWorkflowLink.value)) activeWorkflowLink.value = ''
  if (removed.has(pendingLinkFrom.value)) pendingLinkFrom.value = ''
  if (draggingLink.value && removed.has(draggingLink.value.from)) draggingLink.value = null
  if (batchMaterialPickerNode.value && removed.has(batchMaterialPickerNode.value)) batchMaterialPickerNode.value = ''
  if (generateImageEditState.value.node && removed.has(generateImageEditState.value.node)) resetGenerateImageEditState()
  selectedNodeIds.value = selectedNodeIds.value.filter(node => !removed.has(node) && nodeExists(node))
  if (activeGroupId.value && !groupBoxes.value.some(group => group.id === activeGroupId.value)) activeGroupId.value = ''
}

function outputNodesOwnedByModel(modelNode) {
  const linkedOutputNodes = workflowLinks.value
    .filter(link => link.from === modelNode && nodeType(link.to) === 'output')
    .map(link => link.to)
  const stateOutputNodes = Object.entries(workflowOutputNodeStates.value || {})
    .filter(([, state]) => state?.sourceNode === modelNode)
    .map(([node]) => node)
  return Array.from(new Set([...linkedOutputNodes, ...stateOutputNodes]))
    .filter(node => node === 'output' || nodeType(node) === 'output' || workflowOutputNodeStates.value[node])
}

function shouldRemoveStaticOutputForModel(modelNode) {
  if (!modelNode || !workflowNodes.value.output) return false
  if (workflowOutputNodeStates.value.output?.sourceNode === modelNode) return true
  const linkedModelNodes = workflowLinks.value
    .filter(link => link.to === 'output' && nodeType(link.from) === 'upload')
    .map(link => link.from)
  return linkedModelNodes.includes(modelNode) && linkedModelNodes.every(node => node === modelNode)
}

function cleanupModelOutputNodes(outputNodes = [], { removeStaticOutput = false } = {}) {
  const nodes = Array.from(new Set(outputNodes)).filter(Boolean)
  if (!nodes.length) return

  nodes.forEach(node => removeGeneratedChildNodes(node))

  const dynamicOutputNodes = nodes.filter(node => node !== 'output')
  if (dynamicOutputNodes.length) {
    workflowDynamicNodes.value = workflowDynamicNodes.value.filter(item => !dynamicOutputNodes.includes(item.id))
    dynamicOutputNodes.forEach((outputNode) => {
      const { [outputNode]: removedPosition, ...nextPositions } = nodePositions.value
      nodePositions.value = nextPositions
    })
  }

  const nextOutputStates = { ...workflowOutputNodeStates.value }
  nodes.forEach(node => {
    delete nextOutputStates[node]
  })
  workflowOutputNodeStates.value = nextOutputStates

  const linkRemovalNodes = removeStaticOutput ? nodes : dynamicOutputNodes
  if (linkRemovalNodes.length) {
    workflowLinks.value = workflowLinks.value.filter(link => !linkRemovalNodes.includes(link.from) && !linkRemovalNodes.includes(link.to))
  }

  if (nodes.includes('output')) {
    resultImageUrl.value = ''
    resultUrls.value = {}
    if (removeStaticOutput) {
      workflowNodes.value = {
        ...workflowNodes.value,
        output: false
      }
    }
  }
}

function generatedResultNodes(parentNode) {
  return generateWorkflowNodes()
    .filter(node => ensureGenerateNodeState(node.id).generatedFromNode === parentNode)
    .map(node => node.id)
    .filter(nodeExists)
}

function relayoutGeneratedResultNodes(parentNode, { persist = false } = {}) {
  const nodes = generatedResultNodes(parentNode)
  if (nodes.length < 2) return
  const rects = nodes.map(node => computedNodeRect(node)).filter(Boolean)
  if (rects.length !== nodes.length) return
  const minX = Math.min(...rects.map(rect => rect.x))
  const totalHeight = rects.reduce((sum, rect) => sum + rect.height, 0) + (rects.length - 1) * GENERATED_RESULT_NODE_GAP_Y
  let currentY = Math.min(...rects.map(rect => rect.y)) + (Math.max(...rects.map(rect => rect.y + rect.height)) - Math.min(...rects.map(rect => rect.y)) - totalHeight) / 2
  const nextPositions = { ...nodePositions.value }
  nodes.forEach((node, index) => {
    const rect = rects[index]
    nextPositions[node] = {
      ...(nextPositions[node] || DEFAULT_NODE_POSITIONS.generate),
      x: Math.round(minX),
      y: Math.round(currentY)
    }
    currentY += rect.height + GENERATED_RESULT_NODE_GAP_Y
  })
  nodePositions.value = nextPositions
  if (persist) scheduleNodeLayoutStateSave()
}

function relayoutGeneratedResultNodesForImage(url) {
  const parentNodes = new Set()
  Object.entries(workflowGenerateNodeStates.value || {}).forEach(([node, state]) => {
    if (!state?.generatedFromNode) return
    const images = normalizeGenerationResult(state.result).images.map(resolveImageUrl)
    if (images.includes(url)) parentNodes.add(state.generatedFromNode)
  })
  parentNodes.forEach(parentNode => relayoutGeneratedResultNodes(parentNode, { persist: true }))
}

function createGeneratedImageChildNodes(parentNode, images, meta = {}) {
  const parentRect = computedNodeRect(parentNode) || nodePositions.value[parentNode] || DEFAULT_NODE_POSITIONS.generate
  const parentResult = normalizeGenerationResult(ensureGenerateNodeState(parentNode).result)
  const parentState = ensureGenerateNodeState(parentNode)
  const resultImages = images.filter(Boolean)
  if (!resultImages.length) return []
  const nodeSizes = resultImages.map(url => generateResultNodeSizeForImage(url, DEFAULT_NODE_POSITIONS.generate))
  const maxWidth = Math.max(...nodeSizes.map(size => size.width), DEFAULT_NODE_POSITIONS.generate.width)
  const totalHeight = nodeSizes.reduce((sum, size) => sum + size.height, 0) + Math.max(0, nodeSizes.length - 1) * GENERATED_RESULT_NODE_GAP_Y
  let currentY = parentRect.y + ((parentRect.height || DEFAULT_NODE_POSITIONS.generate.height) - totalHeight) / 2
  const resultX = Math.round(parentRect.x + parentRect.width + GENERATED_RESULT_NODE_GAP_X)
  const childNodes = []
  const incomingLinks = workflowLinks.value.filter(link => link.to === parentNode)
  const outgoingLinks = workflowLinks.value.filter(link => link.from === parentNode)

  resultImages.forEach((url, index) => {
    const size = nodeSizes[index]
    const createdAt = meta.imageCreatedAt?.[index] || meta.createdAt || new Date().toISOString()
    const taskId = meta.taskIds?.[index] || meta.taskIds?.slice(-1)?.[0] || ''
    const id = addWorkflowNode('generate', {
      undo: false,
      persist: false,
      groupId: workflowGroupIdForNode(parentNode),
      position: {
        x: Math.round(resultX + (maxWidth - size.width) / 2),
        y: Math.round(currentY)
      },
      state: {
        prompt: parentState.prompt,
        model: parentState.model,
        count: 1,
        aspectRatio: parentState.aspectRatio,
        resolution: parentState.resolution,
        generatedFromNode: parentNode,
        sceneControlOutputNode: parentState.sceneControlOutputNode || '',
        sceneWorkflowNode: !!parentState.sceneWorkflowNode,
        inlineControlImages: normalizeInlineControlImages(parentState.inlineControlImages),
        modelWorkflowImageType: parentState.modelWorkflowImageType || '',
        imageEditDisabled: !!parentState.imageEditDisabled,
        result: {
          url,
          images: [url],
          imageCreatedAt: [createdAt],
          imageSources: [parentResult.imageSources[index] || 'generated'],
          taskIds: taskId ? [taskId] : [],
          taskId,
          createdAt,
          status: 'completed',
          batchTitle: meta.batchTitle || parentState.batchTitle || parentResult.batchTitle || ''
        },
        uploads: [],
        showResults: false
      }
    })
    if (id) {
      incomingLinks.forEach(link => connectWorkflowNodes(link.from, id, { undo: false, persist: false }))
      outgoingLinks.forEach(link => connectWorkflowNodes(id, link.to, { undo: false, persist: false }))
      childNodes.push(id)
    }
    currentY += size.height + GENERATED_RESULT_NODE_GAP_Y
  })
  return childNodes
}

function splitGenerationResultToNodes(parentNode, images, meta = {}) {
  const finalImages = images.filter(Boolean)
  if (!finalImages.length) return []
  removeGeneratedChildNodes(parentNode)
  return createGeneratedImageChildNodes(parentNode, finalImages, meta)
}

function removeWorkflowNodes(nodes, options = {}) {
  const nodesToRemove = [...new Set((Array.isArray(nodes) ? nodes : [nodes]).filter(nodeExists))]
  if (!nodesToRemove.length) return
  if (options.undo !== false && nodesToRemove.some(runningWorkflowGroupForNode)) {
    error.value = '工作流运行中，暂时不能删除节点'
    return
  }
  if (options.undo !== false) invalidatePausedWorkflowRunsForNodes(nodesToRemove, { structural: true })
  const shouldPersist = options.persist !== false
  if (options.undo !== false) pushWorkflowUndoSnapshot()
  nodesToRemove.forEach(node => removeWorkflowNode(node, { undo: false, persist: false }))
  cleanupReferencesToRemovedNodes(nodesToRemove)
  bumpViewportCheck()
  if (shouldPersist) saveActiveProject()
}

function removeWorkflowNodeWithoutSnapshot(node) {
  removeWorkflowNode(node, { undo: false, persist: false })
}

function removeWorkflowNode(node, options = {}) {
  if (node === 'preview') node = 'upload'
  if (!nodeExists(node)) {
    cleanupReferencesToRemovedNodes([node])
    return
  }
  const shouldPersist = options.persist !== false
  if (options.undo !== false) pushWorkflowUndoSnapshot()
  const type = nodeType(node)
  const isDynamicNode = workflowDynamicNodes.value.some(item => item.id === node)
  if (isDynamicNode) {
    const ownedOutputNodes = type === 'upload' ? outputNodesOwnedByModel(node) : []
    const shouldRemoveStaticOwnedOutput = type === 'upload' && shouldRemoveStaticOutputForModel(node)
    workflowDynamicNodes.value = workflowDynamicNodes.value.filter(item => item.id !== node)
    workflowLinks.value = workflowLinks.value.filter(link => link.from !== node && link.to !== node)
    activeWorkflowLink.value = ''
    const { [node]: removedPosition, ...nextPositions } = nodePositions.value
    nodePositions.value = nextPositions
    if (type === 'generate') {
      const { [node]: removedState, ...nextStates } = workflowGenerateNodeStates.value
      workflowGenerateNodeStates.value = nextStates
    }
    if (type === 'text') {
      stopTextNodeProgress(node)
      window.clearTimeout(textNodeInputSyncTimers.get(node))
      textNodeInputSyncTimers.delete(node)
      const { [node]: removedState, ...nextStates } = workflowTextNodeStates.value
      workflowTextNodeStates.value = nextStates
    }
    if (type === 'llm') {
      stopLlmNodeProgress(node)
      window.clearTimeout(llmNodeInputSyncTimers.get(node))
      llmNodeInputSyncTimers.delete(node)
      const { [node]: removedState, ...nextStates } = workflowLlmNodeStates.value
      workflowLlmNodeStates.value = nextStates
    }
    if (type === 'video') {
      const { [node]: removedState, ...nextStates } = workflowVideoNodeStates.value
      workflowVideoNodeStates.value = nextStates
    }
    if (type === 'batch-recolor') {
      removeGeneratedChildNodes(node)
      const { [node]: removedState, ...nextStates } = workflowBatchRecolorNodeStates.value
      workflowBatchRecolorNodeStates.value = nextStates
      if (batchMaterialPickerNode.value === node) batchMaterialPickerNode.value = ''
    }
    if (type === 'batch-translate') {
      removeGeneratedChildNodes(node)
      const { [node]: removedState, ...nextStates } = workflowBatchTranslateNodeStates.value
      workflowBatchTranslateNodeStates.value = nextStates
    }
    if (type === 'upload') {
      if (ownedOutputNodes.length) {
        cleanupModelOutputNodes(ownedOutputNodes, { removeStaticOutput: shouldRemoveStaticOwnedOutput })
      }
      clearModelMeshLoadPromisesForNode(node)
      cancelModelUploadRun(node)
      if (renderedModelNodeId === node) {
        clearRenderedModelScene({ render: true })
      }
      const { [node]: removedState, ...nextStates } = workflowModelNodeStates.value
      workflowModelNodeStates.value = nextStates
      if (activeModelNodeId.value === node) {
        const nextModel = modelWorkflowNodes().find(item => item.id !== node)?.id || ''
        if (nextModel) {
          activeModelNodeId.value = nextModel
          activateModelNode(nextModel)
        } else {
          activeModelNodeId.value = ''
          fileName.value = ''
          stpFileUrl.value = ''
          meshData.value = null
          modelLoaded.value = false
        }
      }
    }
    if (type === 'output') {
      removeGeneratedChildNodes(node)
      const { [node]: removedState, ...nextStates } = workflowOutputNodeStates.value
      workflowOutputNodeStates.value = nextStates
    }
      if (type === 'generate') {
        const owner = Object.keys(workflowModelNodeStates.value || {}).find((modelNode) => (
          workflowModelNodeStates.value[modelNode]?.sceneStyleNode === node
          || workflowModelNodeStates.value[modelNode]?.sceneMaterialNode === node
        ))
        if (owner) {
          const ownerState = workflowModelNodeStates.value[owner]
          workflowModelNodeStates.value = {
            ...workflowModelNodeStates.value,
            [owner]: normalizeModelNodeState({
              ...ownerState,
              sceneStyleNode: ownerState.sceneStyleNode === node ? '' : ownerState.sceneStyleNode,
              sceneMaterialNode: ownerState.sceneMaterialNode === node ? '' : ownerState.sceneMaterialNode
            })
          }
        }
      }
    pruneWorkflowGroupsAfterNodeRemoval()
    if (pendingLinkFrom.value === node) pendingLinkFrom.value = ''
    if (draggingLink.value?.from === node) draggingLink.value = null
    if (activeWorkflowNode.value === node) {
      activeWorkflowNode.value = ''
    }
    cleanupReferencesToRemovedNodes([node])
    bumpViewportCheck()
    if (shouldPersist) saveActiveProject()
    return
  }
  const ownedOutputNodes = node === 'upload' ? outputNodesOwnedByModel(node) : []
  workflowNodes.value = {
    ...workflowNodes.value,
    [node]: false
  }
  workflowLinks.value = workflowLinks.value.filter(link => link.from !== node && link.to !== node)
  pruneWorkflowGroupsAfterNodeRemoval()
  activeWorkflowLink.value = ''
  if (pendingLinkFrom.value === node) pendingLinkFrom.value = ''
  if (draggingLink.value?.from === node) draggingLink.value = null

  if (node === 'upload') {
    cleanupModelOutputNodes(ownedOutputNodes, { removeStaticOutput: true })
    clearModelMeshLoadPromisesForNode(node)
    cancelModelUploadRun(node)
    clearCurrentModel()
    fileName.value = ''
    stpFileUrl.value = ''
    meshData.value = null
    const { upload, ...nextModelStates } = workflowModelNodeStates.value
    workflowModelNodeStates.value = nextModelStates
    resultImageUrl.value = ''
    resultUrls.value = {}
    if (activeModelNodeId.value === node) {
      const nextModel = modelWorkflowNodes().find(item => item.id !== node)?.id || ''
      if (nextModel) {
        activeModelNodeId.value = nextModel
        activateModelNode(nextModel)
      } else {
        activeModelNodeId.value = ''
        modelLoaded.value = false
      }
    }
  } else if (node === 'output') {
    removeGeneratedChildNodes(node)
    resultImageUrl.value = ''
    resultUrls.value = {}
    const { output, ...nextOutputStates } = workflowOutputNodeStates.value
    workflowOutputNodeStates.value = nextOutputStates
  } else if (node === 'scene') {
    if (sceneImage.value.url) URL.revokeObjectURL(sceneImage.value.url)
    sceneImage.value = { url: '', name: '', width: 0, height: 0 }
  } else if (node === 'generate') {
    workflowGeneratePrompt.value = ''
    workflowGenerateCount.value = 1
    workflowGenerateAspectRatio.value = '1:1'
    workflowGenerateResolution.value = '2K'
    workflowGenerationError.value = ''
    workflowGenerationProgress.value = 0
    workflowGenerationCancelled.value = false
    workflowGenerationResult.value = emptyGenerationResult()
    const { generate, ...nextStates } = workflowGenerateNodeStates.value
    workflowGenerateNodeStates.value = nextStates
  }

  if (activeWorkflowNode.value === node) {
    activeWorkflowNode.value = ''
  }
  cleanupReferencesToRemovedNodes([node])
  bumpViewportCheck()
  if (shouldPersist) saveActiveProject()
}

function resolveImageUrl(url) {
  if (!url) return ''
  if (/^(data:|blob:|https?:\/\/)/i.test(url)) return url
  return url.startsWith('/') ? url : `/${url}`
}

function thumbnailImage(url, size = 480) {
  const resolved = resolveImageUrl(url)
  if (!resolved || /^(data:|blob:)/i.test(resolved)) return resolved
  if (!/^https?:\/\/.+oss-[^/]+\.aliyuncs\.com\//i.test(resolved)) return resolved
  const separator = resolved.includes('?') ? '&' : '?'
  return `${resolved}${separator}x-oss-process=image/resize,l_${size}/quality,q_82/format,webp`
}

function setLazyImageSource(el, src) {
  const nextSrc = src || ''
  el.draggable = false
  el.dataset.lazySrc = nextSrc
  if (!nextSrc) {
    el.removeAttribute('src')
    lazyImageObserver?.unobserve(el)
    return
  }
  if (!lazyImageObserver) {
    el.src = nextSrc
    return
  }
  el.removeAttribute('src')
  lazyImageObserver.unobserve(el)
  lazyImageObserver.observe(el)
}

function generationReferenceImages(node = activeGenerateNodeId()) {
  const refs = []
  const state = workflowGenerateNodeStates.value[node]
  const inlineControlImages = normalizeInlineControlImages(state?.inlineControlImages)
  if (inlineControlImages.white) {
    refs.push({ key: `${node}-inline-white`, label: '白底结构图', url: resolveImageUrl(inlineControlImages.white), source: 'control' })
  }
  if (inlineControlImages.depth) {
    refs.push({ key: `${node}-inline-depth`, label: '深度图', url: resolveImageUrl(inlineControlImages.depth), source: 'control' })
  }
  const sceneOutputNode = state?.sceneControlOutputNode && nodeType(state.sceneControlOutputNode) === 'output'
    ? state.sceneControlOutputNode
    : ''
  if (sceneOutputNode) {
    const urls = outputNodeResultUrls(sceneOutputNode)
    if (urls.white) {
      refs.push({ key: `${sceneOutputNode}-white`, label: '白底结构图', url: resolveImageUrl(urls.white), source: 'control' })
    }
    if (urls.depth) {
      refs.push({ key: `${sceneOutputNode}-depth`, label: '深度图', url: resolveImageUrl(urls.depth), source: 'control' })
    }
  }
  workflowLinks.value
    .filter(link => link.to === node)
    .forEach((link) => {
      const type = nodeType(link.from)
      if (type === 'output') {
        const outputState = outputNodeStateFor(link.from)
        const urls = outputNodeResultUrls(link.from)
        if (outputState.controlMode === 'scene') {
          if (urls.white) {
            refs.push({ key: `${link.from}-white`, label: '白底结构图', url: resolveImageUrl(urls.white), source: 'control' })
          }
          if (urls.depth) {
            refs.push({ key: `${link.from}-depth`, label: '深度图', url: resolveImageUrl(urls.depth), source: 'control' })
          }
          return
        }
        if (urls.white) {
          refs.push({ key: `${link.from}-white`, label: '白底结构图', url: resolveImageUrl(urls.white), source: 'control' })
        }
        if (urls.depth) {
          refs.push({ key: `${link.from}-depth`, label: '深度图', url: resolveImageUrl(urls.depth), source: 'control' })
        }
        return
      }
      if (type === 'scene' && sceneImage.value.url) {
        refs.push({ key: `${link.from}-scene`, label: '场景图', url: resolveImageUrl(sceneImage.value.url) })
        return
      }
      if (type === 'generate') {
        const url = primaryGenerationImage(link.from)
        if (url) {
          const fromResult = normalizeGenerationResult(ensureGenerateNodeState(link.from).result)
          const isStyleReference = fromResult.imageSources?.[0] === 'style-reference'
          const isMaterialReference = fromResult.imageSources?.[0] === 'material-reference'
          refs.push({
            key: `${link.from}-generate-primary`,
            label: isMaterialReference ? '材质参考' : (isStyleReference ? '风格参考' : '生图主图'),
            url,
            source: isMaterialReference ? 'material' : (isStyleReference ? 'style' : 'generate')
          })
        }
        return
      }
      if (type === 'text' || type === 'llm') {
        const text = workflowTextOutput(link.from)
        if (text) {
          refs.push({
            key: `${link.from}-text`,
            type: 'text',
            label: '提示词',
            text
          })
        }
      }
    })

  const seen = new Set()
  return refs.filter((item) => {
    const key = item.type === 'text' ? `text:${item.text}` : item.url
    if (!key || seen.has(key)) return false
    seen.add(key)
    return true
  })
}

function generationReferenceSummary(node = activeGenerateNodeId()) {
  const refs = generationReferenceImages(node)
  const imageCount = refs.filter(item => item.type !== 'text').length
  const textCount = refs.filter(item => item.type === 'text').length
  if (!imageCount && !textCount) return '未接入参考图或提示词'
  return [
    imageCount ? `${imageCount} 张参考图` : '',
    textCount ? `${textCount} 个提示词` : ''
  ].filter(Boolean).join(' + ')
}

function generationPromptReferences(node = activeGenerateNodeId()) {
  return workflowLinks.value
    .filter(link => link.to === node && ['text', 'llm'].includes(nodeType(link.from)))
    .map(link => workflowTextOutput(link.from))
    .filter(Boolean)
}

function videoInputReferenceImages(node) {
  return workflowLinks.value
    .filter(item => item.to === node && nodeType(item.from) === 'generate')
    .map((link, index) => {
      const url = primaryGenerationImage(link.from)
      if (!url) return null
      const fromResult = normalizeGenerationResult(ensureGenerateNodeState(link.from).result)
      const isStyleReference = fromResult.imageSources?.[0] === 'style-reference'
      const isMaterialReference = fromResult.imageSources?.[0] === 'material-reference'
      return {
        key: `${link.from}-video-reference`,
        label: isMaterialReference ? '材质参考' : (isStyleReference ? '风格参考' : `参考图 ${index + 1}`),
        url: resolveImageUrl(url),
        source: 'generate'
      }
    })
    .filter(Boolean)
    .slice(0, 9)
}

function videoReferenceSummary(node) {
  const imageCount = videoImageReferences(node).length
  const textCount = videoTextReferences(node).length
  if (!imageCount && !textCount) return '请连接图片或提示词节点'
  return [
    imageCount ? `${imageCount} 张参考图` : '',
    textCount ? `${textCount} 个提示词` : ''
  ].filter(Boolean).join(' + ')
}

function sceneProductControlForGenerateNode(node = activeGenerateNodeId()) {
  const state = ensureGenerateNodeState(node)
  let outputNode = state.sceneControlOutputNode && nodeType(state.sceneControlOutputNode) === 'output'
    ? state.sceneControlOutputNode
    : ''
  if (!outputNode) {
    const legacyLink = workflowLinks.value.find((item) => {
      if (item.to !== node || nodeType(item.from) !== 'output') return false
      const outputState = outputNodeStateFor(item.from)
      return outputState.controlMode === 'scene' && !!outputState.resultUrls?.white && !!outputState.resultUrls?.depth
    })
    outputNode = legacyLink?.from || ''
    if (outputNode) {
      updateGenerateNode(node, { sceneControlOutputNode: outputNode }, { persist: false })
      workflowLinks.value = workflowLinks.value.filter(link => !(link.from === outputNode && link.to === node))
    }
  }
  if (!outputNode) return null
  const outputState = outputNodeStateFor(outputNode)
  if (outputState.controlMode !== 'scene' || !outputState.resultUrls?.white || !outputState.resultUrls?.depth) return null
  return {
    node: outputNode,
    sceneShot: normalizeModelSceneShot(outputState.sceneShot),
    sceneType: normalizeModelSceneType(outputState.sceneType),
    sceneStyle: normalizeModelSceneStyle(outputState.sceneStyle),
    sceneMaterial: normalizeModelSceneMaterial(outputState.sceneMaterial),
    sceneStyleRef: normalizeModelSceneStyleRef(outputState.sceneStyleRef),
    viewGuide: outputState.viewGuide || null
  }
}

function sceneProductInstruction(node = activeGenerateNodeId()) {
  const control = sceneProductControlForGenerateNode(node)
  if (!control) return ''
  return buildCommercialScenePrompt(control)
}

function withProductWhiteBgGeometryGuard(node, prompt) {
  const text = String(prompt || '').trim()
  if (ensureGenerateNodeState(node).modelWorkflowImageType !== 'product-white') return text
  if (text.includes(PRODUCT_GEOMETRY_RULES)) return text
  return [text, `## 产品几何（最高权重）\n${PRODUCT_GEOMETRY_RULES}`].filter(Boolean).join('\n\n')
}

async function prepareSceneWorkflowPromptForGeneration(node = activeGenerateNodeId()) {
  const control = sceneProductControlForGenerateNode(node)
  if (!control) return effectiveGenerationPrompt(node)
  const outputNode = control.node
  const outputState = outputNodeStateFor(outputNode)
  const modelNode = outputState.sourceNode || ''
  let styleRef = outputState.sceneStyleRef
  if (modelNode) {
    styleRef = await ensureModelSceneStyleColorPrompt(modelNode)
    workflowOutputNodeStates.value = {
      ...workflowOutputNodeStates.value,
      [outputNode]: normalizeOutputNodeState({
        ...outputState,
        sceneStyleRef: styleRef
      })
    }
  }
  const prompt = buildCommercialScenePrompt({
    ...control,
    sceneStyleRef: styleRef || control.sceneStyleRef
  })
  updateGenerateNode(node, { prompt }, { persist: false })
  return prompt
}

function effectiveGenerationPrompt(node = activeGenerateNodeId()) {
  const promptRefs = generationPromptReferences(node)
  const basePrompt = (promptRefs.length ? promptRefs.join('\n') : ensureGenerateNodeState(node).prompt).trim()
  const instruction = basePrompt ? '' : sceneProductInstruction(node)
  return withProductWhiteBgGeometryGuard(node, [basePrompt, instruction].filter(Boolean).join('\n'))
}

function canGenerateWorkflowImage(node = activeGenerateNodeId()) {
  const state = ensureGenerateNodeState(node)
  return !!effectiveDrawingModelName(node) && !state.generating && !!effectiveGenerationPrompt(node)
}

function videoNodeQualityLabel(node) {
  const state = ensureVideoNodeState(node)
  return `${videoAspectShortLabel(state.aspectRatio)} · ${state.resolution.toUpperCase()} · ${state.duration}s`
}

function videoModelLabel(value) {
  return videoModelOptions.find(item => item.value === value)?.label || value || '选择模型'
}

function videoAspectShortLabel(value) {
  if (videoAspectRatioOptions.some(item => item.value === value)) return value
  switch (value) {
    case 'film_horizontal_21_9':
      return '21:9'
    case 'widescreen_16_9':
      return '16:9'
    case 'classic_4_3':
      return '4:3'
    case 'square_1_1':
      return '1:1'
    case 'traditional_3_4':
      return '3:4'
    case 'social_story_9_16':
      return '9:16'
    case 'film_vertical_9_21':
      return '9:21'
    default:
      return '16:9'
  }
}

function videoAspectIconClass(value) {
  return aspectIconClass(videoAspectShortLabel(value))
}

function effectiveVideoPrompt(node) {
  const state = ensureVideoNodeState(node)
  const linkedPrompts = linkedWorkflowTextOutputs(node)
  return (linkedPrompts.length ? linkedPrompts.join('\n') : String(state.prompt || '')).trim()
}

function videoImageReferences(node) {
  return videoInputReferenceImages(node)
    .filter(item => item.url)
}

function videoTextReferences(node) {
  return workflowLinks.value
    .filter(link => link.to === node && ['text', 'llm'].includes(nodeType(link.from)))
    .map(link => ({
      key: `${link.from}-video-text`,
      type: 'text',
      label: nodeType(link.from) === 'llm' ? '大模型提示词' : '文本提示词',
      text: workflowTextOutput(link.from)
    }))
    .filter(item => item.text)
}

function videoEditorReferences(node) {
  return [...videoImageReferences(node), ...videoTextReferences(node)]
}

function canRunVideoNode(node) {
  const state = ensureVideoNodeState(node)
  return !!effectiveVideoPrompt(node) && !state.loading
}

function isVideoTaskCompleteStatus(status) {
  return ['COMPLETED', 'COMPLETE', 'SUCCESS', 'SUCCEEDED', 'FINISHED', 'DONE'].includes(status)
}

function isVideoTaskFailedStatus(status) {
  return ['FAILED', 'FAIL', 'ERROR', 'CANCELED', 'CANCELLED', 'EXPIRED'].includes(status)
}

function extractVideoResultUrl(data = {}) {
  return data.video_url ||
    data.videoUrl ||
    data.output?.video_url ||
    data.output?.videoUrl ||
    data.data?.video_url ||
    data.data?.videoUrl ||
    data.result?.video_url ||
    data.result?.videoUrl ||
    data.output?.url ||
    data.data?.url ||
    data.result?.url ||
    data.url ||
    data.image_url ||
    data.imageUrl ||
    data.source_url ||
    data.sourceUrl ||
    ''
}

async function pollVideoNodeTask(node, model, taskId, meta = {}) {
  for (let i = 0; i < 300; i += 1) {
    const state = workflowVideoNodeStates.value[node]
    if (!state?.loading || state.taskId !== taskId) return ''
    const res = await request.get(`/video/tasks/${encodeURIComponent(model)}/${encodeURIComponent(taskId)}`, {
      params: {
        prompt: meta.prompt || '',
        resolution: meta.resolution || '',
        started_at: meta.startedAt || ''
      }
    })
    const status = String(res.data?.status || '').toUpperCase()
    const videoUrl = extractVideoResultUrl(res.data || {})
    updateVideoNode(node, {
      status: status || state.status,
      rawResponse: res.data?.raw_response || state.rawResponse,
      progress: Math.min(98, Math.max(Number(state.progress) || 0, status === 'IN_PROGRESS' ? 32 + i * 2 : 16 + i))
    }, { persist: false, transient: true })
    if (isVideoTaskCompleteStatus(status)) {
      if (!videoUrl) throw new Error('视频任务已完成，但未返回视频地址')
      return videoUrl
    }
    if (isVideoTaskFailedStatus(status)) throw new Error('视频生成失败')
    if (videoUrl) return videoUrl
    await new Promise(resolve => window.setTimeout(resolve, 3000))
  }
  throw new Error('视频生成超时，请稍后重试')
}

async function videoReferenceImagesPayload(node) {
  const payload = []
  for (const item of videoImageReferences(node).slice(0, 9)) {
    const ref = await imageUrlToRef(item.url)
    if (ref?.data) payload.push(`data:${ref.mime_type || 'image/png'};base64,${ref.data}`)
  }
  return payload
}

async function uploadVideoNodeMedia(node, field, event) {
  const files = Array.from(event.target.files || [])
  try {
    if (field === 'firstFrame' || field === 'lastFrame') {
      const file = files[0]
      if (file?.type.startsWith('image/')) updateVideoNode(node, { [field]: await uploadVideoReferenceFile(file) })
    } else {
      const type = field === 'referenceVideos' ? 'video/' : 'audio/'
      const current = ensureVideoNodeState(node)[field] || []
      const additions = []
      for (const file of files.filter(item => item.type.startsWith(type)).slice(0, 3 - current.length)) additions.push(await uploadVideoReferenceFile(file))
      updateVideoNode(node, { [field]: [...current, ...additions] })
    }
  } catch (e) {
    updateVideoNode(node, { error: e?.message || '读取参考素材失败' }, { persist: false })
  } finally {
    event.target.value = ''
  }
}

async function uploadVideoReferenceFile(file) {
  const form = new FormData()
  form.append('file', file)
  form.append('mime_type', file.type)
  const res = await request.post('/upload/image', form)
  return res.data?.url || ''
}

async function runVideoNode(node, options = {}) {
  const state = ensureVideoNodeState(node)
  if (state.loading) return
  const promptText = effectiveVideoPrompt(node)
  if (!promptText) {
    updateVideoNode(node, { error: '请输入视频提示词' }, { persist: false })
    return
  }
  if (options.undo !== false && runningWorkflowGroupForNode(node)) {
    error.value = '工作流运行中，请在组合上控制运行'
    return
  }
  if (options.undo !== false) invalidatePausedWorkflowRunsForNodes([node], { structural: true })
  const startedAt = Date.now()
  if (options.undo !== false) pushWorkflowUndoSnapshot()
  updateVideoNode(node, {
    loading: true,
    progress: 0,
    error: '',
    videoUrl: '',
    taskId: '',
    status: '',
    rawResponse: null
  }, { persist: false })
  const progressTimer = window.setInterval(() => {
    const current = workflowVideoNodeStates.value[node]
    if (!current?.loading) {
      window.clearInterval(progressTimer)
      return
    }
    updateVideoNode(node, {
      progress: Math.min(96, (Number(current.progress) || 0) + 6)
    }, { persist: false, transient: true })
  }, 500)
  try {
    const current = ensureVideoNodeState(node)
    const referenceImages = await videoReferenceImagesPayload(node)
    const res = await request.post('/video/generate', {
      prompt: promptText,
      model: current.model,
      resolution: current.resolution,
      aspect_ratio: current.aspectRatio,
      duration: current.duration,
      generate_audio: current.generateAudio,
      camera_fixed: current.cameraFixed,
      seed: current.seed,
      image: state.inputMode === 'frames' ? state.firstFrame : '',
      image_end: state.inputMode === 'frames' ? state.lastFrame : '',
      reference_images: state.inputMode === 'reference' ? referenceImages : [],
      reference_videos: state.referenceVideos || [],
      reference_audios: state.referenceAudios || []
    })
    const videoUrl = extractVideoResultUrl(res.data || {})
    const taskId = res.data?.task_id || ''
    const taskModel = res.data?.model || `${current.model}-${current.resolution}`
    if (!videoUrl && !taskId) throw new Error('未获取到生成视频')
    if (taskId && !videoUrl) {
      updateVideoNode(node, {
        taskId,
        status: String(res.data?.status || 'CREATED').toUpperCase(),
        rawResponse: res.data?.raw_response || null
      }, { persist: false })
      const completedUrl = await pollVideoNodeTask(node, taskModel, taskId, {
        prompt: promptText,
        resolution: current.resolution,
        startedAt
      })
      if (!completedUrl) return
      updateVideoNode(node, {
        loading: false,
        progress: 100,
        error: '',
        videoUrl: completedUrl,
        taskId,
        status: 'COMPLETED',
        rawResponse: workflowVideoNodeStates.value[node]?.rawResponse || res.data?.raw_response || null
      })
      appendWorkflowHistoryAssets([{ url: completedUrl, createdAt: new Date().toISOString(), node, taskId, type: 'video' }])
      return
    }
    updateVideoNode(node, {
      loading: false,
      progress: 100,
      error: '',
      videoUrl,
      taskId,
      status: String(res.data?.status || (videoUrl ? 'COMPLETED' : '')).toUpperCase(),
      rawResponse: res.data?.raw_response || null
    })
    if (videoUrl) {
      appendWorkflowHistoryAssets([{ url: videoUrl, createdAt: new Date().toISOString(), node, taskId, type: 'video' }])
    }
  } catch (e) {
    const errData = e?.response?.data
    const errMsg = typeof errData === 'string' ? errData : (errData?.error || e?.message || '视频生成失败')
    updateVideoNode(node, {
      loading: false,
      progress: 0,
      error: errMsg
    })
  } finally {
    window.clearInterval(progressTimer)
  }
}

function canRegenerateGeneratedImageNode(node) {
  if (nodeType(node) !== 'generate') return false
  const state = ensureGenerateNodeState(node)
  const result = normalizeGenerationResult(state.result)
  if (state.generating || state.uploadOnly || isReferenceOnlyGenerateNode(node)) return false
  const generationFailed = ['failed', 'error', 'cancelled'].includes(String(result.status || '').toLowerCase()) || !!state.error
  if (!result.images.length && !generationFailed && state.modelWorkflowImageType !== 'product-white') return false
  // 3D 节点生成的产品白底图（包括 360°图片节点）保留控制图，可直接复用现有参数重新生成。
  if (state.modelWorkflowImageType && state.modelWorkflowImageType !== 'product-white') return false
  const blockedSources = new Set(['batch-recolor', 'batch-translate', 'material-reference', 'style-reference'])
  if (result.imageSources.some(source => blockedSources.has(source))) return false
  return !!effectiveDrawingModelName(node) && !!effectiveGenerationPrompt(node)
}

function regenerateGeneratedImageNode(node) {
  if (!canRegenerateGeneratedImageNode(node)) return
  const hasImageReference = generationReferenceImages(node).some(ref => ref.type !== 'text' && ref.url)
  generateWorkflowImage(node, {
    replaceCurrentNode: true,
    includeSelfReference: !hasImageReference,
    ignoreSavedEditRef: true
  })
}

function generateEditCanvasEl() {
  const refValue = generateEditCanvasRef.value
  return Array.isArray(refValue) ? refValue[0] : refValue
}

function isGenerateImageEditing(node) {
  return generateImageEditState.value.node === node
}

function setGenerateEditTool(tool) {
  generateImageEditState.value = {
    ...generateImageEditState.value,
    tool
  }
}

function resetGenerateImageEditState(patch = {}) {
  generateImageEditState.value = {
    node: '',
    tool: 'brush',
    color: IMAGE_EDIT_BRUSH_DEFAULTS.color,
    size: DEFAULT_GENERATE_EDIT_BRUSH_SIZE,
    drawing: false,
    startX: 0,
    startY: 0,
    lastX: 0,
    lastY: 0,
    undoStack: [],
    redoStack: [],
    ...patch
  }
}

function resizeGenerateEditCanvas(canvas) {
  if (!canvas) return false
  const rect = canvas.getBoundingClientRect()
  const width = Math.max(1, Math.round(rect.width))
  const height = Math.max(1, Math.round(rect.height))
  if (canvas.width === width && canvas.height === height) return true
  canvas.width = width
  canvas.height = height
  generateEditBaseSnapshot = null
  return true
}

function generateEditPoint(e, canvas = generateEditCanvasEl()) {
  const rect = canvas?.getBoundingClientRect?.()
  if (!rect) return { x: 0, y: 0 }
  return {
    x: e.clientX - rect.left,
    y: e.clientY - rect.top
  }
}

function generateEditCtx(canvas = generateEditCanvasEl()) {
  if (!canvas) return null
  resizeGenerateEditCanvas(canvas)
  return canvas.getContext('2d')
}

function pushGenerateEditUndoSnapshot() {
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (!canvas || !ctx) return
  const snapshot = markRaw(ctx.getImageData(0, 0, canvas.width, canvas.height))
  generateImageEditState.value = {
    ...generateImageEditState.value,
    undoStack: [...generateImageEditState.value.undoStack, snapshot].slice(-30),
    redoStack: []
  }
}

function restoreGenerateEditSnapshot(snapshot) {
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (!canvas || !ctx || !snapshot) return
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  ctx.putImageData(snapshot, 0, 0)
}

function undoGenerateImageEdit() {
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (!canvas || !ctx || !generateImageEditState.value.undoStack.length) return
  const current = markRaw(ctx.getImageData(0, 0, canvas.width, canvas.height))
  const undoStack = [...generateImageEditState.value.undoStack]
  const snapshot = undoStack.pop()
  generateImageEditState.value = {
    ...generateImageEditState.value,
    undoStack,
    redoStack: [...generateImageEditState.value.redoStack, current].slice(-30)
  }
  restoreGenerateEditSnapshot(snapshot)
}

function redoGenerateImageEdit() {
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (!canvas || !ctx || !generateImageEditState.value.redoStack.length) return
  const current = markRaw(ctx.getImageData(0, 0, canvas.width, canvas.height))
  const redoStack = [...generateImageEditState.value.redoStack]
  const snapshot = redoStack.pop()
  generateImageEditState.value = {
    ...generateImageEditState.value,
    redoStack,
    undoStack: [...generateImageEditState.value.undoStack, current].slice(-30)
  }
  restoreGenerateEditSnapshot(snapshot)
}

function generateEditMaskColor(alpha = GENERATE_EDIT_MASK_ALPHA) {
  const color = String(generateImageEditState.value.color || '#ff2020').trim()
  const hex = color.replace('#', '')
  if (/^[0-9a-f]{3}$/i.test(hex)) {
    const r = parseInt(hex[0] + hex[0], 16)
    const g = parseInt(hex[1] + hex[1], 16)
    const b = parseInt(hex[2] + hex[2], 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  if (/^[0-9a-f]{6}$/i.test(hex)) {
    const r = parseInt(hex.slice(0, 2), 16)
    const g = parseInt(hex.slice(2, 4), 16)
    const b = parseInt(hex.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  return `rgba(255, 32, 32, ${alpha})`
}

function drawGenerateEditStroke(ctx, points) {
  ctx.save()
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.strokeStyle = generateEditMaskColor()
  ctx.fillStyle = generateEditMaskColor()
  ctx.lineWidth = Number(generateImageEditState.value.size) || DEFAULT_GENERATE_EDIT_BRUSH_SIZE
  drawSmoothBrushStroke(ctx, points, ctx.lineWidth)
  ctx.restore()
}

function drawGenerateEditRect(ctx, startX, startY, endX, endY) {
  ctx.save()
  ctx.strokeStyle = generateEditMaskColor()
  ctx.lineWidth = Math.max(2, Number(generateImageEditState.value.size) || DEFAULT_GENERATE_EDIT_BRUSH_SIZE)
  ctx.setLineDash([8, 6])
  ctx.strokeRect(startX, startY, endX - startX, endY - startY)
  ctx.restore()
}

function drawGenerateEditStraightLine(ctx, startX, startY, endX, endY) {
  ctx.save()
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.strokeStyle = generateEditMaskColor()
  ctx.lineWidth = Math.max(2, Number(generateImageEditState.value.size) || DEFAULT_GENERATE_EDIT_BRUSH_SIZE)
  ctx.beginPath()
  ctx.moveTo(startX, startY)
  ctx.lineTo(endX, endY)
  ctx.stroke()
  ctx.restore()
}

function drawGenerateEditArrow(ctx, startX, startY, endX, endY) {
  const angle = Math.atan2(endY - startY, endX - startX)
  const distance = Math.hypot(endX - startX, endY - startY)
  if (distance < 2) return
  const lineWidth = Math.max(2, Number(generateImageEditState.value.size) || DEFAULT_GENERATE_EDIT_BRUSH_SIZE)
  const headLength = Math.min(Math.max(lineWidth * 3.2, 14), Math.max(14, distance * 0.45))
  ctx.save()
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  const maskColor = generateEditMaskColor()
  ctx.strokeStyle = maskColor
  ctx.fillStyle = maskColor
  ctx.lineWidth = lineWidth
  ctx.beginPath()
  ctx.moveTo(startX, startY)
  ctx.lineTo(endX, endY)
  ctx.stroke()
  ctx.beginPath()
  ctx.moveTo(endX, endY)
  ctx.lineTo(
    endX - headLength * Math.cos(angle - Math.PI / 7),
    endY - headLength * Math.sin(angle - Math.PI / 7)
  )
  ctx.lineTo(
    endX - headLength * Math.cos(angle + Math.PI / 7),
    endY - headLength * Math.sin(angle + Math.PI / 7)
  )
  ctx.closePath()
  ctx.fill()
  ctx.restore()
}

function startGenerateImageEdit(node) {
  if (!canEditGenerateNodeImage(node)) return
  if (!primaryGenerationImage(node)) return
  activeWorkflowNode.value = node
  closeGenerateResults(node)
  resetGenerateImageEditState({ node })
  nextTick(() => {
    const canvas = generateEditCanvasEl()
    if (!canvas) return
    resizeGenerateEditCanvas(canvas)
    const ctx = canvas.getContext('2d')
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    generateEditBaseSnapshot = markRaw(ctx.getImageData(0, 0, canvas.width, canvas.height))
  })
}

function stopGenerateImageEdit() {
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (canvas && ctx) ctx.clearRect(0, 0, canvas.width, canvas.height)
  generateEditBaseSnapshot = null
  generateEditStrokePoints = []
  resetGenerateImageEditState()
}

function startGenerateEditDraw(e) {
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (!canvas || !ctx || e.button !== 0) return
  const point = generateEditPoint(e, canvas)
  pushGenerateEditUndoSnapshot()
  generateEditBaseSnapshot = markRaw(ctx.getImageData(0, 0, canvas.width, canvas.height))
  generateEditStrokePoints = [point]
  generateImageEditState.value = {
    ...generateImageEditState.value,
    drawing: true,
    startX: point.x,
    startY: point.y,
    lastX: point.x,
    lastY: point.y
  }
  if (generateImageEditState.value.tool === 'brush') {
    drawGenerateEditStroke(ctx, generateEditStrokePoints)
  }
  canvas.setPointerCapture?.(e.pointerId)
}

function moveGenerateEditDraw(e) {
  const state = generateImageEditState.value
  if (!state.drawing) return
  const canvas = generateEditCanvasEl()
  const ctx = generateEditCtx(canvas)
  if (!canvas || !ctx) return
  const point = generateEditPoint(e, canvas)
  if (state.tool === 'brush') {
    generateEditStrokePoints.push(point)
    if (generateEditBaseSnapshot) {
      ctx.clearRect(0, 0, canvas.width, canvas.height)
      ctx.putImageData(generateEditBaseSnapshot, 0, 0)
    }
    drawGenerateEditStroke(ctx, generateEditStrokePoints)
    return
  }
  if ((state.tool === 'rect' || state.tool === 'line' || state.tool === 'arrow') && generateEditBaseSnapshot) {
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.putImageData(generateEditBaseSnapshot, 0, 0)
    if (state.tool === 'rect') {
      drawGenerateEditRect(ctx, state.startX, state.startY, point.x, point.y)
    } else if (state.tool === 'line') {
      drawGenerateEditStraightLine(ctx, state.startX, state.startY, point.x, point.y)
    } else {
      drawGenerateEditArrow(ctx, state.startX, state.startY, point.x, point.y)
    }
  }
}

function endGenerateEditDraw(e) {
  const canvas = generateEditCanvasEl()
  const state = generateImageEditState.value
  const ctx = generateEditCtx(canvas)
  if (canvas && ctx && state.drawing && generateEditBaseSnapshot && (state.tool === 'rect' || state.tool === 'line' || state.tool === 'arrow')) {
    const point = generateEditPoint(e, canvas)
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.putImageData(generateEditBaseSnapshot, 0, 0)
    if (state.tool === 'rect') {
      drawGenerateEditRect(ctx, state.startX, state.startY, point.x, point.y)
    } else if (state.tool === 'line') {
      drawGenerateEditStraightLine(ctx, state.startX, state.startY, point.x, point.y)
    } else {
      drawGenerateEditArrow(ctx, state.startX, state.startY, point.x, point.y)
    }
  }
  if (canvas) canvas.releasePointerCapture?.(e.pointerId)
  generateImageEditState.value = {
    ...generateImageEditState.value,
    drawing: false
  }
  generateEditBaseSnapshot = null
  generateEditStrokePoints = []
}

async function createGenerateEditMaskReference(node) {
  const editCanvas = generateEditCanvasEl()
  if (!primaryGenerationImage(node) || !editCanvas) return null
  const width = editCanvas.width
  const height = editCanvas.height
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  ctx.clearRect(0, 0, width, height)
  ctx.drawImage(editCanvas, 0, 0, width, height)
  const dataUrl = canvas.toDataURL('image/png')
  return {
    url: dataUrl,
    data: dataUrl.split(',')[1] || '',
    mime_type: 'image/png'
  }
}

async function createGenerateEditPreviewReference(node, editRef = null) {
  const imageUrl = primaryGenerationImage(node)
  const editCanvas = generateEditCanvasEl()
  if (!imageUrl || (!editCanvas && !editRef?.url)) return null
  const sourceRef = await imageUrlToRef(imageUrl)
  const sourceImage = await loadImage(`data:${sourceRef.mime_type || 'image/png'};base64,${sourceRef.data}`)
  const editSource = editRef?.url ? await loadImage(editRef.url) : editCanvas
  const width = sourceImage.naturalWidth || sourceImage.width || editCanvas?.width || 1
  const height = sourceImage.naturalHeight || sourceImage.height || editCanvas?.height || 1
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  ctx.drawImage(sourceImage, 0, 0, width, height)
  const sourceRatio = width / Math.max(1, height)
  const editWidth = editSource.naturalWidth || editSource.width || editCanvas?.width || width
  const editHeight = editSource.naturalHeight || editSource.height || editCanvas?.height || height
  const canvasRatio = editWidth / Math.max(1, editHeight)
  let drawX = 0
  let drawY = 0
  let drawW = editWidth
  let drawH = editHeight
  if (canvasRatio > sourceRatio) {
    drawH = editHeight
    drawW = drawH * sourceRatio
    drawX = (editWidth - drawW) / 2
  } else {
    drawW = editWidth
    drawH = drawW / sourceRatio
    drawY = (editHeight - drawH) / 2
  }
  ctx.drawImage(editSource, drawX, drawY, drawW, drawH, 0, 0, width, height)
  const dataUrl = canvas.toDataURL('image/png')
  return {
    url: dataUrl,
    data: dataUrl.split(',')[1] || '',
    mime_type: 'image/png'
  }
}

function applyGenerateEditPreviewToNode(node, editPreview, editRef) {
  const state = ensureGenerateNodeState(node)
  const previewUrl = editPreview?.url ? resolveImageUrl(editPreview.url) : ''
  if (!previewUrl) return
  const previousPosition = nodePositions.value[node] ? { ...nodePositions.value[node] } : null
  const hasLinkedGeneratedParent = !!state.generatedFromNode && workflowLinks.value.some(link => link.from === state.generatedFromNode && link.to === node)

  const result = normalizeGenerationResult(state.result)
  if (result.images.length) {
    const nextImages = [...result.images]
    nextImages[0] = previewUrl
    const nextImageSources = [...(result.imageSources || [])]
    nextImageSources[0] = 'edited'
    updateGenerateNode(node, {
      result: {
        ...state.result,
        url: previewUrl,
        images: nextImages,
        imageCreatedAt: nextImages.map((url, index) => result.imageCreatedAt[index] || result.createdAt || new Date().toISOString()),
        imageSources: nextImages.map((url, index) => nextImageSources[index] || 'generated')
      },
      editRef: null,
      editPreview: null,
      generatedFromNode: hasLinkedGeneratedParent ? state.generatedFromNode : '',
      error: ''
    }, { persist: false })
    if (previousPosition) {
      nodePositions.value = {
        ...nodePositions.value,
        [node]: previousPosition
      }
    }
    return
  }

  const uploads = state.uploads || []
  if (uploads.length) {
    updateGenerateNode(node, {
      uploads: [{ ...uploads[0], url: previewUrl }, ...uploads.slice(1)],
      editRef: null,
      editPreview: null,
      error: ''
    }, { persist: false })
    return
  }

  updateGenerateNode(node, {
    editRef,
    editPreview,
    error: ''
  }, { persist: false })
}

async function submitGenerateImageEdit(node) {
  const editedRef = await createGenerateEditMaskReference(node)
  if (!editedRef?.url) return
  stopGenerateImageEdit()
  window.setTimeout(() => {
    createGenerateEditPreviewReference(node, editedRef)
      .then((editPreview) => {
        if (!editPreview?.url) return
        applyGenerateEditPreviewToNode(node, editPreview, editedRef)
        saveActiveProject({ persist: false })
        persistActiveProject({ force: true })
      })
      .catch((e) => {
        updateGenerateNode(node, {
          error: e?.message || '保存编辑失败'
        }, { persist: false })
      })
  }, 0)
}

async function imageUrlToRef(url) {
  if (!url) return null
  let fetchUrl = resolveImageUrl(url)

  if (/^data:/i.test(fetchUrl)) {
    const res = await fetch(fetchUrl)
    const blob = await res.blob()
    const data = await blobToBase64(blob)
    return {
      data,
      mime_type: normalizeImageMime(blob.type, fetchUrl)
    }
  }

  if (fetchUrl.includes('oss-cn-')) {
    const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
    fetchUrl = `${apiBase}/proxy?url=${encodeURIComponent(fetchUrl)}`
  } else if (!/^https?:\/\//i.test(fetchUrl) && !/^blob:/i.test(fetchUrl)) {
    const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
    fetchUrl = apiBase.startsWith('http') ? `${new URL(apiBase).origin}${fetchUrl}` : fetchUrl
  }

  const token = localStorage.getItem('token')
  const res = await fetch(fetchUrl, {
    headers: token ? { Authorization: `Bearer ${token}` } : {}
  })
  if (!res.ok) throw new Error(`参考图加载失败：HTTP ${res.status}`)
  const blob = await res.blob()
  const data = await blobToBase64(blob)
  return {
    data,
    mime_type: normalizeImageMime(blob.type, fetchUrl || url)
  }
}

async function imageUrlToGenerationRef(url) {
  if (!url) return null
  const resolved = resolveImageUrl(url)
  if (/^https?:\/\//i.test(resolved) || resolved.startsWith('/uploads/')) {
    return {
      url: resolved,
      mime_type: normalizeImageMime('', resolved)
    }
  }
  const res = await fetch(resolved)
  if (!res.ok) throw new Error(`参考图加载失败：HTTP ${res.status}`)
  const blob = await res.blob()
  const mimeType = normalizeImageMime(blob.type, resolved || url)
  const form = new FormData()
  form.append('image', blob, `ref.${extensionFromMimeType(mimeType)}`)
  form.append('mime_type', mimeType)
  const uploadRes = await request.post('/upload/image', form)
  const uploadedUrl = uploadRes.data?.url || ''
  return uploadedUrl ? {
    url: uploadedUrl,
    mime_type: mimeType
  } : null
}

function extensionFromMimeType(mimeType = 'image/png') {
  if (mimeType.includes('jpeg') || mimeType.includes('jpg')) return 'jpg'
  if (mimeType.includes('webp')) return 'webp'
  if (mimeType.includes('gif')) return 'gif'
  return 'png'
}

function blobToBase64(blob) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || '').split(',')[1] || '')
    reader.onerror = reject
    reader.readAsDataURL(blob)
  })
}

function normalizeImageMime(mimeType = '', url = '') {
  const normalized = String(mimeType || '').split(';')[0].trim().toLowerCase()
  if (/^image\/(png|jpe?g|webp)$/i.test(normalized)) {
    return normalized === 'image/jpg' ? 'image/jpeg' : normalized
  }

  const cleanUrl = String(url || '').split('?')[0].toLowerCase()
  if (cleanUrl.endsWith('.jpg') || cleanUrl.endsWith('.jpeg')) return 'image/jpeg'
  if (cleanUrl.endsWith('.webp')) return 'image/webp'
  if (cleanUrl.endsWith('.png')) return 'image/png'
  return 'image/png'
}

async function buildGenerationRefPayload(node = activeGenerateNodeId(), selfImage = primaryGenerationImage(node)) {
  const sceneControl = sceneProductControlForGenerateNode(node)
  const allRefs = generationReferenceImages(node)
  const sourceRefs = sceneControl
    ? [
        ...allRefs.filter(ref => ref.source === 'control'),
        ...allRefs.filter(ref => ref.source === 'material')
      ]
    : [
        ...(selfImage ? [{ key: 'self', label: '节点主图', url: resolveImageUrl(selfImage) }] : []),
        ...allRefs
      ]
  const refs = [
    ...sourceRefs
  ].filter(ref => ref?.url && !ref.previewOnly)
  const seen = new Set()
  const payload = []
  const inlineControlImages = normalizeInlineControlImages(ensureGenerateNodeState(node).inlineControlImages)
  const nextInlineControlImages = { ...inlineControlImages }
  let inlineControlChanged = false
  for (const ref of refs) {
    if (!ref.url || seen.has(ref.url)) continue
    seen.add(ref.url)
    const item = await imageUrlToGenerationRef(ref.url)
    if (item) {
      payload.push(item)
      if (ref.key === `${node}-inline-white` && nextInlineControlImages.white !== item.url) {
        nextInlineControlImages.white = item.url
        inlineControlChanged = true
      }
      if (ref.key === `${node}-inline-depth` && nextInlineControlImages.depth !== item.url) {
        nextInlineControlImages.depth = item.url
        inlineControlChanged = true
      }
    }
  }
  if (inlineControlChanged) {
    updateGenerateNode(node, {
      inlineControlImages: nextInlineControlImages
    }, { persist: false })
  }
  return payload
}

function generationResultImages(node = activeGenerateNodeId()) {
  return normalizeGenerationResult(ensureGenerateNodeState(node).result).images
}

function generationUploadImages(node = activeGenerateNodeId()) {
  return ensureGenerateNodeState(node).uploads
    .filter(item => item?.url)
    .map(item => resolveImageUrl(item.url))
}

function generationImages(node = activeGenerateNodeId()) {
  const resultImages = generationResultImages(node)
  const uploadImages = generationUploadImages(node)
  if (isWorkflowImageInputNode(node) && uploadImages.length) return uploadImages
  return resultImages.length ? resultImages : uploadImages
}

function displayGenerationImage(node = activeGenerateNodeId(), index = 0) {
  const images = generationImages(node)
  if (index === 0) {
    const preview = ensureGenerateNodeState(node).editPreview?.url
    if (preview) return resolveImageUrl(preview)
  }
  return images[index] || ''
}

function isUploadOnlyGenerateNode(node) {
  if (nodeType(node) !== 'generate') return false
  return !!ensureGenerateNodeState(node).uploadOnly
}

function isWorkflowImageInputNode(node) {
  if (nodeType(node) !== 'generate') return false
  const state = ensureGenerateNodeState(node)
  if (state.uploadOnly) return true
  if (state.generatedFromNode || isReferenceOnlyGenerateNode(node)) return false
  if (String(state.prompt || '').trim()) return false
  if (!groupBoxes.value.some(group => (group.nodes || []).includes(node))) return false
  if (workflowLinks.value.some(link => link.to === node)) return false
  if (!workflowLinks.value.some(link => link.from === node)) return false
  return (state.uploads || []).some(item => item?.url) || normalizeGenerationResult(state.result).images.length > 0
}

function generateNodeFirstImageSource(node) {
  if (nodeType(node) !== 'generate') return ''
  return normalizeGenerationResult(ensureGenerateNodeState(node).result).imageSources?.[0] || ''
}

function isMaterialReferenceNode(node) {
  return generateNodeFirstImageSource(node) === 'material-reference'
}

function isStyleReferenceNode(node) {
  return generateNodeFirstImageSource(node) === 'style-reference'
}

function isReferenceOnlyGenerateNode(node) {
  return isMaterialReferenceNode(node) || isStyleReferenceNode(node)
}

function canEditGenerateNodeImage(node) {
  return !isReferenceOnlyGenerateNode(node)
}

function canShowGenerateEditorPanel(node) {
  const state = ensureGenerateNodeState(node)
  return activeWorkflowNode.value === node
    && !state.generating
    && !state.uploadOnly
    && !isModelWorkflowGeneratedImageNode(node)
}

function isModelWorkflowGeneratedImageNode(node) {
  if (nodeType(node) !== 'generate') return false
  const state = ensureGenerateNodeState(node)
  if (state.imageEditDisabled || state.modelWorkflowImageType) return true
  if (state.sceneWorkflowNode || state.sceneControlOutputNode) return true
  const prompt = String(state.prompt || '')
  return (
    (prompt.includes('产品白底结构图') && prompt.includes('深度图') && prompt.includes('材质图')) ||
    (prompt.includes('唯一产品造型来源') && prompt.includes('第2张参考图只辅助厚度'))
  )
}

function generateNodeTitle(node) {
  if (isMaterialReferenceNode(node)) return '材质参考'
  if (isStyleReferenceNode(node)) return '风格参考'
  if (generateNodeFirstImageSource(node) === 'depth-of-field') return '景深图'
  if (generateNodeFirstImageSource(node) === 'light-shadow') return '光影优化'
  const state = ensureGenerateNodeState(node)
  const resultTitle = normalizeGenerationResult(state.result).batchTitle || state.batchTitle
  if (resultTitle) return resultTitle
  return '图片'
}

function showGenerateUploadButton(node) {
  if (canRegenerateGeneratedImageNode(node)) return false
  if (isReferenceOnlyGenerateNode(node)) return false
  if (isWorkflowImageInputNode(node)) return activeWorkflowNode.value === node
  if (generationResultImages(node).length) return false
  return activeWorkflowNode.value === node
}

function parseHistoryDate(value) {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function formatHistoryDate(value) {
  const date = parseHistoryDate(value)
  if (!date) return ''
  const yyyy = date.getFullYear()
  const mm = String(date.getMonth() + 1).padStart(2, '0')
  const dd = String(date.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

const historyAssets = computed(() => {
  const items = []
  const seen = new Set()
  normalizeHistoryAssets(workflowHistoryAssets.value).forEach((asset) => {
    const resolved = resolveImageUrl(asset.url)
    if (!resolved || seen.has(resolved) || hiddenHistoryAssetUrls.value.has(resolved)) return
    if (/^(data:|blob:)/i.test(resolved)) return
    const date = parseHistoryDate(asset.createdAt)
    if (!date) return
    seen.add(resolved)
    const createdAtIso = date.toISOString()
    items.push({ url: resolved, date: formatHistoryDate(createdAtIso), createdAt: createdAtIso, type: asset.type })
  })

  return items.sort((a, b) => new Date(b.createdAt || 0) - new Date(a.createdAt || 0))
})

const groupedHistoryAssets = computed(() => {
  const groups = new Map()
  historyAssets.value.forEach((asset) => {
    if (!groups.has(asset.date)) groups.set(asset.date, [])
    groups.get(asset.date).push(asset)
  })
  return Array.from(groups.entries())
    .sort(([a], [b]) => b.localeCompare(a))
    .map(([date, items]) => ({ date, items }))
})

const canvasStyleLibraryTitle = computed(() => (
  canvasStyleLibraryScene.value === 'bathroom' ? '浴室场景' : '厨房场景'
))

const canvasStyleLibraryTagsWithRefs = computed(() => (
  canvasStyleLibraryTags.value.filter(tag => tag.references?.length)
))

const canvasStyleLibrarySelectedTag = computed(() => (
  canvasStyleLibraryTags.value.find(tag => tag.id === canvasStyleLibraryTagId.value) || null
))

const canvasStyleLibraryRefs = computed(() => (
  canvasStyleLibrarySelectedTag.value?.references || []
))

function primaryGenerationImage(node = activeGenerateNodeId()) {
  return displayGenerationImage(node, 0)
}

function hideHistoryAsset(url) {
  const resolved = resolveImageUrl(url)
  if (!resolved) return
  hiddenHistoryAssetUrls.value = new Set([...hiddenHistoryAssetUrls.value, resolved])
}

function appendWorkflowHistoryAssets(assets = []) {
  const nextItems = normalizeHistoryAssets(assets)
  if (!nextItems.length) return
  const seen = new Set(normalizeHistoryAssets(workflowHistoryAssets.value).map(asset => asset.url))
  const additions = nextItems.filter((asset) => {
    if (seen.has(asset.url)) return false
    seen.add(asset.url)
    return true
  })
  if (!additions.length) return
  workflowHistoryAssets.value = normalizeHistoryAssets([
    ...workflowHistoryAssets.value,
    ...additions
  ])
  saveActiveProject()
}

function useHistoryAsset(url) {
  const resolved = resolveImageUrl(url)
  if (!resolved) return
  const canvas = workflowCanvasRef.value
  const rect = canvas?.getBoundingClientRect?.()
  const scale = liveCanvasView.scale || 1
  const position = rect
    ? {
        x: (rect.width / 2 - liveCanvasView.x) / scale - 260,
        y: (rect.height / 2 - liveCanvasView.y) / scale - 150
      }
    : undefined
  addWorkflowNode('generate', {
    position,
    state: {
      result: {
        url: resolved,
        images: [resolved],
        imageCreatedAt: [new Date().toISOString()],
        imageSources: ['imported'],
        createdAt: new Date().toISOString(),
        status: 'imported'
      },
      uploads: [],
      showResults: false
    }
  })
  closeHistoryModal()
}

function styleReferenceImage(ref = {}) {
  return resolveImageUrl(ref.image_url || ref.imageUrl || ref.url || '')
}

function toggleStyleLibraryMenu() {
  showStyleLibraryMenu.value = !showStyleLibraryMenu.value
}

function defaultVisibleNodePosition(type = 'generate') {
  const canvas = workflowCanvasRef.value
  const rect = canvas?.getBoundingClientRect?.()
  const scale = liveCanvasView.scale || 1
  const base = defaultNewNodePosition(type)
  return rect
    ? {
        x: (rect.width / 2 - liveCanvasView.x) / scale - base.width / 2,
        y: (rect.height / 2 - liveCanvasView.y) / scale - base.height / 2
      }
    : undefined
}

async function openCanvasMaterialLibrary() {
  showStyleLibraryMenu.value = false
  showNodeMenu.value = false
  showCanvasStyleLibrary.value = false
  showCanvasMaterialLibrary.value = true
  await ensureModelSceneMaterialsLoaded()
}

function closeCanvasMaterialLibrary() {
  showCanvasMaterialLibrary.value = false
}

function useCanvasMaterialReference(material) {
  const materialUrl = resolveImageUrl(material?.imageUrl || '')
  if (!materialUrl) {
    error.value = '该材质没有可用参考图'
    return
  }
  addWorkflowNode('generate', {
    position: defaultVisibleNodePosition('generate'),
    state: {
      uploadOnly: true,
      showResults: false,
      result: {
        url: materialUrl,
        images: [materialUrl],
        imageCreatedAt: [new Date().toISOString()],
        imageSources: ['material-reference'],
        createdAt: new Date().toISOString(),
        status: 'imported'
      },
      uploads: []
    }
  })
  closeCanvasMaterialLibrary()
}

async function openCanvasStyleLibrary(scene) {
  canvasStyleLibraryScene.value = scene === 'bathroom' ? 'bathroom' : 'kitchen'
  canvasStyleLibraryMode.value = 'insert'
  canvasStyleLibraryTargetNode.value = ''
  showStyleLibraryMenu.value = false
  showNodeMenu.value = false
  showCanvasMaterialLibrary.value = false
  showCanvasStyleLibrary.value = true
  await loadCanvasStyleLibrary()
}

async function openModelStyleLibrary(node = activeModelNodeId.value) {
  const targetNode = node || 'upload'
  const state = ensureModelNodeState(targetNode)
  canvasStyleLibraryScene.value = normalizeModelSceneType(state.sceneType)
  canvasStyleLibraryMode.value = 'model-style'
  canvasStyleLibraryTargetNode.value = targetNode
  showStyleLibraryMenu.value = false
  showNodeMenu.value = false
  showCanvasMaterialLibrary.value = false
  showCanvasStyleLibrary.value = true
  await loadCanvasStyleLibrary()
}

async function switchCanvasStyleLibraryScene(scene) {
  const nextScene = normalizeModelSceneType(scene)
  if (canvasStyleLibraryScene.value === nextScene) return
  canvasStyleLibraryScene.value = nextScene
  if (canvasStyleLibraryMode.value === 'model-style') {
    const node = canvasStyleLibraryTargetNode.value || activeModelNodeId.value || 'upload'
    removeModelStyleReferenceNode(node)
    updateActiveModelExportOptions({
      sceneType: nextScene,
      sceneStyleRef: null,
      controlMode: 'scene'
    }, node)
  }
  await loadCanvasStyleLibrary()
}

function closeCanvasStyleLibrary() {
  showCanvasStyleLibrary.value = false
  canvasStyleLibraryMode.value = 'insert'
  canvasStyleLibraryTargetNode.value = ''
}

async function loadCanvasStyleLibrary() {
  canvasStyleLibraryLoading.value = true
  canvasStyleLibraryTagId.value = null
  try {
    const sceneId = canvasStyleLibraryScene.value === 'bathroom' ? 3 : 4
    const res = await request.get('/scene-style-tags', { params: { scene_id: sceneId } })
    canvasStyleLibraryTags.value = Array.isArray(res.data) ? res.data : []
    const first = canvasStyleLibraryTagsWithRefs.value[0]
    canvasStyleLibraryTagId.value = first?.id || null
  } catch (e) {
    canvasStyleLibraryTags.value = []
    console.warn('[Workflow] load style library failed:', e)
  } finally {
    canvasStyleLibraryLoading.value = false
  }
}

function useCanvasStyleReference(ref) {
  const url = styleReferenceImage(ref)
  if (!url) return
  if (canvasStyleLibraryMode.value === 'model-style') {
    const node = canvasStyleLibraryTargetNode.value || activeModelNodeId.value || 'upload'
    removeModelStyleReferenceNode(node)
    updateActiveModelExportOptions({
      sceneType: normalizeModelSceneType(canvasStyleLibraryScene.value),
      sceneStyleRef: {
        ...ref,
        url,
        tagName: canvasStyleLibrarySelectedTag.value?.name || '',
        sceneType: canvasStyleLibraryScene.value,
        prompt: '',
        promptImageUrl: ''
      },
      sceneStyleNode: '',
      controlMode: 'scene'
    }, node)
    closeCanvasStyleLibrary()
    return
  }
  useHistoryAsset(url)
  closeCanvasStyleLibrary()
}

function imageDataUrl(data = {}) {
  return data.image_data ? `data:${data.mime_type || 'image/png'};base64,${data.image_data}` : ''
}

function extractTaskImage(data = {}) {
  const nestedResult = data.result || {}
  const firstImage = Array.isArray(data.images) ? data.images.find(item => item?.image_url || item?.url || item?.image_data) : null
  return resolveImageUrl(
    data.image_url ||
    data.url ||
    nestedResult.image_url ||
    nestedResult.url ||
    firstImage?.image_url ||
    firstImage?.url ||
    imageDataUrl(data) ||
    imageDataUrl(nestedResult) ||
    imageDataUrl(firstImage || {})
  )
}

function mergeCompletedGenerationImage(node, imageUrl, taskId, taskIds = [], runId = '') {
  const resolved = resolveImageUrl(imageUrl)
  if (!resolved) return []
  const state = runId ? activeGenerateRunState(node, runId) : generateNodeStateIfExists(node)
  if (!state) return []
  if ((runId && state.runId !== runId) || state.cancelled) return []
  const previousImages = normalizeGenerationResult(state.result).images
  const images = uniqueUrls([...previousImages, resolved])
  const result = normalizeGenerationResult(state.result)
  const createdAtMap = new Map(previousImages.map((url, index) => [url, result.imageCreatedAt[index] || result.createdAt || '']))
  const sourceMap = new Map(previousImages.map((url, index) => [url, result.imageSources[index] || 'generated']))
  const now = new Date().toISOString()
  if (!createdAtMap.has(resolved)) createdAtMap.set(resolved, now)
  if (!sourceMap.has(resolved)) sourceMap.set(resolved, 'generated')
  const activeTaskIds = uniqueValues([...(state.result.taskIds || []), ...taskIds, taskId])
  updateGenerateNode(node, {
    result: {
      ...state.result,
      url: images[0] || resolved,
      images,
      imageCreatedAt: images.map(url => createdAtMap.get(url) || ''),
      imageSources: images.map(url => sourceMap.get(url) || 'generated'),
      taskIds: activeTaskIds,
      taskId: taskId || state.result.taskId,
      createdAt: state.result.createdAt || now,
      status: 'running'
    }
  }, { persist: false })
  return images
}

function cancelWorkflowGeneration(node = activeGenerateNodeId()) {
  const state = ensureGenerateNodeState(node)
  updateGenerateNode(node, {
    cancelled: true,
    generating: false,
    runId: '',
    taskProgress: [],
    progress: 0,
    result: {
      ...state.result,
      status: generationImages(node).length ? 'partial' : 'cancelled'
    }
  })
}

async function recordWorkflowImageDownload(url, index = 0, node = activeGenerateNodeId()) {
  const resolved = resolveImageUrl(url)
  if (!resolved || /^(blob:|data:)/i.test(resolved)) return
  try {
    const parsed = new URL(resolved, window.location.origin)
    if (parsed.pathname.startsWith('/uploads/')) return
  } catch {
    if (resolved.startsWith('/uploads/')) return
  }
  const state = ensureGenerateNodeState(node)
  const result = normalizeGenerationResult(state.result)
  try {
    await request.post('/usage-logs/record-download', {
      image_url: resolved,
      model: effectiveDrawingModelName(node),
      prompt: effectiveGenerationPrompt(node),
      resolution: state.resolution,
      aspect_ratio: state.aspectRatio,
      source: 'workflow_canvas',
      task_id: result.taskIds?.[index] || result.taskId || ''
    })
  } catch (e) {
    console.warn('[Workflow] record image download failed:', e)
  }
}

async function downloadGeneratedImage(url, index, node = activeGenerateNodeId()) {
  if (!url) return
  await recordWorkflowImageDownload(url, index, node)
  await downloadImage(resolveImageUrl(url), `generated-${index + 1}.png`)
}

async function pollWorkflowGenerationTask(node, taskId, taskIndex, totalTasks, runId) {
  for (let i = 0; i < 600; i += 1) {
    const state = activeGenerateRunState(node, runId)
    if (!state) throw new Error('生成节点已删除')
    if (runId && state.runId !== runId) {
      throw new Error('已开始新的生成任务')
    }
    if (state.cancelled) {
      throw new Error('已取消生成')
    }
    const res = await request.get(`/image/generate-async/${taskId}`)
    const data = res.data || {}
    const currentImages = generationResultImages(node)
    updateGenerateNode(node, {
      result: {
        ...state.result,
        url: currentImages[0] || state.result.url || '',
        taskId,
        status: data.status || 'running'
      }
    }, { persist: false })

    updateWorkflowGenerationTaskProgress(node, taskIndex, Math.min(92, i * 3))

    const imageUrl = extractTaskImage(data)
    if (data.status === 'completed' || data.status === 'done' || imageUrl) {
      if (!imageUrl) throw new Error('未获取到生成图片')
      updateWorkflowGenerationTaskProgress(node, taskIndex, 100)
      mergeCompletedGenerationImage(node, imageUrl, taskId, [], runId)
      return imageUrl
    }
    if (data.status === 'failed' || data.status === 'error') {
      throw new Error(data.error || '生图失败')
    }
    await new Promise(resolve => window.setTimeout(resolve, 1500))
  }
  throw new Error('生图超时，请稍后查看任务结果')
}

function updateWorkflowGenerationTaskProgress(node, taskIndex, taskProgress) {
  const state = generateNodeStateIfExists(node)
  if (!state) return false
  const nextTaskProgress = [...(state.taskProgress || [])]
  nextTaskProgress[taskIndex] = Math.max(nextTaskProgress[taskIndex] || 0, taskProgress)
  const total = Math.max(1, nextTaskProgress.length || Number(state.count) || 1)
  const progressSum = nextTaskProgress.reduce((sum, value) => sum + (Number(value) || 0), 0)
  updateGenerateNode(node, {
    taskProgress: nextTaskProgress,
    progress: Math.max(state.progress || 0, Math.min(98, Math.floor(progressSum / total)))
  }, { persist: false, transient: true })
  return true
}

function finalizeWorkflowGeneration(node, images, taskIds = [], createdAt = new Date().toISOString(), options = {}) {
  if (!generateNodeStateIfExists(node)) return false
  const finalImages = options.replaceImages
    ? uniqueUrls(images.filter(Boolean))
    : uniqueUrls([
        ...normalizeGenerationResult(ensureGenerateNodeState(node).result).images,
        ...images.filter(Boolean)
      ])
  if (!finalImages.length) return false
  const finalResult = normalizeGenerationResult(ensureGenerateNodeState(node).result)
  const finalCreatedAt = ensureGenerateNodeState(node).result.createdAt || createdAt
  const finalImageCreatedAt = finalImages.map((url, index) => finalResult.imageCreatedAt[index] || createdAt)
  const finalTaskIds = taskIds.filter(Boolean)
  const resultNodeIds = splitGenerationResultToNodes(node, finalImages, {
    createdAt: finalCreatedAt,
    imageCreatedAt: finalImageCreatedAt,
    taskIds: finalTaskIds
  })
  removeWorkflowNode(node, { undo: false, persist: false })
  appendWorkflowHistoryAssets(finalImages.map((url, index) => ({
    url,
    createdAt: finalImageCreatedAt[index] || finalCreatedAt,
    node,
    taskId: finalTaskIds[index] || finalTaskIds.slice(-1)[0] || ''
  })))
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  activeGroupId.value = ''
  selectedNodeIds.value = []
  selectionBox.value = { active: false, fixed: false, x: 0, y: 0, width: 0, height: 0, startX: 0, startY: 0 }
  return true
}

function finalizeWorkflowGenerationInPlace(node, images, taskIds = [], createdAt = new Date().toISOString()) {
  const state = generateNodeStateIfExists(node)
  if (!state) return false
  const finalImages = uniqueUrls(images.filter(Boolean).map(resolveImageUrl))
  if (!finalImages.length) return false
  updateGenerateNode(node, {
    error: '',
    progress: 100,
    result: {
      ...state.result,
      url: finalImages[0],
      images: finalImages,
      imageCreatedAt: finalImages.map(() => createdAt),
      imageSources: finalImages.map(() => 'generated'),
      taskIds: taskIds.filter(Boolean),
      taskId: taskIds.filter(Boolean).slice(-1)[0] || '',
      createdAt,
      status: 'completed'
    },
    editRef: null,
    editPreview: null,
    showResults: false
  }, { persist: false })
  appendWorkflowHistoryAssets(finalImages.map((url, index) => ({
    url,
    createdAt,
    node,
    taskId: taskIds[index] || taskIds.filter(Boolean).slice(-1)[0] || ''
  })))
  return true
}

async function generateWorkflowImage(node = activeGenerateNodeId(), options = {}) {
  const current = ensureGenerateNodeState(node)
  if (current.generating) return
  const modelName = effectiveDrawingModelName(node)
  if (!modelName) {
    updateGenerateNode(node, { error: '暂无可用生图模型' }, { persist: false })
    return
  }
  if (options.undo !== false) pushWorkflowUndoSnapshot()
  removeGeneratedChildNodes(node)
  const replaceCurrentNode = options.replaceCurrentNode === true
  const selfInputImage = options.includeSelfReference === false ? '' : primaryGenerationImage(node)
  const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`
  const createdAt = new Date().toISOString()
  const previousResultForReplace = replaceCurrentNode ? clonePlain(current.result) : null
  updateGenerateNode(node, {
    error: '',
    model: modelName,
    generating: true,
    cancelled: false,
    progress: 0,
    runId,
    taskProgress: [],
    showResults: false,
    result: replaceCurrentNode
      ? {
          ...current.result,
          status: 'queued'
        }
      : {
          url: '',
          images: [],
          imageCreatedAt: [],
          imageSources: [],
          taskId: '',
          taskIds: [],
          createdAt,
          status: 'queued'
        }
  })

  try {
    const state = ensureGenerateNodeState(node)
    const savedEditRef = options.ignoreSavedEditRef ? null : (state.editPreview?.url ? state.editPreview : (state.editRef?.url ? state.editRef : null))
    const activeEditRef = options.editedRef || savedEditRef
    const promptText = sceneProductControlForGenerateNode(node)
      ? await prepareSceneWorkflowPromptForGeneration(node)
      : effectiveGenerationPrompt(node)
    const refImages = activeEditRef
      ? [activeEditRef, ...await buildGenerationRefPayload(node, '')]
      : await buildGenerationRefPayload(node, selfInputImage)
    const total = replaceCurrentNode ? 1 : Math.max(1, Math.min(4, Number(state.count) || 1))
    const basePayload = {
      prompt: promptText,
      model: modelName,
      aspect_ratio: state.aspectRatio,
      resolution: state.resolution,
      count: 1,
      num_images: 1,
      ref_images: refImages
    }

    const taskIds = Array(total).fill('')
    const generationJobs = Array.from({ length: total }, async (_, index) => {
      const activeState = activeGenerateRunState(node, runId)
      if (!activeState || activeState.cancelled) return null
      const res = await request.post('/image/generate-async', {
        ...basePayload,
        request_nonce: `${runId}-${index}`
      })
      if (!activeGenerateRunState(node, runId)) return null
      const taskId = res.data?.task_id
      if (taskId) {
        taskIds[index] = taskId
        const activeTaskIds = taskIds.filter(Boolean)
        updateGenerateNode(node, {
          result: {
            ...activeGenerateRunState(node, runId).result,
            taskIds: activeTaskIds,
            taskId,
            status: 'running'
          }
        }, { persist: false })
        return pollWorkflowGenerationTask(node, taskId, index, total, runId)
      }
      const imageUrl = extractTaskImage(res.data || {})
      if (!imageUrl) throw new Error('未获取到生成图片')
      if (!updateWorkflowGenerationTaskProgress(node, index, 100)) return null
      mergeCompletedGenerationImage(node, imageUrl, '', taskIds.filter(Boolean), runId)
      return imageUrl
    })

    if (!activeGenerateRunState(node, runId)) return
    updateGenerateNode(node, {
      taskProgress: Array(total).fill(0),
      result: {
        ...activeGenerateRunState(node, runId).result,
        status: 'running'
      }
    }, { persist: false })

    const images = (await Promise.all(generationJobs.map(job => typeof job === 'function' ? job() : job))).flat().filter(Boolean)

    const finalState = activeGenerateRunState(node, runId)
    if (!finalState) return
    const completedImages = uniqueUrls(images.filter(Boolean))
    const finalImages = replaceCurrentNode
      ? completedImages
      : uniqueUrls([
          ...normalizeGenerationResult(finalState.result).images,
          ...completedImages
        ])
    if (!finalImages.length && finalState.cancelled) return
    if (!finalImages.length) throw new Error('未获取到生成图片')
    if (replaceCurrentNode) {
      finalizeWorkflowGenerationInPlace(node, finalImages, taskIds.filter(Boolean), createdAt)
    } else {
      finalizeWorkflowGeneration(node, finalImages, taskIds.filter(Boolean), createdAt, { replaceImages: !!sceneProductControlForGenerateNode(node) })
    }
  } catch (e) {
    const state = activeGenerateRunState(node, runId)
    if (!state || state.cancelled) return
    const errData = e?.response?.data
    const errMsg = typeof errData === 'string' ? errData : (errData?.error || e?.message || '生图失败')
    updateGenerateNode(node, {
      error: errMsg,
      ...(previousResultForReplace ? { result: previousResultForReplace } : {})
    })
  } finally {
    if (activeGenerateRunState(node, runId)) {
      updateGenerateNode(node, { generating: false, runId: '' })
    }
  }
}

function shouldResumeGenerateState(state = {}) {
  const result = normalizeGenerationResult(state.result || {})
  const taskIds = result.taskIds?.length ? result.taskIds : (result.taskId ? [result.taskId] : [])
  return !!state.generating && !state.cancelled && taskIds.length > 0 && !['completed', 'failed', 'error', 'cancelled'].includes(result.status)
}

function resumeWorkflowGenerationTasks() {
  Object.keys(workflowGenerateNodeStates.value || {}).forEach((node) => {
    if (!nodeExists(node)) return
    const state = ensureGenerateNodeState(node)
    if (!shouldResumeGenerateState(state)) return
    const result = normalizeGenerationResult(state.result)
    const taskIds = result.taskIds?.length ? result.taskIds : [result.taskId]
    const runId = state.runId || `resume-${Date.now()}-${Math.random().toString(16).slice(2)}`
    updateGenerateNode(node, {
      generating: true,
      cancelled: false,
      runId,
      taskProgress: state.taskProgress?.length ? state.taskProgress : Array(taskIds.length).fill(state.progress || 0),
      result: {
        ...state.result,
        taskIds,
        taskId: taskIds.slice(-1)[0] || '',
        status: result.status || 'running'
      }
    }, { persist: false })
    const total = taskIds.length
    const jobs = taskIds.map((taskId, index) => pollWorkflowGenerationTask(node, taskId, index, total, runId))
    Promise.all(jobs)
      .then(async (images) => {
        const current = activeGenerateRunState(node, runId)
        if (!current || current.cancelled) return
        finalizeWorkflowGeneration(node, images.filter(Boolean), taskIds, result.createdAt || new Date().toISOString(), { replaceImages: !!sceneProductControlForGenerateNode(node) })
      })
      .catch((e) => {
        const current = activeGenerateRunState(node, runId)
        if (!current || current.cancelled) return
        updateGenerateNode(node, {
          generating: false,
          runId: '',
          error: e?.response?.data?.error || e?.message || '生图失败'
        })
      })
  })
}

function onSceneFileSelect(e) {
  const file = e.target.files?.[0]
  if (file) loadSceneImage(file)
}

function onSceneDrop(e) {
  const file = e.dataTransfer.files?.[0]
  if (file) loadSceneImage(file)
}

function loadSceneImage(file) {
  if (!file.type.startsWith('image/')) {
    error.value = '请上传场景图片文件'
    return
  }

  const url = URL.createObjectURL(file)
  const img = new Image()
  img.onload = () => {
    if (sceneImage.value.url) URL.revokeObjectURL(sceneImage.value.url)
    sceneImage.value = {
      url,
      name: file.name,
      width: img.naturalWidth || img.width,
      height: img.naturalHeight || img.height
    }
    resetPlacement()
    setWorkspaceMode('scene')
    saveActiveProject()
  }
  img.onerror = () => {
    URL.revokeObjectURL(url)
    error.value = '场景图片加载失败'
  }
  img.src = url
}

function clampPlacement() {
  placement.value = {
    ...placement.value,
    x: Math.min(100, Math.max(0, placement.value.x)),
    y: Math.min(100, Math.max(0, placement.value.y)),
    scale: Math.min(1.8, Math.max(0.08, placement.value.scale)),
    rotation: Math.min(180, Math.max(-180, placement.value.rotation)),
    opacity: Math.min(1, Math.max(0.15, placement.value.opacity))
  }
}

function startPlacementDrag(event) {
  if (!sceneStageRef.value || !resultUrls.value.white) return
  placementDrag = {
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    baseX: placement.value.x,
    baseY: placement.value.y
  }
  event.currentTarget.setPointerCapture?.(event.pointerId)
}

function dragPlacement(event) {
  if (!placementDrag || placementDrag.pointerId !== event.pointerId || !sceneStageRef.value) return
  const rect = sceneStageRef.value.getBoundingClientRect()
  if (!rect.width || !rect.height) return
  placement.value = {
    ...placement.value,
    x: placementDrag.baseX + ((event.clientX - placementDrag.startX) / rect.width) * 100,
    y: placementDrag.baseY + ((event.clientY - placementDrag.startY) / rect.height) * 100
  }
  clampPlacement()
}

function endPlacementDrag(event) {
  if (!placementDrag || placementDrag.pointerId !== event.pointerId) return
  event.currentTarget.releasePointerCapture?.(event.pointerId)
  placementDrag = null
}

function resetPlacement() {
  placement.value = {
    x: 50,
    y: 62,
    scale: 0.36,
    rotation: 0,
    opacity: 0.78
  }
}

function sceneProductStyle() {
  return {
    left: `${placement.value.x}%`,
    top: `${placement.value.y}%`,
    width: `${placement.value.scale * 100}%`,
    opacity: placement.value.opacity,
    transform: `translate(-50%, -50%) rotate(${placement.value.rotation}deg)`
  }
}

function loadImage(url) {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = url
  })
}

function trimmedImageBounds(img) {
  const width = img.naturalWidth || img.width
  const height = img.naturalHeight || img.height
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  ctx.drawImage(img, 0, 0, width, height)
  const { data } = ctx.getImageData(0, 0, width, height)
  let minX = width
  let minY = height
  let maxX = -1
  let maxY = -1
  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      if (data[(y * width + x) * 4 + 3] <= 8) continue
      if (x < minX) minX = x
      if (y < minY) minY = y
      if (x > maxX) maxX = x
      if (y > maxY) maxY = y
    }
  }
  if (maxX < minX || maxY < minY) {
    return { x: 0, y: 0, width, height }
  }
  return {
    x: minX,
    y: minY,
    width: maxX - minX + 1,
    height: maxY - minY + 1
  }
}

function createWhiteKeyedCanvas(img) {
  const canvas = document.createElement('canvas')
  canvas.width = img.naturalWidth || img.width
  canvas.height = img.naturalHeight || img.height
  const ctx = canvas.getContext('2d')
  ctx.drawImage(img, 0, 0, canvas.width, canvas.height)

  const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height)
  const data = imageData.data
  for (let i = 0; i < data.length; i += 4) {
    const r = data[i]
    const g = data[i + 1]
    const b = data[i + 2]
    const min = Math.min(r, g, b)
    if (min > 245) {
      data[i + 3] = 0
    } else if (min > 228) {
      data[i + 3] = Math.min(data[i + 3], Math.round((245 - min) * 15))
    }
  }
  ctx.putImageData(imageData, 0, 0)
  return canvas
}

async function renderSceneComposite() {
  if (!sceneImage.value.url || !resultUrls.value.white) {
    error.value = '请先上传场景图，并生成白底结构图'
    return
  }

  try {
    const [sceneImg, productImg] = await Promise.all([
      loadImage(sceneImage.value.url),
      loadImage(resultUrls.value.white)
    ])
    const canvas = document.createElement('canvas')
    canvas.width = sceneImg.naturalWidth || sceneImg.width
    canvas.height = sceneImg.naturalHeight || sceneImg.height
    const ctx = canvas.getContext('2d')
    ctx.drawImage(sceneImg, 0, 0, canvas.width, canvas.height)

    const productWidth = canvas.width * placement.value.scale
    const productHeight = productWidth * ((productImg.naturalHeight || productImg.height) / (productImg.naturalWidth || productImg.width))
    const x = canvas.width * placement.value.x / 100
    const y = canvas.height * placement.value.y / 100

    const keyedProduct = createWhiteKeyedCanvas(productImg)
    ctx.save()
    ctx.globalAlpha = placement.value.opacity
    ctx.translate(x, y)
    ctx.rotate(THREE.MathUtils.degToRad(placement.value.rotation))
    ctx.drawImage(keyedProduct, -productWidth / 2, -productHeight / 2, productWidth, productHeight)
    ctx.restore()

    openPreview(canvas.toDataURL('image/png'), '场景放置预览')
  } catch (e) {
    error.value = '场景预合成失败：' + (e.message || e)
  }
}

async function compositeSceneProductImage(sceneUrl, productUrl, sceneShot = 'wide') {
  if (!sceneUrl || !productUrl) return resolveImageUrl(sceneUrl)
  const [sceneImg, productImg] = await Promise.all([
    loadImage(resolveImageUrl(sceneUrl)),
    loadImage(resolveImageUrl(productUrl))
  ])
  const canvas = document.createElement('canvas')
  canvas.width = sceneImg.naturalWidth || sceneImg.width
  canvas.height = sceneImg.naturalHeight || sceneImg.height
  const ctx = canvas.getContext('2d')
  ctx.drawImage(sceneImg, 0, 0, canvas.width, canvas.height)

  const placementConfig = MODEL_SCENE_COMPOSITE_PLACEMENT[normalizeModelSceneShot(sceneShot)] || MODEL_SCENE_COMPOSITE_PLACEMENT.wide
  const productBounds = trimmedImageBounds(productImg)
  const sourceWidth = Math.max(1, productBounds.width)
  const sourceHeight = Math.max(1, productBounds.height)
  const maxProductWidth = canvas.width * placementConfig.scale
  const maxProductHeight = canvas.height * (placementConfig.heightScale || placementConfig.scale)
  const productScale = Math.min(maxProductWidth / sourceWidth, maxProductHeight / sourceHeight)
  const productWidth = sourceWidth * productScale
  const productHeight = sourceHeight * productScale
  const x = canvas.width * placementConfig.x / 100
  const y = canvas.height * placementConfig.y / 100

  ctx.save()
  ctx.translate(x, y)
  ctx.drawImage(
    productImg,
    productBounds.x,
    productBounds.y,
    sourceWidth,
    sourceHeight,
    -productWidth / 2,
    -productHeight,
    productWidth,
    productHeight
  )
  ctx.restore()

  return canvas.toDataURL('image/png')
}

async function applySceneProductCompositeToImages(node, images = []) {
  const control = sceneProductControlForGenerateNode(node)
  if (!control?.productUrl) return images
  const composited = []
  for (const image of images) {
    try {
      const dataUrl = await compositeSceneProductImage(image, control.productUrl, control.sceneShot)
      composited.push(dataUrl)
    } catch (e) {
      console.warn('[Workflow] composite scene product failed:', e)
      composited.push(image)
    }
  }
  return composited
}

async function uploadAndLoad(file, node = activeModelNodeId.value) {
  const targetNode = node || 'upload'
  activateModelNode(targetNode)
  clearModelMeshLoadPromisesForNode(targetNode)
  const uploadRun = beginModelUploadRun(targetNode)
  loading.value = true
  loadingMessage.value = '上传文件...'
  if (renderedModelNodeId === targetNode) {
    clearRenderedModelScene({ render: true })
  }
  workflowModelNodeStates.value = {
    ...workflowModelNodeStates.value,
    [targetNode]: {
      ...ensureModelNodeState(targetNode),
      fileName: file.name,
      stpFileUrl: '',
      meshData: null,
      correction: { x: 0, y: 0, z: 0 },
      cut: { enabled: false, height: 0 },
      camera: null,
      modelLoaded: false,
      loading: true,
      loadingMessage: '上传文件...',
      error: ''
    }
  }
  try {
    activeModelCorrection.value = { x: 0, y: 0, z: 0 }
    activeModelCut.value = { enabled: false, height: 0 }
    const formData = new FormData()
    formData.append('file', file)

    const res = await request.post('/stp/model', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    // axios 鍝嶅簲: res.data 鏄悗绔繑鍥炵殑 JSON锛孶RL 鍦?res.data.data.url
    const url = res.data?.data?.url
    if (!url) {
      throw new Error('上传失败：未获取到文件 URL')
    }
    assertCurrentModelUploadRun(targetNode, uploadRun)

    stpFileUrl.value = url
    fileName.value = file.name
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...ensureModelNodeState(targetNode),
        fileName: file.name,
        stpFileUrl: url,
        meshData: null,
        modelLoaded: false,
        loading: false,
        loadingMessage: '点击控制角度加载模型',
        error: ''
      }
    }
    syncDynamicUploadNodeFileState(targetNode, { fileName: file.name, stpFileUrl: url })
    saveWorkflowStructureState({ persist: true })
    error.value = ''
  } catch (e) {
    if (!isCurrentModelUploadRun(targetNode, uploadRun)) return
    if (e?.message?.includes('模型预览加载已取消')) return
    error.value = e.message
    workflowModelNodeStates.value = {
      ...workflowModelNodeStates.value,
      [targetNode]: {
        ...ensureModelNodeState(targetNode),
        loading: false,
        error: e.message
      }
    }
  } finally {
    if (isCurrentModelUploadRun(targetNode, uploadRun)) {
      cancelModelUploadRun(targetNode)
      loading.value = false
      workflowModelNodeStates.value = {
        ...workflowModelNodeStates.value,
        [targetNode]: {
          ...ensureModelNodeState(targetNode),
          loading: false
        }
      }
    }
  }
}

function modelStatesWithDynamicUploadFallback(states = {}, dynamicNodes = []) {
  const nextStates = normalizeModelNodeStates(states || {})
  normalizeDynamicNodes(dynamicNodes || []).forEach((node) => {
    if (node.type !== 'upload' || !node.id || nextStates[node.id]) return
    if (!node.stpFileUrl && !node.fileName) return
    nextStates[node.id] = normalizeModelNodeState({
      fileName: node.fileName || '',
      stpFileUrl: node.stpFileUrl || '',
      modelLoaded: false,
      loading: false,
      error: ''
    })
  })
  return nextStates
}

function cloneExportNormalization(normalization) {
  if (!normalization) return null
  return {
    offset: normalization.offset?.clone?.() || new THREE.Vector3(),
    box: normalization.box?.clone?.() || new THREE.Box3()
  }
}

function cloneExportMesh(materialFactory, options = {}) {
  if (!mesh) return null

  mesh.updateMatrixWorld(true)
  const normalization = options.groundVisibleBottom
    ? (cloneExportNormalization(options.normalization) || exportModelNormalization())
    : null
  const cloneGeometry = mesh.geometry
  const material = materialFactory()
  const clone = new THREE.Mesh(cloneGeometry, material)
  clone.userData.disposeGeometryAfterRender = false
  applyExportCutToMaterial(clone.material, normalization)
  clone.matrix.copy(mesh.matrixWorld)
  clone.matrix.decompose(clone.position, clone.quaternion, clone.scale)
  if (normalization) {
    clone.position.add(normalization.offset)
  }
  clone.updateMatrixWorld(true)
  return clone
}

function modelCutWorldYForBox(box) {
  const cut = normalizeModelCut(activeModelCut.value)
  if (!cut.enabled || !box || box.isEmpty()) return box?.min?.y ?? 0
  return box.min.y + (box.max.y - box.min.y) * normalizedModelCutHeightRatio(cut)
}

function isVideoAssetUrl(url = '') {
  return /\.(mp4|webm|mov|m4v|avi)(\?|#|$)/i.test(String(url || ''))
}

function getVisibleModelWorldBox() {
  const target = modelRoot || mesh
  if (!target) return null
  target.updateMatrixWorld(true)
  const box = new THREE.Box3().setFromObject(target)
  if (box.isEmpty()) return null
  const cut = normalizeModelCut(activeModelCut.value)
  if (cut.enabled) {
    const cutY = modelCutWorldYForBox(box)
    box.min.y = Math.min(Math.max(box.min.y, cutY), box.max.y)
  }
  return box.isEmpty() ? null : box
}

function getBoxCorners(box) {
  const minY = box.min.y
  const points = [
    new THREE.Vector3(box.min.x, minY, box.min.z),
    new THREE.Vector3(box.min.x, minY, box.max.z),
    new THREE.Vector3(box.min.x, box.max.y, box.min.z),
    new THREE.Vector3(box.min.x, box.max.y, box.max.z),
    new THREE.Vector3(box.max.x, minY, box.min.z),
    new THREE.Vector3(box.max.x, minY, box.max.z),
    new THREE.Vector3(box.max.x, box.max.y, box.min.z),
    new THREE.Vector3(box.max.x, box.max.y, box.max.z)
  ]
  return points
}

function getVisibleMeshExportPoints() {
  if (!mesh?.geometry?.attributes?.position) return []

  mesh.updateMatrixWorld(true)
  const cut = normalizeModelCut(activeModelCut.value)
  const fullBox = new THREE.Box3().setFromObject(modelRoot || mesh)
  const cutY = cut.enabled && !fullBox.isEmpty() ? modelCutWorldYForBox(fullBox) : -Infinity
  return visibleMeshPoints(mesh, cutY)
}

function exportModelNormalization() {
  const visibleBox = getVisibleModelWorldBox()
  if (!visibleBox) return null
  const center = new THREE.Vector3()
  visibleBox.getCenter(center)
  const offset = new THREE.Vector3(-center.x, -visibleBox.min.y, -center.z)
  const box = visibleBox.clone()
  box.min.add(offset)
  box.max.add(offset)
  return { offset, box }
}

function applyExportCutToMaterial(material, normalization = exportModelNormalization()) {
  if (!material) return
  const cut = normalizeModelCut(activeModelCut.value)
  const plane = cut.enabled
    ? new THREE.Plane(
        new THREE.Vector3(0, 1, 0),
        -(currentModelCutWorldY() + (normalization?.offset?.y || 0))
      )
    : null
  const materials = Array.isArray(material) ? material : [material]
  materials.forEach(item => {
    item.clippingPlanes = plane ? [plane] : []
    item.clipIntersection = false
    item.needsUpdate = true
  })
}

function getProjectedBounds(renderCamera, points) {
  let minX = Infinity
  let maxX = -Infinity
  let minY = Infinity
  let maxY = -Infinity
  const projected = new THREE.Vector3()

  points.forEach(point => {
    projected.copy(point).project(renderCamera)
    if (!Number.isFinite(projected.x) || !Number.isFinite(projected.y)) return
    minX = Math.min(minX, projected.x)
    maxX = Math.max(maxX, projected.x)
    minY = Math.min(minY, projected.y)
    maxY = Math.max(maxY, projected.y)
  })

  if (!Number.isFinite(minX) || !Number.isFinite(maxX) || !Number.isFinite(minY) || !Number.isFinite(maxY)) {
    return null
  }

  return {
    fill: Math.max(maxX - minX, maxY - minY) / 2,
    maxAbs: Math.max(Math.abs(minX), Math.abs(maxX), Math.abs(minY), Math.abs(maxY)),
    centerX: (minX + maxX) / 2,
    centerY: (minY + maxY) / 2
  }
}

function centerExportCameraOnProjectedBounds(exportCamera, points, distance) {
  if (!exportCamera || !points?.length || !Number.isFinite(distance) || distance <= 0) return
  centerCameraOnPoints(exportCamera, points)
}

function currentModelExportOptions() {
  const state = ensureModelNodeState(activeModelNodeId.value || 'upload')
  const controlMode = normalizeModelControlMode(state.controlMode)
  const sceneShot = normalizeModelSceneShot(state.sceneShot)
  return {
    controlMode,
    sceneShot,
    targetFill: controlMode === 'scene' ? MODEL_SCENE_SHOT_FILL[sceneShot] : EXPORT_TARGET_FILL,
    offsetY: controlMode === 'scene' ? MODEL_SCENE_SHOT_OFFSET_Y[sceneShot] : 0
  }
}

function fitExportCameraToMesh(exportCamera, options = currentModelExportOptions()) {
  if (!mesh || !camera) return

  const normalization = exportModelNormalization()
  if (!normalization) return
  const box = normalization.box.clone()

  const center = new THREE.Vector3()
  const size = new THREE.Vector3()
  box.getCenter(center)
  box.getSize(size)

  const maxSize = Math.max(size.x, size.y, size.z)
  if (maxSize <= 0) return

  const fitPoints = getVisibleMeshExportPoints()
  const safePoints = getBoxCorners(box)
  const normalizedFitPoints = fitPoints.map(point => point.clone().add(normalization.offset))
  const points = normalizedFitPoints.length ? normalizedFitPoints : safePoints
  const forward = new THREE.Vector3()
  camera.getWorldDirection(forward)
  if (forward.lengthSq() <= 0.000001) return
  forward.normalize()

  const targetFill = THREE.MathUtils.clamp(Number(options.targetFill) || EXPORT_TARGET_FILL, 0.06, EXPORT_TARGET_FILL)
  const fixedDistance = Number(options.fixedDistance) || 0
  let distance = fixedDistance > 0 ? fixedDistance : Math.max(camera.position.distanceTo(center), maxSize)
  const minDistance = maxSize * 0.25

  if (!fixedDistance) {
    for (let i = 0; i < 8; i += 1) {
      exportCamera.position.copy(center).addScaledVector(forward, -distance)
      exportCamera.quaternion.copy(camera.quaternion)
      exportCamera.near = Math.max(distance / 100, 0.01)
      exportCamera.far = Math.max(distance * 100, distance + maxSize * 100)
      exportCamera.updateProjectionMatrix()
      exportCamera.updateMatrixWorld(true)

      const bounds = getProjectedBounds(exportCamera, points)
      if (!bounds || bounds.fill <= 0.0001) break

      const nextDistance = Math.max(minDistance, distance * (bounds.fill / targetFill))
      if (Math.abs(nextDistance - distance) / distance < 0.015) break
      distance = nextDistance
    }
  }

  exportCamera.position.copy(center).addScaledVector(forward, -distance)
  exportCamera.quaternion.copy(camera.quaternion)
  exportCamera.near = Math.max(distance / 100, 0.01)
  exportCamera.far = Math.max(distance * 100, distance + maxSize * 100)
  exportCamera.updateProjectionMatrix()
  exportCamera.updateMatrixWorld(true)
  centerExportCameraOnProjectedBounds(exportCamera, points, distance)

  if (!fixedDistance) {
    for (let i = 0; i < 4; i += 1) {
      exportCamera.near = Math.max(distance / 100, 0.01)
      exportCamera.far = Math.max(distance * 100, distance + maxSize * 100)
      exportCamera.updateProjectionMatrix()
      exportCamera.updateMatrixWorld(true)

      const bounds = getProjectedBounds(exportCamera, safePoints)
      if (!bounds || bounds.maxAbs <= EXPORT_SAFE_NDC) break
      distance *= bounds.maxAbs / EXPORT_SAFE_NDC
      exportCamera.position.copy(center).addScaledVector(forward, -distance)
      exportCamera.quaternion.copy(camera.quaternion)
      exportCamera.updateMatrixWorld(true)
      centerExportCameraOnProjectedBounds(exportCamera, points, distance)
    }
  }

  exportCamera.near = Math.max(distance / 100, 0.01)
  exportCamera.far = Math.max(distance * 100, distance + maxSize * 100)
  exportCamera.updateProjectionMatrix()
  exportCamera.updateMatrixWorld(true)
  centerExportCameraOnProjectedBounds(exportCamera, points, distance)

  const offsetY = Number(options.offsetY) || 0
  if (Math.abs(offsetY) > 0.0001) {
    const up = new THREE.Vector3(0, 1, 0).applyQuaternion(exportCamera.quaternion).normalize()
    const viewHeight = 2 * distance * Math.tan(THREE.MathUtils.degToRad(exportCamera.fov) / 2)
    const shift = up.multiplyScalar(viewHeight * offsetY)
    exportCamera.position.add(shift)
    exportCamera.updateMatrixWorld(true)
  }
  exportCamera.userData.fitDistance = distance
}

function createExportCamera(options = {}) {
  if (!camera) return null

  let exportFov = camera.fov
  if (camera.aspect > 1) {
    const fovRad = THREE.MathUtils.degToRad(camera.fov)
    exportFov = THREE.MathUtils.radToDeg(2 * Math.atan(Math.tan(fovRad / 2) * camera.aspect))
  }

  const exportCamera = new THREE.PerspectiveCamera(exportFov, 1, camera.near, camera.far)
  exportCamera.position.copy(camera.position)
  exportCamera.quaternion.copy(camera.quaternion)
  exportCamera.up.copy(camera.up)
  fitExportCameraToMesh(exportCamera, options)
  exportCamera.updateProjectionMatrix()
  exportCamera.updateMatrixWorld(true)
  return exportCamera
}

function createOffscreenRenderer(bgColor, resolution = RENDER_RESOLUTION) {
  const offscreen = new THREE.WebGLRenderer({
    antialias: true,
    alpha: false,
    preserveDrawingBuffer: true,
    powerPreference: 'high-performance'
  })
  offscreen.setPixelRatio(1)
  offscreen.setSize(resolution, resolution, false)
  offscreen.setClearColor(bgColor, 1)
  offscreen.outputColorSpace = THREE.SRGBColorSpace
  offscreen.localClippingEnabled = true
  return offscreen
}

function renderSceneToDataUrl(renderScene, renderCamera, bgColor, resolution = RENDER_RESOLUTION) {
  const offscreen = createOffscreenRenderer(bgColor, resolution)
  offscreen.render(renderScene, renderCamera)
  const dataUrl = offscreen.domElement.toDataURL('image/png')
  offscreen.dispose()
  return dataUrl
}

function addExportLights(renderScene) {
  renderScene.add(new THREE.AmbientLight(0xffffff, 0.68))

  const key = new THREE.DirectionalLight(0xffffff, 1.05)
  key.position.set(4, 6, 6)
  renderScene.add(key)

  const fill = new THREE.DirectionalLight(0xffffff, 0.55)
  fill.position.set(-5, 3, -4)
  renderScene.add(fill)

  const rim = new THREE.DirectionalLight(0xffffff, 0.42)
  rim.position.set(-3, 5, 6)
  renderScene.add(rim)
}

function exportGroundSize() {
  const visibleBox = getVisibleModelWorldBox()
  if (!visibleBox) return 8
  const size = new THREE.Vector3()
  visibleBox.getSize(size)
  return Math.max(8, Math.max(size.x, size.z) * 3.4, size.y * 2.4)
}

function addExportContactPlane(renderScene) {
  const size = exportGroundSize()
  const plane = new THREE.Mesh(
    new THREE.PlaneGeometry(size, size, 24, 24),
    new THREE.MeshStandardMaterial({
      color: 0xf3f4f6,
      roughness: 0.82,
      metalness: 0,
      side: THREE.DoubleSide
    })
  )
  plane.name = 'export-contact-plane'
  plane.rotation.x = -Math.PI / 2
  plane.position.y = -0.001
  renderScene.add(plane)

  const grid = new THREE.GridHelper(size, 36, 0xd1d5db, 0xe5e7eb)
  grid.name = 'export-contact-grid'
  grid.position.y = 0
  grid.material.transparent = true
  grid.material.opacity = 0.72
  renderScene.add(grid)
}

function renderWhiteProductImage(renderCamera, options = {}) {
  const renderScene = new THREE.Scene()
  const exportBackground = 0xffffff
  renderScene.background = new THREE.Color(exportBackground)
  addExportLights(renderScene)

  const product = cloneExportMesh(() => new THREE.MeshStandardMaterial({
    color: 0xf5f5f5,
    metalness: 0.02,
    roughness: 0.72,
    side: THREE.DoubleSide
  }), {
    groundVisibleBottom: true,
    normalization: options.normalization,
  })
  if (!product) throw new Error('No model mesh')
  renderScene.add(product)

  const dataUrl = renderSceneToDataUrl(renderScene, renderCamera, exportBackground, CONTROL_RENDER_RESOLUTION)
  if (product.userData.disposeGeometryAfterRender) product.geometry?.dispose?.()
  product.material?.dispose?.()
  return dataUrl
}

function renderWireframeImage(renderCamera) {
  const renderScene = new THREE.Scene()
  renderScene.background = new THREE.Color(0xffffff)

  const fill = cloneExportMesh(() => new THREE.MeshBasicMaterial({
    color: 0xffffff,
    side: THREE.DoubleSide,
    polygonOffset: true,
    polygonOffsetFactor: 1,
    polygonOffsetUnits: 1
  }), { groundVisibleBottom: true })
  if (!fill) throw new Error('No model mesh')
  renderScene.add(fill)

  mesh.updateMatrixWorld(true)
  const edges = new THREE.EdgesGeometry(mesh.geometry, 18)
  const lines = new THREE.LineSegments(edges, new THREE.LineBasicMaterial({
    color: 0x000000,
    depthTest: true,
    depthWrite: false
  }))
  lines.matrix.copy(mesh.matrixWorld)
  lines.matrix.decompose(lines.position, lines.quaternion, lines.scale)
  const normalization = exportModelNormalization()
  if (normalization) lines.position.add(normalization.offset)
  renderScene.add(lines)

  const dataUrl = renderSceneToDataUrl(renderScene, renderCamera, 0xffffff)
  edges.dispose()
  return dataUrl
}

function getCameraDepthRange(renderCamera, options = {}) {
  const normalization = cloneExportNormalization(options.normalization) || exportModelNormalization()
  if (!normalization) return { nearDepth: 0.1, farDepth: 10 }
  const currentBox = getVisibleModelWorldBox()
  const box = currentBox && !currentBox.isEmpty()
    ? currentBox.clone().translate(normalization.offset)
    : normalization.box.clone()

  const points = [
    [box.min.x, box.min.y, box.min.z],
    [box.min.x, box.min.y, box.max.z],
    [box.min.x, box.max.y, box.min.z],
    [box.min.x, box.max.y, box.max.z],
    [box.max.x, box.min.y, box.min.z],
    [box.max.x, box.min.y, box.max.z],
    [box.max.x, box.max.y, box.min.z],
    [box.max.x, box.max.y, box.max.z]
  ].map(([x, y, z]) => new THREE.Vector3(x, y, z).applyMatrix4(renderCamera.matrixWorldInverse))

  const depths = points.map(point => -point.z).filter(value => Number.isFinite(value) && value > 0)
  if (!depths.length) return { nearDepth: 0.1, farDepth: 10 }

  const nearDepth = Math.max(0.001, Math.min(...depths))
  const farDepth = Math.max(nearDepth + 0.001, Math.max(...depths))
  return { nearDepth, farDepth }
}

function renderDepthImage(renderCamera, options = {}) {
  const { nearDepth, farDepth } = getCameraDepthRange(renderCamera, options)
  const depthMaterial = new THREE.ShaderMaterial({
    side: THREE.DoubleSide,
    clipping: true,
    uniforms: {
      nearDepth: { value: nearDepth },
      farDepth: { value: farDepth }
    },
    vertexShader: `
      #include <clipping_planes_pars_vertex>
      varying float vViewDepth;
      void main() {
        vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
        vViewDepth = -mvPosition.z;
        gl_Position = projectionMatrix * mvPosition;
        #include <clipping_planes_vertex>
      }
    `,
    fragmentShader: `
      uniform float nearDepth;
      uniform float farDepth;
      #include <clipping_planes_pars_fragment>
      varying float vViewDepth;
      void main() {
        vec4 diffuseColor = vec4(1.0);
        #include <clipping_planes_fragment>
        float value = clamp((farDepth - vViewDepth) / max(farDepth - nearDepth, 0.0001), 0.0, 1.0);
        gl_FragColor = vec4(vec3(value), 1.0);
      }
    `
  })

  const renderScene = new THREE.Scene()
  renderScene.background = new THREE.Color(0x000000)

  const product = cloneExportMesh(() => depthMaterial, { groundVisibleBottom: true, normalization: options.normalization })
  if (!product) throw new Error('No model mesh')
  renderScene.add(product)

  const dataUrl = renderSceneToDataUrl(renderScene, renderCamera, 0x000000, CONTROL_RENDER_RESOLUTION)
  depthMaterial.dispose()
  return dataUrl
}

async function renderControlImagesFromPreview(options = {}) {
  if (!mesh || !camera) {
    throw new Error('Model is not ready')
  }

  controls?.update?.()
  camera.updateMatrixWorld(true)
  modelRoot?.updateMatrixWorld(true)
  mesh.updateMatrixWorld(true)

  const exportCamera = createExportCamera(options)
  if (!exportCamera) throw new Error('Camera is not ready')

  await new Promise(resolve => requestAnimationFrame(resolve))
  return {
    white: renderWhiteProductImage(exportCamera, { normalization: options.normalization }),
    depth: renderDepthImage(exportCamera, { normalization: options.normalization })
  }
}

function normalizeCorrectionAngle(value) {
  let next = Number(value) || 0
  while (next > 180) next -= 360
  while (next < -180) next += 360
  return next
}

function model360WhiteBgAngles(baseY) {
  const step = 360 / MODEL_360_WHITE_BG_COUNT
  return Array.from({ length: MODEL_360_WHITE_BG_COUNT }, (_, index) => normalizeCorrectionAngle(baseY + index * step))
}

function model360AngleTitle(index) {
  const angle = Math.round(index * (360 / MODEL_360_WHITE_BG_COUNT))
  return `360°白底 ${angle}°`
}

async function generateWhiteBg(node = activeModelNodeId.value) {
  if (generating.value) return
  generating.value = true
  renderTaskMessage.value = '生成控制图...'
  const targetNode = node || 'upload'
  error.value = ''

  try {
    await prepareRenderedModelForExport(targetNode)
    if (renderedModelNodeId !== targetNode || !hasRenderedModelMesh()) throw new Error('目标 3D 节点模型还没有加载完成')
    const images = await renderControlImagesFromPreview()
    const modelState = ensureModelNodeState(targetNode)
    const viewGuide = currentModelViewGuide()
    const taskOffsetY = productWhiteTaskCountForModel(targetNode) * 360
    const outputNode = ensureOutputNodeForModel(targetNode, {
      forceNew: true,
      offsetY: taskOffsetY
    })
    const outputState = normalizeOutputNodeState({
      ...ensureOutputNodeState(outputNode),
      sourceNode: targetNode,
      resultUrls: images,
      controlMode: modelState.controlMode,
      sceneType: modelState.sceneType,
      sceneShot: modelState.sceneShot,
      sceneStyle: modelState.sceneStyle,
      sceneMaterial: modelState.sceneMaterial,
      sceneStyleRef: modelState.sceneStyleRef,
      viewGuide
    })
    if (outputNode === 'output') {
      resultUrls.value = images
      resultImageUrl.value = images.white
    }
    workflowOutputNodeStates.value = {
      ...workflowOutputNodeStates.value,
      [outputNode]: outputState
    }
    connectWorkflowNodes(targetNode, outputNode, { undo: false, persist: false })
    const generateNode = createProductWhiteBgGenerateNode(targetNode, outputNode)
    if (modelPreviewOpen.value) {
      await closeModelPreview()
    }
    activeWorkflowNode.value = generateNode || outputNode
    saveActiveProject({ persist: false })
    if (generateNode) {
      renderTaskMessage.value = '提交生图...'
      await generateWorkflowImage(generateNode)
    }
  } catch (e) {
    error.value = e.message || '生成白底图失败'
  } finally {
    generating.value = false
    renderTaskMessage.value = ''
  }
}

async function generateModel360WhiteBg(node = activeModelNodeId.value) {
  if (generating.value) return
  const targetNode = node || 'upload'
  const originalCorrection = normalizeModelCorrection(activeModelCorrection.value)
  const originalCut = normalizeModelCut(activeModelCut.value)
  const angles = model360WhiteBgAngles(originalCorrection.y)
  generating.value = true
  renderTaskMessage.value = '准备360°白底图...'
  error.value = ''

  try {
    await prepareRenderedModelForExport(targetNode)
    if (renderedModelNodeId !== targetNode || !hasRenderedModelMesh()) throw new Error('目标 3D 节点模型还没有加载完成')
    if (modelPreviewOpen.value) {
      await closeModelPreview()
    }
    const baseViewGuide = currentModelViewGuide()
    const startOffset = productWhiteGenerateCountForModel(targetNode)
    let fixedExportDistance = 0
    // Use the widest required framing across the full turn, not just frame 0.
    for (const angle of angles) {
      activeModelCut.value = originalCut
      activeModelCorrection.value = { ...originalCorrection, y: angle }
      applyModelCorrection({ refit: false, persist: false })
      const exportCamera = createExportCamera()
      if (!exportCamera) throw new Error('Camera is not ready')
      fixedExportDistance = Math.max(fixedExportDistance, Number(exportCamera.userData.fitDistance) || 0)
      await new Promise(resolve => requestAnimationFrame(resolve))
    }
    for (let index = 0; index < angles.length; index += 1) {
      const angle = angles[index]
      const title = model360AngleTitle(index)
      renderTaskMessage.value = `360°控制图 ${index + 1}/${angles.length}`
      activeModelCut.value = originalCut
      activeModelCorrection.value = {
        ...originalCorrection,
        y: angle
      }
      applyModelCorrection({ refit: false, persist: false })
      const images = await renderControlImagesFromPreview({
        fixedDistance: fixedExportDistance
      })
      const generateNode = createDirectProductWhiteBgGenerateNode(targetNode, images, {
        batchTitle: title,
        offsetY: (startOffset + index) * 360,
        viewGuide: baseViewGuide
      })
      activeWorkflowNode.value = generateNode || targetNode
      saveActiveProject({ persist: false })
      if (generateNode) {
        renderTaskMessage.value = `360°生图 ${index + 1}/${angles.length}`
        await generateWorkflowImage(generateNode, { undo: false, replaceCurrentNode: true })
        if (generateNodeStateIfExists(generateNode)) await persistActiveProject({ force: true })
      }
    }
  } catch (e) {
    error.value = e.message || '生成360°白底图失败'
  } finally {
    activeModelCut.value = originalCut
    activeModelCorrection.value = originalCorrection
    applyModelCorrection({ refit: false, persist: false })
    generating.value = false
    renderTaskMessage.value = ''
  }
}

async function generateModelWorkflow(node = activeModelNodeId.value) {
  const targetNode = node || 'upload'
  const modelState = ensureModelNodeState(targetNode)
  if (modelState.controlMode !== 'scene') {
    await generateWhiteBg(targetNode)
    return
  }
  if (generating.value) return
  generating.value = true
  renderTaskMessage.value = '生成控制图...'
  error.value = ''

  try {
    await prepareRenderedModelForExport(targetNode)
    if (renderedModelNodeId !== targetNode || !hasRenderedModelMesh()) throw new Error('目标 3D 节点模型还没有加载完成')
    const images = await renderControlImagesFromPreview()
    const currentState = ensureModelNodeState(targetNode)
    const viewGuide = currentModelViewGuide()
    const outputNode = ensureOutputNodeForModel(targetNode, {
      forceNew: true
    })
    const outputState = normalizeOutputNodeState({
      ...ensureOutputNodeState(outputNode),
      sourceNode: targetNode,
      resultUrls: images,
      controlMode: currentState.controlMode,
      sceneType: currentState.sceneType,
      sceneShot: currentState.sceneShot,
      sceneStyle: currentState.sceneStyle,
      sceneMaterial: currentState.sceneMaterial,
      sceneStyleRef: currentState.sceneStyleRef,
      viewGuide
    })
    if (outputNode === 'output') {
      resultUrls.value = images
      resultImageUrl.value = images.white
    }
    workflowOutputNodeStates.value = {
      ...workflowOutputNodeStates.value,
      [outputNode]: outputState
    }
    connectWorkflowNodes(targetNode, outputNode, { undo: false, persist: false })
    const generateNode = ensureScenePresetGenerateNode(targetNode, outputNode, { forceNew: true })
    if (modelPreviewOpen.value) {
      await closeModelPreview()
    }
    activeWorkflowNode.value = generateNode || outputNode
    saveActiveProject({ persist: false })
    if (generateNode) {
      renderTaskMessage.value = '提交生图...'
      await generateWorkflowImage(generateNode)
    }
  } catch (e) {
    error.value = e.message || '生成场景图失败'
  } finally {
    generating.value = false
    renderTaskMessage.value = ''
  }
}

function downloadWhite() {
  if (!resultUrls.value.white) return
  const a = document.createElement('a')
  a.href = resultUrls.value.white
  a.download = `stp-white-${Date.now()}.png`
  a.click()
}

function downloadDepth() {
  if (!resultUrls.value.depth) return
  const a = document.createElement('a')
  a.href = resultUrls.value.depth
  a.download = `stp-depth-${Date.now()}.png`
  a.click()
}

async function loadDrawingModels() {
  try {
    const res = await request.get('/models', { params: { type: 'drawing', _: Date.now() } })
    drawingModels.value = (res.data || []).filter(model => model.is_active && modelHasType(model, 'drawing'))
    const current = drawingModels.value.find(model => model.name === workflowGenerateModel.value)
    if (!current) {
      const defaultModel = drawingModels.value.find(model => model.is_default)
      workflowGenerateModel.value = defaultModel?.name || drawingModels.value[0]?.name || ''
    }
    if (workflowGenerateModel.value) {
      const validModelNames = new Set(drawingModels.value.map(model => model.name))
      const nextStates = { ...workflowGenerateNodeStates.value }
      Object.keys(nextStates).forEach(key => {
        const currentModel = nextStates[key]?.model
        if (!currentModel || !validModelNames.has(currentModel)) {
          const aspectOptions = generateAspectRatioOptionsForModel(workflowGenerateModel.value)
          const currentAspectRatio = nextStates[key]?.aspectRatio
          nextStates[key] = {
            ...nextStates[key],
            model: workflowGenerateModel.value,
            aspectRatio: aspectOptions.includes(currentAspectRatio) ? currentAspectRatio : aspectOptions[0] || '1:1'
          }
        }
      })
      workflowGenerateNodeStates.value = nextStates
      const nextModelStates = { ...workflowModelNodeStates.value }
      let modelStateChanged = false
      Object.keys(nextModelStates).forEach((key) => {
        const currentRatio = nextModelStates[key]?.sceneAspectRatio
        const nextRatio = normalizeModelSceneAspectRatio(currentRatio)
        if (currentRatio !== nextRatio) {
          nextModelStates[key] = {
            ...nextModelStates[key],
            sceneAspectRatio: nextRatio
          }
          modelStateChanged = true
        }
      })
      if (modelStateChanged) workflowModelNodeStates.value = nextModelStates
    }
  } catch {
    drawingModels.value = []
  }
}

function refreshDrawingModelsWhenVisible() {
  if (document.visibilityState === 'visible') loadDrawingModels()
}

function modelHasType(model, type) {
  if (Array.isArray(model?.types)) return model.types.includes(type)
  if (typeof model?.types === 'string') {
    try {
      const parsed = JSON.parse(model.types)
      if (Array.isArray(parsed)) return parsed.includes(type)
    } catch {
      return model.types.includes(type)
    }
  }
  return false
}

function openPreview(url, title, type = 'image') {
  activeWorkflowNode.value = ''
  activeWorkflowLink.value = ''
  closeAllGenerateMenus()
  previewZoom.value = 1
  previewImage.value = { url, title, type: type === 'video' || isVideoAssetUrl(url) ? 'video' : 'image' }
}

async function loadChatModels() {
  try {
    const res = await request.get('/models', { params: { type: 'chat' } })
    chatModels.value = (res.data || []).filter(model => model.is_active && modelHasType(model, 'chat'))
    const validModelNames = new Set(chatModels.value.map(model => model.name))
    const fallbackModel = defaultChatModelName()
    const nextStates = { ...workflowLlmNodeStates.value }
    Object.keys(nextStates).forEach((node) => {
      if (!nextStates[node]?.model || !validModelNames.has(nextStates[node].model)) {
        nextStates[node] = { ...nextStates[node], model: fallbackModel }
      }
    })
    workflowLlmNodeStates.value = nextStates
  } catch {
    chatModels.value = []
  }
}

function openVideoPreview(url, title = '视频生成结果') {
  openPreview(url, title, 'video')
  nextTick(() => {
    const video = previewVideoRef.value
    if (!video) return
    video.currentTime = 0
    video.play?.().catch(() => {})
  })
}

function onVideoPreviewClick(url, title = '视频生成结果') {
  if (suppressNodeClick) {
    suppressNodeClick = false
    return
  }
  openVideoPreview(url, title)
}

async function downloadVideoNode(node) {
  const url = videoNodeState(node).videoUrl
  if (!url) return
  const filename = `generated-video-${Date.now()}.mp4`
  await downloadImage(resolveImageUrl(url), filename, { forceSkuPicker: true, filename })
}

function openGenerateNodePreview(node) {
  const url = primaryGenerationImage(node)
  if (!url) return
  openPreview(url, '生图结果')
}

function closePreview() {
  previewImage.value = { url: '', title: '', type: 'image' }
  previewZoom.value = 1
}

function setPreviewZoom(value) {
  previewZoom.value = Math.round(THREE.MathUtils.clamp(Number(value) || 1, 0.25, 4) * 100) / 100
}

function handlePreviewWheel(e) {
  setPreviewZoom(previewZoom.value + (e.deltaY < 0 ? 0.12 : -0.12))
}

function handleWorkflowKeydown(e) {
  if (!activeProjectId.value) return
  if (showShortcutModal.value) {
    if (e.key === 'Escape') {
      e.preventDefault()
      closeShortcutModal()
    }
    return
  }
  if (e.key === 'Shift') {
    isShiftPressed = true
    syncModelControlMouseButtons()
    return
  }
  const target = e.target
  if (target?.closest?.('textarea.text-node-input[readonly], textarea.llm-node-textarea[readonly]') && activeWorkflowNode.value && ['Backspace', 'Delete'].includes(e.key)) {
    e.preventDefault()
    removeWorkflowNode(activeWorkflowNode.value)
    return
  }
  if (target?.closest?.('input, textarea, select, [contenteditable="true"]')) return

  if (e.code === 'Space') {
    e.preventDefault()
    isSpacePressed = true
    return
  }

  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z' && !e.shiftKey) {
    e.preventDefault()
    undoWorkflowChange()
    return
  }

  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c') {
    const nodes = currentClipboardNodeIds()
    if (nodes.length) {
      e.preventDefault()
      copyWorkflowSelection()
    }
    return
  }

  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'v') {
    if (workflowClipboard.value?.nodes?.length) {
      e.preventDefault()
      pasteWorkflowSelection()
    }
    return
  }

  if (!activeWorkflowNode.value && !activeWorkflowLink.value && !activeGroupId.value && !selectedNodeIds.value.length) return
  if (!['Backspace', 'Delete'].includes(e.key)) return
  e.preventDefault()
  if (selectedNodeIds.value.length) {
    removeSelectedWorkflowNodes()
    return
  }
  if (activeGroupId.value) {
    removeWorkflowGroupWithNodes(activeGroupId.value)
    return
  }
  if (activeWorkflowLink.value) {
    removeWorkflowLink(activeWorkflowLink.value)
    return
  }
  removeWorkflowNode(activeWorkflowNode.value)
}

function handleWorkflowKeyup(e) {
  if (e.code === 'Space') {
    isSpacePressed = false
  }
  if (e.key === 'Shift') {
    isShiftPressed = false
    syncModelControlMouseButtons()
  }
}

function cleanup() {
  if (animationId) cancelAnimationFrame(animationId)
  animationId = null
  modelRenderPending = false
  window.removeEventListener('pointerup', endViewCubeDrag)
  window.removeEventListener('pointercancel', cancelViewCubeDrag)
  if (controls) controls.dispose()
  if (renderer) {
    renderer.dispose()
    renderer.forceContextLoss()
    renderer.domElement?.parentNode?.removeChild(renderer.domElement)
  }
  if (sceneGroundGrid) {
    sceneGroundGrid.geometry?.dispose?.()
    sceneGroundGrid.material?.dispose?.()
    sceneGroundGrid = null
  }
  if (modelRoot) {
    disposeObject3D(modelRoot)
    scene?.remove?.(modelRoot)
    modelRoot = null
    mesh = null
  } else if (mesh) {
    mesh.geometry.dispose()
    if (mesh.material) mesh.material.dispose()
  }
  window.removeEventListener('resize', onResize)
  window.removeEventListener('resize', bumpViewportCheck)
  window.removeEventListener('keydown', handleWorkflowKeydown)
  window.removeEventListener('keyup', handleWorkflowKeyup)
  window.removeEventListener('pagehide', flushProjectSaveBeforeUnload)
  window.removeEventListener('beforeunload', flushProjectSaveBeforeUnload)
  window.removeEventListener('pointerdown', closeNodeMenuOnOutside)
  window.removeEventListener('pointerdown', closeTextNodeEditOnOutside, true)
  window.removeEventListener('pointermove', dragNode)
  window.removeEventListener('pointerup', endNodeDrag)
  window.removeEventListener('pointercancel', endNodeDrag)
  window.removeEventListener('pointermove', dragTextNodeResize)
  window.removeEventListener('pointerup', endTextNodeResize)
  window.removeEventListener('pointercancel', endTextNodeResize)
}

onMounted(async () => {
  await loadDrawingModels()
  modelPollTimer = window.setInterval(loadDrawingModels, 10_000)
  document.addEventListener('visibilitychange', refreshDrawingModelsWhenVisible)
  await loadChatModels()
  loadModelSceneMaterials()
  window.addEventListener('keydown', handleWorkflowKeydown)
  window.addEventListener('keyup', handleWorkflowKeyup)
  window.addEventListener('pagehide', flushProjectSaveBeforeUnload)
  window.addEventListener('beforeunload', flushProjectSaveBeforeUnload)
  window.addEventListener('pointerdown', closeNodeMenuOnOutside)
  window.addEventListener('pointerdown', closeTextNodeEditOnOutside, true)
  window.addEventListener('resize', bumpViewportCheck)
  await loadWorkflowProjects()
  await nextTick()
  if (activeProjectId.value) canvasReady.value = true
  bumpViewportCheck()
  if (!activeProjectId.value || modelWorkflowNodes().length) initThreeJS()
})

onUnmounted(() => {
  if (modelPollTimer) window.clearInterval(modelPollTimer)
  document.removeEventListener('visibilitychange', refreshDrawingModelsWhenVisible)
  if (canvasPanFrame) window.cancelAnimationFrame(canvasPanFrame)
  if (canvasWheelFrame) window.cancelAnimationFrame(canvasWheelFrame)
  if (canvasFloatingScaleTimer) window.clearTimeout(canvasFloatingScaleTimer)
  if (nodeDragFrame) window.cancelAnimationFrame(nodeDragFrame)
  if (selectionMoveFrame) window.cancelAnimationFrame(selectionMoveFrame)
  if (groupDragFrame) window.cancelAnimationFrame(groupDragFrame)
  canvasPanFrame = null
  canvasWheelFrame = null
  canvasFloatingScaleTimer = null
  nodeDragFrame = null
  selectionMoveFrame = null
  groupDragFrame = null
  pendingCanvasPanEvent = null
  pendingCanvasWheel = null
  pendingNodeDragEvent = null
  pendingSelectionMoveEvent = null
  pendingGroupDragEvent = null
  isSpacePressed = false
  if (canvasSaveTimer) {
    window.clearTimeout(canvasSaveTimer)
    canvasSaveTimer = null
  }
  if (layoutSaveTimer) {
    window.clearTimeout(layoutSaveTimer)
    layoutSaveTimer = null
  }
  if (viewportCheckTimer) {
    window.clearTimeout(viewportCheckTimer)
    viewportCheckTimer = null
  }
  textNodeInputSyncTimers.forEach(timer => window.clearTimeout(timer))
  llmNodeInputSyncTimers.forEach(timer => window.clearTimeout(timer))
  textNodeInputSyncTimers.clear()
  llmNodeInputSyncTimers.clear()
  if (errorDismissTimer) {
    window.clearTimeout(errorDismissTimer)
    errorDismissTimer = null
  }
  syncCanvasViewFromLive({ persist: false, viewport: true })
  saveActiveProject()
  flushProjectDraft()
  if (projectSaveTimer) {
    window.clearTimeout(projectSaveTimer)
    projectSaveTimer = null
    persistActiveProject()
  }
  cleanup()
})
</script>

<style scoped>
.stp-white-bg {
  display: flex;
  height: calc(100vh - 64px);
  background: var(--color-bg);
}

.preview-panel {
  flex: 1 1 auto;
  min-width: 0;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  background:
    radial-gradient(circle at 50% 38%, rgba(255, 255, 255, 0.72), rgba(238, 242, 246, 0.12) 42%, rgba(226, 232, 240, 0.36) 100%),
    #eef2f6;
  display: flex;
  align-items: center;
  justify-content: center;
}

.workspace-toolbar {
  position: absolute;
  top: 16px;
  left: 16px;
  right: 168px;
  z-index: 12;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  pointer-events: none;
}

.mode-tabs,
.workflow-steps {
  pointer-events: auto;
  display: flex;
  align-items: center;
  padding: 4px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(12px);
}

.mode-tabs button {
  height: 32px;
  padding: 0 14px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.mode-tabs button.active {
  background: #111827;
  color: #fff;
}

.workflow-steps {
  gap: 6px;
  padding: 6px 8px;
  color: #64748b;
  font-size: 12px;
}

.workflow-steps span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.workflow-steps span::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #cbd5e1;
}

.workflow-steps span.done {
  color: #0f172a;
}

.workflow-steps span.done::before {
  background: #22c55e;
}

.model-workspace,
.workflow-workspace {
  position: absolute;
  inset: 0;
}

.ground-grid {
  position: absolute;
  left: -12%;
  right: -12%;
  bottom: -8%;
  height: 42%;
  z-index: 0;
  pointer-events: none;
  display: block;
  background-image:
    linear-gradient(rgba(71, 85, 105, 0.42) 1px, transparent 1px),
    linear-gradient(90deg, rgba(71, 85, 105, 0.42) 1px, transparent 1px);
  background-size: 40px 40px;
  transform: perspective(900px) rotateX(66deg);
  transform-origin: center bottom;
  opacity: 0.68;
  -webkit-mask-image: linear-gradient(to top, rgba(0, 0, 0, 0.9), rgba(0, 0, 0, 0.4) 72%, transparent);
  mask-image: linear-gradient(to top, rgba(0, 0, 0, 0.9), rgba(0, 0, 0, 0.4) 72%, transparent);
}

.viewer-container {
  width: 100%;
  height: 100%;
  display: none;
  position: relative;
  z-index: 1;
}

.viewer-container.visible {
  display: block;
  cursor: inherit;
}

.view-cube {
  position: absolute;
  top: 24px;
  right: 24px;
  width: 132px;
  height: 132px;
  z-index: 6;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.view-cube__stage {
  width: 92px;
  height: 92px;
  perspective: 520px;
  filter: drop-shadow(0 12px 24px rgba(15, 23, 42, 0.12));
}

.view-cube__body {
  position: relative;
  width: 92px;
  height: 92px;
  transform-style: preserve-3d;
  transform: rotateX(-24deg) rotateY(36deg);
  pointer-events: auto;
  touch-action: none;
}

.view-cube__face {
  position: absolute;
  inset: 0;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-template-rows: repeat(3, 1fr);
  border: 1px solid rgba(255, 255, 255, 0.92);
  background: rgba(231, 239, 252, 0.94);
  color: #334155;
  font-size: 15px;
  font-weight: 700;
  line-height: 1;
  user-select: none;
  transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
  backface-visibility: hidden;
}

.view-cube__cell {
  min-width: 0;
  min-height: 0;
  padding: 0;
  border: 0;
  border-right: 1px solid rgba(148, 163, 184, 0.14);
  border-bottom: 1px solid rgba(148, 163, 184, 0.14);
  background: transparent;
  color: inherit;
  font: inherit;
  line-height: 1;
  cursor: pointer;
}

.view-cube__cell:nth-child(3n) {
  border-right: 0;
}

.view-cube__cell:nth-child(n + 7) {
  border-bottom: 0;
}

.view-cube__cell:hover {
  background: rgba(147, 197, 253, 0.35);
}

.view-cube__cell--center {
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.12);
}

.view-cube__cell--center:hover {
  color: #111827;
  box-shadow: inset 0 0 0 2px rgba(59, 130, 246, 0.22);
}

.view-cube__face--front {
  transform: translateZ(46px);
}

.view-cube__face--back {
  transform: rotateY(180deg) translateZ(46px);
}

.view-cube__face--right {
  transform: rotateY(90deg) translateZ(46px);
}

.view-cube__face--left {
  transform: rotateY(-90deg) translateZ(46px);
}

.view-cube__face--top {
  transform: rotateX(90deg) translateZ(46px);
}

.view-cube__face--bottom {
  transform: rotateX(-90deg) translateZ(46px);
}

.upload-hint, .loading-state {
  text-align: center;
  color: #666;
}

.workflow-workspace {
  z-index: 2;
  padding: 72px 32px 32px;
  overflow: auto;
}

.workflow-canvas {
  position: relative;
  width: min(1180px, 100%);
  min-width: 960px;
  height: 560px;
  margin: 0 auto;
  border: 1px solid rgba(148, 163, 184, 0.26);
  border-radius: 18px;
  background:
    linear-gradient(rgba(148, 163, 184, 0.12) 1px, transparent 1px),
    linear-gradient(90deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px),
    #f8fafc;
  background-size: 28px 28px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.9), 0 24px 80px rgba(15, 23, 42, 0.1);
}

.workflow-lines {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.workflow-lines path {
  fill: none;
  stroke: #94a3b8;
  stroke-width: 2;
  stroke-dasharray: 8 8;
}

.workflow-node {
  position: absolute;
  width: 220px;
  min-height: 156px;
  padding: 16px;
  border: 1px solid #dbe3ef;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.94);
  color: #111827;
  text-align: left;
  box-shadow: 0 18px 50px rgba(15, 23, 42, 0.1);
  cursor: pointer;
  transition: border-color 0.18s, box-shadow 0.18s, transform 0.18s;
}

button.workflow-node {
  font: inherit;
}

.workflow-node:hover,
.workflow-node.active {
  border-color: #111827;
  box-shadow: 0 22px 60px rgba(15, 23, 42, 0.16);
  transform: translateY(-2px);
}

.workflow-node.done::after {
  content: none;
}

.workflow-node--input { left: 38px; top: 80px; }
.workflow-node--camera { left: 368px; top: 80px; }
.workflow-node--control { left: 720px; top: 80px; }
.workflow-node--scene { left: 200px; top: 330px; }
.workflow-node--placement { left: 700px; top: 306px; width: 320px; min-height: 212px; }

.node-index {
  display: inline-flex;
  margin-bottom: 10px;
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.workflow-node h3 {
  margin: 0 0 8px;
  color: #0f172a;
  font-size: 17px;
  line-height: 1.2;
}

.workflow-node p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
  line-height: 1.55;
}

.node-status {
  position: absolute;
  left: 16px;
  bottom: 14px;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
}

.node-action,
.node-upload,
.secondary-btn {
  height: 38px;
  padding: 0 16px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.node-action {
  margin-top: 14px;
}

.node-action:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.node-action.primary,
.node-upload {
  background: #111827;
  color: #fff;
  border-color: #111827;
}

.workflow-node input[type="file"] {
  display: none;
}

.node-thumbs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.node-thumbs img,
.node-scene-preview img {
  width: 100%;
  aspect-ratio: 1 / 1;
  object-fit: contain;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.node-scene-preview {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
  gap: 10px;
}

.node-scene-preview button {
  height: 34px;
  padding: 0 12px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.node-placement-stage {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
  overflow: hidden;
}

.scene-base-image {
  display: block;
  width: 100%;
  max-height: min(78vh, 820px);
  object-fit: contain;
  background: #111;
  user-select: none;
  pointer-events: none;
}

.scene-safe-frame {
  position: absolute;
  inset: 6%;
  border: 1px dashed rgba(59, 130, 246, 0.38);
  pointer-events: none;
}

.scene-product-layer {
  position: absolute;
  touch-action: none;
  cursor: move;
  outline: 1px solid rgba(37, 99, 235, 0.7);
  outline-offset: 4px;
  transform-origin: center;
}

.scene-product-layer img {
  display: block;
  width: 100%;
  height: auto;
  background: #fff;
  user-select: none;
  pointer-events: none;
  mix-blend-mode: multiply;
}

.placement-handle {
  position: absolute;
  width: 10px;
  height: 10px;
  border: 2px solid #2563eb;
  border-radius: 50%;
  background: #fff;
}

.placement-handle--tl { left: -9px; top: -9px; }
.placement-handle--tr { right: -9px; top: -9px; }
.placement-handle--bl { left: -9px; bottom: -9px; }
.placement-handle--br { right: -9px; bottom: -9px; }

.scene-product-placeholder {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  padding: 10px 14px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.72);
  color: #fff;
  font-size: 13px;
}

.control-panel {
  flex: 0 0 clamp(320px, 28vw, 420px);
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel-section { }
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  margin-bottom: 12px;
}

.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 120px;
  border: 2px dashed var(--color-border);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.drop-zone:hover, .drop-zone.dragging {
  border-color: var(--color-primary);
  background: var(--color-primary-alpha);
}

.drop-zone input[type="file"] {
  display: none;
}

.drop-content { text-align: center; }
.file-info { display: flex; align-items: center; gap: 8px; }
.file-name { font-size: 14px; color: var(--color-text); }

.angle-display {
  display: flex;
  gap: 16px;
  background: var(--color-bg);
  border-radius: 8px;
  padding: 16px;
}

.angle-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.angle-item .label {
  font-size: 12px;
  color: #666;
}

.angle-item .value {
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.generate-btn {
  width: 100%;
  padding: 14px;
  background: #111;
  color: white;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.generate-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-ring-sm {
  width: 16px;
  height: 16px;
  border: 2px solid transparent;
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-orbit {
  width: 48px;
  height: 48px;
  position: relative;
  margin: 0 auto;
}

.loading-ring {
  position: absolute;
  inset: 0;
  border: 3px solid transparent;
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-core {
  position: absolute;
  inset: 8px;
  background: #fff;
  border-radius: 50%;
  opacity: 0.4;
}

.error-msg {
  color: #ef4444;
  font-size: 13px;
  text-align: center;
}

.scene-panel-section {
  padding: 14px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #f8fafc;
}

.scene-file-name {
  margin: 8px 0 0;
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.placement-controls {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 14px;
}

.placement-controls label {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  color: #475569;
  font-size: 12px;
}

.placement-controls input[type="range"] {
  width: 100%;
  accent-color: #111827;
}

.placement-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 14px;
}

.result-panel {
  position: fixed;
  inset: 64px 0 0;
  background: white;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(420px, 100%), 1fr));
  grid-template-rows: minmax(0, 1fr) auto;
  align-items: center;
  gap: 20px;
  z-index: 50;
  padding: 20px;
  overflow-y: auto;
}

.result-panel--inline {
  position: static;
  inset: auto;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 14px;
  padding: 14px;
  overflow: visible;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  z-index: auto;
}

.result-image-item {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.result-panel:has(.result-image-item:nth-child(2)) {
  justify-content: initial;
}

.result-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
}

.result-image-wrapper {
  width: min(100%, 720px);
  max-height: min(70vh, 720px);
  aspect-ratio: 1 / 1;
  background: white;
  box-shadow: 0 4px 24px rgba(0,0,0,0.1);
  padding: clamp(8px, 1.5vw, 16px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.result-panel--inline .result-image-wrapper {
  width: 100%;
  max-height: none;
  padding: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
  box-shadow: none;
  cursor: zoom-in;
}

.result-image {
  max-width: 100%;
  max-height: 100%;
  display: block;
}

.result-panel--inline .result-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.result-actions {
  grid-column: 1 / -1;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
  padding-bottom: max(0px, env(safe-area-inset-bottom));
}

.result-panel--inline .result-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 0;
}

.action-btn {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid #ddd;
  background: white;
  transition: all 0.15s;
}

.action-btn:hover { background: #f5f5f5; }

.image-modal {
  position: fixed;
  inset: 0;
  z-index: 220;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(28px, 5vw, 64px);
  overflow: hidden;
  background: rgba(0, 0, 0, 0.78);
}

.image-modal__close {
  position: fixed;
  right: 28px;
  top: 24px;
  z-index: 2;
  width: 38px;
  height: 38px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: rgba(255, 255, 255, 0.72);
  font-size: 34px;
  line-height: 1;
  cursor: pointer;
}

.image-modal__close:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.image-modal__img {
  display: block;
  max-width: min(82vw, 1120px);
  max-height: min(82vh, 820px);
  object-fit: contain;
  transform-origin: center center;
  transition: transform 120ms ease;
}

.image-modal__toolbar {
  position: fixed;
  left: 50%;
  bottom: 28px;
  z-index: 3;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 10px;
  transform: translateX(-50%);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  background: rgba(22, 22, 22, 0.86);
  color: #f8fafc;
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.38);
  backdrop-filter: blur(14px);
}

.image-modal__toolbar button {
  min-width: 34px;
  height: 30px;
  padding: 0 10px;
  border: 0;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: 14px;
  font-weight: 800;
  cursor: pointer;
}

.image-modal__toolbar button:hover {
  background: rgba(255, 255, 255, 0.18);
}

.image-modal__toolbar span {
  min-width: 54px;
  color: #e5e7eb;
  font-size: 13px;
  font-weight: 800;
  text-align: center;
}

.model-modal {
  position: fixed;
  inset: 0;
  z-index: 85;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 18px;
  background: rgba(0, 0, 0, 0.82);
}

.model-modal__content {
  width: min(96vw, 1520px);
  height: min(92vh, 940px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  border-radius: 14px;
  background: #6f7174;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.32);
}

.model-modal__header {
  height: 60px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(18, 18, 18, 0.96);
  color: #f8fafc;
}

.model-modal__header div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.model-modal__header strong,
.model-modal__header span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-modal__header strong {
  font-size: 15px;
}

.model-modal__header span {
  color: #a1a1aa;
  font-size: 12px;
}

.model-modal__body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background:
    radial-gradient(circle at 50% 36%, rgba(184, 186, 190, 0.72), rgba(124, 126, 130, 0.88) 48%, rgba(90, 92, 96, 0.98) 100%),
    #85878b;
}

.model-modal__viewer {
  position: absolute;
  inset: 0 400px 0 0;
  z-index: 2;
  pointer-events: auto;
  cursor: grab;
}

.model-modal__viewer:active {
  cursor: grabbing;
}

.model-modal__viewer canvas {
  background: #8a8a8a !important;
  pointer-events: auto;
}

.model-modal-loading {
  position: absolute;
  inset: 0 400px 0 0;
  z-index: 6;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 32px;
  background: rgba(25, 25, 25, 0.26);
  color: #f8fafc;
  text-align: center;
  pointer-events: auto;
}

.model-modal-loading strong {
  font-size: 18px;
  font-weight: 900;
}

.model-modal-loading p {
  max-width: 360px;
  margin: 0;
  color: #e5e7eb;
  font-size: 13px;
  font-weight: 700;
  line-height: 1.6;
}

.model-modal-loading button {
  height: 38px;
  padding: 0 18px;
  border: 0;
  border-radius: 10px;
  background: #f8fafc;
  color: #111827;
  font-size: 13px;
  font-weight: 900;
  cursor: pointer;
}

.model-modal-loading--error {
  background: rgba(24, 24, 27, 0.42);
}

.model-modal-hint {
  position: absolute;
  left: 24px;
  bottom: 22px;
  z-index: 7;
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 999px;
  background: rgba(17, 24, 39, 0.48);
  color: #f8fafc;
  font-size: 12px;
  font-weight: 800;
  pointer-events: none;
  backdrop-filter: blur(10px);
}

.model-modal .view-cube {
  top: 24px;
  right: 424px;
  z-index: 7;
}

.model-orientation-panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: 8;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 400px;
  padding: 16px;
  overflow: hidden;
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(13, 13, 13, 0.94);
  color: #f8fafc;
  box-shadow: -18px 0 50px rgba(0, 0, 0, 0.34), inset 1px 0 0 rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(16px);
}

.model-orientation-panel__scroll {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  padding-right: 4px;
}

.model-orientation-panel__scroll::-webkit-scrollbar {
  width: 6px;
}

.model-orientation-panel__scroll::-webkit-scrollbar-track {
  background: transparent;
}

.model-orientation-panel__scroll::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.22);
}

.model-orientation-section,
.model-cut-panel,
.model-export-panel {
  padding: 14px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.045);
}

.model-orientation-panel__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
}

.model-orientation-panel__head strong,
.model-cut-panel__head strong {
  color: #f8fafc;
  font-size: 15px;
  font-weight: 900;
}

.model-orientation-panel__head span,
.model-cut-panel__head span {
  color: #a1a1aa;
  font-size: 12px;
  font-weight: 700;
}

.model-orientation-row {
  display: grid;
  grid-template-columns: 44px 58px minmax(0, 1fr) 58px 54px;
  align-items: center;
  gap: 10px;
  min-height: 36px;
  margin-top: 10px;
  font-size: 12px;
}

.model-orientation-row > span {
  color: #d4d4d8;
  font-weight: 800;
}

.model-orientation-row button,
.model-orientation-actions button {
  height: 34px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.1);
  color: #f8fafc;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease;
}

.model-orientation-row button:hover,
.model-orientation-actions button:hover {
  border-color: rgba(255, 255, 255, 0.22);
  background: rgba(255, 255, 255, 0.18);
}

.model-orientation-row input {
  width: 100%;
  height: 4px;
  accent-color: #fff;
  cursor: pointer;
}

.model-orientation-row input:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.model-orientation-row em {
  color: #e5e7eb;
  font-size: 12px;
  font-style: normal;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.model-orientation-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}

.model-orientation-actions button {
  padding: 0 14px;
}

.model-cut-panel,
.model-export-panel {
  margin-top: 0;
}

.model-cut-panel__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.model-cut-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.08);
  color: #e5e7eb;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
}

.model-cut-toggle input {
  width: 14px;
  height: 14px;
  accent-color: #fff;
}

.model-cut-row {
  margin-top: 12px;
}

.model-segment {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  gap: 6px;
  padding: 4px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.07);
}

.model-segment button {
  height: 34px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: #a1a1aa;
  font-size: 12px;
  font-weight: 900;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease, box-shadow 140ms ease;
}

.model-segment button:hover {
  color: #f8fafc;
}

.model-segment button.active {
  background: #f8fafc;
  color: #111827;
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.28);
}

.model-shot-options {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.model-field-label {
  display: block;
  margin-top: 12px;
  color: #d4d4d8;
  font-size: 13px;
  font-weight: 800;
  line-height: 1.2;
}

.model-segment + .model-field-label,
.model-material-options + .model-field-label,
.model-shot-options + .model-field-label {
  margin-top: 14px;
}

.model-material-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 8px;
}

.model-material-options button {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
  height: 44px;
  padding: 6px 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.06);
  color: #d4d4d8;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease, color 140ms ease, box-shadow 140ms ease;
}

.model-material-options button:hover {
  border-color: rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.1);
  color: #f8fafc;
}

.model-material-options button.active {
  border-color: #f8fafc;
  background: rgba(248, 250, 252, 0.14);
  box-shadow: inset 0 0 0 1px rgba(248, 250, 252, 0.4);
  color: #fff;
}

.model-material-options i {
  display: grid;
  overflow: hidden;
  width: 28px;
  height: 28px;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 8px;
  background: linear-gradient(135deg, #3f3f46, #18181b);
  color: #f8fafc;
  font-size: 12px;
  font-style: normal;
  font-weight: 900;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.24), 0 6px 16px rgba(0, 0, 0, 0.28);
}

.model-material-options img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.model-material-options strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 900;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-aspect-options {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.model-aspect-options button {
  display: grid;
  min-width: 0;
  height: 50px;
  place-items: center;
  gap: 4px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.06);
  color: #a1a1aa;
  cursor: pointer;
}

.model-aspect-options button:hover {
  border-color: rgba(255, 255, 255, 0.18);
  color: #f8fafc;
}

.model-aspect-options button.active {
  border-color: #f8fafc;
  background: #f8fafc;
  color: #111827;
}

.model-aspect-options strong {
  font-size: 11px;
  font-weight: 900;
  line-height: 1;
}

.model-aspect-options .generate-aspect-icon {
  width: var(--aspect-icon-width, 18px);
  height: var(--aspect-icon-height, 18px);
}

.model-style-picker {
  display: grid;
  width: 100%;
  grid-template-columns: 64px minmax(0, 1fr);
  grid-template-rows: auto auto;
  align-items: center;
  gap: 4px 10px;
  min-height: 64px;
  padding: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.07);
  color: #f8fafc;
  text-align: left;
  cursor: pointer;
}

.model-style-picker:hover {
  border-color: rgba(255, 255, 255, 0.18);
  background: rgba(255, 255, 255, 0.1);
}

.model-style-picker img {
  grid-row: 1 / span 2;
  width: 64px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
  background: rgba(255, 255, 255, 0.08);
}

.model-style-picker strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-style-picker small {
  overflow: hidden;
  color: #a1a1aa;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-modal__generate {
  flex: 0 0 auto;
  width: 100%;
  height: 46px;
  margin-top: 2px;
  padding: 0 22px;
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 12px;
  background: #f8fafc;
  box-shadow: 0 -10px 24px rgba(0, 0, 0, 0.2), 0 12px 28px rgba(0, 0, 0, 0.24);
  color: #111827;
  font-size: 14px;
  font-weight: 900;
  cursor: pointer;
}

.model-modal__generate:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.model-modal__generate--secondary {
  margin-top: 8px;
  background: rgba(255, 255, 255, 0.08);
  color: #f8fafc;
}

.model-modal__generate--secondary:not(:disabled):hover {
  border-color: rgba(255, 255, 255, 0.28);
  background: rgba(255, 255, 255, 0.12);
}

.action-btn.primary {
  background: #111;
  color: white;
  border-color: #111;
}

.action-btn.primary:hover { background: #333; }

@media (max-width: 1024px) {
  .model-modal__viewer {
    inset: 0 340px 0 0;
  }

  .model-modal-loading {
    inset: 0 340px 0 0;
  }

  .model-orientation-panel {
    width: 340px;
    padding: 12px;
  }

  .model-modal .view-cube {
    right: 364px;
  }

  .model-orientation-row {
    grid-template-columns: 38px 50px minmax(0, 1fr) 50px 46px;
    gap: 8px;
  }

  .stp-white-bg {
    flex-direction: column;
    min-height: calc(100vh - 64px);
    height: auto;
  }

  .preview-panel {
    flex: none;
    height: min(58vh, 560px);
    min-height: 360px;
  }

  .control-panel {
    flex: none;
    border-left: 0;
    border-top: 1px solid var(--color-border);
  }

  .workspace-toolbar {
    right: 16px;
    flex-wrap: wrap;
    align-items: flex-start;
  }

  .workflow-workspace {
    padding: 92px 16px 20px;
  }

  .workflow-canvas {
    min-width: 920px;
  }
}

@media (max-width: 900px) {
  .result-panel:not(.result-panel--inline) {
    grid-template-columns: 1fr;
    align-items: start;
    gap: 16px;
    padding: 14px;
  }

  .result-panel:not(.result-panel--inline) .result-image-wrapper {
    width: min(100%, 560px);
    max-height: 58vh;
  }
}

@media (max-width: 640px) {
  .preview-panel {
    height: 52vh;
    min-height: 300px;
  }

  .view-cube {
    top: 12px;
    right: 12px;
    width: 104px;
    height: 104px;
    transform: scale(0.82);
    transform-origin: top right;
  }

  .control-panel {
    padding: 16px;
    gap: 16px;
  }

  .workspace-toolbar {
    top: 12px;
    left: 12px;
    right: 92px;
    gap: 8px;
  }

  .workflow-steps {
    display: none;
  }

  .mode-tabs button {
    height: 30px;
    padding: 0 10px;
    font-size: 12px;
  }

  .workflow-workspace {
    padding: 62px 12px 16px;
  }

  .workflow-canvas {
    min-width: 860px;
    height: 540px;
  }

  .placement-actions,
  .result-panel--inline .result-actions {
    grid-template-columns: 1fr;
  }

  .angle-display {
    gap: 8px;
    padding: 12px;
  }

  .result-panel:not(.result-panel--inline) {
    inset: 64px 0 0;
    padding: 12px;
  }

  .result-panel:not(.result-panel--inline) .result-image-wrapper {
    max-height: 44vh;
  }

  .result-panel:not(.result-panel--inline) .result-actions {
    position: sticky;
    bottom: 0;
    margin: 0 -12px -12px;
    padding: 10px 12px 12px;
    background: rgba(255, 255, 255, 0.96);
    border-top: 1px solid #eee;
  }

  .action-btn {
    min-width: 0;
  }
}
.workflow-editor {
  position: relative;
  height: calc(100vh - 64px);
  min-height: 720px;
  overflow: hidden;
  background:
    radial-gradient(circle, rgba(148, 163, 184, 0.2) 1px, transparent 1.4px),
    #101112;
  background-size: 24px 24px;
  color: #e5e7eb;
}

.hidden-input {
  display: none;
}

.project-create-page {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background:
    radial-gradient(circle at 50% 36%, rgba(59, 130, 246, 0.1), transparent 34%),
    rgba(16, 17, 18, 0.96);
}

.project-create-card {
  width: min(520px, 100%);
  padding: 34px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 16px;
  background: #202020;
  box-shadow: 0 24px 90px rgba(0, 0, 0, 0.4);
}

.project-create-card__eyebrow {
  color: #9ca3af;
  font-size: 12px;
  font-weight: 700;
}

.project-create-card h1 {
  margin: 12px 0 10px;
  color: #f8fafc;
  font-size: 26px;
  line-height: 1.2;
}

.project-create-card p {
  margin: 0;
  color: #a1a1aa;
  font-size: 14px;
  line-height: 1.7;
}

.project-create-form {
  display: flex;
  gap: 10px;
  margin-top: 24px;
}

.project-create-form input {
  flex: 1;
  min-width: 0;
  height: 42px;
  padding: 0 12px;
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 10px;
  background: #151515;
  color: #f8fafc;
  outline: none;
}

.project-create-form button,
.project-create-list button,
.project-sidebar__header button,
.project-item__delete,
.project-name-modal__actions button {
  border: 0;
  cursor: pointer;
}

.project-create-form button {
  height: 42px;
  padding: 0 16px;
  border-radius: 10px;
  background: #f8fafc;
  color: #111827;
  font-weight: 800;
}

.project-create-list {
  display: grid;
  gap: 8px;
  margin-top: 22px;
}

.project-create-list span {
  color: #8b929e;
  font-size: 12px;
}

.project-create-list button {
  height: 38px;
  padding: 0 12px;
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.06);
  color: #e5e7eb;
  text-align: left;
}

.project-create-list button:hover {
  background: rgba(255, 255, 255, 0.1);
}

.project-sidebar {
  position: absolute;
  left: 16px;
  top: 16px;
  bottom: 16px;
  z-index: 25;
  width: 210px;
  padding: 12px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 14px;
  background: rgba(31, 31, 31, 0.92);
  box-shadow: 0 20px 70px rgba(0, 0, 0, 0.34);
  backdrop-filter: blur(12px);
}

.project-sidebar__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: #f8fafc;
  font-size: 13px;
  font-weight: 800;
}

.project-sidebar__header button {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: #f8fafc;
  color: #111827;
  font-size: 18px;
  line-height: 1;
}

.project-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: #e5e7eb;
  text-align: left;
  cursor: pointer;
}

.project-item__text {
  flex: 1;
  min-width: 0;
}

.project-item + .project-item {
  margin-top: 6px;
}

.project-item:hover,
.project-item.active {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.08);
}

.project-item strong {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-item strong {
  font-size: 13px;
}

.project-rename-input {
  width: 100%;
  height: 26px;
  padding: 0 8px;
  border: 1px solid rgba(248, 250, 252, 0.32);
  border-radius: 7px;
  background: #151515;
  color: #f8fafc;
  font-size: 13px;
  font-weight: 700;
  outline: none;
}

.project-item__delete {
  width: 24px;
  height: 24px;
  flex: 0 0 auto;
  border-radius: 7px;
  background: transparent;
  color: #8b929e;
  font-size: 18px;
  line-height: 1;
  opacity: 0;
}

.project-item:hover .project-item__delete {
  opacity: 1;
}

.project-item__delete:hover {
  background: rgba(248, 113, 113, 0.14);
  color: #fecaca;
}

.project-name-modal {
  position: absolute;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(0, 0, 0, 0.48);
  backdrop-filter: blur(6px);
}

.project-name-modal__card {
  width: min(420px, 100%);
  padding: 22px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 14px;
  background: #202020;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.44);
}

.project-name-modal__card h3 {
  margin: 0;
  color: #f8fafc;
  font-size: 18px;
}

.project-name-modal__card p {
  margin: 8px 0 14px;
  color: #a1a1aa;
  font-size: 13px;
}

.project-name-modal__card input {
  width: 100%;
  height: 42px;
  padding: 0 12px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 10px;
  background: #151515;
  color: #f8fafc;
  outline: none;
}

.project-name-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.project-name-modal__actions button {
  height: 36px;
  padding: 0 14px;
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.08);
  color: #e5e7eb;
}

.project-name-modal__actions button.primary {
  background: #f8fafc;
  color: #111827;
  font-weight: 800;
}

.project-name-modal__actions button.danger {
  background: #ef4444;
  color: #fff;
  font-weight: 800;
}

.project-delete-card p {
  line-height: 1.65;
}

.canvas-topbar {
  position: absolute;
  top: 22px;
  left: 28px;
  right: 28px;
  z-index: 20;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  pointer-events: none;
}

.canvas-topbar h1 {
  margin: 0;
  color: #f8fafc;
  font-size: 18px;
  font-weight: 700;
}

.canvas-topbar p {
  margin: 6px 0 0;
  color: #8b929e;
  font-size: 12px;
}

.canvas-status {
  display: flex;
  gap: 8px;
  pointer-events: auto;
}

.canvas-status span {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 999px;
  background: rgba(32, 33, 35, 0.82);
  color: #7f8792;
  font-size: 12px;
}

.canvas-status span.done {
  color: #f8fafc;
  border-color: rgba(34, 197, 94, 0.5);
}

.node-canvas {
  position: absolute;
  inset: 0 0 0 240px;
  min-width: 1400px;
  min-height: 860px;
  cursor: grab;
  opacity: 0;
  transition: opacity 80ms ease;
}

.node-canvas--ready {
  opacity: 1;
}

.node-canvas:active {
  cursor: grabbing;
}

.node-canvas--batch-selecting,
.node-canvas--batch-selecting:active {
  cursor: crosshair;
}

.node-canvas__stage {
  position: absolute;
  left: 0;
  top: 0;
  width: 6000px;
  height: 4000px;
  transform-origin: 0 0;
  will-change: transform;
  contain: layout style;
}

.viewport-empty-tip {
  position: absolute;
  left: 50%;
  bottom: 88px;
  z-index: 45;
  display: inline-flex;
  align-items: center;
  gap: 18px;
  max-width: min(86vw, 760px);
  padding: 12px 14px 12px 24px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 999px;
  background: rgba(38, 38, 38, 0.96);
  color: #f4f4f5;
  box-shadow: 0 18px 52px rgba(0, 0, 0, 0.42);
  transform: translateX(-50%);
  backdrop-filter: blur(14px);
  pointer-events: auto;
}

.viewport-empty-tip span {
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
}

.viewport-empty-tip button {
  flex: 0 0 auto;
  height: 40px;
  padding: 0 24px;
  border: 0;
  border-radius: 999px;
  background: #f8fafc;
  color: #111827;
  font-size: 15px;
  font-weight: 800;
  cursor: pointer;
}

.node-links {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
  pointer-events: none;
}

.node-links path {
  fill: none;
}

.node-link {
  pointer-events: auto;
  cursor: pointer;
}

.node-link__hit {
  stroke: transparent;
  stroke-width: 18;
}

.node-link__line {
  stroke: rgba(148, 163, 184, 0.45);
  stroke-width: 2;
  pointer-events: none;
  transition: stroke 140ms ease, stroke-width 140ms ease, filter 140ms ease;
}

.node-link:hover .node-link__line {
  stroke: rgba(191, 219, 254, 0.82);
  stroke-width: 3;
  filter: drop-shadow(0 0 5px rgba(96, 165, 250, 0.28));
}

.node-link.selected .node-link__line {
  stroke: #60a5fa;
  stroke-width: 3;
  filter: drop-shadow(0 0 6px rgba(96, 165, 250, 0.55));
}

.node-selection-box {
  position: absolute;
  z-index: 170;
  border: 1px dashed rgba(226, 232, 240, 0.9);
  background: rgba(96, 165, 250, 0.08);
  pointer-events: auto;
  cursor: move;
}

.node-selection-box:not(.fixed) {
  pointer-events: none;
  cursor: crosshair;
}

.node-selection-toolbar,
.node-group-toolbar {
  position: absolute;
  left: 50%;
  top: -46px;
  z-index: 2;
  display: flex;
  transform: translateX(-50%);
  padding: 6px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  background: rgba(28, 28, 28, 0.96);
  box-shadow: 0 16px 44px rgba(0, 0, 0, 0.42);
}

.node-selection-toolbar button,
.node-group-toolbar button {
  height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
  color: #f8fafc;
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
}

.node-link.transmitting .node-link__line {
  stroke: #3b82f6;
  stroke-width: 3;
  stroke-dasharray: 10 12;
  filter: drop-shadow(0 0 7px rgba(59, 130, 246, 0.72));
  animation: workflow-link-transmit 750ms linear infinite;
}

@keyframes workflow-link-transmit { to { stroke-dashoffset: -22; } }

.node-group-toolbar button + button { margin-left: 4px; }
.node-group-toolbar button:disabled { opacity: .38; cursor: not-allowed; }
.node-group-toolbar__error { max-width: 112px; align-self: center; overflow: hidden; margin: 0 5px 0 2px; color: #fca5a5; font-size: 11px; font-weight: 750; text-overflow: ellipsis; white-space: nowrap; }
.node-group-toolbar .node-group-toolbar__run { display: inline-flex; align-items: center; gap: 5px; background: #2563eb; color: #fff; }
.node-group-toolbar .node-group-toolbar__run:hover:not(:disabled) { background: #3b82f6; }
.node-group-toolbar .node-group-toolbar__run svg { width: 13px; height: 13px; }
.node-group-toolbar .node-group-toolbar__restart { display: inline-flex; align-items: center; gap: 5px; }
.node-group-toolbar .node-group-toolbar__restart svg { width: 13px; height: 13px; }
.node-group-toolbar .node-group-toolbar__pause { display: inline-flex; align-items: center; gap: 6px; }
.node-group-toolbar .node-group-toolbar__pause > span { width: 9px; height: 10px; border-left: 3px solid currentColor; border-right: 3px solid currentColor; }
.node-group-toolbar .node-group-toolbar__save { background: #f4f4f5; color: #18181b; }
.node-group-toolbar .node-group-toolbar__save:hover { background: #fff; }

.node-group-box {
  position: absolute;
  z-index: 4;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 18px;
  background: rgba(82, 82, 91, 0.26);
  pointer-events: auto;
  cursor: move;
}

.node-group-box.active {
  border-color: rgba(226, 232, 240, 0.48);
  background: rgba(82, 82, 91, 0.34);
}

.node-group-box.is-dragging {
  transition: none;
  will-change: transform;
}

.node-group-toolbar {
  pointer-events: auto;
}

.node-links .node-links__draft {
  stroke: #60a5fa;
  stroke-width: 2.5;
  stroke-dasharray: 7 7;
  filter: drop-shadow(0 0 5px rgba(96, 165, 250, 0.34));
}

.link-create-menu {
  position: absolute;
  z-index: 260;
  width: 244px;
  padding: 8px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(43, 43, 43, 0.98), rgba(27, 27, 27, 0.98));
  box-shadow: 0 20px 56px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255, 255, 255, 0.07);
  backdrop-filter: blur(16px);
}

.link-create-menu::before {
  content: "";
  position: absolute;
  left: 18px;
  top: -7px;
  width: 12px;
  height: 12px;
  border-left: 1px solid rgba(255, 255, 255, 0.14);
  border-top: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(43, 43, 43, 0.98);
  transform: rotate(45deg);
}

.link-create-menu button {
  position: relative;
  display: grid;
  width: 100%;
  grid-template-columns: 38px 1fr;
  grid-template-rows: auto auto;
  align-items: center;
  column-gap: 10px;
  padding: 10px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: #e5e7eb;
  text-align: left;
  cursor: pointer;
}

.link-create-menu button:hover {
  background: rgba(255, 255, 255, 0.09);
}

.link-create-menu strong {
  color: #f8fafc;
  font-size: 14px;
  font-weight: 800;
}

.link-create-menu small {
  margin-top: 2px;
  color: #a1a1aa;
  font-size: 12px;
}

.empty-canvas-hint {
  position: absolute;
  left: 50%;
  top: 46%;
  display: grid;
  gap: 8px;
  transform: translate(-50%, -50%);
  color: #8b929e;
  text-align: center;
  pointer-events: none;
}

.empty-canvas-hint strong {
  color: #e5e7eb;
  font-size: 16px;
}

.empty-canvas-hint span {
  font-size: 13px;
}

.canvas-node {
  position: absolute;
  z-index: 10;
  box-sizing: border-box;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 12px;
  background: #232323;
  color: #e5e7eb;
  box-shadow: 0 18px 70px rgba(0, 0, 0, 0.35);
  transition: border-color 0.18s, box-shadow 0.18s;
  cursor: grab;
  user-select: none;
  touch-action: none;
}

.canvas-node:hover {
  border-color: rgba(226, 232, 240, 0.32);
  box-shadow: 0 22px 82px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(255, 255, 255, 0.035);
}

.canvas-node:active {
  cursor: grabbing;
}

.canvas-node.is-dragging {
  z-index: 220;
  border-color: rgba(226, 232, 240, 0.55);
  box-shadow: 0 14px 38px rgba(0, 0, 0, 0.42), 0 0 0 2px rgba(96, 165, 250, 0.18);
  transition: none;
  will-change: transform;
}

.canvas-node.is-group-dragging {
  transition: none;
  will-change: transform;
}

.canvas-node.selected {
  border-color: rgba(255, 255, 255, 0.42);
  box-shadow: 0 22px 90px rgba(0, 0, 0, 0.45), 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.canvas-node.multi-selected {
  border-color: rgba(96, 165, 250, 0.62);
  box-shadow: 0 20px 82px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(96, 165, 250, 0.16);
}

.canvas-node.link-source {
  border-color: rgba(96, 165, 250, 0.9);
  box-shadow: 0 22px 90px rgba(0, 0, 0, 0.46), 0 0 0 2px rgba(96, 165, 250, 0.22);
}

.canvas-node.link-target {
  border-color: rgba(191, 219, 254, 0.9);
  box-shadow: 0 22px 90px rgba(0, 0, 0, 0.46), 0 0 0 2px rgba(147, 197, 253, 0.24);
}

.node-canvas--linking .canvas-node.link-candidate {
  border-color: rgba(148, 163, 184, 0.28);
}

.node-upload-3d {
  height: 318px;
  border-color: rgba(148, 163, 184, 0.2);
  background: #262626;
}

.node-control-output {
  min-height: 320px;
  padding: 18px;
}

.node-generic-workflow {
  min-height: 240px;
  padding: 18px;
}

.node-generic-workflow .node-body {
  height: 100%;
}

.node-text-workflow {
  min-height: 300px;
  padding: 0;
  border-color: rgba(255, 255, 255, 0.08);
  background: #242424;
}

.node-text-workflow .node-body {
  padding: 0;
}

.node-text-workflow .node-body--center {
  min-height: 100%;
  align-items: stretch;
  justify-content: stretch;
}

.node-scene-input {
  min-height: 238px;
}

.node-final-output {
  min-height: 302px;
}

.node-label {
  position: absolute;
  top: -26px;
  left: 0;
  color: #8b929e;
  font-size: 13px;
  font-weight: 600;
}

.node-group-box.running { border-color: rgba(59,130,246,.72); box-shadow: 0 0 0 1px rgba(59,130,246,.15), 0 0 26px rgba(37,99,235,.14); }
.node-group-box.failed { border-color: rgba(248,113,113,.72); box-shadow: 0 0 0 1px rgba(248,113,113,.13), 0 0 22px rgba(185,28,28,.12); }

.node-label--with-action { display: inline-flex; align-items: center; gap: 7px; }
.node-delete-button { display: inline-grid; width: 18px; height: 18px; place-items: center; border: 0; border-radius: 5px; background: transparent; color: #89919c; font-size: 16px; line-height: 1; cursor: pointer; }
.node-delete-button:hover { background: rgba(248,113,113,.16); color: #fca5a5; }

.node-body {
  padding: 24px;
}

.node-body--center {
  min-height: 166px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 10px;
}

.node-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: #303030;
  color: #a3a3a3;
  font-size: 13px;
  font-weight: 800;
}

.canvas-node h3 {
  margin: 0;
  color: #f8fafc;
  font-size: 15px;
  font-weight: 700;
}

.canvas-node p {
  margin: 0;
  color: #a1a1aa;
  font-size: 12px;
  line-height: 1.55;
}

.node-primary-btn,
.node-ghost-btn {
  height: 34px;
  padding: 0 14px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.node-primary-btn {
  border: 1px solid #e5e7eb;
  background: #f8fafc;
  color: #111827;
}

.node-primary-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.node-ghost-btn {
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: transparent;
  color: #e5e7eb;
}

.node-upload-3d .node-primary-btn,
.node-scene-input .node-primary-btn {
  margin: 0 24px 24px;
}

.node-port {
  position: absolute;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 2px solid #9ca3af;
  border-radius: 50%;
  background: #101112;
  cursor: crosshair;
  z-index: 120;
  opacity: 0;
  pointer-events: auto;
  transition: opacity 140ms ease, border-color 140ms ease, background 140ms ease, box-shadow 140ms ease;
}

.canvas-node:hover .node-port,
.canvas-node.selected .node-port,
.canvas-node.multi-selected .node-port,
.node-canvas--linking .canvas-node.link-candidate .node-port--in,
.node-port:hover,
.node-port.active {
  opacity: 1;
  pointer-events: auto;
}

.node-port::before {
  content: "";
  position: absolute;
  inset: -34px;
  border-radius: 50%;
}

.node-port:hover,
.node-port.active {
  border-color: #60a5fa;
  background: #1d4ed8;
  box-shadow: 0 0 0 5px rgba(96, 165, 250, 0.18);
}

.node-canvas--linking .canvas-node.link-candidate .node-port--in {
  border-color: rgba(191, 219, 254, 0.76);
  background: #172554;
  box-shadow: 0 0 0 5px rgba(96, 165, 250, 0.11);
}

.canvas-node.link-target .node-port--in {
  border-color: #bfdbfe;
  background: #1d4ed8;
  box-shadow: 0 0 0 6px rgba(96, 165, 250, 0.22);
}

.node-port--in {
  left: -24px;
  top: 50%;
  transform: translateY(-50%);
}

.node-port--out {
  right: -24px;
  top: 50%;
  transform: translateY(-50%);
}

.node-port--bottom {
  right: 50%;
  top: auto;
  bottom: -18px;
  transform: translateX(50%);
}

.node-preview-shell {
  position: absolute;
  inset: 0;
  overflow: hidden;
  border-radius: 12px;
  background:
    radial-gradient(circle at 50% 38%, rgba(255, 255, 255, 0.98), rgba(241, 245, 249, 0.94) 56%, rgba(226, 232, 240, 0.96) 100%),
    #fff;
}

.node-upload-3d .node-preview-shell {
  display: flex;
  flex-direction: column;
}

.node-empty-preview,
.node-loading {
  position: absolute;
  inset: 0;
  z-index: 9;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #a1a1aa;
  font-size: 13px;
}

.node-empty-preview {
  bottom: 82px;
  flex-direction: column;
  gap: 12px;
  padding: 28px;
  text-align: center;
}

.model-file-icon {
  display: grid;
  width: 74px;
  height: 58px;
  place-items: center;
  border-radius: 12px;
  background: #e5e7eb;
  color: #475569;
}

.model-file-icon svg {
  width: 34px;
  height: 34px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.node-empty-preview h3 {
  margin-top: 4px;
  color: #111827;
  font-size: 16px;
}

.node-empty-preview p {
  max-width: 250px;
  color: #64748b;
}

.model-node-summary {
  position: absolute;
  inset: 0 0 82px;
  z-index: 5;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px;
  color: #111827;
  text-align: center;
  pointer-events: none;
}

.model-node-summary .model-file-icon {
  background: rgba(17, 24, 39, 0.1);
  color: #111827;
}

.model-node-summary strong {
  max-width: 280px;
  overflow: hidden;
  color: #111827;
  font-size: 15px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-node-summary span {
  color: #64748b;
  font-size: 12px;
}

.node-preview-shell .viewer-container {
  display: none;
  width: 100%;
  height: 100%;
}

.node-preview-shell .viewer-container.visible {
  display: block;
  pointer-events: none;
}

.node-preview-shell .viewer-container canvas {
  background:
    radial-gradient(circle at 50% 38%, rgba(255, 255, 255, 0.98), rgba(241, 245, 249, 0.94) 56%, rgba(226, 232, 240, 0.96) 100%);
}

.node-preview-shell .ground-grid {
  opacity: 0.24;
}

.model-node-footer {
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: 10px;
  z-index: 12;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  min-height: 64px;
  padding: 10px;
  border: 1px solid rgba(15, 23, 42, 0.1);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
  color: #111827;
  box-shadow: 0 14px 38px rgba(15, 23, 42, 0.16);
  backdrop-filter: blur(12px);
}

.model-node-file {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.model-node-file strong {
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-node-file span {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
}

.model-node-actions {
  display: flex;
  gap: 6px;
}

.model-node-action {
  height: 34px;
  padding: 0 10px;
  border: 1px solid rgba(15, 23, 42, 0.12);
  border-radius: 9px;
  background: #f8fafc;
  color: #111827;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
}

.model-node-action:hover {
  background: #e5e7eb;
}

.model-node-action--primary {
  border-color: #111827;
  background: #111827;
  color: #fff;
}

.model-node-action--primary:hover {
  background: #020617;
}

.model-node-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.angle-chip {
  position: absolute;
  left: 12px;
  bottom: 12px;
  z-index: 6;
  padding: 6px 8px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.72);
  color: #fff;
  font-size: 11px;
}

.control-output-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
  height: 100%;
  gap: 16px;
}

.control-output-grid button {
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  min-height: 0;
  height: 100%;
  padding: 0;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 10px;
  background: #181818;
  overflow: hidden;
  cursor: zoom-in;
}

.control-output-grid img {
  display: block;
  width: 100%;
  height: 100%;
  min-height: 0;
  object-fit: cover;
  background: #fff;
}

img[data-lazy-src] {
  display: block;
}

.control-output-grid span {
  display: block;
  padding: 8px;
  color: #d4d4d8;
  font-size: 12px;
}

.node-actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 18px 18px;
}

.scene-node-preview {
  padding: 18px;
}

.scene-node-preview img {
  display: block;
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  background: #111;
  border-radius: 10px;
}

.scene-node-preview p {
  margin-top: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.final-stage {
  position: relative;
  margin: 18px;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: 10px;
  background: #111;
}

.final-stage .scene-base-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.node-final-output .placement-controls.compact {
  margin: 0 18px 12px;
  gap: 8px;
}

.node-final-output .placement-controls.compact label {
  grid-template-columns: 42px minmax(0, 1fr);
  color: #d4d4d8;
}

.node-final-output > .node-primary-btn {
  margin: 0 18px 18px;
}

.node-image-generate {
  height: 282px;
  border-radius: 14px;
  background: #262626;
  transition: box-shadow 160ms ease, border-color 160ms ease;
}

.node-image-generate .node-port {
  z-index: 140;
}

.node-video-workflow .node-body {
  height: 100%;
  padding: 0;
}

.node-video-workflow .node-body--center {
  min-height: 100%;
  align-items: stretch;
  justify-content: stretch;
  gap: 0;
  text-align: left;
}

.node-image-generate.selected {
  z-index: 90;
  box-shadow: 0 22px 90px rgba(0, 0, 0, 0.45), 0 0 0 2px rgba(255, 255, 255, 0.46);
}

.generate-card {
  position: absolute;
  inset: 0;
  padding: 0;
}

.generate-node__preview,
.generate-node__placeholder,
.generate-node__loading,
.generate-node__error {
  display: flex;
  width: 100%;
  height: 100%;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: #242424;
}

.generate-node__error {
  flex-direction: column;
  gap: 8px;
  padding: 24px;
  color: #fecaca;
  text-align: center;
}

.generate-node__error strong {
  color: #fee2e2;
  font-size: 16px;
}

.generate-node__error span {
  max-width: 86%;
  overflow: hidden;
  color: #fca5a5;
  font-size: 13px;
  line-height: 1.5;
  text-overflow: ellipsis;
}

.generate-node__loading {
  position: relative;
  overflow: hidden;
  border: 2px solid rgba(255, 255, 255, 0.48);
  isolation: isolate;
}

.generate-wave {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  border-radius: 10px;
  opacity: 0.85;
  background:
    radial-gradient(circle at 22% 72%, rgba(96, 165, 250, 0.18), transparent 30%),
    radial-gradient(circle at 78% 28%, rgba(255, 255, 255, 0.12), transparent 28%),
    linear-gradient(115deg, rgba(30, 64, 175, 0.08), rgba(96, 165, 250, 0.14), rgba(14, 165, 233, 0.08));
  background-size: 120% 120%, 140% 140%, 220% 100%;
  animation: generateWaterFlow 6s ease-in-out infinite;
  pointer-events: none;
}

.generate-wave::before,
.generate-wave::after {
  content: "";
  position: absolute;
  inset: -20%;
  background:
    repeating-linear-gradient(
      0deg,
      transparent 0,
      transparent 16px,
      rgba(255, 255, 255, 0.08) 17px,
      transparent 19px,
      transparent 34px
    );
  filter: blur(2px);
  transform: translate3d(-8%, 0, 0) skewY(-8deg);
  animation: generateRipple 4.8s ease-in-out infinite;
}

.generate-wave::after {
  opacity: 0.62;
  filter: blur(4px);
  transform: translate3d(8%, 0, 0) skewY(7deg);
  animation-duration: 6.6s;
}

.generate-progress-panel {
  position: relative;
  z-index: 1;
  display: flex;
  width: min(260px, 72%);
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
}

.generate-progress-pill {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 44px;
  padding: 0 18px;
  border: 1px solid rgba(255, 255, 255, 0.34);
  border-radius: 12px;
  background: rgba(17, 17, 17, 0.86);
  color: #f4f4f5;
  cursor: pointer;
}

.generate-progress-track {
  width: 100%;
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.2);
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.32);
}

.generate-progress-track span {
  display: block;
  width: 0;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #60a5fa, #f8fafc);
  transition: width 180ms ease;
}

@keyframes generateWaterFlow {
  0%,
  100% {
    background-position: 0% 50%, 100% 40%, 0% 50%;
  }

  50% {
    background-position: 100% 46%, 0% 60%, 100% 50%;
  }
}

@keyframes generateRipple {
  0%,
  100% {
    transform: translate3d(-8%, 0, 0) skewY(-8deg);
  }

  50% {
    transform: translate3d(8%, -8px, 0) skewY(-4deg);
  }
}

.generate-progress-pill strong {
  font-size: 15px;
}

.generate-progress-pill span {
  color: #a1a1aa;
  font-size: 14px;
}

.generate-progress-pill:hover span {
  color: #f4f4f5;
}

.generate-node__preview {
  position: relative;
}

.generate-node__preview.is-stack::before,
.generate-node__preview.is-stack::after {
  content: "";
  position: absolute;
  inset: 16px 6px -10px 18px;
  z-index: -1;
  border-radius: 10px;
  background: rgba(120, 89, 62, 0.62);
  transform: rotate(5deg);
}

.generate-node__preview.is-stack::after {
  inset: 10px 0 -5px 10px;
  background: rgba(86, 63, 49, 0.74);
  transform: rotate(3deg);
}

.generate-node__image-button {
  position: relative;
  z-index: 2;
  width: 100%;
  height: 100%;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 10px;
  background: #171717;
  cursor: pointer;
}

.generate-node__preview img,
.generate-node__preview video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #111;
  user-select: none;
  -webkit-user-drag: none;
}

.generate-node__image-button img,
.generate-node__image-button video {
  pointer-events: none;
}

.generate-node__image-button--video video {
  display: block;
  object-fit: cover;
}

.generate-node__image-button--video {
  cursor: grab;
}

.video-node-play-button {
  position: absolute;
  left: 50%;
  top: 50%;
  z-index: 4;
  display: grid;
  width: 56px;
  height: 56px;
  padding: 0;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 999px;
  background: rgba(17, 24, 39, 0.7);
  color: #fff;
  backdrop-filter: blur(10px);
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.32);
  transform: translate(-50%, -50%);
  transition: transform 160ms ease, background 160ms ease, border-color 160ms ease;
  cursor: pointer;
  pointer-events: auto;
}

.video-node-play-button::before {
  content: "";
  position: absolute;
  inset: 7px;
  border-radius: inherit;
  background: rgba(255, 255, 255, 0.16);
}

.video-node-play-icon {
  position: relative;
  z-index: 1;
  width: 28px;
  height: 28px;
  transform: translateX(2px);
}

.video-node-play-button:hover,
.video-node-play-button:focus-visible {
  border-color: rgba(255, 255, 255, 0.94);
  background: rgba(17, 24, 39, 0.86);
  transform: translate(-50%, -50%) scale(1.06);
}

.video-node-play-button:focus-visible {
  outline: 2px solid rgba(255, 255, 255, 0.82);
  outline-offset: 3px;
}

.generate-node__size {
  position: absolute;
  right: 0;
  top: -26px;
  z-index: 8;
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  padding: 0 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 7px;
  background: rgba(38, 38, 38, 0.92);
  color: #aeb4bf;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  pointer-events: none;
  box-shadow: 0 8px 18px rgba(0, 0, 0, 0.18);
}

.generate-edit-canvas {
  position: absolute;
  inset: 0;
  z-index: 4;
  width: 100%;
  height: 100%;
  border-radius: 10px;
  cursor: crosshair;
  touch-action: none;
}

.generate-node__count {
  position: absolute;
  right: 12px;
  top: 12px;
  z-index: 5;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 34px;
  padding: 0 12px;
  border: 0;
  border-radius: 10px;
  background: rgba(39, 39, 42, 0.9);
  color: #fff;
  font-size: 15px;
  font-weight: 800;
  cursor: pointer;
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.26);
}

.stack-expand-icon {
  position: relative;
  display: inline-block;
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.stack-expand-icon::before,
.stack-expand-icon::after {
  content: "";
  position: absolute;
  border-color: currentColor;
  border-style: solid;
}

.stack-expand-icon::before {
  right: 1px;
  top: 1px;
  width: 8px;
  height: 8px;
  border-width: 2px 2px 0 0;
}

.stack-expand-icon::after {
  left: 1px;
  bottom: 1px;
  width: 11px;
  height: 11px;
  border-width: 0 0 2px 2px;
}

.generate-node__count:hover {
  background: rgba(63, 63, 70, 0.96);
}

.image-node-toolbar {
  position: absolute;
  left: 50%;
  top: -74px;
  z-index: 40;
  display: flex;
  align-items: center;
  gap: 10px;
  height: 58px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(47, 47, 47, 0.98), rgba(28, 28, 28, 0.98));
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.48), inset 0 1px 0 rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(16px);
  transform: translateX(-50%) scale(var(--canvas-editor-scale, 1));
  transform-origin: bottom center;
}

.image-node-toolbar button {
  position: relative;
  display: inline-flex;
  width: 46px;
  height: 46px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: #e7e7ea;
  font-size: 0;
  line-height: 1;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease, transform 140ms ease, box-shadow 140ms ease;
}

.image-node-toolbar button:hover {
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
}

.image-node-toolbar button:active {
  transform: translateY(1px);
}

.image-node-toolbar button:focus-visible {
  outline: 2px solid rgba(255, 255, 255, 0.5);
  outline-offset: 2px;
}

.image-node-toolbar .toolbar-icon {
  width: 26px;
  height: 26px;
}

.generate-edit-toolbar {
  position: absolute;
  left: 50%;
  top: -82px;
  z-index: 48;
  display: flex;
  height: 64px;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(39, 39, 42, 0.98), rgba(24, 24, 27, 0.98));
  box-shadow: 0 22px 58px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255, 255, 255, 0.09);
  backdrop-filter: blur(18px);
  transform: translateX(-50%) scale(var(--canvas-editor-scale, 1));
  transform-origin: bottom center;
}

.generate-edit-toolbar button {
  position: relative;
  display: inline-flex;
  min-width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.04);
  color: #d4d4d8;
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease, color 140ms ease, transform 140ms ease, box-shadow 140ms ease;
}

.generate-edit-toolbar button:hover {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.generate-edit-toolbar button.active {
  border-color: rgba(255, 255, 255, 0.2);
  background: #f4f4f5;
  color: #18181b;
  box-shadow: 0 10px 22px rgba(244, 244, 245, 0.12);
}

.generate-edit-toolbar button:active {
  transform: translateY(1px);
}

.generate-edit-toolbar button:focus-visible,
.generate-edit-color:focus-within,
.generate-edit-size:focus-within {
  outline: 2px solid rgba(255, 255, 255, 0.46);
  outline-offset: 2px;
}

.generate-edit-toolbar button:disabled {
  cursor: default;
  opacity: 0.34;
}

.generate-edit-toolbar button:disabled:hover {
  border-color: transparent;
  background: rgba(255, 255, 255, 0.04);
  color: #d4d4d8;
  transform: none;
  box-shadow: none;
}

.generate-edit-toolbar__exit {
  min-width: 112px !important;
  padding: 0 13px 0 11px;
  color: #f4f4f5 !important;
  font-size: 14px;
  font-weight: 800;
}

.generate-edit-toolbar__exit strong {
  font: inherit;
  white-space: nowrap;
}

.generate-edit-toolbar__divider {
  width: 1px;
  height: 38px;
  background: rgba(255, 255, 255, 0.12);
}

.generate-edit-toolbar__save {
  min-width: 78px !important;
  padding: 0 18px;
  border-color: rgba(255, 255, 255, 0.88) !important;
  border-radius: 12px !important;
  background: #fafafa !important;
  color: #111 !important;
  font-size: 15px;
  font-weight: 900;
  box-shadow: 0 10px 24px rgba(255, 255, 255, 0.14);
}

.generate-edit-toolbar__save:hover {
  background: #fff !important;
  color: #000 !important;
}

.generate-edit-color {
  position: relative;
  display: inline-flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.04);
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease;
}

.generate-edit-color:hover {
  border-color: rgba(255, 255, 255, 0.16);
  background: rgba(255, 255, 255, 0.12);
}

.generate-edit-color input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.generate-edit-color span {
  display: block;
  width: 26px;
  height: 26px;
  border: 2px solid rgba(255, 255, 255, 0.88);
  border-radius: 999px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.7), inset 0 0 0 1px rgba(0, 0, 0, 0.18);
}

.generate-edit-size {
  display: inline-flex;
  width: 164px;
  height: 42px;
  align-items: center;
  gap: 10px;
  padding: 0 11px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.04);
  color: #e4e4e7;
}

.generate-edit-size input {
  width: 104px;
  accent-color: #f4f4f5;
  cursor: pointer;
}

.generate-edit-size input::-webkit-slider-runnable-track {
  height: 5px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.26);
}

.generate-edit-size input::-webkit-slider-thumb {
  margin-top: -5px;
}

.generate-edit-icon {
  display: block;
  width: 22px;
  height: 22px;
  flex: 0 0 22px;
  stroke-width: 1.9;
  pointer-events: none;
}

.generate-edit-icon--redo {
  transform: scaleX(-1);
}

.generate-edit-size .generate-edit-icon {
  width: 20px;
  height: 20px;
  flex-basis: 20px;
  color: #f4f4f5;
}

.toolbar-icon {
  display: block;
  width: 24px;
  height: 24px;
  stroke-width: 1.9;
  color: currentColor;
  pointer-events: none;
}

.generate-node__placeholder {
  flex-direction: column;
  gap: 14px;
  color: #f4f4f5;
}

.generate-node__placeholder-icon {
  display: flex;
  width: 74px;
  height: 58px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: #353535;
  color: #777;
}

.generate-node__placeholder-icon svg {
  width: 34px;
  height: 34px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.generate-editor-panel {
  position: absolute;
  left: 50%;
  top: calc(100% + 30px);
  z-index: 30;
  display: flex;
  width: min(960px, 90vw);
  min-height: 320px;
  max-height: min(620px, 70vh);
  transform: translateX(-50%) scale(var(--canvas-editor-scale, 1));
  transform-origin: top center;
  flex-direction: column;
  gap: 18px;
  padding: 28px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 16px;
  background: rgba(38, 38, 38, 0.98);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.42);
  cursor: default;
}

.text-node-card {
  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 100%;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(226, 232, 240, 0.16);
  border-radius: 14px;
  background: #181818;
}

.text-node-input {
  width: 100%;
  height: 100%;
  min-height: 100%;
  padding: 18px 20px 34px;
  resize: none;
  border: 0;
  background: transparent;
  color: #f4f4f5;
  font-size: 19px;
  line-height: 1.65;
  cursor: text;
}

.text-node-input[readonly] {
  cursor: grab;
  user-select: none;
}

.text-node-input[readonly]:active {
  cursor: grabbing;
}

.text-node-input::placeholder {
  color: rgba(212, 212, 216, 0.46);
}

.text-node-input:focus {
  outline: none;
}

.text-node-card:focus-within {
  border-color: rgba(226, 232, 240, 0.34);
  box-shadow: inset 0 0 0 1px rgba(226, 232, 240, 0.1);
}

.text-node-loading-panel {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 28px;
  background:
    linear-gradient(180deg, rgba(32, 32, 32, 0.98), rgba(20, 20, 20, 0.98));
  color: #f4f4f5;
  text-align: center;
}

.text-node-loading-panel strong {
  font-size: 15px;
  font-weight: 900;
}

.text-node-loading-panel em {
  color: #d4d4d8;
  font-size: 13px;
  font-style: normal;
  font-weight: 800;
}

.text-node-progress-track {
  width: min(210px, 80%);
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
}

.text-node-progress-track span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #f4f4f5;
  transition: width 220ms ease;
}

.text-node-error {
  position: absolute;
  left: 12px;
  right: 62px;
  bottom: 10px;
  overflow: hidden;
  color: #fecaca;
  font-size: 12px;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.text-node-resize {
  position: absolute;
  right: 8px;
  bottom: 8px;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: nwse-resize;
}

.text-node-resize::before,
.text-node-resize::after {
  content: "";
  position: absolute;
  right: 4px;
  bottom: 4px;
  width: 13px;
  height: 2px;
  border-radius: 999px;
  background: rgba(226, 232, 240, 0.62);
  transform: rotate(-45deg);
  transform-origin: right center;
}

.text-node-resize::after {
  right: 4px;
  bottom: 10px;
  width: 9px;
}

.batch-node-card {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 100%;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  gap: 12px;
  text-align: left;
}

.batch-node-head {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.batch-node-head span {
  overflow: hidden;
  color: #f8fafc;
  font-size: 15px;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.batch-node-head strong {
  flex: 0 0 auto;
  color: #a1a1aa;
  font-size: 12px;
  font-weight: 800;
}

.batch-field {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.batch-field span {
  color: #9ca3af;
  font-size: 12px;
  font-weight: 800;
}

.batch-field input,
.batch-field select {
  width: 100%;
  height: 38px;
  min-width: 0;
  padding: 0 12px;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 9px;
  outline: none;
  background: #151515;
  color: #f8fafc;
  font-size: 13px;
  font-weight: 700;
}

.batch-field input:focus,
.batch-field select:focus {
  border-color: rgba(226, 232, 240, 0.58);
}

.batch-language-grid {
  display: grid;
  min-height: 116px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
  overflow: visible;
}

.batch-language-grid button {
  min-width: 0;
  height: 34px;
  padding: 0 8px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.06);
  color: #d4d4d8;
  font-size: 12px;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.batch-language-grid button.active {
  border-color: rgba(248, 250, 252, 0.8);
  background: #f8fafc;
  color: #111827;
}

.batch-status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 34px;
  padding: 0 10px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.045);
}

.batch-status-row span {
  min-width: 0;
  overflow: hidden;
  color: #d4d4d8;
  font-size: 12px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.batch-status-row button {
  flex: 0 0 auto;
  height: 26px;
  padding: 0 9px;
  border: 0;
  border-radius: 8px;
  background: #f8fafc;
  color: #111827;
  font-size: 12px;
  font-weight: 900;
  cursor: pointer;
}

.batch-quality-select,
.batch-quality-select .generate-select__trigger {
  width: 100%;
}

.batch-quality-select {
  z-index: 180;
}

.batch-quality-select .generate-select__trigger {
  height: 38px;
  min-width: 0;
}

.batch-quality-menu {
  left: 50%;
  z-index: 240;
  width: min(492px, 86vw);
  transform: translateX(-50%);
}

.batch-material-grid {
  display: grid;
  max-height: 92px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  overflow: auto;
}

.batch-material-card,
.batch-add-material {
  min-width: 0;
  height: 42px;
  border-radius: 10px;
  cursor: pointer;
}

.batch-material-card {
  position: relative;
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr);
  align-items: center;
  gap: 7px;
  padding: 5px 8px 5px 5px;
  border: 1px solid rgba(226, 232, 240, 0.24);
  background: rgba(255, 255, 255, 0.075);
  color: #f8fafc;
}

.batch-material-card img,
.batch-material-card > span {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  overflow: hidden;
  border-radius: 8px;
  background: #0f0f0f;
  color: #e5e7eb;
  font-size: 12px;
  font-weight: 900;
  object-fit: cover;
}

.batch-material-card strong {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.batch-material-card em {
  position: absolute;
  right: 4px;
  top: 3px;
  color: rgba(244, 244, 245, 0.7);
  font-size: 12px;
  font-style: normal;
  line-height: 1;
}

.batch-add-material {
  border: 1px dashed rgba(226, 232, 240, 0.34);
  background: transparent;
  color: #e5e7eb;
  font-size: 12px;
  font-weight: 900;
}

.batch-task-panel {
  display: grid;
  max-height: 122px;
  gap: 7px;
  overflow: auto;
}

.batch-task-row {
  display: grid;
  min-width: 0;
  grid-template-columns: 34px minmax(0, 1fr) auto;
  grid-template-rows: auto auto;
  column-gap: 8px;
  align-items: center;
  padding: 6px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.055);
}

.batch-task-row img,
.batch-task-thumb {
  display: grid;
  width: 34px;
  height: 34px;
  grid-row: 1 / 3;
  place-items: center;
  overflow: hidden;
  border-radius: 8px;
  background: #111;
  color: #e5e7eb;
  font-size: 12px;
  font-weight: 900;
  object-fit: cover;
}

.batch-task-row strong,
.batch-task-row small {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.batch-task-row strong {
  color: #f8fafc;
  font-size: 12px;
  font-weight: 900;
}

.batch-task-row small {
  color: #9ca3af;
  font-size: 11px;
  font-weight: 700;
}

.batch-status-badge {
  grid-column: 3;
  grid-row: 1 / 3;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.14);
  color: #cbd5e1;
  font-size: 11px;
  font-style: normal;
  font-weight: 900;
  line-height: 24px;
}

.batch-status-badge.is-running {
  background: rgba(251, 191, 36, 0.16);
  color: #fde68a;
}

.batch-status-badge.is-completed {
  background: rgba(52, 211, 153, 0.16);
  color: #a7f3d0;
}

.batch-status-badge.is-failed,
.batch-status-badge.is-error {
  background: rgba(248, 113, 113, 0.18);
  color: #fecaca;
}

.batch-run-btn {
  height: 48px;
  margin-top: auto;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #f8fafc;
  color: #111827;
  font-size: 14px;
  font-weight: 900;
  cursor: pointer;
}

.batch-run-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.batch-node-error {
  padding: 8px 10px;
  border: 1px solid rgba(248, 113, 113, 0.36);
  border-radius: 10px;
  background: rgba(127, 29, 29, 0.22);
  color: #fecaca;
  font-size: 12px;
  font-weight: 800;
  line-height: 1.45;
}

.generate-editor-panel:not(.has-refs) {
  justify-content: flex-start;
}

.generate-results-panel {
  position: absolute;
  inset: 0;
  z-index: 70;
  overflow: visible;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  cursor: default;
  pointer-events: none;
}

.generate-upload-btn {
  position: absolute;
  right: 10px;
  top: 10px;
  z-index: 42;
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  padding: 0;
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  background:
    linear-gradient(180deg, rgba(50, 50, 52, 0.94), rgba(24, 24, 25, 0.94));
  color: #f8fafc;
  cursor: pointer;
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.34), inset 0 1px 0 rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(12px);
  transition: background 140ms ease, border-color 140ms ease, color 140ms ease, transform 140ms ease, box-shadow 140ms ease;
}

.generate-upload-btn:hover {
  border-color: rgba(255, 255, 255, 0.34);
  background:
    linear-gradient(180deg, rgba(70, 70, 73, 0.98), rgba(34, 34, 36, 0.98));
  color: #fff;
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.4), inset 0 1px 0 rgba(255, 255, 255, 0.14);
}

.generate-upload-btn:active {
  transform: translateY(1px);
}

.generate-upload-btn:focus-visible {
  outline: 2px solid rgba(255, 255, 255, 0.55);
  outline-offset: 2px;
}

.generate-upload-btn--top {
  left: 50%;
  right: auto;
  top: -50px;
  height: 38px;
  width: auto;
  min-width: 78px;
  padding: 0 20px;
  border-radius: 14px;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0;
  transform: translateX(-50%);
}

.generate-upload-btn--top:active {
  transform: translateX(-50%) translateY(1px);
}

.generate-node__refs {
  display: flex;
  gap: 10px;
  min-height: 62px;
}

.video-editor-prompt {
  min-height: 132px;
}

.generate-input-summary {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.07);
  color: #d4d4d8;
  font-size: 14px;
  font-weight: 700;
}

.generate-input-summary span {
  color: #8b929e;
}

.generate-input-summary strong {
  color: #f4f4f5;
  font-weight: 800;
}

.generate-result-grid {
  position: absolute;
  inset: 0;
  display: block;
  padding: 0;
  overflow: visible;
  background: transparent;
  pointer-events: none;
}

.generate-result-item {
  position: absolute;
  width: 100%;
  height: 100%;
  aspect-ratio: 1;
  min-height: 0;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 12px;
  background: #111;
  cursor: zoom-in;
  pointer-events: auto;
  box-shadow: 0 22px 70px rgba(0, 0, 0, 0.46);
}

.generate-result-item--0 {
  left: 0;
  top: calc(-100% - 14px);
}

.generate-result-item--1 {
  left: calc(100% + 14px);
  top: calc(-100% - 14px);
}

.generate-result-item--2 {
  left: calc(100% + 14px);
  top: 0;
}

.generate-result-item img {
  display: block;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: inherit;
  object-fit: cover;
  background: #111;
}

.generate-result-actions {
  position: absolute;
  right: 8px;
  top: 8px;
  z-index: 4;
  display: flex;
  gap: 6px;
  max-width: calc(100% - 16px);
}

.generate-result-actions em {
  flex: 0 0 auto;
  padding: 6px 9px;
  border-radius: 8px;
  background: rgba(24, 24, 27, 0.88);
  color: #fff;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}

.generate-result-actions em:hover {
  background: rgba(63, 63, 70, 0.94);
}

.generate-ref {
  position: relative;
  width: 74px;
  height: 62px;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 10px;
  background: #171717;
  cursor: zoom-in;
}

.generate-ref--text {
  display: grid;
  width: 74px;
  height: 62px;
  place-items: center;
  overflow: visible;
  border: 0;
  background: #535353;
  cursor: default;
}

.generate-ref-text-icon {
  position: relative;
  width: 25px;
  height: 21px;
}

.generate-ref-text-icon::before {
  content: "";
  position: absolute;
  left: 0;
  top: 1px;
  width: 20px;
  height: 3px;
  background: #f4f4f5;
  box-shadow:
    0 6px 0 #f4f4f5,
    0 12px 0 #f4f4f5;
}

.generate-ref-text-icon::after {
  content: "";
  position: absolute;
  left: 0;
  top: 19px;
  width: 12px;
  height: 3px;
  background: #f4f4f5;
}

.generate-ref-text-count {
  position: absolute;
  right: -5px;
  top: -6px;
  display: grid;
  min-width: 17px;
  height: 17px;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 999px;
  background: #0f0f0f;
  color: #f8fafc;
  font-size: 11px;
  font-weight: 800;
  line-height: 1;
}

.generate-ref img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #fff;
}

.video-node__refs {
  min-height: 78px;
}

.video-ref-item {
  position: relative;
  width: 74px;
  height: 78px;
  flex: 0 0 auto;
}

.video-ref-item .generate-ref {
  width: 74px;
  height: 62px;
}

.video-ref-index {
  position: absolute;
  right: 5px;
  bottom: 11px;
  z-index: 3;
  display: grid;
  width: 18px;
  height: 18px;
  place-items: center;
  border-radius: 5px;
  background: rgba(24, 24, 27, 0.92);
  color: #a1a1aa;
  font-size: 9px;
  font-weight: 800;
}

.video-media-inputs {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin: 7px 0;
}

.video-media-inputs button,
.video-media-inputs label {
  position: relative;
  display: inline-flex;
  min-height: 25px;
  align-items: center;
  border: 1px solid #d9dde1;
  border-radius: 5px;
  background: #fff;
  padding: 0 8px;
  color: #666d75;
  font-size: 10px;
  cursor: pointer;
}

.toolbar-download {
  position: relative;
}

.toolbar-download svg {
  width: 14px;
  height: 14px;
}

.toolbar-download span {
  min-width: 12px;
  margin-left: -2px;
  color: #bfdbfe;
  font-size: 10px;
  font-weight: 800;
  line-height: 1;
}

.toolbar-download:disabled {
  opacity: 0.38;
  cursor: not-allowed;
}

.toolbar-download.active {
  color: #bfdbfe;
}

.batch-selection-order {
  position: absolute;
  top: -12px;
  right: -12px;
  z-index: 8;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 2px solid #0f172a;
  border-radius: 50%;
  background: #2563eb;
  color: #fff;
  font-size: 12px;
  font-weight: 800;
  line-height: 1;
  pointer-events: none;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.35);
}

.video-media-inputs button.active {
  border-color: #7e9d86;
  background: #f1f7f3;
  color: #166534;
}

.video-media-inputs input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.video-duration-range {
  display: grid;
  grid-template-columns: minmax(90px, 1fr) 52px auto;
  align-items: center;
  gap: 7px;
}

.video-duration-range input[type="number"] {
  width: 52px;
  height: 28px;
  box-sizing: border-box;
  border: 1px solid #d9dde1;
  border-radius: 5px;
  padding: 0 6px;
  font-size: 11px;
}

.model-preview-meta {
  position: absolute;
  left: 12px;
  top: 12px;
  z-index: 7;
  display: flex;
  max-width: 218px;
  flex-direction: column;
  gap: 3px;
  padding: 8px 10px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 10px;
  background: rgba(17, 24, 39, 0.62);
  color: #fff;
  backdrop-filter: blur(10px);
  pointer-events: none;
}

.model-preview-meta strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.model-preview-meta span {
  color: rgba(255, 255, 255, 0.66);
  font-size: 11px;
}

.generate-prompt {
  flex: 1;
  min-height: 150px;
  resize: none;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: #f4f4f5;
  padding: 0 2px;
  font-size: 18px;
  line-height: 1.8;
  outline: none;
}

.generate-prompt::placeholder {
  color: #8b929e;
}

.generate-error {
  margin: 0;
  color: #fecaca;
  font-size: 12px;
}

.generate-node__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding-top: 8px;
  border-top: 1px solid rgba(148, 163, 184, 0.18);
}

.generate-select {
  position: relative;
  min-width: 0;
}

.generate-select__trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  gap: 8px;
  height: 38px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 10px;
  background: rgba(64, 64, 64, 0.8);
  color: #d4d4d8;
  padding: 0 12px;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
}

.generate-select__value {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.generate-select__trigger:hover {
  border-color: rgba(255, 255, 255, 0.28);
  background: rgba(82, 82, 82, 0.9);
}

.generate-select--model {
  flex: 1 1 220px;
}

.generate-select--model .generate-select__trigger {
  width: 100%;
  justify-content: flex-start;
  height: 42px;
}

.generate-select__icon {
  display: inline-flex;
  width: 28px;
  height: 28px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: 18px;
}

.generate-select__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  align-items: flex-start;
  line-height: 1.15;
}

.generate-select__copy strong {
  max-width: 178px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #f4f4f5;
  font-size: 15px;
}

.generate-select__copy small,
.generate-select__trigger small {
  color: #a1a1aa;
  font-size: 13px;
  font-weight: 500;
}

.generate-select__chevron {
  color: #a1a1aa;
  transition: transform 140ms ease;
}

.generate-select.open .generate-select__chevron {
  transform: rotate(180deg);
}

.video-quality-menu {
  width: min(506px, 86vw);
  max-height: min(520px, 72vh);
  padding: 16px;
}

.video-quality-menu .generate-quality-section p {
  font-size: 15px;
  font-weight: 800;
}

.video-aspect-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.video-aspect-grid button {
  min-height: 86px;
  border-radius: 11px;
}

.video-aspect-grid button strong {
  font-size: 16px;
  font-weight: 800;
}

.video-resolution-row {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.video-resolution-row button {
  height: 48px;
  font-size: 17px;
  font-weight: 800;
  text-transform: uppercase;
}

.generate-select--compact .generate-select__trigger,
.generate-select--count .generate-select__trigger {
  min-width: 82px;
  flex-direction: column;
  gap: 0;
  line-height: 1.05;
}

.generate-select--count .generate-select__trigger {
  min-width: 66px;
}

.generate-select--quality .generate-select__trigger {
  min-width: 210px;
  justify-content: space-between;
  white-space: nowrap;
}

.generate-quality-check {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  border: 2px solid #f4f4f5;
  border-radius: 3px;
}

.generate-select-menu {
  position: absolute;
  left: 0;
  bottom: calc(100% + 8px);
  z-index: 80;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 14px;
  background: rgba(36, 36, 36, 0.98);
  box-shadow: 0 22px 70px rgba(0, 0, 0, 0.46);
}

.generate-model-menu {
  width: min(390px, 82vw);
  max-height: 300px;
  overflow-y: auto;
  padding: 8px;
}

.generate-model-option {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: #f4f4f5;
  text-align: left;
  cursor: pointer;
}

.generate-model-option:hover,
.generate-model-option.active {
  background: rgba(255, 255, 255, 0.1);
}

.generate-model-option__icon {
  display: inline-flex;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.generate-model-option span:nth-child(2) {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.generate-model-option strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.generate-model-option small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #a1a1aa;
  font-size: 12px;
}

.generate-model-option em {
  flex: 0 0 auto;
  padding: 5px 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  color: #a1a1aa;
  font-style: normal;
  font-size: 12px;
}

.generate-quality-menu {
  width: min(492px, 86vw);
  max-height: min(560px, 72vh);
  overflow-y: auto;
  padding: 14px;
}

.generate-quality-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.generate-quality-section + .generate-quality-section {
  margin-top: 14px;
}

.generate-quality-section p {
  margin: 0;
  color: #a1a1aa;
  font-size: 14px;
  font-weight: 600;
}

.generate-quality-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.generate-quality-row button,
.generate-aspect-grid button {
  border: 1px solid rgba(148, 163, 184, 0.26);
  border-radius: 10px;
  background: rgba(38, 38, 38, 0.92);
  color: #a1a1aa;
  cursor: pointer;
}

.generate-quality-row button {
  height: 48px;
  font-size: 15px;
  font-weight: 700;
}

.generate-quality-row button:hover,
.generate-quality-row button.active,
.generate-aspect-grid button:hover,
.generate-aspect-grid button.active {
  border-color: rgba(255, 255, 255, 0.72);
  color: #f4f4f5;
  background: rgba(64, 64, 64, 0.96);
}

.generate-aspect-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.generate-aspect-grid button {
  display: flex;
  min-height: 82px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 9px;
  padding: 10px 8px;
}

.generate-aspect-grid button strong {
  font-size: 14px;
  font-weight: 600;
}

.generate-aspect-icon {
  display: block;
  width: var(--aspect-icon-width, 18px);
  height: var(--aspect-icon-height, 18px);
  border: 2px solid currentColor;
  border-radius: 3px;
}

.generate-aspect-icon--1-2,
.generate-aspect-icon--9-16,
.generate-aspect-icon--3-4,
.generate-aspect-icon--2-3,
.generate-aspect-icon--4-5,
.generate-aspect-icon--9-21,
.generate-aspect-icon--1-4,
.generate-aspect-icon--1-8 {
  --aspect-icon-width: 11px;
  --aspect-icon-height: 22px;
}

.generate-aspect-icon--1-4,
.generate-aspect-icon--1-8 {
  --aspect-icon-width: 8px;
  --aspect-icon-height: 24px;
}

.generate-aspect-icon--2-1,
.generate-aspect-icon--16-9,
.generate-aspect-icon--4-3,
.generate-aspect-icon--3-2,
.generate-aspect-icon--5-4,
.generate-aspect-icon--21-9,
.generate-aspect-icon--4-1,
.generate-aspect-icon--8-1 {
  --aspect-icon-width: 24px;
  --aspect-icon-height: 12px;
}

.generate-aspect-icon--4-1,
.generate-aspect-icon--8-1 {
  --aspect-icon-width: 26px;
  --aspect-icon-height: 8px;
}

.generate-compact-menu,
.generate-count-menu {
  min-width: 86px;
  padding: 6px;
}

.generate-compact-menu button,
.generate-count-menu button {
  display: block;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: #a1a1aa;
  text-align: left;
  font-size: 13px;
  cursor: pointer;
}

.generate-compact-menu button:hover,
.generate-compact-menu button.active,
.generate-count-menu button:hover,
.generate-count-menu button.active {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.generate-select-empty {
  padding: 14px;
  color: #a1a1aa;
  font-size: 13px;
}

.canvas-toolbar {
  position: absolute;
  left: 50%;
  bottom: 24px;
  z-index: 40;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 14px;
  background: rgba(36, 36, 36, 0.94);
  box-shadow: 0 16px 50px rgba(0, 0, 0, 0.36);
}

.canvas-toolbar button {
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: #d4d4d8;
  font-size: 18px;
  cursor: pointer;
}

.canvas-toolbar .toolbar-main {
  position: relative;
  display: grid;
  place-items: center;
  background: #f8fafc;
  color: #111827;
  font-size: 0;
  line-height: 1;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.22);
  transition: background 140ms ease, color 140ms ease, transform 140ms ease, box-shadow 140ms ease;
}

.canvas-toolbar .toolbar-main:hover {
  background: #fff;
  box-shadow: 0 7px 18px rgba(0, 0, 0, 0.28);
  transform: translateY(-1px);
}

.canvas-toolbar .toolbar-main:active,
.canvas-toolbar .toolbar-action:active,
.canvas-toolbar .toolbar-history:active {
  transform: translateY(1px) scale(0.96);
}

.canvas-toolbar .toolbar-main::before,
.canvas-toolbar .toolbar-main::after {
  content: "";
  position: absolute;
  width: 16px;
  height: 2px;
  border-radius: 999px;
  background: currentColor;
  transition: transform 160ms ease;
}

.canvas-toolbar .toolbar-main::after {
  transform: rotate(90deg);
}

.canvas-toolbar .toolbar-main.active::before {
  transform: rotate(45deg);
}

.canvas-toolbar .toolbar-main.active::after {
  transform: rotate(-45deg);
}

.canvas-toolbar .toolbar-history,
.canvas-toolbar .toolbar-action {
  display: grid;
  place-items: center;
  background: transparent;
}

.canvas-toolbar .toolbar-history:focus,
.canvas-toolbar .toolbar-history:active,
.canvas-toolbar .toolbar-action:focus,
.canvas-toolbar .toolbar-action:active,
.canvas-toolbar .toolbar-action.active {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.canvas-toolbar .toolbar-history:hover,
.canvas-toolbar .toolbar-action:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.canvas-toolbar button:focus-visible,
.node-menu button:focus-visible,
.node-menu-submenu button:focus-visible {
  outline: 2px solid rgba(147, 197, 253, 0.86);
  outline-offset: 2px;
}

.toolbar-history svg,
.toolbar-action svg {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.workflow-template-menu {
  position: absolute;
  left: 50%;
  bottom: 56px;
  width: 316px;
  padding: 10px;
  transform: translateX(-50%);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  background: rgba(31, 31, 31, 0.98);
  box-shadow: 0 16px 50px rgba(0, 0, 0, 0.42);
}

.workflow-template-menu button {
  display: grid;
  width: 100%;
  height: auto;
  grid-template-columns: 40px 1fr;
  grid-template-rows: auto auto;
  align-items: center;
  column-gap: 10px;
  padding: 11px;
  border-radius: 12px;
  color: #e5e7eb;
  text-align: left;
}

.workflow-template-menu button:hover {
  background: rgba(255, 255, 255, 0.08);
}

.workflow-template-menu__icon {
  grid-row: 1 / span 2;
  display: grid;
  width: 40px !important;
  height: 40px !important;
  place-items: center;
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.08) !important;
  color: #bfdbfe;
}

.workflow-template-menu__icon svg {
  width: 22px;
  height: 22px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.workflow-template-menu strong {
  color: #f8fafc;
  font-size: 14px;
  font-weight: 800;
}

.workflow-template-menu small {
  margin-top: 3px;
  color: #a1a1aa;
  font-size: 12px;
  line-height: 1.35;
}

.workflow-template-modal { position: fixed; inset: 0; z-index: 130; display: grid; place-items: center; padding: 24px; background: rgba(4,5,7,.72); backdrop-filter: blur(8px); }
.workflow-template-modal__panel { width: min(1160px, 100%); height: min(860px, calc(100vh - 48px)); display: flex; flex-direction: column; overflow: hidden; border: 1px solid #35373c; border-radius: 8px; background: #1b1c1f; box-shadow: 0 32px 100px rgba(0,0,0,.64); color: #f4f4f5; }
.workflow-template-modal__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; padding: 24px 28px 19px; }
.workflow-template-modal__header h3 { margin: 0; font-size: 20px; font-weight: 760; letter-spacing: 0; }
.workflow-template-modal__header p { margin: 6px 0 0; color: #7c838d; font-size: 12px; }
.workflow-template-modal__header > button { display: grid; width: 32px; height: 32px; place-items: center; border-radius: 6px; color: #7d848e; }
.workflow-template-modal__header > button:hover { background: #292b2f; color: #f1f3f5; }
.workflow-template-modal__header > button:focus-visible { outline: 2px solid #609af0; outline-offset: 2px; }
.workflow-template-modal__header > button svg { width: 18px; height: 18px; }
.workflow-template-controls { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; padding: 0 28px; border-bottom: 1px solid #303237; }
.workflow-template-tabs { display: flex; align-items: center; gap: 24px; }
.workflow-template-tabs button { position: relative; height: 46px; display: inline-flex; align-items: center; gap: 7px; color: #858c96; font-size: 13px; font-weight: 720; }
.workflow-template-tabs button::after { position: absolute; right: 0; bottom: -1px; left: 0; height: 2px; background: transparent; content: ''; }
.workflow-template-tabs button:hover { color: #c7cbd1; }
.workflow-template-tabs button.active { color: #f4f5f6; }
.workflow-template-tabs button.active::after { background: #4a8fe3; }
.workflow-template-tabs small { display: grid; min-width: 19px; height: 19px; place-items: center; padding: 0 5px; border-radius: 4px; background: #292b30; color: #737b86; font-size: 9px; }
.workflow-template-tabs button.active small { background: rgba(59,130,246,.14); color: #82aee6; }
.workflow-template-search { display: flex; width: 248px; height: 36px; align-items: center; gap: 8px; margin-bottom: 9px; padding: 0 11px; border: 1px solid #383a40; border-radius: 6px; background: #222327; transition: border-color 140ms ease, box-shadow 140ms ease; }
.workflow-template-search:focus-within { border-color: #568ddd; box-shadow: 0 0 0 3px rgba(59,130,246,.11); }
.workflow-template-search svg { width: 15px; height: 15px; flex: 0 0 auto; fill: none; stroke: #737b86; stroke-width: 1.8; }
.workflow-template-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: #f1f2f4; font-size: 12px; }
.workflow-template-search input::placeholder { color: #666d77; }
.workflow-template-grid { min-height: 0; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); grid-auto-rows: max-content; align-content: start; gap: 14px; flex: 1; overflow-y: auto; padding: 22px 28px 30px; scrollbar-color: #41444b transparent; }
.workflow-template-grid::-webkit-scrollbar { width: 8px; }
.workflow-template-grid::-webkit-scrollbar-track { background: transparent; }
.workflow-template-grid::-webkit-scrollbar-thumb { border: 2px solid #1b1c1f; border-radius: 4px; background: #3d4046; }
.workflow-template-grid::-webkit-scrollbar-thumb:hover { background: #4b4e55; }
.workflow-template-empty { grid-column: 1 / -1; min-height: 390px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; color: #6f7680; text-align: center; }
.workflow-template-empty__icon { display: grid; width: 44px; height: 44px; margin-bottom: 4px; place-items: center; border: 1px solid #373a40; border-radius: 8px; background: #222327; color: #858d98; }
.workflow-template-empty__icon svg { width: 20px; height: 20px; }
.workflow-template-empty strong { color: #c8ccd2; font-size: 13px; }
.workflow-template-empty small { font-size: 10px; }
.workflow-template-card { position: relative; min-width: 0; display: flex; flex-direction: column; overflow: hidden; border: 1px solid #34363b; border-radius: 8px; background: #202125; transition: border-color 150ms ease, background 150ms ease, box-shadow 150ms ease; }
.workflow-template-card:hover { border-color: #4a4d54; background: #222428; box-shadow: 0 10px 28px rgba(0,0,0,.2); }
.workflow-template-card__topline { height: 38px; display: flex; align-items: center; justify-content: space-between; padding: 0 12px; }
.workflow-template-card__topline > span { color: #8e969f; font-size: 10px; font-weight: 760; }
.workflow-template-card__topline > span.official { color: #80aee7; }
.workflow-template-card__delete { display: grid; width: 27px; height: 27px; place-items: center; border-radius: 5px; color: #636b76; opacity: 0; transition: opacity 140ms ease, background 140ms ease, color 140ms ease; }
.workflow-template-card:hover .workflow-template-card__delete, .workflow-template-card__delete:focus-visible { opacity: 1; }
.workflow-template-card__delete:hover { background: rgba(239,68,68,.1); color: #e98d8d; }
.workflow-template-card__delete:focus-visible { outline: 2px solid #609af0; outline-offset: 1px; }
.workflow-template-card__delete svg { width: 14px; height: 14px; }
.workflow-template-card__preview { height: 126px; display: grid; place-items: center; margin: 0 12px; overflow: hidden; border: 1px solid #2f3136; border-radius: 6px; background-color: #18191c; background-image: radial-gradient(#35383e 1px, transparent 1px); background-size: 14px 14px; }
.workflow-preview-flow { max-width: calc(100% - 28px); display: flex; align-items: center; justify-content: center; }
.workflow-preview-item { position: relative; display: flex; align-items: center; }
.workflow-preview-item:not(:last-of-type)::after { width: 20px; height: 1px; flex: 0 0 auto; background: #535963; content: ''; }
.workflow-preview-node { position: relative; z-index: 1; min-width: 43px; height: 33px; display: grid; place-items: center; padding: 0 7px; border: 1px solid #515863; border-radius: 5px; background: #292d33; color: #c9ced5; font-size: 9px; font-weight: 760; white-space: nowrap; box-shadow: 0 3px 10px rgba(0,0,0,.25); }
.workflow-preview-node.is-model { border-color: #477aa8; background: #22384a; color: #9ec6e8; }
.workflow-preview-node.is-output { border-color: #49806a; background: #213a31; color: #9ed0b8; }
.workflow-preview-node.is-image { border-color: #76699d; background: #332e46; color: #c0b6df; }
.workflow-preview-node.is-text { border-color: #6a7079; background: #303238; color: #ced1d6; }
.workflow-preview-node.is-llm { border-color: #3d7d83; background: #203a3d; color: #99d0d3; }
.workflow-preview-node.is-video { border-color: #9a6b4e; background: #442f25; color: #dbb296; }
.workflow-preview-node.is-color { border-color: #8b5867; background: #412832; color: #d5a3b1; }
.workflow-preview-node.is-translate { border-color: #557293; background: #273749; color: #a9c4df; }
.workflow-preview-more { min-width: 26px; margin-left: 8px; color: #737b86; font-size: 9px; font-weight: 760; }
.workflow-template-card__body { min-width: 0; padding: 15px 14px 0; }
.workflow-template-card__body strong { display: block; overflow: hidden; color: #eef0f2; font-size: 13px; font-weight: 760; text-overflow: ellipsis; white-space: nowrap; }
.workflow-template-card__body p { height: 34px; display: -webkit-box; overflow: hidden; margin: 6px 0 0; color: #7d848e; font-size: 11px; line-height: 1.65; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.workflow-template-card__meta { display: flex; align-items: center; gap: 7px; margin-top: auto; padding: 11px 14px 13px; color: #686f79; font-size: 10px; }
.workflow-template-card__meta i { width: 3px; height: 3px; border-radius: 50%; background: #4d535c; }
.workflow-template-card__use { height: 38px; display: flex; align-items: center; justify-content: center; gap: 6px; margin: 0 12px 12px; border-radius: 6px; background: #303238; color: #dce0e5; font-size: 11px; font-weight: 760; transition: background 140ms ease, color 140ms ease; }
.workflow-template-card__use:hover { background: #3c82d9; color: #fff; }
.workflow-template-card__use:focus-visible { outline: 2px solid #609af0; outline-offset: 2px; }
.workflow-template-card__use svg { width: 12px; height: 12px; }
.workflow-template-skeleton { min-height: 324px; display: flex; flex-direction: column; gap: 11px; padding: 12px; border: 1px solid #303237; border-radius: 8px; background: #202125; }
.workflow-template-skeleton i, .workflow-template-skeleton span, .workflow-template-skeleton button { border-radius: 5px; background: linear-gradient(90deg, #292b30 20%, #33353a 45%, #292b30 70%); background-size: 220% 100%; animation: workflow-template-shimmer 1.4s ease infinite; }
.workflow-template-skeleton i { height: 126px; }
.workflow-template-skeleton span { width: 70%; height: 13px; }
.workflow-template-skeleton span + span { width: 48%; height: 10px; }
.workflow-template-skeleton button { height: 38px; margin-top: auto; }
.workflow-template-delete-modal { position: fixed; inset: 0; z-index: 150; display: grid; place-items: center; padding: 20px; background: rgba(4,5,7,.64); backdrop-filter: blur(5px); }
.workflow-template-delete-modal__panel { width: min(390px, 100%); display: grid; grid-template-columns: 36px 1fr; gap: 12px; padding: 20px; border: 1px solid #3a3c42; border-radius: 8px; background: #1d1e22; box-shadow: 0 24px 70px rgba(0,0,0,.55); color: #f2f3f5; }
.workflow-template-delete-modal__icon { display: grid; width: 36px; height: 36px; place-items: center; border-radius: 7px; background: rgba(239,68,68,.1); color: #e58c8c; }
.workflow-template-delete-modal__icon svg { width: 17px; height: 17px; }
.workflow-template-delete-modal h3 { margin: 1px 0 0; font-size: 14px; letter-spacing: 0; }
.workflow-template-delete-modal p { margin: 7px 0 0; color: #858c96; font-size: 10px; line-height: 1.5; }
.workflow-template-delete-modal footer { grid-column: 1 / -1; display: flex; justify-content: flex-end; gap: 8px; margin-top: 7px; }
.workflow-template-delete-modal footer button { height: 34px; padding: 0 14px; border-radius: 6px; color: #aeb4bd; font-size: 10px; font-weight: 760; }
.workflow-template-delete-modal footer button:hover:not(:disabled) { background: #292b30; color: #f2f3f5; }
.workflow-template-delete-modal footer .danger { background: #b84545; color: #fff; }
.workflow-template-delete-modal footer .danger:hover:not(:disabled) { background: #c65353; }
.workflow-template-delete-modal footer button:disabled { opacity: .45; }
.workflow-library-pop-enter-active, .workflow-library-pop-leave-active { transition: opacity 150ms ease; }
.workflow-library-pop-enter-active .workflow-template-modal__panel, .workflow-library-pop-leave-active .workflow-template-modal__panel, .workflow-library-pop-enter-active .workflow-template-delete-modal__panel, .workflow-library-pop-leave-active .workflow-template-delete-modal__panel { transition: opacity 150ms ease, transform 170ms ease; }
.workflow-library-pop-enter-from, .workflow-library-pop-leave-to { opacity: 0; }
.workflow-library-pop-enter-from .workflow-template-modal__panel, .workflow-library-pop-leave-to .workflow-template-modal__panel, .workflow-library-pop-enter-from .workflow-template-delete-modal__panel, .workflow-library-pop-leave-to .workflow-template-delete-modal__panel { opacity: 0; transform: translateY(8px) scale(.99); }
@keyframes workflow-template-shimmer { to { background-position: -220% 0; } }
@media (max-width: 900px) {
  .workflow-template-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 640px) {
  .workflow-template-modal { padding: 10px; }
  .workflow-template-modal__panel { height: calc(100vh - 20px); }
  .workflow-template-modal__header { padding: 20px 18px 16px; }
  .workflow-template-controls { align-items: stretch; flex-direction: column-reverse; gap: 6px; padding: 0 18px; }
  .workflow-template-search { width: 100%; margin-bottom: 0; }
  .workflow-template-tabs { justify-content: space-between; }
  .workflow-template-tabs button { flex: 1; justify-content: center; }
  .workflow-template-grid { grid-template-columns: 1fr; padding: 16px 18px 24px; }
  .workflow-template-card__delete { opacity: 1; }
}
.workflow-save-modal { position: fixed; inset: 0; z-index: 140; display: grid; place-items: center; padding: 20px; background: rgba(4,5,7,.68); backdrop-filter: blur(6px); }
.workflow-save-modal__panel { width: min(500px, 100%); overflow: hidden; border: 1px solid #34363b; border-radius: 8px; background: #1b1c1f; box-shadow: 0 30px 90px rgba(0,0,0,.58), 0 0 0 1px rgba(255,255,255,.025); color: #f4f4f5; }
.workflow-save-modal__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; padding: 24px 26px 19px; }
.workflow-save-modal__title { display: flex; min-width: 0; align-items: center; gap: 9px; }
.workflow-save-modal__title h3 { margin: 0; font-size: 18px; font-weight: 760; letter-spacing: 0; }
.workflow-save-modal__title span { flex: 0 0 auto; padding: 3px 7px; border: 1px solid #3b3d43; border-radius: 4px; background: #242529; color: #949aa4; font-size: 10px; font-weight: 700; }
.workflow-save-modal__header p { margin: 7px 0 0; color: #777e89; font-size: 11px; line-height: 1.45; }
.workflow-save-modal__header > button { display: grid; width: 30px; height: 30px; flex: 0 0 auto; place-items: center; border-radius: 6px; color: #767d87; transition: background 140ms ease, color 140ms ease; }
.workflow-save-modal__header > button:hover:not(:disabled) { background: #292b2f; color: #eceef1; }
.workflow-save-modal__header > button:focus-visible { outline: 2px solid #609af0; outline-offset: 2px; }
.workflow-save-modal__header > button svg { width: 17px; height: 17px; }
.workflow-save-modal__body { display: flex; flex-direction: column; gap: 22px; padding: 4px 26px 26px; }
.workflow-save-name { display: flex; flex-direction: column; gap: 8px; }
.workflow-save-name > span, .workflow-save-scope legend, .workflow-save-destination > span { color: #b8bdc6; font-size: 11px; font-weight: 720; }
.workflow-save-name > div { position: relative; }
.workflow-save-name input { width: 100%; height: 44px; border: 1px solid #3a3c42; border-radius: 6px; outline: 0; background: #222327; padding: 0 52px 0 13px; color: #f2f3f5; font-size: 13px; transition: border-color 140ms ease, box-shadow 140ms ease, background 140ms ease; }
.workflow-save-name input:hover { border-color: #4a4d54; }
.workflow-save-name input:focus { border-color: #568ddd; background: #24262a; box-shadow: 0 0 0 3px rgba(59,130,246,.13); }
.workflow-save-name input::placeholder { color: #686e77; }
.workflow-save-name small { position: absolute; right: 12px; top: 50%; color: #626873; font-size: 9px; transform: translateY(-50%); }
.workflow-save-scope { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; border: 0; }
.workflow-save-scope > div { overflow: hidden; border: 1px solid #35373c; border-radius: 6px; background: #202125; }
.workflow-save-scope button { width: 100%; min-height: 52px; display: flex; align-items: center; gap: 11px; padding: 0 13px; color: #aeb4bd; text-align: left; transition: background 140ms ease; }
.workflow-save-scope button + button { border-top: 1px solid #33353a; }
.workflow-save-scope button:hover { background: #27292d; }
.workflow-save-scope button.active { background: #282c33; color: #f3f4f6; }
.workflow-save-scope button > i, .workflow-save-destination div > i { display: grid; width: 16px; height: 16px; flex: 0 0 auto; place-items: center; border: 1px solid #5c626d; border-radius: 50%; }
.workflow-save-scope button.active > i, .workflow-save-destination div > i { border-color: #6ea3ef; }
.workflow-save-scope button > i span, .workflow-save-destination div > i span { width: 8px; height: 8px; border-radius: 50%; background: transparent; }
.workflow-save-scope button.active > i span, .workflow-save-destination div > i span { background: #6ea3ef; }
.workflow-save-scope button > span { min-width: 0; display: flex; flex: 1; align-items: center; justify-content: space-between; gap: 10px; }
.workflow-save-scope strong, .workflow-save-destination strong { font-size: 12px; font-weight: 720; }
.workflow-save-scope small, .workflow-save-destination small { color: #747b86; font-size: 10px; }
.workflow-save-destination { display: flex; flex-direction: column; gap: 8px; }
.workflow-save-destination > div { min-height: 52px; display: flex; align-items: center; gap: 11px; padding: 0 13px; border: 1px solid #35373c; border-radius: 6px; background: #24262a; }
.workflow-save-destination > div small { margin-left: auto; }
.workflow-save-modal__actions { display: flex; justify-content: flex-end; gap: 9px; padding: 15px 26px; border-top: 1px solid #303237; background: #18191c; }
.workflow-save-modal__actions button { height: 37px; display: inline-flex; align-items: center; justify-content: center; gap: 7px; padding: 0 17px; border-radius: 6px; font-size: 11px; font-weight: 760; transition: background 140ms ease, border-color 140ms ease, color 140ms ease; }
.workflow-save-modal__actions .secondary { border: 1px solid #393b41; color: #aeb4bd; }
.workflow-save-modal__actions .secondary:hover:not(:disabled) { border-color: #4b4e55; background: #24262a; color: #eef0f2; }
.workflow-save-modal__actions .primary { min-width: 80px; background: #377fd8; color: #fff; box-shadow: 0 1px 0 rgba(255,255,255,.12) inset; }
.workflow-save-modal__actions .primary:hover:not(:disabled) { background: #438ce5; }
.workflow-save-modal__actions button:focus-visible { outline: 2px solid #609af0; outline-offset: 2px; }
.workflow-save-modal__actions button:disabled { cursor: not-allowed; opacity: .42; }
.workflow-save-spinner { width: 13px; height: 13px; border: 2px solid rgba(255,255,255,.35); border-top-color: #fff; border-radius: 50%; animation: workflow-save-spin .7s linear infinite; }
.workflow-save-pop-enter-active, .workflow-save-pop-leave-active { transition: opacity 150ms ease; }
.workflow-save-pop-enter-active .workflow-save-modal__panel, .workflow-save-pop-leave-active .workflow-save-modal__panel { transition: opacity 150ms ease, transform 170ms ease; }
.workflow-save-pop-enter-from, .workflow-save-pop-leave-to { opacity: 0; }
.workflow-save-pop-enter-from .workflow-save-modal__panel, .workflow-save-pop-leave-to .workflow-save-modal__panel { opacity: 0; transform: translateY(8px) scale(.985); }
@keyframes workflow-save-spin { to { transform: rotate(360deg); } }
@media (max-width: 520px) {
  .workflow-save-modal { align-items: end; padding: 12px; }
  .workflow-save-modal__panel { width: 100%; }
  .workflow-save-modal__header { padding: 21px 18px 17px; }
  .workflow-save-modal__body { padding: 3px 18px 22px; }
  .workflow-save-modal__actions { padding: 13px 18px; }
  .workflow-save-modal__actions button { flex: 1; }
}
@media (prefers-reduced-motion: reduce) {
  .workflow-save-pop-enter-active, .workflow-save-pop-leave-active, .workflow-save-pop-enter-active .workflow-save-modal__panel, .workflow-save-pop-leave-active .workflow-save-modal__panel { transition: none; }
  .workflow-save-spinner { animation-duration: 1.2s; }
}
.llm-node-card { width: 100%; height: 100%; display: flex; flex-direction: column; gap: 9px; padding: 13px; text-align: left; }
.llm-node-model-row { display: flex; min-height: 34px; align-items: center; gap: 8px; }
.llm-node-mark { display: grid; width: 28px; height: 28px; place-items: center; border: 1px solid rgba(147,197,253,.25); border-radius: 8px; background: rgba(59,130,246,.13); color: #93c5fd; }
.llm-node-mark svg { width: 16px; height: 16px; }
.llm-node-model-row label { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 1px; }
.llm-node-model-row label small { color: #7e8793; font-size: 9px; }
.llm-node-model-row select { max-width: 150px; border: 0; outline: 0; background: transparent; color: #f4f4f5; font-size: 12px; font-weight: 800; }
.llm-node-status { padding: 4px 7px; border: 1px solid rgba(255,255,255,.1); border-radius: 99px; color: #8b929e; font-size: 9px; font-weight: 850; }
.llm-node-status.running { border-color: rgba(147,197,253,.35); color: #93c5fd; }
.llm-node-status.done { border-color: rgba(134,239,172,.3); color: #86efac; }
.llm-node-context { display: flex; align-items: center; justify-content: space-between; padding: 9px 10px; border: 1px solid rgba(255,255,255,.08); border-radius: 7px; background: #202125; }
.llm-node-context div { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.llm-node-context div span { color: #d4d4d8; font-size: 11px; font-weight: 850; }
.llm-node-context div small { overflow: hidden; color: #777f8b; font-size: 9px; text-overflow: ellipsis; white-space: nowrap; }
.llm-context-count { display: grid; width: 21px; height: 21px; place-items: center; border-radius: 6px; background: #30343c; color: #cbd5e1; font-size: 10px; font-weight: 900; }
.llm-node-prompt { display: flex; min-height: 118px; flex: 1; flex-direction: column; gap: 5px; }
.llm-node-prompt > span { display: flex; justify-content: space-between; color: #d4d4d8; font-size: 11px; font-weight: 850; }
.llm-node-prompt > span small { color: #6f7782; font-size: 9px; font-weight: 600; }
.llm-node-prompt textarea { min-height: 90px; flex: 1; resize: none; border: 1px solid rgba(255,255,255,.1); border-radius: 7px; background: #17181a; padding: 10px; color: #f4f4f5; font-size: 11px; line-height: 1.5; }
.llm-node-advanced-trigger { display: flex; align-items: center; gap: 8px; padding: 7px 0; border: 0; color: #8b929e; font-size: 10px; text-align: left; }
.llm-node-advanced-trigger span:first-child { color: #b9c0ca; font-weight: 800; }
.llm-node-advanced-trigger small { margin-left: auto; color: #77808d; }
.llm-node-advanced-trigger span:last-child { transition: transform 150ms ease; }
.llm-node-advanced-trigger.open span:last-child { transform: rotate(180deg); }
.llm-node-advanced { padding: 7px 0 3px; }
.llm-node-advanced label { display: flex; align-items: center; gap: 7px; color: #727b87; font-size: 9px; }
.llm-node-advanced input { min-width: 0; flex: 1; accent-color: #9ca3af; }
.llm-node-result { max-height: 96px; overflow: auto; padding: 9px 10px; border: 1px solid rgba(134,239,172,.16); border-radius: 7px; background: rgba(20,48,37,.35); }
.llm-node-result.error { border-color: rgba(248,113,113,.25); background: rgba(69,25,25,.4); }
.llm-node-result > div { display: flex; align-items: center; justify-content: space-between; color: #86efac; font-size: 10px; font-weight: 850; }
.llm-node-result.error > div { color: #fca5a5; }
.llm-node-result button { color: inherit; font-size: 16px; }
.llm-node-result p { margin: 5px 0 0; color: #cbd5e1; font-size: 10px; line-height: 1.45; white-space: pre-wrap; }
.llm-node-footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding-top: 3px; }
.llm-node-footer small { color: #6f7782; font-size: 9px; }
.llm-node-footer button { display: inline-flex; height: 29px; align-items: center; gap: 5px; padding: 0 10px; border-radius: 6px; background: #f4f4f5; color: #18181b; font-size: 10px; font-weight: 900; }
.llm-node-footer button:disabled { opacity: .4; }
.llm-node-footer button svg { width: 12px; height: 12px; }
.llm-node-card { gap: 8px; padding: 10px; background: #1f2023; }
.llm-node-output { min-height: 160px; flex: 1; overflow: auto; padding: 12px; border: 1px solid rgba(255,255,255,.08); border-radius: 9px; background: #17181a; color: #e4e4e7; font-size: 12px; line-height: 1.65; white-space: pre-wrap; }
.llm-node-output.empty { display: flex; align-items: center; justify-content: center; color: #666d77; font-size: 11px; }
.llm-node-output.loading { color: #a9c6e8; }
.llm-node-output__loading { position: relative; padding-left: 16px; }
.llm-node-output__loading::before { position: absolute; left: 0; top: 5px; width: 8px; height: 8px; border: 1px solid #93c5fd; border-top-color: transparent; border-radius: 50%; content: ''; animation: llm-spin .8s linear infinite; }
.llm-node-output__error { color: #fca5a5; }
.llm-node-output__placeholder { opacity: .8; }
.llm-node-input { position: relative; display: flex; align-items: flex-end; min-height: 76px; border: 1px solid rgba(255,255,255,.13); border-radius: 9px; background: #25272b; }
.llm-node-input:focus-within { border-color: rgba(147,197,253,.55); box-shadow: 0 0 0 2px rgba(59,130,246,.12); }
.llm-node-input textarea { width: 100%; min-height: 74px; padding: 11px 42px 10px 12px; resize: none; border: 0; outline: 0; background: transparent; color: #f4f4f5; font-size: 12px; line-height: 1.5; }
.llm-node-input textarea::placeholder { color: #6f7782; }
.llm-node-send { position: absolute; right: 8px; bottom: 8px; display: grid; width: 28px; height: 28px; place-items: center; border-radius: 7px; background: #f4f4f5; color: #18181b; }
.llm-node-send:disabled { opacity: .35; }
.llm-node-send svg { width: 15px; height: 15px; }
@keyframes llm-spin { to { transform: rotate(360deg); } }
.node-llm-workflow > .node-label { display: none; }
.node-llm-workflow .node-body { min-height: 0; padding: 0; }
.llm-node-card { gap: 8px; padding: 10px; background: #202124; }
.llm-node-head { display: flex; height: 27px; align-items: center; gap: 7px; }
.llm-node-head__icon { display: grid; width: 22px; height: 22px; place-items: center; border-radius: 6px; background: rgba(147,197,253,.12); color: #93c5fd; }
.llm-node-head__icon svg { width: 14px; height: 14px; }
.llm-node-head strong { color: #f4f4f5; font-size: 12px; font-weight: 850; }
.llm-node-head__state { margin-left: auto; color: #77808d; font-size: 10px; }
.llm-node-head__state.loading { color: #93c5fd; }
.llm-node-editor { position: relative; min-height: 195px; display: flex; flex-direction: column; overflow: hidden; border: 1px solid rgba(255,255,255,.12); border-radius: 9px; background: #17181a; }
.llm-node-editor:focus-within { border-color: rgba(147,197,253,.5); box-shadow: 0 0 0 2px rgba(59,130,246,.12); }
.llm-node-editor.loading { border-color: rgba(147,197,253,.35); }
.llm-node-editor.error { border-color: rgba(248,113,113,.38); }
.llm-node-editor textarea { width: 100%; min-height: 150px; flex: 1; padding: 12px 12px 30px; resize: none; border: 0; outline: 0; background: transparent; color: #f4f4f5; font-size: 12px; line-height: 1.65; }
.llm-node-editor textarea::placeholder { color: #626a75; }
.llm-node-editor__hint { position: absolute; left: 12px; right: 44px; bottom: 9px; overflow: hidden; color: #626a75; font-size: 9px; text-overflow: ellipsis; white-space: nowrap; pointer-events: none; }
.llm-node-editor.error .llm-node-editor__hint { color: #fca5a5; }
.llm-node-send { position: absolute; right: 8px; bottom: 7px; display: grid; width: 27px; height: 27px; place-items: center; border-radius: 7px; background: #f4f4f5; color: #18181b; }
.llm-node-send:disabled { opacity: .35; }
.llm-node-send svg { width: 14px; height: 14px; }
.node-llm-workflow > .node-label { display: block; }
.node-llm-workflow .node-body { min-height: 100%; padding: 0; }
.llm-node-textarea { width: 100%; height: 100%; min-height: 250px; padding: 18px; resize: none; border: 0; outline: 0; border-radius: inherit; background: #202124; color: #f4f4f5; font-size: 14px; line-height: 1.7; }
.llm-node-textarea::placeholder { color: #6f7782; }
.llm-node-textarea:focus { background: #222326; box-shadow: none; }
.llm-node-textarea.loading { color: #93c5fd; cursor: wait; }
.llm-node-textarea.error { color: #fca5a5; }
.node-menu__icon--llm { color: #93c5fd; }
.node-llm-workflow .node-body--center { position: relative; display: block; }
.node-llm-workflow .llm-node-textarea { padding: 18px 18px 58px; }
.llm-node-loading-panel { position: absolute; inset: 0; min-height: 0; border-radius: inherit; }
.llm-node-run { position: absolute; right: 12px; bottom: 12px; z-index: 2; display: grid; width: 34px; height: 34px; place-items: center; border-radius: 7px; background: #f4f4f5; color: #191a1d; box-shadow: 0 4px 14px rgba(0,0,0,.24); transition: background 140ms ease, color 140ms ease, transform 140ms ease; }
.llm-node-run.has-resize { right: 46px; }
.llm-node-run:hover:not(:disabled) { background: #fff; transform: translateY(-1px); }
.llm-node-run:focus-visible { outline: 2px solid #60a5fa; outline-offset: 2px; }
.llm-node-run:disabled { cursor: not-allowed; opacity: .35; transform: none; }
.llm-node-run svg { width: 15px; height: 15px; }
.llm-node-run.loading > span { width: 14px; height: 14px; border: 2px solid rgba(24,24,27,.28); border-top-color: #18181b; border-radius: 50%; animation: llm-spin .75s linear infinite; }
.llm-node-run-state { position: absolute; right: 56px; bottom: 19px; left: 16px; z-index: 1; display: flex; min-width: 0; align-items: center; gap: 6px; overflow: hidden; color: #8eb8eb; font-size: 10px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; pointer-events: none; }
.llm-node-run-state i { width: 7px; height: 7px; flex: 0 0 auto; border: 1px solid currentColor; border-top-color: transparent; border-radius: 50%; animation: llm-spin .75s linear infinite; }
.llm-node-run-state.error { color: #f09a9a; }
.llm-node-resize { z-index: 3; }

.shortcut-modal {
  position: fixed;
  inset: 0;
  z-index: 108;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 36px 72px 84px;
  background: rgba(0, 0, 0, 0.18);
  pointer-events: auto;
}

.shortcut-modal__panel {
  position: relative;
  display: grid;
  width: min(1720px, calc(100vw - 144px));
  max-height: min(650px, calc(100vh - 140px));
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
  overflow: hidden;
  padding: 36px 34px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 22px;
  background: #252525;
  color: #f8fafc;
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.48);
}

.shortcut-modal__close {
  position: absolute;
  top: 18px;
  right: 22px;
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: #d4d4d8;
  font-size: 30px;
  line-height: 1;
  cursor: pointer;
}

.shortcut-modal__close:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.shortcut-section {
  min-width: 0;
  padding: 0 36px;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
}

.shortcut-section:first-of-type {
  padding-left: 2px;
}

.shortcut-section:last-of-type {
  padding-right: 0;
  border-right: 0;
}

.shortcut-section h3 {
  margin: 0 0 26px;
  color: #06d9ff;
  font-size: 16px;
  font-weight: 800;
}

.shortcut-section dl {
  display: grid;
  gap: 18px;
  margin: 0;
}

.shortcut-section dl > div {
  display: grid;
  min-height: 24px;
  grid-template-columns: minmax(96px, 1fr) auto;
  align-items: center;
  gap: 14px;
}

.shortcut-section dt {
  overflow: hidden;
  margin: 0;
  color: #9ca3af;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shortcut-section dd {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 9px;
  margin: 0;
  color: #f8fafc;
  font-size: 13px;
  font-weight: 800;
  line-height: 1.35;
  text-align: right;
  white-space: nowrap;
}

.shortcut-section dd span {
  color: #71717a;
  font-size: 13px;
}

.shortcut-section kbd {
  display: inline-flex;
  min-width: 30px;
  height: 32px;
  align-items: center;
  justify-content: center;
  padding: 0 9px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.04);
  color: #f8fafc;
  font-family: inherit;
  font-size: 15px;
  font-weight: 900;
  box-shadow: inset 0 -1px 0 rgba(255, 255, 255, 0.06);
}

.history-modal {
  position: fixed;
  inset: 0;
  z-index: 110;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(18px, 4vw, 48px);
  background: rgba(0, 0, 0, 0.72);
}

.history-modal__panel {
  display: flex;
  min-width: 0;
  width: min(1480px, 96vw);
  height: min(900px, 92vh);
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: #242424;
  color: #e5e7eb;
  box-shadow: 0 28px 90px rgba(0, 0, 0, 0.58);
}

.history-modal__header {
  display: flex;
  min-height: 58px;
  align-items: center;
  justify-content: space-between;
  padding: 0 18px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.history-modal__header h3 {
  margin: 0;
  color: #f8fafc;
  font-size: 18px;
  font-weight: 800;
}

.history-modal__tools {
  display: flex;
  align-items: center;
  gap: 4px;
}

.history-modal__tools button {
  height: 32px;
  border: 0;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.06);
  color: #d4d4d8;
  cursor: pointer;
}

.history-modal__tools button {
  min-width: 32px;
  padding: 0 9px;
  font-size: 18px;
  font-weight: 800;
}

.history-modal__tools span {
  min-width: 54px;
  color: #e5e7eb;
  text-align: center;
  font-size: 13px;
  font-weight: 800;
}

.history-modal__tools button:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.history-modal__close {
  margin-left: 12px;
  font-size: 22px !important;
}

.history-modal__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 18px;
}

.history-group + .history-group {
  margin-top: 28px;
}

.history-group h4 {
  margin: 0 0 14px;
  color: #f4f4f5;
  font-size: 15px;
  font-weight: 800;
}

.history-grid {
  --history-thumb-scale: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(calc(118px * var(--history-thumb-scale)), calc(118px * var(--history-thumb-scale))));
  gap: 14px;
}

.history-grid button {
  position: relative;
  width: calc(118px * var(--history-thumb-scale));
  aspect-ratio: 1;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 8px;
  background: #181818;
  cursor: default;
}

.history-grid img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.history-card__video {
  display: block;
  width: 100%;
  height: 100%;
  background: #050505;
  object-fit: cover;
}

.history-card__shade {
  position: absolute;
  inset: 0;
  display: block;
  background: rgba(0, 0, 0, 0.52);
  opacity: 0;
  transition: opacity 150ms ease;
}

.history-card__delete {
  position: absolute;
  right: 8px;
  top: 8px;
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border-radius: 7px;
  background: rgba(24, 24, 27, 0.82);
  color: #e5e7eb;
  font-size: 18px;
  line-height: 1;
  opacity: 0;
  transition: opacity 150ms ease, background 150ms ease;
}

.history-card__actions {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  color: #fff;
  font-size: 14px;
  font-weight: 800;
  opacity: 0;
  transition: opacity 150ms ease;
}

.history-card__actions em {
  font-style: normal;
}

.history-grid button:hover .history-card__shade,
.history-grid button:hover .history-card__delete,
.history-grid button:hover .history-card__actions {
  opacity: 1;
}

.history-empty {
  display: grid;
  min-height: 260px;
  place-items: center;
  color: #8b929e;
  font-size: 14px;
}

.style-library-modal {
  position: fixed;
  inset: 0;
  z-index: 112;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(18px, 4vw, 48px);
  background: rgba(0, 0, 0, 0.72);
}

.style-library-panel {
  display: flex;
  width: min(1180px, 96vw);
  height: min(780px, 90vh);
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 18px;
  background: #242424;
  color: #e5e7eb;
  box-shadow: 0 28px 90px rgba(0, 0, 0, 0.58);
}

.style-library-header {
  display: flex;
  min-height: 68px;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.style-library-header span {
  color: #9ca3af;
  font-size: 12px;
  font-weight: 800;
}

.style-library-header h3 {
  margin: 4px 0 0;
  color: #f8fafc;
  font-size: 20px;
}

.style-library-header small {
  display: block;
  margin-top: 4px;
  color: #9ca3af;
  font-size: 12px;
}

.style-library-scene-switch {
  margin-left: auto;
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.07);
}

.style-library-scene-switch button {
  width: auto;
  min-width: 62px;
  padding: 0 12px;
  border-radius: 8px;
  background: transparent;
  color: #a1a1aa;
  font-size: 13px;
  font-weight: 800;
}

.style-library-scene-switch button.active,
.style-library-scene-switch button:hover {
  background: #f8fafc;
  color: #111827;
}

.style-library-header > button {
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.08);
  color: #f4f4f5;
  font-size: 22px;
  cursor: pointer;
}

.style-library-body {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: 240px minmax(0, 1fr);
}

.style-library-tags {
  min-height: 0;
  overflow: auto;
  padding: 14px;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
}

.style-library-tags button {
  display: flex;
  width: 100%;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 12px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: #d4d4d8;
  cursor: pointer;
}

.style-library-tags button.active,
.style-library-tags button:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.style-library-tags strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.style-library-tags span {
  color: #9ca3af;
  font-size: 12px;
}

.style-library-refs {
  display: grid;
  min-height: 0;
  grid-template-columns: repeat(auto-fill, 180px);
  align-content: start;
  justify-content: start;
  gap: 20px;
  overflow: auto;
  padding: 22px;
}

.style-library-card {
  width: 180px;
  height: 180px;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 14px;
  background: #181818;
  color: #f4f4f5;
  cursor: pointer;
}

.style-library-card img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: transparent;
}

.style-library-card:hover {
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.22);
}

.material-library-panel {
  width: min(920px, calc(100vw - 96px));
  height: min(720px, calc(100vh - 96px));
}

.material-library-body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 22px;
}

.material-library-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
}

.material-library-card {
  display: grid;
  gap: 10px;
  min-width: 0;
  padding: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.05);
  color: #f4f4f5;
  cursor: pointer;
  text-align: left;
  transition: border-color 150ms ease, background 150ms ease, transform 150ms ease;
}

.material-library-card:hover {
  border-color: rgba(255, 255, 255, 0.24);
  background: rgba(255, 255, 255, 0.09);
  transform: translateY(-1px);
}

.material-library-card.selected {
  border-color: rgba(248, 250, 252, 0.82);
  background: rgba(255, 255, 255, 0.13);
  box-shadow: 0 0 0 2px rgba(248, 250, 252, 0.14);
}

.material-library-card__image {
  display: grid;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 1;
  place-items: center;
  border-radius: 10px;
  background: #101010;
}

.material-library-card__image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.material-library-card__image em {
  color: #f8fafc;
  font-size: 28px;
  font-style: normal;
  font-weight: 900;
}

.material-library-card strong {
  overflow: hidden;
  color: #f8fafc;
  font-size: 14px;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.batch-material-picker {
  display: flex;
  flex-direction: column;
}

.batch-material-picker .material-library-body {
  padding-bottom: 14px;
}

.batch-material-picker__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.batch-material-picker__footer span {
  color: #d4d4d8;
  font-size: 13px;
  font-weight: 800;
}

.batch-material-picker__footer button {
  height: 36px;
  padding: 0 18px;
  border: 0;
  border-radius: 10px;
  background: #f8fafc;
  color: #111827;
  font-size: 13px;
  font-weight: 900;
  cursor: pointer;
}

.style-library-empty {
  display: grid;
  min-height: 160px;
  place-items: center;
  color: #8b929e;
  font-size: 13px;
}

.canvas-toolbar .toolbar-separator {
  width: 1px;
  height: 22px;
  margin: 0 6px;
  background: rgba(148, 163, 184, 0.2);
}

.node-menu {
  position: absolute;
  left: 50%;
  bottom: 56px;
  width: 280px;
  padding: 10px;
  transform: translateX(-50%);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  background: rgba(31, 31, 31, 0.98);
  box-shadow: 0 16px 50px rgba(0, 0, 0, 0.42);
  transform-origin: bottom center;
  animation: canvas-menu-in 140ms ease-out both;
}

.node-menu button {
  width: 100%;
  height: auto;
  display: grid;
  grid-template-columns: 38px 1fr;
  grid-template-rows: auto auto;
  align-items: center;
  column-gap: 10px;
  padding: 10px;
  border-radius: 12px;
  color: #e5e7eb;
  text-align: left;
}

.node-menu__library.open {
  background: rgba(255, 255, 255, 0.1);
}

.node-menu button:hover {
  background: rgba(255, 255, 255, 0.08);
}

.node-menu button:active {
  background: rgba(255, 255, 255, 0.14);
  transform: scale(0.99);
}

@keyframes canvas-menu-in {
  from { opacity: 0; transform: translateX(-50%) translateY(6px) scale(0.98); }
  to { opacity: 1; transform: translateX(-50%) translateY(0) scale(1); }
}

.node-menu-submenu {
  position: absolute;
  left: calc(100% + 12px);
  top: 84px;
  display: flex;
  width: 180px;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 18px;
  background: rgba(35, 35, 36, 0.98);
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.34);
}

.node-menu-submenu button {
  display: flex;
  height: 42px;
  align-items: center;
  justify-content: flex-start;
  padding: 0 10px;
  border-radius: 10px;
  color: #f4f4f5;
  font-size: 15px;
  font-weight: 800;
}

.node-menu-submenu button:hover {
  background: rgba(255, 255, 255, 0.1);
}

.node-menu__icon {
  grid-row: 1 / span 2;
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.07);
  color: #f8fafc;
}

.canvas-toolbar .node-menu__icon {
  width: 38px;
  height: 38px;
  background: rgba(255, 255, 255, 0.07);
}

.node-menu__icon svg {
  width: 21px;
  height: 21px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.node-menu__icon--model {
  color: #bfdbfe;
}

.node-menu__icon--image {
  color: #bbf7d0;
}

.node-menu__icon--video {
  color: #fecdd3;
}

.node-menu__icon--asset {
  color: #a7f3d0;
}

.node-menu__icon--style {
  color: #fde68a;
}

.node-menu__icon--text {
  color: #e9d5ff;
}

.node-menu button strong {
  color: #f8fafc;
  font-size: 14px;
}

.node-menu button small {
  margin-top: 2px;
  color: #a1a1aa;
  font-size: 12px;
}

.node-menu button:hover {
  background: rgba(255, 255, 255, 0.08);
}

.floating-error {
  position: absolute;
  left: 28px;
  bottom: 28px;
  z-index: 45;
  display: flex;
  max-width: 520px;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 10px 10px 12px;
  border: 1px solid rgba(248, 113, 113, 0.35);
  border-radius: 10px;
  background: rgba(127, 29, 29, 0.82);
  color: #fecaca;
  font-size: 13px;
}

.floating-error span {
  min-width: 0;
  line-height: 1.45;
}

.floating-error button {
  display: grid;
  width: 22px;
  height: 22px;
  flex: 0 0 auto;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: currentColor;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}

.floating-error button:hover {
  background: rgba(255, 255, 255, 0.12);
}

@media (max-width: 1024px) {
  .workflow-editor {
    overflow: auto;
  }

  .canvas-topbar {
    position: fixed;
  }

  .canvas-toolbar {
    position: fixed;
  }

  .shortcut-modal {
    padding: 18px 14px 78px;
  }

  .shortcut-modal__panel {
    width: 100%;
    max-height: calc(100vh - 110px);
    grid-template-columns: repeat(2, minmax(0, 1fr));
    overflow-y: auto;
    padding: 28px 20px;
  }

  .shortcut-section {
    padding: 18px;
    border-right: 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  }

  .shortcut-section:nth-last-of-type(-n + 2) {
    border-bottom: 0;
  }

  .image-modal__img {
    max-width: 92vw;
    max-height: 78vh;
  }

  .history-modal {
    padding: 12px;
  }

  .history-modal__panel {
    width: 100%;
    height: 82vh;
  }

  .history-grid {
    grid-template-columns: repeat(auto-fill, minmax(104px, 104px));
  }

  .history-grid button {
    width: 104px;
  }
}

@media (max-width: 760px) {
  .shortcut-modal__panel {
    grid-template-columns: 1fr;
  }

  .shortcut-section,
  .shortcut-section:nth-last-of-type(-n + 2) {
    padding: 16px 4px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  }

  .shortcut-section:last-of-type {
    border-bottom: 0;
  }

  .shortcut-section dl > div {
    grid-template-columns: minmax(86px, 1fr) auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .canvas-node,
  .node-link__line,
  .node-port,
  .canvas-toolbar button,
  .node-menu button {
    transition: none;
  }

  .node-menu,
  .node-link.transmitting .node-link__line {
    animation: none;
  }
}
</style>
