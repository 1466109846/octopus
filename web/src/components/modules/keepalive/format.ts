import type { ScheduledProbeRow } from '@/api/scheduled-probe';

// 本文件把一行保活结论说成界面上的几个短词: 状态、HTTP 状态码、耗时。
// 与监控那边的 format.ts 分开: 保活展示的是"上游回了什么、多快", 监控展示的是"通/不通 + 糖果",
// 两套说法共用一份工具只会让每个函数都长出 if (keepalive) 的分支。

// KeepaliveTone 是状态 chip 的色调。
// running(绿): 上一次保活打通; failed(红): 上一次保活没打通; pending(灰): 还没跑过; paused(灰): 任务已停用。
export type KeepaliveTone = 'running' | 'failed' | 'pending' | 'paused';

// keepaliveToneOf 决定一行的状态 chip。
//
// 停用优先于结论: 任务关了就不会再有新的心跳, 继续显示「保活中」会让人以为它还在跑;
// 结论(上一次的 HTTP 状态码与耗时)照常保留在旁边, 停用前最后一次打成什么样仍然可查。
export function keepaliveToneOf(row: ScheduledProbeRow, enabled: boolean): KeepaliveTone {
    if (!enabled) return 'paused';
    if (!row.probed) return 'pending';
    return row.ok ? 'running' : 'failed';
}

// TONE_CLASS 是 chip 的配色; 只在背景上着色、字色跟随, 与截图里的「保活中」同一副面孔。
export const TONE_CLASS: Record<KeepaliveTone, string> = {
    running: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    failed: 'bg-rose-500/10 text-rose-600 dark:text-rose-400',
    pending: 'bg-muted text-muted-foreground',
    paused: 'bg-muted text-muted-foreground',
};

// statusCodeClass 给 HTTP 状态码上色: 2xx 绿、429/5xx 琥珀(上游响应了但拒绝, 排队类场景的常态)、
// 其余 4xx 红(多半是配置问题: 鉴权、模型名)、0 灰(根本没拿到响应: 超时、网络、本地配置)。
//
// 429 与 5xx 单独一档而不是一律标红: 保活场景里"队列满"恰恰说明心跳已经打到了上游,
// 与"钥匙填错了"是两种完全不同的处境 —— 前者等一等就好, 后者要去改配置。
export function statusCodeClass(code: number): string {
    if (code >= 200 && code < 300) return 'text-emerald-600 dark:text-emerald-400';
    if (code === 429 || code >= 500) return 'text-amber-600 dark:text-amber-400';
    if (code >= 400) return 'text-rose-600 dark:text-rose-400';
    return 'text-muted-foreground';
}

// formatLatencySeconds 把毫秒耗时说成"4.32 秒"这样的两位小数秒数。
// 用秒而不是毫秒: 保活的耗时动辄数秒(排队时更长), "4320ms" 要人自己换算, 秒数一眼可读。
export function formatLatencySeconds(latencyMS: number, t: (key: string, values?: Record<string, string | number>) => string): string {
    return t('seconds', { value: (Math.max(0, latencyMS) / 1000).toFixed(2) });
}
