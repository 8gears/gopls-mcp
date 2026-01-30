import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	site: 'https://github.com/xieyuschen/gopls-mcp',
	integrations: [
		starlight({
			title: 'gopls-mcp',
			description: 'Semantic Go code understanding for AI assistants',
			pagination: false,
			social: [
				{ label: 'GitHub', href: 'https://github.com/xieyuschen/gopls-mcp', icon: 'github' }
			],
			sidebar: [
				{
					label: 'Getting Started',
					autogenerate: { directory: 'quick-start' },
				},
				{
					label: 'Case Studies',
					autogenerate: { directory: 'case-studies' },
				},
				{
					label: 'Configuration',
					items: [
						{ label: 'Configuration', link: 'config' },
					],
				},
				{
					label: 'Reference',
					autogenerate: { directory: 'reference' },
				},
				{
					label: 'Benchmarks',
					autogenerate: { directory: 'benchmarks' },
				},
			],
		}),
	],
});