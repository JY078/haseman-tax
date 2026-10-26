import test from 'node:test'
import assert from 'node:assert/strict'

import { sanitizeWorkflowTemplateState } from './workflowTemplate.js'

function templateState() {
  return {
    workflowNodes: {},
    workflowDynamicNodes: [
      { id: 'generated-image', type: 'generate' },
      { id: 'imported-image', type: 'generate' },
      { id: 'batch', type: 'batch-recolor' },
      { id: 'batch-result', type: 'generate' },
      { id: 'output', type: 'output' },
      { id: 'video', type: 'video' },
    ],
    resultImageUrl: 'blob:composite',
    resultUrls: { white: 'blob:white', depth: 'blob:depth' },
    workflowHistoryAssets: [{ url: 'blob:generated' }],
    workflowGenerateNodeStates: {
      'generated-image': {
        prompt: 'product photo',
        generating: true,
        progress: 72,
        runId: 'run-1',
        result: {
          url: 'blob:generated',
          images: ['blob:generated'],
          imageSources: ['generated'],
          taskIds: ['task-1'],
          taskId: 'task-1',
          status: 'completed',
        },
        uploads: [{ url: 'blob:input' }],
        editRef: { url: 'blob:generated' },
        inlineControlImages: { white: 'blob:white', depth: 'blob:depth' },
      },
      'imported-image': {
        uploadOnly: true,
        result: {
          url: 'oss://input',
          images: ['oss://input'],
          imageSources: ['imported'],
          status: 'imported',
        },
      },
      'batch-result': {
        generatedFromNode: 'batch',
        result: {
          url: 'blob:batch-result',
          images: ['blob:batch-result'],
          imageSources: ['batch-recolor'],
          batchItems: [{ imageUrl: 'blob:batch-result' }],
          batchTitle: 'Black',
          status: 'completed',
        },
      },
    },
    workflowOutputNodeStates: {
      output: { sourceNode: 'generated-image', resultUrls: { white: 'blob:white' } },
    },
    workflowBatchRecolorNodeStates: {
      batch: { target: 'faucet', tasks: [{ imageUrl: 'blob:batch-result' }], loading: true, progress: 80, resultNode: 'batch-result', runId: 'run-2' },
    },
    workflowBatchTranslateNodeStates: {},
    workflowVideoNodeStates: {
      video: { prompt: 'camera move', videoUrl: 'blob:video', videoWidth: 1280, videoHeight: 720, taskId: 'task-video', status: 'completed' },
    },
  }
}

test('generated images become empty reusable image nodes in workflow templates', () => {
  const source = templateState()
  const sanitized = sanitizeWorkflowTemplateState(source)
  const generated = sanitized.workflowGenerateNodeStates['generated-image']

  assert.equal(generated.prompt, 'product photo')
  assert.deepEqual(generated.uploads, [{ url: 'blob:input' }])
  assert.deepEqual(generated.result.images, [])
  assert.deepEqual(generated.result.imageSources, ['generated'])
  assert.equal(generated.result.url, '')
  assert.equal(generated.generating, false)
  assert.deepEqual(generated.inlineControlImages, { white: '', depth: '' })
  assert.equal(source.workflowGenerateNodeStates['generated-image'].result.url, 'blob:generated')
})

test('workflow template sanitizing preserves image inputs and derived node structure', () => {
  const sanitized = sanitizeWorkflowTemplateState(templateState())
  const imported = sanitized.workflowGenerateNodeStates['imported-image']
  const batchResult = sanitized.workflowGenerateNodeStates['batch-result']

  assert.equal(imported.result.url, 'oss://input')
  assert.deepEqual(imported.result.images, ['oss://input'])
  assert.equal(batchResult.generatedFromNode, 'batch')
  assert.deepEqual(batchResult.result.imageSources, ['batch-recolor'])
  assert.deepEqual(batchResult.result.images, [])
  assert.equal(batchResult.result.batchTitle, 'Black')
})

test('workflow template sanitizing clears output task state before asset upload', () => {
  const sanitized = sanitizeWorkflowTemplateState(templateState())
  const serialized = JSON.stringify(sanitized)

  assert.equal(sanitized.resultImageUrl, '')
  assert.deepEqual(sanitized.resultUrls, {})
  assert.deepEqual(sanitized.workflowHistoryAssets, [])
  assert.deepEqual(sanitized.workflowOutputNodeStates.output.resultUrls, {})
  assert.deepEqual(sanitized.workflowBatchRecolorNodeStates.batch.tasks, [])
  assert.equal(sanitized.workflowBatchRecolorNodeStates.batch.target, 'faucet')
  assert.equal(sanitized.workflowVideoNodeStates.video.videoUrl, '')
  assert.equal(sanitized.workflowVideoNodeStates.video.prompt, 'camera move')
  assert.equal(serialized.includes('blob:generated'), false)
  assert.equal(serialized.includes('blob:batch-result'), false)
  assert.equal(serialized.includes('blob:video'), false)
  assert.equal(serialized.includes('blob:white'), false)
  assert.equal(serialized.includes('blob:depth'), false)
})
