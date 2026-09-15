import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const manifests = new Map([
  ['win32:x64', { directory: 'win32-x64', packageName: '@maxiedev/dovik-win32-x64', suffix: '.exe' }],
  ['linux:x64', { directory: 'linux-x64', packageName: '@maxiedev/dovik-linux-x64', suffix: '' }],
  ['darwin:x64', { directory: 'darwin-x64', packageName: '@maxiedev/dovik-darwin-x64', suffix: '' }],
  ['darwin:arm64', { directory: 'darwin-arm64', packageName: '@maxiedev/dovik-darwin-arm64', suffix: '' }],
]);
const target = manifests.get(`${process.platform}:${process.arch}`);
assert.ok(target, `unsupported verification platform ${process.platform}/${process.arch}`);
const npmRunner = process.env.npm_execpath
  ? { command: process.execPath, prefix: [process.env.npm_execpath] }
  : { command: 'npm', prefix: [] };

async function run(command, args, options = {}) {
  const child = spawn(command, args, {
    cwd: options.cwd ?? root,
    env: options.env ?? process.env,
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  });
  let stdout = '';
  let stderr = '';
  child.stdout.setEncoding('utf8');
  child.stderr.setEncoding('utf8');
  child.stdout.on('data', (data) => { stdout += data; });
  child.stderr.on('data', (data) => { stderr += data; });
  const code = await new Promise((resolveExit, rejectExit) => {
    child.once('error', rejectExit);
    child.once('exit', resolveExit);
  });
  if (code !== 0) throw new Error(`${command} ${args.join(' ')} failed (${code})\n${stdout}${stderr}`);
  return { stdout, stderr };
}

async function pack(directory, destination) {
  const result = await run(npmRunner.command, [
    ...npmRunner.prefix,
    'pack',
    directory,
    '--silent',
    '--ignore-scripts',
    '--pack-destination',
    destination,
  ]);
  const filename = result.stdout.trim().split(/\r?\n/).at(-1);
  assert.ok(filename, `npm pack returned no archive for ${directory}`);
  return resolve(destination, filename);
}

const manifest = JSON.parse(await readFile(resolve(root, 'package.json'), 'utf8'));
assert.equal(manifest.optionalDependencies[target.packageName], manifest.version);
const nativeManifest = JSON.parse(
  await readFile(resolve(root, 'packages', target.directory, 'package.json'), 'utf8'),
);
assert.equal(nativeManifest.version, manifest.version);

const privateDirectory = resolve(root, '.local', 'npm-package-check');
await mkdir(privateDirectory, { recursive: true });
const workspace = await mkdtemp(join(privateDirectory, 'run-'));
try {
  const consumer = resolve(workspace, 'consumer');
  const cache = resolve(workspace, 'cache');
  await mkdir(consumer);
  await writeFile(resolve(consumer, 'package.json'), `${JSON.stringify({ private: true }, null, 2)}\n`);

  if (process.argv.includes('--published')) {
    await run(
      npmRunner.command,
      [...npmRunner.prefix, 'install', '--ignore-scripts', '--no-audit', '--no-fund', '--no-package-lock', `dovik@${manifest.version}`],
      { cwd: consumer, env: { ...process.env, npm_config_cache: cache } },
    );
  } else {
    const rootArchive = await pack(root, workspace);
    const nativeArchive = await pack(resolve(root, 'packages', target.directory), workspace);
    await run(
      npmRunner.command,
      [...npmRunner.prefix, 'install', '--ignore-scripts', '--no-audit', '--no-fund', '--no-package-lock', rootArchive, nativeArchive],
      { cwd: consumer, env: { ...process.env, npm_config_cache: cache } },
    );
  }

  const launcher = resolve(consumer, 'node_modules', 'dovik', 'npm');
  const dovik = resolve(launcher, 'dovik.js');
  const dovikd = resolve(launcher, 'dovikd.js');
  const clientVersion = await run(process.execPath, [dovik, '--version'], { cwd: consumer });
  const daemonVersion = await run(process.execPath, [dovikd, '--version'], { cwd: consumer });
  assert.equal(clientVersion.stdout.trim(), `dovik ${manifest.version}`);
  assert.equal(daemonVersion.stdout.trim(), `dovikd ${manifest.version}`);
  await run(process.execPath, [dovik, '--help'], { cwd: consumer });
  await run(process.execPath, [dovikd, '--help'], { cwd: consumer });

  const proxy = resolve(
    consumer,
    'node_modules',
    target.packageName,
    'bin',
    `gh${target.suffix}`,
  );
  const identity = await run(proxy, ['--dovik-proxy-identify'], { cwd: consumer });
  assert.equal(identity.stdout.trim(), 'dovik-gh-proxy-v1');
  process.stdout.write(`verified dovik@${manifest.version} and ${target.packageName}\n`);
} finally {
  await rm(workspace, { force: true, recursive: true, maxRetries: 5, retryDelay: 100 });
}
