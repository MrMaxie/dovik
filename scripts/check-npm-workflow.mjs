import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const source = await readFile(resolve(root, '.github', 'workflows', 'npm.yml'), 'utf8');
const ci = await readFile(resolve(root, '.github', 'workflows', 'ci.yml'), 'utf8');
const candidate = await readFile(resolve(root, '.github', 'workflows', 'candidate.yml'), 'utf8');
const release = await readFile(resolve(root, '.github', 'workflows', 'release.yml'), 'utf8');
const stage = await readFile(resolve(root, 'scripts', 'stage-npm-release.mjs'), 'utf8');
assert.match(source, /release:\s*\n\s*types: \[published\]/);
assert.match(source, /workflow_dispatch:/);
assert.match(source, /stage:\s*\n[\s\S]*?default: false/);
assert.match(source, /permissions:\s*\n\s*contents: read/);
assert.match(source, /environment: npm-production/);
assert.match(source, /id-token: write/);
assert.match(source, /if: github\.event_name == 'release' \|\| inputs\.stage == true/);
assert.match(source, /package_version="\$\(node -p "require\('\.\/package\.json'\)\.version"\)"/);
assert.equal(source.includes('node -p \\"'), false);
assert.equal(source.includes('NPM_TOKEN'), false);
assert.equal(source.includes('NODE_AUTH_TOKEN'), false);
assert.equal(source.includes('npm publish '), false);
assert.match(stage, /\['stage', 'publish', archive/);
assert.match(stage, /'--provenance'/);
for (const match of source.matchAll(/uses:\s+([^\s#]+)/g)) {
  assert.match(match[1], /^[^@]+@[0-9a-f]{40}$/, `Action is not pinned: ${match[1]}`);
}
for (const workflow of [ci, candidate, release]) {
  assert.equal(workflow.includes('dovik-v1.0.0'), false);
  assert.match(workflow, /dovik-v\$\{\{ (?:steps\.version|needs\.source)\.outputs\.version \}\}-\$\{\{ matrix\.target \}\}\.\$\{\{ matrix\.extension \}\}/);
}
assert.equal(release.includes('Publish GHCR image'), false);
assert.equal(release.includes('Verify Scoop install'), false);
assert.equal(release.includes('dist/dovik.json'), false);
assert.equal(release.includes('ghcr.io/mrmaxie/dovik'), false);
process.stdout.write('verified npm and release workflow contracts\n');
