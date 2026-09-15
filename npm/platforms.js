const platformTargets = Object.freeze([
  Object.freeze({ archiveTarget: 'windows-x64', cpu: Object.freeze(['x64']), directory: 'win32-x64', executableSuffix: '.exe', os: Object.freeze(['win32']), packageName: '@maxiedev/dovik-win32-x64' }),
  Object.freeze({ archiveTarget: 'linux-x64', cpu: Object.freeze(['x64']), directory: 'linux-x64', executableSuffix: '', os: Object.freeze(['linux']), packageName: '@maxiedev/dovik-linux-x64' }),
  Object.freeze({ archiveTarget: 'macos-x64', cpu: Object.freeze(['x64']), directory: 'darwin-x64', executableSuffix: '', os: Object.freeze(['darwin']), packageName: '@maxiedev/dovik-darwin-x64' }),
  Object.freeze({ archiveTarget: 'macos-arm64', cpu: Object.freeze(['arm64']), directory: 'darwin-arm64', executableSuffix: '', os: Object.freeze(['darwin']), packageName: '@maxiedev/dovik-darwin-arm64' }),
]);

function selectPlatformTarget(platform = process.platform, architecture = process.arch) {
  return platformTargets.find((target) => target.os[0] === platform && target.cpu[0] === architecture);
}

module.exports = { platformTargets, selectPlatformTarget };
