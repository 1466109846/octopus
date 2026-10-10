import { useTranslations } from 'use-intl';
import { Info, Tag, AlertTriangle, Download, Loader2, Monitor, GitCommitHorizontal } from 'lucide-react';
import Github from '@thesvg/react/github';
import { useLatestInfo, useBuildInfo, useUpdateCore } from '@/api/update';
import { hasBuildMismatch, isNewerVersion } from '@/lib/build-info';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

const FRONTEND_BUILD = {
    version: import.meta.env.VITE_APP_VERSION || 'unknown',
    repository: import.meta.env.VITE_GITHUB_REPO || 'https://github.com/1466109846/octopus',
};

export function SettingInfo() {
    const t = useTranslations('setting');
    const latestInfoQuery = useLatestInfo();
    const buildInfoQuery = useBuildInfo();
    const updateCore = useUpdateCore();
    const backend = buildInfoQuery.data;
    const backendVersion = backend?.version || '';
    const latestVersion = latestInfoQuery.data?.tag_name || '';
    const repository = backend?.repository || FRONTEND_BUILD.repository;
    const mismatch = hasBuildMismatch(FRONTEND_BUILD, backend);
    const hasNewVersion = isNewerVersion(latestVersion, backendVersion);

    const clearCacheAndReload = async () => {
        const scope = new URL('./', document.baseURI);
        if ('caches' in window) {
            const names = await caches.keys();
            await Promise.all(names.filter((name) => name.startsWith(`octopus-${encodeURIComponent(scope.pathname)}-`)).map((name) => caches.delete(name)));
        }
        if ('serviceWorker' in navigator) {
            const registration = await navigator.serviceWorker.getRegistration(scope.href);
            if (registration?.scope === scope.href) await registration.unregister();
        }
        window.location.reload();
    };

    const handleUpdate = () => {
        updateCore.mutate(undefined, {
            onSuccess: () => {
                toast.success(t('info.updateSuccess'));
                setTimeout(() => { void clearCacheAndReload(); }, 3000);
            },
            onError: () => toast.error(t('info.updateFailed')),
        });
    };

    const labelClass = 'flex min-w-0 items-center gap-3 text-sm font-medium';
    const valueClass = 'min-w-0 text-right text-sm font-mono text-muted-foreground tabular-nums break-all';

    return (
        <section aria-label={t('info.title')} className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Info className="size-5 shrink-0" />
                {t('info.title')}
            </h2>
            <dl className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-4 gap-y-4">
                <dt className={labelClass}>
                    <Github variant="mono" className="size-5 shrink-0 text-muted-foreground" />
                    {t('info.github')}
                </dt>
                <dd className="min-w-0 text-right text-sm break-all">
                    <a href={repository} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                        {repository.replace('https://github.com/', '')}
                    </a>
                </dd>
                <dt className={labelClass}>
                    <Tag className="size-5 shrink-0 text-muted-foreground" />
                    {t('info.currentVersion')}
                </dt>
                <dd className={valueClass}>
                    {buildInfoQuery.isLoading ? <Loader2 aria-label={t('info.loading')} className="ml-auto size-4 animate-spin" /> : <code>{backendVersion || t('info.unknown')}</code>}
                </dd>
                <dt className={labelClass}>
                    <Monitor className="size-5 shrink-0 text-muted-foreground" />
                    {t('info.frontendVersion')}
                </dt>
                <dd className={valueClass}><code>{FRONTEND_BUILD.version}</code></dd>
                <dt className={labelClass}>
                    <Download className="size-5 shrink-0 text-muted-foreground" />
                    {t('info.latestVersion')}
                </dt>
                <dd className={valueClass}>
                    {latestInfoQuery.isLoading ? <Loader2 aria-label={t('info.loading')} className="ml-auto size-4 animate-spin" /> : <code>{latestVersion || t('info.unknown')}</code>}
                </dd>
                <dt className={labelClass}>
                    <GitCommitHorizontal className="size-5 shrink-0 text-muted-foreground" />
                    {t('info.buildCommit')}
                </dt>
                <dd className={valueClass}><code>{backend?.commit || t('info.unknown')}</code></dd>
            </dl>

            {mismatch && (
                <div role="alert" className="p-3 bg-destructive/10 border border-destructive/20 rounded-xl space-y-2">
                    <div className="flex items-start gap-3">
                        <AlertTriangle className="size-5 text-destructive shrink-0 mt-0.5" />
                        <div className="min-w-0 flex-1 space-y-1">
                            <p className="text-sm text-destructive font-medium">{t('info.versionMismatch')}</p>
                            <p className="text-xs text-muted-foreground break-words">
                                {t('info.versionMismatchHint', { frontend: FRONTEND_BUILD.version, backend: backendVersion })}
                            </p>
                            {backend?.repository !== FRONTEND_BUILD.repository && (
                                <p className="text-xs text-muted-foreground break-all">
                                    {t('info.repositoryMismatch', { frontend: FRONTEND_BUILD.repository, backend: repository })}
                                </p>
                            )}
                        </div>
                    </div>
                    <div className="flex justify-end">
                        <Button variant="destructive" size="sm" onClick={() => { void clearCacheAndReload(); }} className="rounded-xl">
                            {t('info.forceRefresh')}
                        </Button>
                    </div>
                </div>
            )}
            {hasNewVersion && (
                <div className="p-3 bg-primary/10 border border-primary/20 rounded-xl space-y-2">
                    <div className="flex items-start gap-3">
                        <Download className="size-5 text-primary shrink-0 mt-0.5" />
                        <div className="min-w-0 flex-1 space-y-1">
                            <p className="text-sm text-primary font-medium">{t('info.newVersionAvailable')}</p>
                            <p className="text-xs text-muted-foreground">{t('info.newVersionAvailableHint')}</p>
                        </div>
                    </div>
                    <div className="flex justify-end">
                        <Button variant="default" size="sm" onClick={handleUpdate} disabled={updateCore.isPending} className="rounded-xl">
                            {updateCore.isPending ? t('info.updating') : t('info.updateNow')}
                        </Button>
                    </div>
                </div>
            )}
        </section>
    );
}
