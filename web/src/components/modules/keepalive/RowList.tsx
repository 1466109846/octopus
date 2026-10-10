import { HeartPulse, LoaderCircle, X } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { toast } from 'sonner';
import { useKeepaliveGrantNow } from '@/api/keepalive';
import { useSetScheduledProbeCredential, type ScheduledProbeRow } from '@/api/scheduled-probe';
import { IconButton } from '@/components/common/IconButton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { describeProbedAt } from '@/components/modules/scheduled-probe/format';
import { TONE_CLASS, formatLatencySeconds, keepaliveToneOf, statusCodeClass } from './format';

// RowList 逐行列出一条保活任务的各条凭据: 一行 = 一条凭据。
//
// 与监控页的行不同, 这里没有拖动排序: 保活关心的是"每条凭据都持续有心跳", 轮转次序对它没有意义;
// 少一个握把, 窄卡片里也就多留出一段位置给状态码与耗时 —— 那才是保活这一页要读的信息。
export function RowList({ probeId, rows, enabled, now }: { probeId: number; rows: ScheduledProbeRow[]; enabled: boolean; now: number }) {
    const t = useTranslations('keepalive');

    if (rows.length === 0) {
        return <p className="px-1 py-2 text-xs text-muted-foreground/70">{t('noTarget')}</p>;
    }

    return (
        <ul className="flex flex-col gap-1">
            {rows.map((row) => (
                <Row key={row.grant_id} probeId={probeId} row={row} enabled={enabled} now={now} />
            ))}
        </ul>
    );
}

// Row 渲染一条凭据: 左侧是「渠道名/凭据名/模型名」, 右侧依次是 状态 chip、HTTP 状态码 + 耗时、
// 多久以前, 以及行尾的心跳(只保活这一条)与 ×(不再保活这一条)。
function Row({ probeId, row, enabled, now }: { probeId: number; row: ScheduledProbeRow; enabled: boolean; now: number }) {
    const t = useTranslations('keepalive');
    const keepaliveGrant = useKeepaliveGrantNow();
    const setCredential = useSetScheduledProbeCredential();
    const label = row.key_name || `#${row.grant_id}`;
    const tone = keepaliveToneOf(row, enabled);

    const handleKeepalive = () => {
        keepaliveGrant.mutate(row.grant_id, {
            onSuccess: (verdict) => {
                if (verdict.ok) {
                    toast.success(t('toast.grantDone', { status: verdict.status_code || '-' }));
                    return;
                }
                // 失败也说清状态码: 429/503 是"上游响应了但拒绝", 与连不上是两回事。
                toast.error(t('toast.grantFailed', { message: verdict.message || t('unknownError') }));
            },
            onError: (error) => toast.error(error.message),
        });
    };

    // 删除不再确认：这个动作只影响当前凭据行，不会删除渠道本身的授权（与监控页同一口径）。
    const handleRemove = () => {
        setCredential.mutate(
            {
                id: probeId,
                channelId: row.channel_id,
                modelName: row.model_name,
                keyName: row.key_name,
                excluded: true,
            },
            {
                onSuccess: () => toast.success(t('toast.credentialRemoved')),
                onError: (error) => toast.error(error.message),
            },
        );
    };

    // 悬停提示给出完整结论: 正文只放短词, 错误正文(可能很长)只在这里出现。
    const detail = !row.probed
        ? t('notProbed')
        : row.ok
            ? t('detailOk', { status: row.status_code || '-', latency: formatLatencySeconds(row.latency_ms, t) })
            : t('detailFailed', { status: row.status_code || '-', message: row.message || t('unknownError') });

    return (
        <li className="flex items-center gap-1.5 rounded-lg border border-border/60 px-1.5 py-1">
            {/* 三段身份与监控页同一副面孔: 渠道名与凭据名保住(shrink-0), 模型名最先被省略。 */}
            <Tooltip>
                <TooltipTrigger asChild>
                    <span className="flex min-w-0 flex-1 items-center overflow-hidden text-sm">
                        <span className="shrink-0 text-foreground">{row.channel_name || `#${row.channel_id}`}</span>
                        <span className="shrink-0 text-muted-foreground/50">/</span>
                        <span className="shrink-0 text-foreground">{label}</span>
                        {row.model_name && (
                            <>
                                <span className="shrink-0 text-muted-foreground/50">/</span>
                                <span className="min-w-0 truncate text-muted-foreground">{row.model_name}</span>
                            </>
                        )}
                    </span>
                </TooltipTrigger>
                <TooltipContent side="top" sideOffset={10} align="start">
                    {`${row.channel_name || `#${row.channel_id}`} / ${label}`}
                    {row.model_name ? ` / ${row.model_name}` : ''}
                </TooltipContent>
            </Tooltip>

            {/* 状态 chip 与「HTTP 200 / 4.32 秒」两行: 截图里的那一组, 结论一眼可读。 */}
            <Tooltip>
                <TooltipTrigger asChild>
                    <span className="flex shrink-0 items-center gap-1.5">
                        <span className={cn('rounded-md px-1.5 py-0.5 text-[11px] font-medium leading-none', TONE_CLASS[tone])}>
                            {t(`status.${tone}`)}
                        </span>
                        {row.probed && (
                            <span className="flex flex-col items-end leading-tight tabular-nums">
                                <span className={cn('text-[11px] font-medium', statusCodeClass(row.status_code))}>
                                    {row.status_code > 0 ? `HTTP ${row.status_code}` : t('noStatus')}
                                </span>
                                <span className="text-[10px] text-muted-foreground/70">
                                    {formatLatencySeconds(row.latency_ms, t)}
                                </span>
                            </span>
                        )}
                    </span>
                </TooltipTrigger>
                <TooltipContent side="top" sideOffset={10} align="end" className="max-w-80 break-all">
                    {detail}
                </TooltipContent>
            </Tooltip>

            {/* 多久以前留在正文: 只看颜色分不出"刚才的心跳"与"一小时前的心跳"。 */}
            {row.probed && (
                <span className="shrink-0 text-[11px] text-muted-foreground/60 tabular-nums">
                    {describeProbedAt(row.probed_at, now, t)}
                </span>
            )}

            <IconButton
                onClick={handleKeepalive}
                disabled={keepaliveGrant.isPending}
                tip={t('keepaliveThis')}
                aria-label={t('keepaliveThis')}
                className="size-6 shrink-0"
            >
                {keepaliveGrant.isPending
                    ? <LoaderCircle className="size-3 animate-spin" />
                    : <HeartPulse className="size-3" />}
            </IconButton>

            <IconButton
                onClick={handleRemove}
                disabled={setCredential.isPending}
                tip={t('removeCredential')}
                aria-label={t('removeCredential')}
                className="size-6 shrink-0 hover:text-destructive"
            >
                <X className="size-3" />
            </IconButton>
        </li>
    );
}
