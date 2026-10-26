export const IMAGE_EDIT_BRUSH_DEFAULTS = Object.freeze({
  color: '#ff3b30',
  opacity: 0.7,
  size: 20,
  maxSize: 80,
})

export function drawSmoothBrushStroke(ctx, points, brushSize) {
  if (!points.length) return

  if (points.length === 1) {
    const point = points[0]
    ctx.beginPath()
    ctx.arc(point.x, point.y, brushSize / 2, 0, Math.PI * 2)
    ctx.fill()
    return
  }

  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)

  if (points.length === 2) {
    ctx.lineTo(points[1].x, points[1].y)
  } else {
    for (let index = 1; index < points.length - 1; index += 1) {
      const current = points[index]
      const next = points[index + 1]
      ctx.quadraticCurveTo(
        current.x,
        current.y,
        (current.x + next.x) / 2,
        (current.y + next.y) / 2,
      )
    }
    const last = points[points.length - 1]
    ctx.lineTo(last.x, last.y)
  }

  ctx.stroke()
}
