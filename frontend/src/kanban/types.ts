export type KanbanStatus = 'todo' | 'in_progress' | 'done' | 'archived';

export interface KanbanTask {
  id: string;
  title: string;
  description: string;
  status: KanbanStatus;
  priority: number;
  tags: string[];
  due_date: string | null;
  order_index: number;
  created_at: string;
  updated_at: string;
}

export interface KanbanComment {
  id: string;
  task_id: string;
  content: string;
  created_at: string;
}

export interface KanbanApiResponse<T> {
  success: boolean;
  data: T;
}
