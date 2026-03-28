import { useState, useCallback, useMemo } from 'react';
import {
  DndContext,
  DragOverlay,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
  type UniqueIdentifier,
} from '@dnd-kit/core';
import { Plus, RefreshCw, Inbox } from 'lucide-react';
import { useKanbanTasks } from './useKanbanTasks';
import { TaskColumn } from './TaskColumn';
import { TaskCard } from './TaskCard';
import { TaskDialog } from './TaskDialog';
import type { KanbanTask, KanbanStatus } from './types';
import type { TaskFormData } from './TaskDialog';
import './kanban.css';

interface ColumnConfig {
  key: KanbanStatus;
  title: string;
}

const COLUMNS: ColumnConfig[] = [
  { key: 'todo', title: 'Todo' },
  { key: 'in_progress', title: 'In Progress' },
  { key: 'done', title: 'Done' },
  { key: 'archived', title: 'Archived' },
];

function KanbanBoard() {
  const {
    tasks,
    loading,
    error,
    fetchTasks,
    createTask,
    updateTask,
    moveTask,
    deleteTask,
    fetchComments,
    createComment,
  } = useKanbanTasks();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<KanbanTask | null>(null);
  const [defaultStatus, setDefaultStatus] = useState<KanbanStatus>('todo');
  const [activeId, setActiveId] = useState<UniqueIdentifier | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
  );

  const tasksByStatus = useMemo(() => {
    const map: Record<KanbanStatus, KanbanTask[]> = {
      todo: [],
      in_progress: [],
      done: [],
      archived: [],
    };
    for (const task of tasks) {
      map[task.status].push(task);
    }
    for (const key of Object.keys(map) as KanbanStatus[]) {
      map[key].sort((a, b) => a.order_index - b.order_index);
    }
    return map;
  }, [tasks]);

  const activeTask = useMemo(
    () => (activeId ? tasks.find((t) => t.id === activeId) ?? null : null),
    [activeId, tasks],
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveId(event.active.id);
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveId(null);
      const { active, over } = event;
      if (!over) return;

      const taskId = active.id as string;
      const overId = over.id as string;

      // Determine target column: dropped on column header or on another task
      const columnStatus = COLUMNS.find((col) => col.key === overId)?.key;
      const overTask = tasks.find((t) => t.id === overId);
      const targetStatus = (columnStatus ?? overTask?.status) as KanbanStatus | undefined;

      if (!targetStatus) return;

      const currentTask = tasks.find((t) => t.id === taskId);
      if (!currentTask) return;

      // Same column and position — skip
      if (currentTask.status === targetStatus && currentTask.id === overId) return;

      // Calculate order index
      const targetColumnTasks = tasksByStatus[targetStatus] ?? [];
      let newIndex = 0;

      if (targetStatus === currentTask.status) {
        // Reordering within same column
        const currentIndex = targetColumnTasks.findIndex((t) => t.id === taskId);
        const overIndex = targetColumnTasks.findIndex((t) => t.id === overId);
        if (currentIndex >= 0 && overIndex >= 0 && currentIndex !== overIndex) {
          newIndex = overIndex;
        } else {
          newIndex = currentIndex >= 0 ? currentIndex : 0;
        }
      } else {
        // Moving to different column
        const overIndex = targetColumnTasks.findIndex((t) => t.id === overId);
        newIndex = overIndex >= 0 ? overIndex : targetColumnTasks.length;
      }

      let orderIndex = 1000;

      if (newIndex === 0 && targetColumnTasks.length > 0) {
        orderIndex = targetColumnTasks[0]?.order_index != null
          ? targetColumnTasks[0].order_index - 1000
          : 1000;
      } else if (newIndex >= targetColumnTasks.length && targetColumnTasks.length > 0) {
        const last = targetColumnTasks[targetColumnTasks.length - 1];
        orderIndex = (last?.order_index ?? 0) + 1000;
      } else if (newIndex > 0 && newIndex < targetColumnTasks.length) {
        const prev = targetColumnTasks[newIndex - 1];
        const next = targetColumnTasks[newIndex];
        if (prev && next) {
          orderIndex = Math.round((prev.order_index + next.order_index) / 2);
        } else {
          orderIndex = prev?.order_index != null ? prev.order_index + 1000 : 1000;
        }
      }

      void moveTask(taskId, targetStatus, orderIndex);
    },
    [tasks, tasksByStatus, moveTask],
  );

  const handleSelectTask = useCallback((task: KanbanTask) => {
    setEditingTask(task);
    setDefaultStatus(task.status);
    setDialogOpen(true);
  }, []);

  const handleAddTask = useCallback((status: KanbanStatus) => {
    setEditingTask(null);
    setDefaultStatus(status);
    setDialogOpen(true);
  }, []);

  const handleDialogClose = useCallback(() => {
    setDialogOpen(false);
    setEditingTask(null);
  }, []);

  const handleCreate = useCallback(
    async (fields: Partial<TaskFormData>) => {
      return await createTask({
        title: fields.title ?? '',
        description: fields.description ?? '',
        status: fields.status ?? defaultStatus,
        priority: fields.priority,
        tags: fields.tags,
        due_date: fields.due_date,
      });
    },
    [createTask, defaultStatus],
  );

  const handleSave = useCallback(
    async (fields: TaskFormData) => {
      if (!editingTask) return null;
      return await updateTask(editingTask.id, {
        title: fields.title,
        description: fields.description,
        status: fields.status,
        priority: fields.priority,
        tags: fields.tags,
        due_date: fields.due_date,
      });
    },
    [editingTask, updateTask],
  );

  if (error) {
    return (
      <div className="kanban-board">
        <div className="kanban-board__empty">
          <p style={{ color: 'var(--kanban-danger)' }}>{error}</p>
          <button
            className="kanban-board__btn"
            onClick={() => void fetchTasks()}
            type="button"
          >
            <RefreshCw size={13} />
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="kanban-board">
      <div className="kanban-board__header">
        <div>
          <h2 className="kanban-board__title">Kanban Board</h2>
          <div className="kanban-board__subtitle">
            Drag and drop tasks between columns
          </div>
        </div>
        <div className="kanban-board__actions">
          <button
            className="kanban-board__btn kanban-board__btn--icon-only"
            onClick={() => void fetchTasks()}
            disabled={loading}
            title="Refresh tasks"
            type="button"
          >
            <RefreshCw size={14} className={loading ? 'status-icon--spinning' : ''} />
          </button>
          <button
            className="kanban-board__btn"
            onClick={() => handleAddTask('todo')}
            type="button"
          >
            <Plus size={14} />
            Add Task
          </button>
        </div>
      </div>

      <div className="kanban-board__body">
        {loading && tasks.length === 0 ? (
          <div className="kanban-board__empty">
            <RefreshCw size={24} className="status-icon--spinning" />
            <span>Loading tasks...</span>
          </div>
        ) : tasks.length === 0 ? (
          <div className="kanban-board__empty" style={{ gridColumn: '1 / -1' }}>
            <Inbox size={32} />
            <span>No tasks yet. Create one to get started.</span>
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCorners}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
          >
            {COLUMNS.map((col) => (
              <TaskColumn
                key={col.key}
                status={col.key}
                title={col.title}
                tasks={tasksByStatus[col.key]}
                onSelectTask={handleSelectTask}
                onAddTask={handleAddTask}
              />
            ))}

            <DragOverlay>
              {activeTask ? (
                <div className="kanban-drag-overlay">
                  <TaskCard task={activeTask} onSelect={() => {}} />
                </div>
              ) : null}
            </DragOverlay>
          </DndContext>
        )}
      </div>

      <TaskDialog
        open={dialogOpen}
        task={editingTask}
        defaultStatus={defaultStatus}
        onSave={handleSave}
        onCreate={handleCreate}
        onDelete={deleteTask}
        onClose={handleDialogClose}
        fetchComments={fetchComments}
        createComment={createComment}
      />
    </div>
  );
}

export { KanbanBoard };
