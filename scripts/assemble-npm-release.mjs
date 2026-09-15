import assert from 'node:assert/strict';
import { chmod, copyFile, cp, mkdir, mkdtemp, readFile, rm, stat, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import platformModule from '../npm/platforms.js';
import {
  commandNames,
  documentationFiles,
  ensureNewDirectory,
  listFiles,
  parseArguments,
  readJson,
  requiredArgument,
  run,
  sha256,
} from './npm-package-utils.mjs';

const { platformTargets } = platformModule;
const root = resolve(import.meta.dirname, '..');

async function extractArchive(archive, destination) {
  if (archive.endsWith('.zip') && process.platform !== 'win32') {
    await run('unzip', ['-q', archive, '-d', destination]);
    return;
  }
  await run('tar', ['-xf', archive, '-C', destination]);
}

async function verifyChecksum(archive) {
  const checksum = `${archive}.sha256`;
  const expected = (await readFile(checksum, 'utf8')).trim().split(/\s+/)[0].toLowerCase();
  assert.match(expected, /^[0-9a-f]{64}$/);
  assert.equal(await sha256(archive), expected, `SHA-256 mismatch for ${archive}`);
  return checksum;
}

async function copyRootPackage(output, version) {
  const destination = resolve(output, 'root');
  await mkdir(destination);
  const manifest = await readJson(resolve(root, 'package.json'));
  assert.equal(manifest.version, version);
  await copyFile(resolve(root, 'package.json'), resolve(destination, 'package.json'));
  await cp(resolve(root, 'npm'), resolve(destination, 'npm'), { recursive: true });
  await cp(resolve(root, 'skills'), resolve(destination, 'skills'), { recursive: true });
  for (const file of documentationFiles) await copyFile(resolve(root, file), resolve(destination, file));
}

async function preparePlatformPackage(assets, output, target, version, verifyAttestations) {
  const extension = target.executableSuffix === '.exe' ? '.zip' : '.tar.gz';
  const archiveName = `dovik-v${version}-${target.archiveTarget}${extension}`;
  const archive = resolve(assets, archiveName);
  const checksum = await verifyChecksum(archive);
  if (verifyAttestations) {
    await run('gh', ['attestation', 'verify', archive, '--repo', 'MrMaxie/dovik']);
    await run('gh', ['attestation', 'verify', checksum, '--repo', 'MrMaxie/dovik']);
  }

  const extraction = await mkdtemp(resolve(output, `.extract-${target.directory}-`));
  try {
    await extractArchive(archive, extraction);
    const content = resolve(extraction, `dovik-v${version}-${target.archiveTarget}`);
    const expected = [
      'CHANGELOG.md',
      'LICENSE',
      'README.md',
      'THIRD_PARTY_LICENSES.md',
      ...commandNames.map((command) => `${command}${target.executableSuffix}`),
      'skills/README.md',
      'skills/claude/SKILL.md',
      'skills/codex/SKILL.md',
    ].sort();
    assert.deepEqual(await listFiles(content), expected, `${archiveName} has unexpected contents.`);

    const destination = resolve(output, target.directory);
    const bin = resolve(destination, 'bin');
    await mkdir(bin, { recursive: true });
    const manifestPath = resolve(root, 'packages', target.directory, 'package.json');
    const manifest = await readJson(manifestPath);
    assert.equal(manifest.version, version);
    await copyFile(manifestPath, resolve(destination, 'package.json'));
    for (const file of ['LICENSE', 'THIRD_PARTY_LICENSES.md']) {
      await copyFile(resolve(content, file), resolve(destination, file));
    }

    const digests = {};
    for (const command of commandNames) {
      const fileName = `${command}${target.executableSuffix}`;
      const source = resolve(content, fileName);
      const destinationPath = resolve(bin, fileName);
      if (target.os[0] !== 'win32' && process.platform !== 'win32') {
        assert.notEqual((await stat(source)).mode & 0o111, 0);
      }
      await copyFile(source, destinationPath);
      if (target.os[0] !== 'win32') await chmod(destinationPath, 0o755);
      digests[command] = await sha256(source);
      assert.equal(await sha256(destinationPath), digests[command]);
    }
    return { archive: archiveName, directory: target.directory, name: manifest.name, digests };
  } finally {
    await rm(extraction, { recursive: true, force: true });
  }
}

const argumentsMap = parseArguments(process.argv.slice(2));
const assets = resolve(requiredArgument(argumentsMap, 'assets'));
const output = await ensureNewDirectory(requiredArgument(argumentsMap, 'output'));
const version = requiredArgument(argumentsMap, 'version');
const testMode = argumentsMap.get('test-mode') === true;
if (argumentsMap.has('test-mode') && !testMode) throw new Error('--test-mode does not accept a value.');

await copyRootPackage(output, version);
const packages = [];
for (const target of platformTargets) {
  packages.push(await preparePlatformPackage(assets, output, target, version, !testMode));
}
await writeFile(
  resolve(output, 'assembly.json'),
  `${JSON.stringify({ version, repository: 'MrMaxie/dovik', attestationsVerified: !testMode, packages }, null, 2)}\n`,
);
process.stdout.write(`assembled Dovik npm packages for ${version}${testMode ? ' in explicit test mode' : ''}\n`);
