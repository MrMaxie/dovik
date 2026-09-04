export function createTerminalEnvironment(parentEnvironment, logEndpoint) {
  const environment = {
    ...parentEnvironment,
    TERM: "xterm-256color",
    COLORTERM: "truecolor",
    DOVIK_TUI_DEV_LOG_PIPE: logEndpoint,
  };
  delete environment.NO_COLOR;
  return environment;
}
