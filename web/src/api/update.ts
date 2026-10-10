import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from './client';
import type { BuildInfo } from '@/lib/build-info';

export function useBuildInfo() {
    return useQuery({
        queryKey: ['update', 'build-info'],
        queryFn: () => apiRequest<BuildInfo>('/api/v1/update/build-info'),
        refetchInterval: 300000,
        refetchOnMount: 'always',
    });
}

/**
 * 后端 /api/v1/update 返回的最新发布信息
 */
interface LatestInfo {
    tag_name: string;
    published_at: string;
    body: string;
    message: string;
}

/**
 * 获取最新发布信息 Hook
 * 
 * @example
 * const { data: latestInfo, isLoading, error } = useLatestInfo();
 * 
 * if (isLoading) return <Loading />;
 * if (error) return <Error message={error.message} />;
 * 
 * console.log('Latest tag:', latestInfo?.tag_name);
 */
export function useLatestInfo() {
    return useQuery({
        queryKey: ['update', 'latest'],
        queryFn: () => apiRequest<LatestInfo>('/api/v1/update'),
        refetchInterval: 300000, // 5 分钟
        refetchOnMount: 'always',
    });
}

/**
 * 获取后端当前版本 Hook
 *
 * 后端: GET /api/v1/update/now-version -> string
 */
export function useNowVersion() {
    return useQuery({
        queryKey: ['update', 'now-version'],
        queryFn: () => apiRequest<string>('/api/v1/update/now-version'),
        refetchInterval: 300000, // 5 分钟
        refetchOnMount: 'always',
    });
}

/**
 * 执行更新 Hook
 * 
 * @example
 * const updateCore = useUpdateCore();
 * 
 * updateCore.mutate(undefined, {
 *   onSuccess: () => {
 *     console.log('Update started successfully');
 *   },
 * });
 */
export function useUpdateCore() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: () => apiRequest<string>('/api/v1/update', { method: 'POST' }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['update', 'latest'] });
            queryClient.invalidateQueries({ queryKey: ['update', 'now-version'] });
            // 更新替换的就是二进制本身, 版本/仓库/提交这些构建元数据都会变, 一起失效重拉。
            queryClient.invalidateQueries({ queryKey: ['update', 'build-info'] });
        },
    });
}
