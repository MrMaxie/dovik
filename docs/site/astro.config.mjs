import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://maxie.dev/',
  base: '/dovik',
  trailingSlash: 'always',
  compressHTML: true,
  devToolbar: { enabled: false },
  integrations: [
    starlight({
      title: 'Dovik',
      description: 'Dovik supervises local development processes through one daemon, CLI, TUI, and MCP control plane.',
      logo: {
        src: '../../assets/brand/dovik-mark-square.png',
        alt: 'Dovik logo',
      },
      favicon: 'favicon.png',
      customCss: ['./src/styles/dovik.css'],
      components: {
        Footer: './src/components/DovikFooter.astro',
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/MrMaxie/dovik' },
      ],
      sidebar: [
        {
          label: 'Start',
          items: [
            { label: 'Overview', slug: 'index' },
            { label: 'Getting started', slug: 'quick-start/getting-started' },
          ],
        },
        {
          label: 'Use Dovik',
          items: [
            { label: 'Daily workflow', slug: 'guides/daily-workflow' },
            { label: 'CLI and TUI', slug: 'guides/cli-and-tui' },
            { label: 'MCP and agents', slug: 'guides/mcp-and-agents' },
            { label: 'Identity and gh', slug: 'guides/identity' },
          ],
        },
        {
          label: 'Platforms',
          items: [
            { label: 'Windows', slug: 'platforms/windows' },
            { label: 'Linux', slug: 'platforms/linux' },
            { label: 'macOS', slug: 'platforms/macos' },
            { label: 'Docker and Podman', slug: 'platforms/containers' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Security and privacy', slug: 'reference/security' },
            { label: 'Updates', slug: 'reference/updates' },
            { label: 'Releases', slug: 'reference/releases' },
          ],
        },
      ],
    }),
  ],
});
