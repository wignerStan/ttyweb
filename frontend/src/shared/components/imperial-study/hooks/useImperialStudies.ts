import { useState, useEffect, useCallback } from 'react';
import type { ImperialStudy } from '../types';
import { BUTLER_API_BASE } from '../constants';
import { getAuthHeader } from '../../../../utils/auth';

function toError(e: unknown): Error {
    return e instanceof Error ? e : new Error(String(e));
}

export function useImperialStudies() {
    const [studies, setStudies] = useState<ImperialStudy[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const refetch = useCallback(async () => {
        setLoading(true);
        try {
            const authHeader = getAuthHeader();
            const res = await fetch(`${BUTLER_API_BASE}/imperial_studies?status=active`, {
                headers: authHeader ? { 'Authorization': authHeader } : undefined,
            });
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const json = await res.json();
            const data: ImperialStudy[] = json?.data?.imperial_studies ?? [];
            setStudies(data);
            setError(null);
        } catch (e: unknown) {
            setError(toError(e));
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        refetch();
    }, [refetch]);

    return { studies, loading, error, refetch };
}
