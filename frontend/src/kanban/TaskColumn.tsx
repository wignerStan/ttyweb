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
    <div
      ref={setNodeRef}
      className={`flex min-h-0 flex-col overflow-hidden rounded-box border border-base-200 bg-base-300 ${
        isOver ? 'task-column--over' : ''
      }`}
    >
      {children}
    </div>
  )
}

function TaskColumn({ status, title, tasks, onSelectTask, onAddTask }: TaskColumnProps) {
  const taskIds = tasks.map((t) => t.id)

  return (
    <DroppableColumn status={status}>
      <div className="flex flex-shrink-0 items-center justify-between border-b border-base-200 bg-base-200 rounded-t-box px-3.5 py-2.5">
        <div className="flex items-center gap-2">
          <h3 className="m-0 text-[13px] font-semibold text-base-content">{title}</h3>
          <span className="badge badge-sm badge-ghost">{tasks.length}</span>
        </div>
        <button
          className="inline-flex h-6 w-6 items-center justify-center rounded text-base-content/50 transition-colors hover:bg-base-200 hover:text-base-content"
          onClick={() => onAddTask(status)}
          title={`Add task to ${title}`}
          type="button"
        >
          <Plus size={14} />
        </button>
      </div>

      <div className="flex min-h-[60px] flex-col gap-2 overflow-y-auto p-2 rounded-b-box">
        <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} onSelect={onSelectTask} />
          ))}
        </SortableContext>

        {tasks.length === 0 && (
          <div className="flex flex-1 min-h-[48px] items-center justify-center rounded-md border-2 border-dashed border-base-300 text-[11px] text-base-content/50">
            Drop tasks here
          </div>
        )}
      </div>
    </DroppableColumn>
  )
}

export type { TaskColumnProps }
export { TaskColumn }
