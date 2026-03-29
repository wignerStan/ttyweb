import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { Plus } from 'lucide-react'
import type { ReactNode } from 'react'
import { TaskCard } from './TaskCard'
import type { KanbanStatus, KanbanTask } from './types'

interface TaskColumnProps {
  status: KanbanStatus
  title: string
  tasks: KanbanTask[]
  onSelectTask: (task: KanbanTask) => void
  onAddTask: (status: KanbanStatus) => void
}

function DroppableColumn({ status, children }: { status: KanbanStatus; children: ReactNode }) {
  const { setNodeRef, isOver } = useDroppable({ id: status })
  return (
    <div ref={setNodeRef} className={`task-column ${isOver ? 'task-column--over' : ''}`}>
      {children}
    </div>
  )
}

function TaskColumn({ status, title, tasks, onSelectTask, onAddTask }: TaskColumnProps) {
  const taskIds = tasks.map((t) => t.id)

  return (
    <DroppableColumn status={status}>
      <div className="task-column__header">
        <div className="task-column__title-group">
          <h3 className="task-column__title">{title}</h3>
          <span className="task-column__badge">{tasks.length}</span>
        </div>
        <button
          className="task-column__add-btn"
          onClick={() => onAddTask(status)}
          title={`Add task to ${title}`}
          type="button"
        >
          <Plus size={14} />
        </button>
      </div>

      <div className="task-column__body">
        <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} onSelect={onSelectTask} />
          ))}
        </SortableContext>

        {tasks.length === 0 && <div className="task-column__placeholder">Drop tasks here</div>}
      </div>
    </DroppableColumn>
  )
}

export type { TaskColumnProps }
export { TaskColumn }
