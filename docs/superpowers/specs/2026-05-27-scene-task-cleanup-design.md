# 场景图生成任务超时自动清理设计方案

## 背景

场景图生成任务（`scene_generation_tasks`）在 `running` 状态下可能因网络问题、API 超时等原因卡住，长时间无法完成。需要自动清理超过30分钟仍处于 `running` 状态的任务。

## 目标

- 自动清理超过30分钟未完成（`running` 状态）的场景图生成任务
- 同时清理关联的 `SceneGenerationImage` 记录
- 不影响已完成（`done`）或失败（`failed`）的任务

## 实现方案

### 方案A：启动时清理 + 定时清理

#### 实现位置
`backend/internal/handler/scene.go`

#### 清理逻辑

1. **清理函数 `cleanupExpiredSceneTasks()`**
   - 查询条件：`status = 'running' AND updated_at < now - 30分钟`
   - 级联删除关联的 `SceneGenerationImage` 记录
   - 删除超时的 `SceneGenerationTask` 记录

2. **启动时机**
   - 应用启动时立即执行一次清理（处理历史超时任务）
   - 启动 goroutine 每 1 分钟执行一次清理（处理新增超时任务）

#### 代码设计

```go
const sceneTaskCleanupInterval = 1 * time.Minute
const sceneTaskCleanupTimeout = 30 * time.Minute

// cleanupExpiredSceneTasks 清理超时的场景图生成任务
func cleanupExpiredSceneTasks() {
    cutoff := time.Now().Add(-sceneTaskCleanupTimeout)

    var tasks []model.SceneGenerationTask
    if err := database.DB.Where("status = ? AND updated_at < ?", "running", cutoff).Find(&tasks).Error; err != nil {
        log.Printf("[cleanup] 查询超时任务失败: %v", err)
        return
    }

    if len(tasks) == 0 {
        return
    }

    log.Printf("[cleanup] 发现 %d 个超时任务，开始清理...", len(tasks))

    for _, task := range tasks {
        tx := database.DB.Begin()

        // 删除关联图片
        if err := tx.Where("task_id_ref = ?", task.TaskID).Delete(&model.SceneGenerationImage{}).Error; err != nil {
            tx.Rollback()
            log.Printf("[cleanup] 删除任务 %s 的关联图片失败: %v", task.TaskID, err)
            continue
        }

        // 删除任务
        if err := tx.Delete(&task).Error; err != nil {
            tx.Rollback()
            log.Printf("[cleanup] 删除任务 %s 失败: %v", task.TaskID, err)
            continue
        }

        tx.Commit()
        log.Printf("[cleanup] 已清理超时任务: %s", task.TaskID)
    }

    log.Printf("[cleanup] 本次清理完成，共清理 %d 个任务", len(tasks))
}

// startSceneTaskCleanup 启动定时清理
func startSceneTaskCleanup() {
    // 启动时立即执行一次
    cleanupExpiredSceneTasks()

    // 定时执行
    ticker := time.NewTicker(sceneTaskCleanupInterval)
    go func() {
        for range ticker.C {
            cleanupExpiredSceneTasks()
        }
    }()
}
```

#### 启动调用

在 `backend/main.go` 中应用启动时调用：

```go
// 启动场景任务超时清理
handler.StartSceneTaskCleanup()
```

## 注意事项

1. **不影响已完成任务**：只清理 `running` 状态，不影响 `done`、`failed`、`queued` 状态
2. **事务保护**：删除任务和关联图片使用事务，确保数据一致性
3. **日志记录**：便于排查问题和监控清理效果
4. **无需资源回滚**：清理任务状态时不需要执行额外的资源回滚

## 影响范围

- 修改文件：`backend/internal/handler/scene.go`
- 修改文件：`backend/main.go`
- 新增逻辑：清理函数和定时任务启动

## 测试计划

1. 创建任务后手动修改数据库状态为 `running`，验证30分钟后被清理
2. 检查清理日志输出正确
3. 检查关联图片同时被删除
