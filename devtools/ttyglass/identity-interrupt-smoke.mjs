import { execFile, spawn as spawnProcess } from 'node:child_process';
import { mkdtemp, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { promisify } from 'node:util';

const run = promisify(execFile);
const root = path.resolve(import.meta.dirname, '..', '..');
const suffix = process.platform === 'win32' ? '.exe' : '';
const dovik = path.join(root, 'build', 'identity-interrupt', `dovik${suffix}`);
const daemon = path.join(root, 'build', 'identity-interrupt', `dovikd${suffix}`);
const ttyglass = path.join(import.meta.dirname, 'node_modules', 'ttyglass', 'dist', 'cli.js');
const temporary = await mkdtemp(path.join(tmpdir(), 'dovik-identity-interrupt-'));
const repository = path.join(temporary, 'project');
const registry = path.join(temporary, 'registry.json');
const endpoint = process.platform === 'win32'
  ? `\\\\.\\pipe\\dovik-ttyglass-${process.pid}`
  : path.join(temporary, 'control.sock');
await mkdir(repository, { recursive: true });

const environment = {
  ...process.env,
  DOVIK_TTYGLASS_CONTROL_ENDPOINT: endpoint,
  DOVIK_TTYGLASS_REGISTRY_PATH: registry,
};

async function dovikCommand(args) {
  return run(dovik, ['--endpoint', endpoint, ...args], { windowsHide: true });
}

async function ttyglassCommand(args) {
  return run(process.execPath, [ttyglass, ...args], { windowsHide: true });
}

async function waitForScreen(sessionId, text) {
  const deadline = Date.now() + 15_000;
  while (Date.now() < deadline) {
    const screen = JSON.parse((await ttyglassCommand(['screen', sessionId, '--json'])).stdout);
    if (screen.lines.join('\n').includes(text)) return;
    const status = JSON.parse((await ttyglassCommand(['status', sessionId, '--json'])).stdout);
    if (['exited', 'failed', 'stopped'].includes(status.state)) {
      throw new Error(`questionnaire exited before showing ${text}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`questionnaire did not show ${text}`);
}

let sessionId = '';
try {
  await run('git', ['init', '--quiet', repository], { windowsHide: true });
  const daemonProcess = spawnProcess(daemon, [], {
    env: environment,
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  });
  let daemonOutput = '';
  daemonProcess.stdout.on('data', (chunk) => { daemonOutput += chunk; });
  daemonProcess.stderr.on('data', (chunk) => { daemonOutput += chunk; });

  try {
    const deadline = Date.now() + 10_000;
    while (true) {
      try {
        await dovikCommand(['--json', 'project', 'list']);
        break;
      } catch {
        if (Date.now() >= deadline) throw new Error(`daemon did not become ready: ${daemonOutput}`);
        await new Promise((resolve) => setTimeout(resolve, 100));
      }
    }

    const started = JSON.parse((await ttyglassCommand([
      'start', '--json', '--name', 'Dovik identity interrupt', '--cwd', repository,
      '--', dovik, '--endpoint', endpoint, 'project', 'configure', '--root', repository,
    ])).stdout);
    sessionId = started.sessionId;
    await waitForScreen(sessionId, 'Project ID');
    await ttyglassCommand(['input', sessionId, '--base64', 'Aw==']);

    const exitDeadline = Date.now() + 10_000;
    let status;
    while (Date.now() < exitDeadline) {
      status = JSON.parse((await ttyglassCommand(['status', sessionId, '--json'])).stdout);
      if (['exited', 'failed', 'stopped'].includes(status.state)) break;
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
    if (!status || !['exited', 'failed', 'stopped'].includes(status.state)) {
      throw new Error('questionnaire did not exit after Ctrl+C');
    }
    if (status.exitCode !== 0) throw new Error(`Ctrl+C exited with code ${status.exitCode}`);

    const projects = JSON.parse((await dovikCommand(['--json', 'project', 'list'])).stdout);
    const identities = JSON.parse((await dovikCommand(['--json', 'identity', 'list'])).stdout);
    if (projects.length !== 0 || identities.state.projects.length !== 0) {
      throw new Error('Ctrl+C changed project or identity state');
    }
    console.log('ttyglass Ctrl+C exited cleanly and left project and identity state unchanged');
  } finally {
    if (sessionId) {
      try { await ttyglassCommand(['stop', sessionId]); } catch {}
    }
    if (daemonProcess.exitCode === null) {
      const exited = new Promise((resolve) => daemonProcess.once('exit', resolve));
      daemonProcess.kill();
      await exited;
    }
  }
} finally {
  await rm(temporary, { recursive: true, force: true });
}
