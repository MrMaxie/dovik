const { spawn } = require('node:child_process');
const { createRequire } = require('node:module');
const { dirname, resolve } = require('node:path');
const { selectPlatformTarget } = require('./platforms.js');

module.exports = function runNative(command) {
  const target = selectPlatformTarget(process.platform, process.arch);
  if (target === undefined) {
    process.stderr.write(`dovik: unsupported platform ${process.platform}/${process.arch}\n`);
    process.exitCode = 1;
    return;
  }

  const requireFromPackage = createRequire(__filename);
  let manifestPath;
  try {
    manifestPath = requireFromPackage.resolve(`${target.packageName}/package.json`);
  } catch {
    process.stderr.write(
      `dovik: the native package ${target.packageName} is missing. Reinstall dovik without --no-optional.\n`,
    );
    process.exitCode = 1;
    return;
  }

  const executable = resolve(dirname(manifestPath), 'bin', `${command}${target.executableSuffix}`);
  const child = spawn(executable, process.argv.slice(2), {
    stdio: 'inherit',
    windowsHide: true,
  });
  child.once('error', (error) => {
    process.stderr.write(`dovik: could not start ${command}: ${error.message}\n`);
    process.exitCode = 1;
  });
  child.once('exit', (code, signal) => {
    if (signal !== null && process.platform !== 'win32') {
      process.kill(process.pid, signal);
      return;
    }
    process.exitCode = code ?? 1;
  });
};
