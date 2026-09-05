export function createTerminalEnvironment(parentEnvironment, logEndpoint, daemonConfiguration = {}) {
  const environment = {
    ...parentEnvironment,
    TERM: "xterm-256color",
    COLORTERM: "truecolor",
    DOVIK_TUI_DEV_LOG_PIPE: logEndpoint,
    ...daemonConfiguration,
  };
  delete environment.NO_COLOR;
  return environment;
}
