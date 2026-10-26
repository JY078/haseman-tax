import * as THREE from 'three'

// Include cut-plane intersections: a triangle can cross the cut even when
// its lowest vertices are hidden.
export function visibleMeshPoints(mesh, cutY = -Infinity) {
  const position = mesh.geometry.attributes.position
  mesh.updateMatrixWorld(true)
  const vertices = Array.from({ length: position.count }, (_, index) =>
    new THREE.Vector3().fromBufferAttribute(position, index).applyMatrix4(mesh.matrixWorld))
  if (!Number.isFinite(cutY)) return vertices
  const points = vertices.filter(point => point.y >= cutY)
  const indices = mesh.geometry.index
  const count = indices ? indices.count : position.count
  for (let index = 0; index + 2 < count; index += 3) {
    const triangle = [0, 1, 2].map(offset => vertices[indices ? indices.getX(index + offset) : index + offset])
    for (let edge = 0; edge < 3; edge += 1) {
      const a = triangle[edge]
      const b = triangle[(edge + 1) % 3]
      if ((a.y < cutY) === (b.y < cutY)) continue
      points.push(a.clone().lerp(b, (cutY - a.y) / (b.y - a.y)))
    }
  }
  return points
}

// Shift the projection window, not the viewing direction. This centers the
// actual silhouette exactly even for asymmetric objects with varying depth.
export function centerCameraOnPoints(camera, points) {
  camera.clearViewOffset()
  camera.updateMatrixWorld(true)
  let minX = Infinity
  let maxX = -Infinity
  let minY = Infinity
  let maxY = -Infinity
  const projected = new THREE.Vector3()
  for (const point of points) {
    projected.copy(point).project(camera)
    if (!Number.isFinite(projected.x) || !Number.isFinite(projected.y)) continue
    minX = Math.min(minX, projected.x)
    maxX = Math.max(maxX, projected.x)
    minY = Math.min(minY, projected.y)
    maxY = Math.max(maxY, projected.y)
  }
  if (!Number.isFinite(minX)) return
  const aspect = camera.aspect
  camera.setViewOffset(2 * aspect, 2, (minX + maxX) * aspect / 2, -(minY + maxY) / 2, 2 * aspect, 2)
}
