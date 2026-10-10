import { useMemo } from 'react';
import { Plus } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { useKeepaliveList } from '@/api/keepalive';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { buttonVariants } from '@/components/ui/button';
import {
    MorphingDialog,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogTrigger,
} from '@/components/ui/morphing-dialog';
import { useProbeClock } from '@/components/modules/scheduled-probe/clock';
import { CreateDialogContent } from './Create';
import { Item } from './Item';

// KeepaliveActions 向稳定顶栏提供新建保活任务的入口。
export function KeepaliveActions() {
    const t = useTranslations('keepalive');

    return (
        <MorphingDialog>
            <MorphingDialogTrigger
                className={buttonVariants({ variant: 'ghost', size: 'icon', className: 'rounded-xl transition-none text-muted-foreground hover:bg-transparent hover:text-foreground' })}
                aria-label={t('createTitle')}
            >
                <Plus className="size-4 transition-colors duration-300" />
            </MorphingDialogTrigger>
            <MorphingDialogContainer>
                {/* 宽度必须显式给, 理由同监控页: 居中 flex 容器里的子项不给宽度会塌成一条窄缝。 */}
                <MorphingDialogContent
                    dismissOnClickOutside={false}
                    className="relative flex h-[calc(100dvh-2rem)] w-screen max-w-full flex-col overflow-hidden rounded-3xl bg-card px-4 py-3 text-card-foreground md:h-auto md:max-h-[calc(100vh-2rem)] md:max-w-2xl md:px-6 md:py-4"
                >
                    <CreateDialogContent />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}

// Keepalive 渲染保活任务列表: 每张卡片一条任务, 每行一条凭据的最近一次心跳。
//
// 保活结论只用于展示: 失败(如排队已满的 429)不会冷却渠道、也不会改动选路顺序 ——
// 排队场景里失败是常态, 照监控语义处理会把正常渠道越保越死。需要选路效果请另建模型监控任务。
export function Keepalive() {
    const t = useTranslations('keepalive');
    const { data: probes } = useKeepaliveList();
    // 页面级时钟: 各张卡片共享同一个"现在", 用来把「多久以前」算准(见 useProbeClock)。
    const now = useProbeClock();

    // 顺序直接用后端给的（主键升序）: 那就是轮转顺序, 界面上不该重排。
    const items = useMemo(() => probes ?? [], [probes]);

    if (items.length === 0) {
        return (
            <div className="flex h-full min-h-0 items-center justify-center px-6">
                <p className="max-w-md text-center text-sm text-muted-foreground">{t('empty')}</p>
            </div>
        );
    }

    return (
        <VirtualizedGrid
            items={items}
            layout="grid"
            // 一行 2 个而不是监控页的 3 个: 保活行比监控行多出「状态 + HTTP 状态码 + 耗时」一整组,
            // 三列宽度放不下, 省略号会先吞掉渠道名与凭据名这两段身份信息。
            columns={{ default: 1, md: 2 }}
            estimateItemHeight={156}
            getItemKey={(probe) => `keepalive-${probe.id}`}
            renderItem={(probe) => <Item key={probe.id} probe={probe} now={now} />}
        />
    );
}
