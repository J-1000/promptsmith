import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'PromptSmith',
  description: 'Version control and testing for LLM prompts',
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'CLI Reference', link: '/cli-reference' },
      { text: 'Web UI', link: '/web-ui' },
      { text: 'API Reference', link: '/api-reference' },
    ],
    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Getting Started', link: '/getting-started' },
          { text: 'CLI Reference', link: '/cli-reference' },
          { text: 'Web UI', link: '/web-ui' },
          { text: 'API Reference', link: '/api-reference' },
          { text: 'Contributing', link: '/contributing' },
        ],
      },
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/promptsmith/promptsmith' },
    ],
  },
})
