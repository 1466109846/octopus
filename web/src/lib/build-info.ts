export type BuildInfo = {
    version: string;
    repository: string;
    commit?: string;
    build_time?: string;
};

function versionParts(version: string): number[] | undefined {
    if (!/^v?\d+\.\d+\.\d+(?:\.\d+)?$/.test(version)) return undefined;
    const parts = version.replace(/^v/, '').split('.').map(Number);
    return parts.every(Number.isSafeInteger) ? parts : undefined;
}

// Fork releases use a fourth numeric segment; an older latest tag must not offer a downgrade.
export function isNewerVersion(latest: string, current: string): boolean {
    const left = versionParts(latest);
    const right = versionParts(current);
    if (!left || !right) return false;
    for (let i = 0; i < Math.max(left.length, right.length); i++) {
        if ((left[i] ?? 0) !== (right[i] ?? 0)) return (left[i] ?? 0) > (right[i] ?? 0);
    }
    return false;
}

export function hasBuildMismatch(frontend: BuildInfo, backend: BuildInfo | undefined): boolean {
    if (!backend) return false;
    const versionMismatch = versionParts(frontend.version) && versionParts(backend.version)
        && frontend.version !== backend.version;
    const repoMismatch = frontend.repository && backend.repository && frontend.repository !== backend.repository;
    return Boolean(versionMismatch || repoMismatch);
}
