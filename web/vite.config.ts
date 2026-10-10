import path from 'node:path';
import { execSync } from 'node:child_process';
import fs from 'node:fs';
import babel from '@rolldown/plugin-babel';
import react, { reactCompilerPreset } from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { compression, defineAlgorithm } from 'vite-plugin-compression2';

const root = path.resolve(import.meta.dirname, '..');

function git(command: string): string {
  try {
    return execSync(command, { cwd: root, stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim();
  } catch {
    return '';
  }
}

// The source marker is available before tagging and when building a source archive.
const sourceVersion = fs.existsSync(path.join(root, 'main.go'))
  ? fs.readFileSync(path.join(root, 'main.go'), 'utf-8').match(/^\/\/ Version (\S+)/m)?.[1]
  : undefined;
const appVersion = process.env.VITE_APP_VERSION || sourceVersion || 'dev';
const appCommit = process.env.VITE_APP_COMMIT || git('git rev-parse --short HEAD') || 'unknown';
const githubRepo = (process.env.VITE_GITHUB_REPO || git('git remote get-url origin') || 'https://github.com/1466109846/octopus')
  .replace(/^git@github\.com:/, 'https://github.com/').replace(/\.git$/, '').replace(/\/$/, '');
const buildInfo = { version: appVersion, repository: githubRepo, commit: appCommit };

export default defineConfig({
  base: './',
  define: {
    // 用 define 而非令 Vite 读取环境变量: 取值有 git 回退逻辑, 在这里定稿才能保证两侧一致。
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(appVersion),
    'import.meta.env.VITE_GITHUB_REPO': JSON.stringify(githubRepo),
  },
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset()] }),
    // 把版本、仓库与提交号写进构建产物: sw.js 的缓存版本号, 以及 build-info.json(供发布校验与直接查阅)。
    // 走 closeBundle 落盘而不是 emitFile: public/sw.js 会被 Vite 原样拷贝到输出目录,
    // 再 emit 一个同名资产会撞名; 写盘则是在拷贝完成后原地替换, 顺序稳定。
    {
      name: 'inject-build-metadata',
      closeBundle() {
        const outDir = path.resolve(import.meta.dirname, '../static/out');
        const swPath = path.join(outDir, 'sw.js');
        if (fs.existsSync(swPath)) {
          const content = fs.readFileSync(swPath, 'utf-8');
          fs.writeFileSync(swPath, content.replace(/__APP_VERSION__/g, `${appVersion}-${appCommit}`));
        }
        fs.writeFileSync(path.join(outDir, 'build-info.json'), JSON.stringify(buildInfo, null, 2) + '\n');
      },
    },
    compression({
      algorithms: [defineAlgorithm('gzip', { level: 9 })],
      include: /\.(html|css|js|mjs|json|svg|txt|xml)$/,
      deleteOriginalAssets: true,
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  build: {
    outDir: path.resolve(import.meta.dirname, '../static/out'),
    emptyOutDir: true,
  },
  server: {
    hmr: process.env.DISABLE_HMR !== 'true',
    watch: process.env.DISABLE_HMR === 'true' ? null : {},
    proxy: {
      '/api': {
        target: process.env.VITE_PROXY_TARGET || 'http://127.0.0.1:18874',
        changeOrigin: false,
      },
    },
  },
});
