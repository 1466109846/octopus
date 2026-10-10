import { useState } from 'react';
import { Check, HeartPulse, Pencil, Trash2, X } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { toast } from 'sonner';
import { keepaliveInputOf, useKeepaliveScheduledNow } from '@/api/keepalive';
import { useDeleteScheduledProbe, useUpdateScheduledProbe, type ScheduledProbe } from '@/api/scheduled-probe';
import { Switch } from '@/components/ui/switch';
import { IconButton } from '@/components/common/IconButton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { getModelIcon } from '@/lib/model-icons';
import { cn } from '@/lib/utils';
import {
    MorphingDialog,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogTitle,
    MorphingDialogTrigger,
} from '@/components/ui/morphing-dialog';
import { RowList } from './RowList';
import { EditDialogContent } from './Edit';

// Item 展示一条保活任务: 一个自定义名字下挂多行凭据, 每行显示最近一次心跳的状态、HTTP 状态码与耗时。
//
// 控件与监控卡片同一排布(心跳一次 / 编辑 / 删除 / 启用开关), 只是少了糖果按钮:
// 保活只发最小请求, 没有"问一道题看它笨不笨"这件事。
export function Item({ probe, now }: { probe: ScheduledProbe; now: number }) {
    const t = useTranslations('keepalive');
    const updateProbe = useUpdateScheduledProbe();
    const deleteProbe = useDeleteScheduledProbe();
    const keepaliveNow = useKeepaliveScheduledNow();
    const [confirmDelete, setConfirmDelete] = useState(false);

    const { Icon, className: iconClassName, color: brandColor } = getModelIcon(probe.name);
    const titleHint = `${probe.name} · ${t('intervalValue', { minutes: probe.interval_minutes })}`;

    // 启停走整体替换: 更新是全量提交, 漏传字段会被清空, 故用 keepaliveInputOf 还原完整提交体。
    const handleEnableChange = (checked: boolean) => {
        updateProbe.mutate(
            {
                ...keepaliveInputOf({
                    name: probe.name,
                    targets: probe.targets,
                    interval_minutes: probe.interval_minutes,
                    enabled: checked,
                }),
                id: probe.id,
            },
            {
                onSuccess: () => toast.success(checked ? t('enabled') : t('disabled')),
                onError: (error) => toast.error(error.message),
            },
        );
    };

    const handleConfirmDelete = () => {
        deleteProbe.mutate(probe.id, {
            onSuccess: () => {
                setConfirmDelete(false);
                toast.success(t('toast.deleted'));
            },
            onError: (error) => {
                setConfirmDelete(false);
                toast.error(error.message);
            },
        });
    };

    // handleKeepaliveAll 把这条任务的每条凭据都心跳一遍, 逐条汇报而不是只说"完成":
    // 多凭据时用户要看的就是哪条保住了、哪条被挡回。
    const handleKeepaliveAll = () => {
        keepaliveNow.mutate(probe.id, {
            onSuccess: (rows) => {
                const failed = rows.filter((row) => !row.ok);
                if (failed.length === 0) {
                    toast.success(t('toast.allDone', { count: rows.length }));
                    return;
                }
                toast.error(t('toast.partialFailed', { failed: failed.length, total: rows.length }), {
                    description: failed[0]?.message || undefined,
                });
            },
            onError: (error) => toast.error(error.message),
        });
    };

    return (
        <article className="flex h-full flex-col gap-1.5 rounded-2xl border border-border bg-card p-2.5 text-card-foreground">
            <div className="flex items-center gap-1.5">
                <Icon aria-hidden="true" className={cn('size-5 shrink-0', iconClassName)} style={{ color: brandColor }} />
                <Tooltip>
                    <TooltipTrigger asChild>
                        <span className="min-w-0 flex-1 truncate text-sm font-semibold leading-tight">{probe.name}</span>
                    </TooltipTrigger>
                    <TooltipContent side="top" sideOffset={10} align="start">{titleHint}</TooltipContent>
                </Tooltip>

                <div className="flex shrink-0 items-center gap-0.5">
                    <IconButton
                        onClick={handleKeepaliveAll}
                        disabled={keepaliveNow.isPending || probe.rows.length === 0}
                        tip={t('keepaliveNow')}
                        aria-label={t('keepaliveNow')}
                        className="size-7"
                    >
                        <HeartPulse className={cn('size-3.5', keepaliveNow.isPending && 'animate-pulse')} />
                    </IconButton>

                    <MorphingDialog>
                        <IconButton asChild tip={t('edit')} className="size-7">
                            <MorphingDialogTrigger aria-label={t('edit')}>
                                <Pencil className="size-3.5" />
                            </MorphingDialogTrigger>
                        </IconButton>
                        <MorphingDialogContainer>
                            {/* 宽度写法与监控页同一套: 不显式给宽度, 弹窗会缩到内容宽度而塌成窄缝。 */}
                            <MorphingDialogContent
                                dismissOnClickOutside={false}
                                className="relative flex h-[calc(100dvh-2rem)] w-screen max-w-full flex-col overflow-hidden rounded-3xl bg-card px-4 py-3 text-card-foreground md:h-auto md:max-h-[calc(100vh-2rem)] md:max-w-2xl md:px-6 md:py-4"
                            >
                                <MorphingDialogTitle className="shrink-0">{t('editTitle', { name: probe.name })}</MorphingDialogTitle>
                                <EditDialogContent initial={probe} />
                            </MorphingDialogContent>
                        </MorphingDialogContainer>
                    </MorphingDialog>

                    {confirmDelete ? (
                        <>
                            <IconButton
                                onClick={() => setConfirmDelete(false)}
                                tip={t('cancel')}
                                aria-label={t('cancel')}
                                className="size-7"
                            >
                                <X className="size-3.5" />
                            </IconButton>
                            <IconButton
                                onClick={handleConfirmDelete}
                                disabled={deleteProbe.isPending}
                                tip={t('confirmDelete')}
                                aria-label={t('confirmDelete')}
                                className="size-7 text-destructive hover:text-destructive/70"
                            >
                                <Check className="size-3.5" />
                            </IconButton>
                        </>
                    ) : (
                        <IconButton
                            onClick={() => setConfirmDelete(true)}
                            disabled={deleteProbe.isPending}
                            tip={t('delete')}
                            aria-label={t('delete')}
                            className="size-7 hover:text-destructive"
                        >
                            <Trash2 className="size-3.5" />
                        </IconButton>
                    )}

                    <Switch
                        checked={probe.enabled}
                        onCheckedChange={handleEnableChange}
                        disabled={updateProbe.isPending || deleteProbe.isPending}
                        aria-label={t('enable')}
                    />
                </div>
            </div>

            <RowList probeId={probe.id} rows={probe.rows} enabled={probe.enabled} now={now} />
        </article>
    );
}
