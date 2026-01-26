import { readFileSync, writeFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Read the source reference.md
const sourcePath = resolve(__dirname, '../../gopls/mcpbridge/core/reference.md');
const fileContent = readFileSync(sourcePath, 'utf-8');

// Skip the header (first 7 lines) to get just the tool list content
const lines = fileContent.split('\n');
const referenceContent = lines.slice(7).join('\n');

// Output file (the .md file)
const outputPath = resolve(__dirname, '../src/content/docs/reference/index.md');

// Read the existing .mdx file
const mdxContent = readFileSync(outputPath, 'utf-8');

// Replace the placeholder with the actual markdown content
const updatedContent = mdxContent.replace(
  /<!-- REFERENCE_CONTENT -->/,
  referenceContent
);

writeFileSync(outputPath, updatedContent, 'utf-8');

console.log('✅ Generated tool reference documentation');
