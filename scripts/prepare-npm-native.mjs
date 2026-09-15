import { spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';

const targets = new Map([
  ['win32:x64', { directory: 'win32-x64', suffix: '.exe' }],
  ['linux:x64', { directory: 'linux-x64', suffix: '' }],
  ['darwin:x64', { directory: 'darwin-x64', suffix: '' }],
  ['darwin:arm64', { directory: 'darwin-arm64', suffix: '' }],
]);

const target = targets.get(`${process.platform}:${process.arch}`);
if (target === undefined) {
  throw new Error(`Dovik does not support native npm builds for ${process.platform}/${process.arch}`);
}

const packageDirectory = resolve('packages', target.directory);
mkdirSync(resolve(packageDirectory, 'bin'), { recursive: true });
copyFileSync(resolve('LICENSE'), resolve(packageDirectory, 'LICENSE'));
copyFileSync(resolve('THIRD_PARTY_LICENSES.md'), resolve(packageDirectory, 'THIRD_PARTY_LICENSES.md'));

for (const command of ['dovik', 'dovikd', 'gh']) {
  const output = resolve(packageDirectory, 'bin', `${command}${target.suffix}`);
  mkdirSync(dirname(output), { recursive: true });
  const result = spawnSync(
    'go',
    ['build', '-trimpath', '-ldflags=-s -w', '-o', output, `./cmd/${command}`],
    { shell: false, stdio: 'inherit', windowsHide: true },
  );
  if (result.error !== undefined) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}

process.stdout.write(`prepared ${packageDirectory}\n`);
