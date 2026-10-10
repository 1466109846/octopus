import { useTranslations } from 'use-intl';
import { toast } from 'sonner';
import { KEEPALIVE_DEFAULT_INTERVAL, keepaliveInputOf } from '@/api/keepalive';
import { useCreateScheduledProbe } from '@/api/scheduled-probe';
import { MorphingDialogDescription, MorphingDialogTitle, useMorphingDialog } from '@/components/ui/morphing-dialog';
import { ProbeForm, type ProbeFormValues } from '@/components/modules/scheduled-probe/ProbeForm';

// CreateDialogContent 渲染新建保活任务的表单。
// 表单与监控共用(ProbeForm), 以 purpose="keepalive" 去掉糖果开关与时间窗; 提交时补上 kind 与保活的固定缺省。
export function CreateDialogContent() {
    const { setIsOpen } = useMorphingDialog();
    const t = useTranslations('keepalive');
    const tProbe = useTranslations('scheduledProbe');
    const createProbe = useCreateScheduledProbe();

    const handleSubmit = async (values: ProbeFormValues) => {
        try {
            // 新建一律先启用: 用户点了创建就是要它开始心跳, 不该在停用状态下躺进列表。
            await createProbe.mutateAsync(keepaliveInputOf({ ...values, enabled: true }));
            toast.success(t('toast.created'));
            setIsOpen(false);
        } catch (error) {
            toast.error((error as Error).message);
        }
    };

    return (
        <>
            <MorphingDialogTitle className="shrink-0">{t('createTitle')}</MorphingDialogTitle>
            {/* 这一层必须自己就是可收缩的 flex 列: 否则表单撑高后会把提交按钮顶出弹窗、被裁掉(见监控页同名组件)。 */}
            <MorphingDialogDescription className="flex min-h-0 flex-1 flex-col">
                <ProbeForm
                    purpose="keepalive"
                    defaultInterval={KEEPALIVE_DEFAULT_INTERVAL}
                    submitText={tProbe('submit')}
                    submittingText={tProbe('submitting')}
                    isSubmitting={createProbe.isPending}
                    onCancel={() => setIsOpen(false)}
                    onSubmit={handleSubmit}
                />
            </MorphingDialogDescription>
        </>
    );
}
