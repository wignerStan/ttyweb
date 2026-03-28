export interface Project {
  id: string;
  name: string;
  path: string;
  createdAt: string;
  updatedAt: string;
}

export interface Worktree {
  id: string;
  projectId: string;
  branchName: string;
  path: string;
  isMain: boolean;
  headCommit?: string;
  headCommitMessage?: string;
  statusAhead: number;
  statusBehind: number;
  statusModified: number;
  statusStaged: number;
  statusUntracked: number;
  statusConflicts: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

export interface CreateProjectParams {
  path: string;
}

export interface CreateWorktreeParams {
  branchName: string;
  createBranch?: boolean;
  baseBranch?: string;
}
