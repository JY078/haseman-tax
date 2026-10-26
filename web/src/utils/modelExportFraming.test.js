import test from 'node:test'
import assert from 'node:assert/strict'
import * as THREE from 'three'
import { visibleMeshPoints, centerCameraOnPoints } from './modelExportFraming.js'

function bounds(camera, points) {
  const projected = points.map(point => point.clone().project(camera))
  return {
    left: Math.min(...projected.map(point => point.x)),
    right: Math.max(...projected.map(point => point.x)),
    bottom: Math.min(...projected.map(point => point.y)),
    top: Math.max(...projected.map(point => point.y)),
  }
}

test('asymmetric product stays centered at every 15 degree turn with a fixed distance', () => {
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.Float32BufferAttribute([
    -4, 0, -1, 2, 0, 1, -3, 7, 0,
    2, 0, 1, 1, 2, 4, -3, 7, 0,
  ], 3))
  const mesh = new THREE.Mesh(geometry)
  const camera = new THREE.PerspectiveCamera(40, 1, 0.01, 1000)
  camera.position.set(15, 12, 25)
  camera.lookAt(0, 3, 0)
  const position = camera.position.clone()
  const orientation = camera.quaternion.clone()
  for (let angle = 0; angle < 360; angle += 15) {
    mesh.rotation.set(0.2, THREE.MathUtils.degToRad(angle), -0.1)
    const points = visibleMeshPoints(mesh)
    centerCameraOnPoints(camera, points)
    const result = bounds(camera, points)
    assert.ok(Math.abs(result.left + result.right) < 1e-10, `horizontal center at ${angle}`)
    assert.ok(Math.abs(result.bottom + result.top) < 1e-10, `vertical center at ${angle}`)
    assert.ok(Math.max(Math.abs(result.left), Math.abs(result.right), Math.abs(result.top), Math.abs(result.bottom)) < 0.9)
    assert.ok(camera.position.equals(position))
    assert.ok(camera.quaternion.equals(orientation))
  }
})

test('cut geometry includes boundary intersections for indexed and non-indexed triangles', () => {
  for (const indexed of [false, true]) {
    const geometry = new THREE.BufferGeometry()
    geometry.setAttribute('position', new THREE.Float32BufferAttribute([-6, -2, 0, 6, -2, 0, 0, 2, 0], 3))
    if (indexed) geometry.setIndex([0, 1, 2])
    const mesh = new THREE.Mesh(geometry)
    mesh.position.x = 10
    const points = visibleMeshPoints(mesh, 0)
    assert.equal(points.length, 3)
    assert.equal(Math.min(...points.map(point => point.x)), 7)
    assert.equal(Math.max(...points.map(point => point.x)), 13)
    assert.equal(Math.min(...points.map(point => point.y)), 0)
  }
})

test('framing includes extremities beyond the old 3200 vertex sampling limit', () => {
  const positions = new Float32Array(9600 * 3)
  positions[3] = 10
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
  const points = visibleMeshPoints(new THREE.Mesh(geometry))
  assert.equal(points.length, 9600)
  assert.equal(Math.max(...points.map(point => point.x)), 10)
})
