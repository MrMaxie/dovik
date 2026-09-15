import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import platformModule from '../npm/platforms.js';
import {
  ensureNewDirectory,
  npmArchiveName,
  parseArguments,
  parseNpmPackMetadata,
  readJson,
  requiredArgument,
  runNpm,
  sha256,
} from './npm-package-utils.mjs';

const { platformTargets } = platformModule;

function expectedFiles(target) {
  if (target === undefined) {
    return [
      'CHANGELOG.md', 'LICENSE', 'README.md', 'THIRD_PARTY_LICENSES.md',
      'npm/dovik.js', 'npm/dovikd.js', 'npm/platforms.js', 'npm/run-native.js',
      'package.json', 'skills/README.md', 'skills/claude/SKILL.md', 'skills/codex/SKILL.md',
    ].sort();
  }
  return [
    'LICENSE', 'THIRD_PARTY_LICENSES.md', 'package.json',
    `bin/dovik${target.executableSuffix}`,
    `bin/dovikd${target.executableSuffix}`,
    `bin/gh${target.executableSuffix}`,
  ].sort();
}

async function packOnce(directory, output) {
  const result = await runNpm(['pack', directory, '--json', '--ignore-scripts', '--pack-destination', output]);
  return parseNpmPackMetadata(result.stdout);
}

async function verifyAndPack(directory, output, target, version) {
  const manifest = await readJson(resolve(directory, 'package.json'));
  assert.equal(manifest.version, version);
  const metadata = await packOnce(directory, output);
  assert.deepEqual(metadata.files.map((file) => file.path).sort(), expectedFiles(target));
  const filename = npmArchiveName(manifest.name, version);
  const repeat = await mkdtemp(resolve(output, '.repeat-'));
  try {
    await packOnce(directory, repeat);
    assert.equal(await sha256(resolve(output, filename)), await sha256(resolve(repeat, filename)));
  } finally {
    await rm(repeat, { recursive: true, force: true });
  }
  return { name: manifest.name, version, file: filename, sha256: await sha256(resolve(output, filename)) };
}

const argumentsMap = parseArguments(process.argv.slice(2));
const stage = resolve(requiredArgument(argumentsMap, 'stage'));
const output = await ensureNewDirectory(requiredArgument(argumentsMap, 'output'));
const assembly = JSON.parse(await readFile(resolve(stage, 'assembly.json'), 'utf8'));
assert.equal(assembly.packages.length, platformTargets.length);

const packages = [];
for (const target of platformTargets) {
  packages.push(await verifyAndPack(resolve(stage, target.directory), output, target, assembly.version));
}
packages.push(await verifyAndPack(resolve(stage, 'root'), output, undefined, assembly.version));
await writeFile(
  resolve(output, 'tarballs.json'),
  `${JSON.stringify({
    version: assembly.version,
    attestationsVerified: assembly.attestationsVerified,
    approvalOrder: packages.map((entry) => entry.name),
    packages,
  }, null, 2)}\n`,
);
process.stdout.write(`packed ${packages.length} deterministic Dovik npm tarballs for ${assembly.version}\n`);
