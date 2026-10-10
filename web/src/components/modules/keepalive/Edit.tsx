import { useTranslations } from 'use-intl';
import { toast } from 'sonner';
import { keepaliveInputOf } from '@/api/keepalive';
import { useUpdateScheduledProbe, type ScheduledProbe } from '@/api/scheduled-probe';
import { MorphingDialogDescription, useMorphingDialog } from '@/components/ui/morphing-dialog';
import { ProbeForm, type ProbeFormValues } from '@/components/modules/scheduled-probe/ProbeForm';

// EditDialogContent 把一条已有保活任务还原到 ProbeForm, 提交时整体替换。
// 种类(kind)由后端在更新时保留, 这里照常带上只为提交体与语义一致。
export function EditDialogContent({ initial }: { initial: ScheduledProbe }) {
    const { setIsOpen } = useMorphingDialog();
    const t = useTranslations('keepalive');
    const tProbe = useTranslations('scheduledProbe');
    const updateProbe = useUpdateScheduledProbe();

    const handleSubmit = async (values: ProbeFormValues) => {
        try {
            // 编辑保留卡片上的启停状态: 开关在卡片上, 表单不参与启停。
            await updateProbe.mutateAsync({ ...keepaliveInputOf(values), id: initial.id });
            toast.success(t('toast.updated'));
            setIsOpen(false);
        } catch (error) {
            toast.error((error as Error).message);
        }
    };

    return (
        <MorphingDialogDescription className="flex min-h-0 flex-1 flex-col">
            <ProbeForm
                purpose="keepalive"
                initial={{
                    name: initial.name,
                    targets: initial.targets,
                    interval_minutes: initial.interval_minutes,
                    enabled: initial.enabled,
                    iq_disabled: true,
                    weekdays: 0,
                    start_hour: 0,
                    end_hour: 0,
                }}
                submitText={tProbe('submit')}
                submittingText={tProbe('submitting')}
                isSubmitting={updateProbe.isPending}
                onCancel={() => setIsOpen(false)}
                onSubmit={handleSubmit}
            />
        </MorphingDialogDescription>
    );
}
