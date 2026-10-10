const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');
const ts = require('typescript');

const source = fs.readFileSync(path.join(__dirname, '../src/lib/build-info.ts'), 'utf8');
const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
});
const context = { exports: {} };
vm.runInNewContext(outputText, context);
const { isNewerVersion, hasBuildMismatch } = context.exports;

test('four-part fork versions are ordered numerically', () => {
    assert.equal(isNewerVersion('v0.13.10.2', 'v0.13.10.1'), true);
    assert.equal(isNewerVersion('v0.13.10.10', 'v0.13.10.9'), true);
    assert.equal(isNewerVersion('v0.13.10.1', 'v0.13.10.2'), false);
    assert.equal(isNewerVersion('v0.13.10.2', 'v0.13.10.2'), false);
    assert.equal(isNewerVersion('v0.13.10', 'v0.13.10.0'), false);
    assert.equal(isNewerVersion('dev', 'v0.13.10.2'), false);
    assert.equal(isNewerVersion('v0.13.10.2', ''), false);
});

test('build version and repository mismatches remain visible', () => {
    const front = { version: 'v0.13.10.2', repository: 'https://github.com/1466109846/octopus' };
    assert.equal(hasBuildMismatch(front, front), false);
    assert.equal(hasBuildMismatch(front, { ...front, version: 'v0.13.10.1' }), true);
    assert.equal(hasBuildMismatch(front, { ...front, repository: 'https://github.com/xiaocqaq/octopus' }), true);
    assert.equal(hasBuildMismatch(front, undefined), false);
});
