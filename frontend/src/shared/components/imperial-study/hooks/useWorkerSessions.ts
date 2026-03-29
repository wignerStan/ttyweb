import { useState, useEffect, useCallback } from 'react';
import type { WorkerSession } from '../types';
import { POLL_WORKERS_MS, BUTLER_API_BASE } from '../constants';
import { getAuthHeader } from '../../../../utils/auth';

function toError(e: unknown): Error {
    return e instanceof Error ? e : new Error(String(e));
}

export function useWorkerSessions(studyId?: string) {
    const [workers, setWorkers] = useState<WorkerSession[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const refetch = useCallback(async () => {
        setLoading(true);
        try {
            const url = `${BUTLER_API_BASE}/worker_sessions${studyId ? `?study_id=${studyId}` : ''}`;
            const authHeader = getAuthHeader();
            const res = await fetch(url, {
                headers: authHeader ? { 'Authorization': authHeader } : undefined,
            });
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const json = await res.json();
            const data: WorkerSession[] = json?.data?.worker_sessions ?? [];
            setWorkers(data);
            setError(null);
        } catch (e: unknown) {
            setError(toError(e));
        } finally {
            setLoading(false);
        }
    }, [studyId]);

    useEffect(() => {
        refetch();
        const interval = setInterval(refetch, POLL_WORKERS_MS);
        return () => clearInterval(interval);
    }, [refetch]);

    return { workers, loading, error, refetch };
}
