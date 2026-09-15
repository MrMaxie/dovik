import assert from 'node:assert/strict';
import { appendFile, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import platformModule from '../npm/platforms.js';
import { parseArguments, requiredArgument, runNpm, sha256 } from './npm-package-utils.mjs';

const { platformTargets } = platformModule;
const argumentsMap = parseArguments(process.argv.slice(2));
const tarballs = resolve(requiredArgument(argumentsMap, 'tarballs'));
const metadata = JSON.parse(await readFile(resolve(tarballs, 'tarballs.json'), 'utf8'));
assert.equal(metadata.attestationsVerified, true, 'Refusing to stage packages without verified attestations.');
const expectedOrder = [...platformTargets.map((target) => target.packageName), 'dovik'];
assert.deepEqual(metadata.approvalOrder, expectedOrder);
assert.deepEqual(metadata.packages.map((entry) => entry.name), expectedOrder);

const summary = ['## Dovik npm staging', '', `Version: ${metadata.version}`, '', 'Required 2FA approval order:', ''];
for (const [index, packageMetadata] of metadata.packages.entries()) {
  const archive = resolve(tarballs, packageMetadata.file);
  assert.equal(await sha256(archive), packageMetadata.sha256);
  const result = await runNpm(['stage', 'publish', archive, '--access', 'public', '--provenance']);
  process.stdout.write(result.stdout);
  process.stderr.write(result.stderr);
  summary.push(`${index + 1}. ${packageMetadata.name}@${packageMetadata.version}`);
}
summary.push('', 'Review every staged tarball. Approve all platform packages before dovik.');
if (process.env.GITHUB_STEP_SUMMARY) await appendFile(process.env.GITHUB_STEP_SUMMARY, `${summary.join('\n')}\n`);
process.stdout.write(`staged ${metadata.packages.length} Dovik npm packages for maintainer review\n`);
