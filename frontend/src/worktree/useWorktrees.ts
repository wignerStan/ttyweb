import { useState, useCallback } from 'react';
import { getAuthHeader } from '../utils/auth';
import type {
  Project,
  Worktree,
  ApiResponse,
  CreateProjectParams,
  CreateWorktreeParams,
} from './types';

function buildHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  const auth = getAuthHeader();
  if (auth) {
    headers['Authorization'] = auth;
  }
  return headers;
}

async function apiRequest<T>(
  url: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: {
      ...buildHeaders(),
      ...(options?.headers as Record<string, string> | undefined),
    },
  });
  const json: ApiResponse<T> = await res.json();
  if (!json.success) {
    throw new Error(json.error ?? `Request failed: ${res.status}`);
  }
  return json.data;
}

interface UseWorktreesState {
  projects: Project[];
  worktreesByProject: Record<string, Worktree[]>;
  loading: boolean;
  error: string | null;
}

interface UseWorktreesActions {
  fetchProjects: () => Promise<Project[]>;
  createProject: (params: CreateProjectParams) => Promise<Project>;
  deleteProject: (projectId: string) => Promise<void>;
  syncProject: (projectId: string) => Promise<void>;
  fetchWorktrees: (projectId: string) => Promise<Worktree[]>;
  createWorktree: (
    projectId: string,
    params: CreateWorktreeParams,
  ) => Promise<Worktree>;
  deleteWorktree: (
    projectId: string,
    worktreeId: string,
  ) => Promise<void>;
  refreshStatus: (
    projectId: string,
    worktreeId: string,
  ) => Promise<Worktree>;
  commitAll: (
    projectId: string,
    worktreeId: string,
    message: string,
  ) => Promise<void>;
  syncAll: (projectId: string) => Promise<void>;
  clearError: () => void;
}

export function useWorktrees(): UseWorktreesState & UseWorktreesActions {
  const [projects, setProjects] = useState<Project[]>([]);
  const [worktreesByProject, setWorktreesByProject] = useState<
    Record<string, Worktree[]>
  >({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => setError(null), []);

  const fetchProjects = useCallback(async (): Promise<Project[]> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<Project[]>('/api/projects');
      setProjects(data);
      return data;
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : 'Failed to fetch projects';
      setError(msg);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const createProject = useCallback(
    async (params: CreateProjectParams): Promise<Project> => {
      setLoading(true);
      setError(null);
      try {
        const data = await apiRequest<Project>('/api/projects', {
          method: 'POST',
          body: JSON.stringify(params),
        });
        setProjects((prev) => [...prev, data]);
        return data;
      } catch (err) {
        const msg =
          err instanceof Error ? err.message : 'Failed to create project';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const deleteProject = useCallback(
    async (projectId: string): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        await apiRequest<null>(`/api/projects/${projectId}`, {
          method: 'DELETE',
        });
        setProjects((prev) => prev.filter((p) => p.id !== projectId));
        setWorktreesByProject((prev) => {
          const next = { ...prev };
          delete next[projectId];
          return next;
        });
      } catch (err) {
        const msg =
          err instanceof Error ? err.message : 'Failed to delete project';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const syncProject = useCallback(
    async (projectId: string): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        await apiRequest<null>(`/api/projects/${projectId}/sync`, {
          method: 'POST',
        });
      } catch (err) {
        const msg =
          err instanceof Error ? err.message : 'Failed to sync project';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const fetchWorktrees = useCallback(
    async (projectId: string): Promise<Worktree[]> => {
      setLoading(true);
      setError(null);
      try {
        const data = await apiRequest<Worktree[]>(
          `/api/worktree/projects/${projectId}/worktrees`,
        );
        setWorktreesByProject((prev) => ({ ...prev, [projectId]: data }));
        return data;
      } catch (err) {
        const msg =
          err instanceof Error ? err.message : 'Failed to fetch worktrees';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const createWorktree = useCallback(
    async (
      projectId: string,
      params: CreateWorktreeParams,
    ): Promise<Worktree> => {
      setLoading(true);
      setError(null);
      try {
        const data = await apiRequest<Worktree>(
          `/api/worktree/projects/${projectId}/worktrees`,
          {
            method: 'POST',
            body: JSON.stringify(params),
          },
        );
        setWorktreesByProject((prev) => ({
          ...prev,
          [projectId]: [...(prev[projectId] ?? []), data],
        }));
        return data;
      } catch (err) {
        const msg =
          err instanceof Error
            ? err.message
            : 'Failed to create worktree';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const deleteWorktree = useCallback(
    async (
      projectId: string,
      worktreeId: string,
    ): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        await apiRequest<null>(
          `/api/worktree/projects/${projectId}/worktrees/${worktreeId}`,
          { method: 'DELETE' },
        );
        setWorktreesByProject((prev) => ({
          ...prev,
          [projectId]: (prev[projectId] ?? []).filter(
            (wt) => wt.id !== worktreeId,
          ),
        }));
      } catch (err) {
        const msg =
          err instanceof Error
            ? err.message
            : 'Failed to delete worktree';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const refreshStatus = useCallback(
    async (
      projectId: string,
      worktreeId: string,
    ): Promise<Worktree> => {
      setError(null);
      try {
        const data = await apiRequest<Worktree>(
          `/api/worktree/projects/${projectId}/worktrees/${worktreeId}/refresh`,
          { method: 'POST' },
        );
        setWorktreesByProject((prev) => ({
          ...prev,
          [projectId]: (prev[projectId] ?? []).map((wt) =>
            wt.id === worktreeId ? data : wt,
          ),
        }));
        return data;
      } catch (err) {
        const msg =
          err instanceof Error
            ? err.message
            : 'Failed to refresh status';
        setError(msg);
        throw err;
      }
    },
    [],
  );

  const commitAll = useCallback(
    async (
      projectId: string,
      worktreeId: string,
      message: string,
    ): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        await apiRequest<null>(
          `/api/worktree/projects/${projectId}/worktrees/${worktreeId}/commit`,
          {
            method: 'POST',
            body: JSON.stringify({ message }),
          },
        );
        // Refresh status after commit
        await refreshStatus(projectId, worktreeId);
      } catch (err) {
        const msg =
          err instanceof Error ? err.message : 'Failed to commit';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [refreshStatus],
  );

  const syncAll = useCallback(
    async (projectId: string): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        await apiRequest<null>(
          `/api/worktree/projects/${projectId}/worktrees/sync`,
          { method: 'POST' },
        );
        // Re-fetch worktrees to get updated state
        await fetchWorktrees(projectId);
      } catch (err) {
        const msg =
          err instanceof Error ? err.message : 'Failed to sync';
        setError(msg);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [fetchWorktrees],
  );

  return {
    projects,
    worktreesByProject,
    loading,
    error,
    fetchProjects,
    createProject,
    deleteProject,
    syncProject,
    fetchWorktrees,
    createWorktree,
    deleteWorktree,
    refreshStatus,
    commitAll,
    syncAll,
    clearError,
  };
}
