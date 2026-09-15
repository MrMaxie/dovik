import assert from 'node:assert/strict';
import { access, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';
import platformModule from '../npm/platforms.js';
import { parseArguments, requiredArgument, run, runNpm } from './npm-package-utils.mjs';

const { selectPlatformTarget } = platformModule;
const target = selectPlatformTarget();
assert.ok(target, `unsupported verification platform ${process.platform}/${process.arch}`);
const argumentsMap = parseArguments(process.argv.slice(2));
const tarballs = resolve(requiredArgument(argumentsMap, 'tarballs'));
const metadata = JSON.parse(await readFile(resolve(tarballs, 'tarballs.json'), 'utf8'));
const platformPackage = metadata.packages.find((entry) => entry.name === target.packageName);
const rootPackage = metadata.packages.find((entry) => entry.name === 'dovik');
assert.ok(platformPackage);
assert.ok(rootPackage);

async function runShim(consumer, command, argumentsValue) {
  return runNpm(['exec', '--offline', '--', command, ...argumentsValue], { cwd: consumer });
}

const workspace = await mkdtemp(resolve(tmpdir(), 'dovik-npm-consumer-'));
try {
  const consumer = resolve(workspace, 'consumer');
  const cache = resolve(workspace, 'cache');
  await mkdir(consumer);
  await mkdir(cache);
  await writeFile(resolve(consumer, 'package.json'), `${JSON.stringify({ private: true }, null, 2)}\n`);
  await runNpm([
    'install', '--ignore-scripts', '--no-audit', '--no-fund', '--no-package-lock',
    resolve(tarballs, rootPackage.file), resolve(tarballs, platformPackage.file),
  ], { cwd: consumer, env: { ...process.env, npm_config_cache: cache } });

  const bin = resolve(consumer, 'node_modules', '.bin');
  for (const command of ['dovik', 'dovikd']) {
    const version = await runShim(consumer, command, ['--version']);
    assert.equal(version.stdout.trim(), `${command} ${metadata.version}`);
    await runShim(consumer, command, ['--help']);
  }
  await assert.rejects(access(resolve(bin, process.platform === 'win32' ? 'gh.cmd' : 'gh')));

  const proxy = resolve(consumer, 'node_modules', target.packageName, 'bin', `gh${target.executableSuffix}`);
  const identity = await run(proxy, ['--dovik-proxy-identify'], { cwd: consumer });
  assert.equal(identity.stdout.trim(), 'dovik-gh-proxy-v1');
  process.stdout.write(`verified dovik@${metadata.version} in an isolated ${target.directory} consumer\n`);
} finally {
  await rm(workspace, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
}
