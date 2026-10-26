const REUSABLE_IMAGE_SOURCES = new Set([
  'imported',
  'material-reference',
  'style-reference',
])

function workflowTemplateNodeIds(state = {}) {
  return new Set([
    ...Object.entries(state.workflowNodes || {})
      .filter(([, enabled]) => enabled)
      .map(([node]) => node),
    ...(state.workflowDynamicNodes || []).map(node => node?.id).filter(Boolean),
  ])
}

function isReusableImageResult(result = {}) {
  const hasImage = Boolean(result.url) || (Array.isArray(result.images) && result.images.some(Boolean))
  if (!hasImage) return false
  if (result.status === 'imported') return true
  const sources = Array.isArray(result.imageSources)
    ? result.imageSources
    : (Array.isArray(result.image_sources) ? result.image_sources : [])
  return sources.length > 0 && sources.every(source => REUSABLE_IMAGE_SOURCES.has(source))
}

function emptyRuntimeImageResult(result = {}) {
  const imageSources = Array.isArray(result.imageSources)
    ? result.imageSources
    : (Array.isArray(result.image_sources) ? result.image_sources : [])
  return {
    url: '',
    images: [],
    imageCreatedAt: [],
    imageSources: [...imageSources],
    taskIds: [],
    taskId: '',
    status: '',
    createdAt: '',
    batchItems: [],
    batchTitle: result.batchTitle || '',
  }
}

function sanitizeGenerateNodeState(state = {}, includedNodes) {
  const { workflowGenerationResult, ...nodeState } = state
  const result = state.result || workflowGenerationResult || {}
  return {
    ...nodeState,
    menu: '',
    generating: false,
    error: '',
    progress: 0,
    cancelled: false,
    runId: '',
    taskProgress: [],
    result: isReusableImageResult(result) ? { ...result } : emptyRuntimeImageResult(result),
    editRef: null,
    editPreview: null,
    showResults: false,
    generatedFromNode: includedNodes.has(state.generatedFromNode) ? state.generatedFromNode : '',
    sceneControlOutputNode: includedNodes.has(state.sceneControlOutputNode) ? state.sceneControlOutputNode : '',
    inlineControlImages: { white: '', depth: '' },
  }
}

function mapValues(map = {}, mapper) {
  return Object.fromEntries(Object.entries(map).map(([key, value]) => [key, mapper(value || {})]))
}

export function sanitizeWorkflowTemplateState(state = {}) {
  const includedNodes = workflowTemplateNodeIds(state)
  return {
    ...state,
    resultImageUrl: '',
    resultUrls: {},
    workflowHistoryAssets: [],
    workflowGenerateNodeStates: mapValues(
      state.workflowGenerateNodeStates,
      nodeState => sanitizeGenerateNodeState(nodeState, includedNodes),
    ),
    workflowOutputNodeStates: mapValues(state.workflowOutputNodeStates, nodeState => ({
      ...nodeState,
      resultUrls: {},
      sourceNode: includedNodes.has(nodeState.sourceNode) ? nodeState.sourceNode : '',
    })),
    workflowBatchRecolorNodeStates: mapValues(state.workflowBatchRecolorNodeStates, nodeState => ({
      ...nodeState,
      menu: '',
      tasks: [],
      loading: false,
      progress: 0,
      resultNode: '',
      error: '',
      runId: '',
    })),
    workflowBatchTranslateNodeStates: mapValues(state.workflowBatchTranslateNodeStates, nodeState => ({
      ...nodeState,
      menu: '',
      tasks: [],
      loading: false,
      progress: 0,
      resultNode: '',
      error: '',
      runId: '',
    })),
    workflowVideoNodeStates: mapValues(state.workflowVideoNodeStates, nodeState => ({
      ...nodeState,
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
    })),
  }
}
