import { useState, useEffect, useCallback } from 'react';
import {
  Plus,
  FolderOpen,
  Trash2,
  ChevronRight,
  ChevronDown,
  RefreshCw,
  Search,
  ArrowLeftRight,
} from 'lucide-react';
import type { Project } from './types';
import { useWorktrees } from './useWorktrees';
import { WorktreeList } from './WorktreeList';
import './worktree.css';

export function ProjectList() {
  const {
    projects,
    worktreesByProject,
    loading,
    error,
    fetchProjects,
    createProject,
    deleteProject,
    fetchWorktrees,
    createWorktree,
    deleteWorktree,
    refreshStatus,
    commitAll,
    syncAll,
    clearError,
  } = useWorktrees();

  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createPath, setCreatePath] = useState('');

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const handleToggle = useCallback(
    async (project: Project) => {
      if (expandedId === project.id) {
        setExpandedId(null);
      } else {
        setExpandedId(project.id);
        try {
          await fetchWorktrees(project.id);
        } catch {
          // Error handled by hook
        }
      }
    },
    [expandedId, fetchWorktrees],
  );

  const handleCreateProject = useCallback(async () => {
    if (!createPath.trim()) return;
    try {
      await createProject({ path: createPath.trim() });
      setCreatePath('');
      setShowCreateForm(false);
    } catch {
      // Error handled by hook
    }
  }, [createPath, createProject]);

  const handleDeleteProject = useCallback(
    async (project: Project) => {
      const confirmed = window.confirm(
        `Delete project "${project.name}"? This will also remove all associated worktrees.`,
      );
      if (!confirmed) return;
      try {
        await deleteProject(project.id);
        if (expandedId === project.id) {
          setExpandedId(null);
        }
      } catch {
        // Error handled by hook
      }
    },
    [deleteProject, expandedId],
  );

  const handleSyncProject = useCallback(
    async (projectId: string) => {
      try {
        await syncAll(projectId);
      } catch {
        // Error handled by hook
      }
    },
    [syncAll],
  );

  const filteredProjects = projects.filter((p) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      p.name.toLowerCase().includes(q) ||
      p.path.toLowerCase().includes(q)
    );
  });

  return (
    <div className="pl-container">
      <div className="pl-header">
        <div className="pl-header-row">
          <h2 className="pl-title">Projects</h2>
          <div className="pl-header-actions">
            <button
              className="wt-icon-btn"
              title="Refresh projects"
              onClick={() => fetchProjects()}
              disabled={loading}
            >
              <RefreshCw size={14} />
            </button>
            <button
              className="wt-btn wt-btn-primary"
              onClick={() => setShowCreateForm(!showCreateForm)}
            >
              <Plus size={14} />
              Add Project
            </button>
          </div>
        </div>

        <div className="pl-search">
          <Search size={14} />
          <input
            className="pl-search-input"
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search projects..."
          />
        </div>
      </div>

      {showCreateForm && (
        <div className="pl-create-form">
          <div className="pl-form-row">
            <input
              className="wt-form-input"
              type="text"
              value={createPath}
              onChange={(e) => setCreatePath(e.target.value)}
              placeholder="/path/to/project"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateProject();
              }}
            />
          </div>
          <div className="pl-form-actions">
            <button
              className="wt-btn wt-btn-secondary wt-btn-sm"
              onClick={() => setShowCreateForm(false)}
            >
              Cancel
            </button>
            <button
              className="wt-btn wt-btn-primary wt-btn-sm"
              onClick={handleCreateProject}
              disabled={loading || !createPath.trim()}
            >
              Create
            </button>
          </div>
        </div>
      )}

      {error && (
        <div className="pl-error">
          <span>{error}</span>
          <button className="pl-error-dismiss" onClick={clearError}>
            &times;
          </button>
        </div>
      )}

      <div className="pl-list">
        {filteredProjects.length === 0 && !loading && (
          <div className="pl-empty">
            {searchQuery
              ? 'No projects match your search'
              : 'No projects yet. Add one to get started.'}
          </div>
        )}

        {filteredProjects.map((project) => {
          const isExpanded = expandedId === project.id;
          const worktrees = worktreesByProject[project.id] ?? [];

          return (
            <div key={project.id} className="pl-project">
              <div
                className="pl-project-card"
                onClick={() => handleToggle(project)}
              >
                <div className="pl-project-left">
                  <span className="pl-expand-icon">
                    {isExpanded ? (
                      <ChevronDown size={14} />
                    ) : (
                      <ChevronRight size={14} />
                    )}
                  </span>
                  <FolderOpen size={16} />
                  <div className="pl-project-info">
                    <span className="pl-project-name">{project.name}</span>
                    <span className="pl-project-path" title={project.path}>
                      {project.path}
                    </span>
                  </div>
                </div>
                <div className="pl-project-meta">
                  <span className="pl-wt-count">{worktrees.length} wt</span>
                  <button
                    className="wt-icon-btn"
                    title="Sync project"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleSyncProject(project.id);
                    }}
                    disabled={loading}
                  >
                    <ArrowLeftRight size={13} />
                  </button>
                  <button
                    className="wt-icon-btn wt-icon-btn-danger"
                    title="Delete project"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteProject(project);
                    }}
                    disabled={loading}
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>

              {isExpanded && (
                <div className="pl-worktrees">
                  <WorktreeList
                    project={project}
                    worktrees={worktrees}
                    loading={loading}
                    onCreate={(params) => {
                      createWorktree(project.id, params);
                    }}
                    onDelete={(wtId) => deleteWorktree(project.id, wtId)}
                    onRefreshStatus={(wtId) => {
                      refreshStatus(project.id, wtId);
                    }}
                    onCommitAll={(wtId, message) =>
                      commitAll(project.id, wtId, message)
                    }
                    onSyncAll={() => syncAll(project.id)}
                    onRefreshWorktrees={() => {
                      fetchWorktrees(project.id);
                    }}
                  />
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
