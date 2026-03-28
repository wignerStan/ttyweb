export interface Project {
  id: string;
  name: string;
  path: string;
  description?: string;
  default_branch?: string;
  worktree_base_path?: string;
  remote_url?: string;
  last_sync_at?: string;
}

export interface Worktree {
  id: string;
  project_id: string;
  branch_name: string;
  path: string;
  is_main: boolean;
  head_commit?: string;
  head_commit_message?: string;
  status_ahead: number;
  status_behind: number;
  status_modified: number;
  status_staged: number;
  status_untracked: number;
  status_conflicts: number;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

export interface CreateProjectParams {
  name: string;
  path: string;
  description?: string;
}

export interface CreateWorktreeParams {
  branch_name: string;
  create_branch?: boolean;
  base_branch?: string;
}
