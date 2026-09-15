import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import platformModule from '../npm/platforms.js';

const { platformTargets } = platformModule;
const root = resolve(import.meta.dirname, '..');
const repository = { type: 'git', url: 'git+https://github.com/MrMaxie/dovik.git' };
const lifecycleScripts = new Set([
  'preinstall', 'install', 'postinstall', 'prepare', 'prepack', 'postpack',
  'prepublish', 'prepublishOnly', 'publish', 'postpublish',
]);
const readJson = async (path) => JSON.parse(await readFile(path, 'utf8'));

function verifyIdentity(manifest, name, version) {
  assert.equal(manifest.name, name);
  assert.equal(manifest.version, version);
  assert.equal(manifest.license, 'Apache-2.0');
  assert.deepEqual(manifest.repository, repository);
  assert.deepEqual(manifest.publishConfig, { access: 'public', provenance: true });
  for (const script of Object.keys(manifest.scripts ?? {})) {
    assert.equal(lifecycleScripts.has(script), false, `${manifest.name} defines lifecycle script ${script}`);
  }
}

const manifest = await readJson(resolve(root, 'package.json'));
assert.match(manifest.version, /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/);
verifyIdentity(manifest, 'dovik', manifest.version);
assert.equal(manifest.packageManager, 'npm@12.0.2');
assert.deepEqual(manifest.bin, { dovik: 'npm/dovik.js', dovikd: 'npm/dovikd.js' });
assert.deepEqual(
  manifest.optionalDependencies,
  Object.fromEntries(platformTargets.map((target) => [target.packageName, manifest.version]).sort()),
);
for (const target of platformTargets) {
  const native = await readJson(resolve(root, 'packages', target.directory, 'package.json'));
  verifyIdentity(native, target.packageName, manifest.version);
  assert.deepEqual(native.os, target.os);
  assert.deepEqual(native.cpu, target.cpu);
  assert.equal(native.bin, undefined);
  assert.equal(native.dependencies, undefined);
  assert.equal(native.optionalDependencies, undefined);
}
process.stdout.write('verified Dovik npm package manifests\n');
