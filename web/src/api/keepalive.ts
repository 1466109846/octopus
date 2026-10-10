import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from './client';
import {
    scheduledProbeListQueryOptions,
    type ScheduledProbe,
    type ScheduledProbeInput,
    type ScheduledProbeRow,
} from './scheduled-probe';

// 保活（keepalive）与模型监控共用同一张任务表、同一套后端接口，靠 kind 字段分流：
// 本模块只负责"把保活那一份挑出来、把保活的两个手动入口接上", 任务 CRUD 完全复用监控的 hooks。

// KEEPALIVE_KIND 与后端 model.ScheduledProbeKindKeepalive 一致。
export const KEEPALIVE_KIND = 'keepalive';

// isKeepalive 判断一条任务是不是保活任务。
// 缺省（undefined/空串）按监控处理：与后端 normalizeProbeKind 同一口径 —— 存量行与老请求体
// 都不带这个字段，落回 monitor 是唯一安全的解释。
export function isKeepalive(probe: ScheduledProbe): boolean {
    return probe.kind === KEEPALIVE_KIND;
}

// keepaliveTasksOf 从全部任务里挑出保活任务；监控页各挑自己那一份，两边不会互相串台。
export function keepaliveTasksOf(probes: ScheduledProbe[]): ScheduledProbe[] {
    return probes.filter(isKeepalive);
}

// useKeepaliveList 读取全部保活任务。
//
// 与监控列表共用同一个查询定义（同 key、同接口、同轮询）：后端返回的是全部任务，
// 两页各自在客户端过滤。分开拉两次相同的数据没有意义，而共用一份缓存让
// "在监控页删了一条任务"这类失效动作对两个页面同时生效。
export function useKeepaliveList() {
    const query = useQuery(scheduledProbeListQueryOptions);
    return { ...query, data: query.data ? keepaliveTasksOf(query.data) : query.data };
}

// KEEPALIVE_DEFAULT_INTERVAL 与后端 model.KeepaliveDefaultIntervalMinutes 一致。
// 比监控的 10 分钟紧一档：排队场景下队列掉出的速度通常比通道变坏快。
export const KEEPALIVE_DEFAULT_INTERVAL = 5;

// keepaliveInputOf 把保活特有的缺省值补进提交体：保活任务固定全天可测、跳过糖果。
//
// 保活的请求就是最小成本的通信测试话术，糖果题的回答再短也大得多，与"只花最少流量维持活跃"相悖；
// 时间窗同理 —— 保活要的是"一直有请求打进上游"，设了时段就有了空窗，队列位反而保不住。
// 后端对保活任务已强制关了糖果（见 ScheduledProbe.IQProbeEnabled），这里显式带上是为了
// 让提交体与实际语义一致，接口层面看不出"传了但被忽略"的歧义。
export function keepaliveInputOf(values: {
    name: string;
    targets: ScheduledProbeInput['targets'];
    interval_minutes: number;
    enabled: boolean;
}): ScheduledProbeInput {
    return {
        name: values.name,
        kind: KEEPALIVE_KIND,
        targets: values.targets,
        interval_minutes: values.interval_minutes,
        enabled: values.enabled,
        iq_disabled: true,
        weekdays: 0,
        start_hour: 0,
        end_hour: 0,
    };
}

// useKeepaliveScheduledNow 手动把一条保活任务的每条凭据都保活一遍。
// 返回整行结论（形状与监控的行一致），供调用方逐条汇报。
export function useKeepaliveScheduledNow() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: number) =>
            apiRequest<ScheduledProbeRow[]>(`/api/v1/scheduled-probe/keepalive/${id}`, { method: 'POST', body: {} }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: scheduledProbeListQueryOptions.queryKey }),
    });
}

// useKeepaliveGrantNow 只对一条凭据做一次保活，供保活页一行末尾的心跳按钮使用。
// 必须传一个空对象当 body 而不是省略它：省略后 apiRequest 不会设 Content-Type，
// 后端 RequireJSON 会以 415 拒绝 —— 按钮就会永远点不动（与监控的同类 hook 同一原因）。
export function useKeepaliveGrantNow() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (grantId: number) =>
            apiRequest<ScheduledProbeRow>(`/api/v1/scheduled-probe/keepalive-grant/${grantId}`, { method: 'POST', body: {} }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: scheduledProbeListQueryOptions.queryKey }),
    });
}
