import { readFileSync } from 'fs';

const md = readFileSync(new URL('./flowchart.md', import.meta.url), 'utf8');
const match = md.match(/```mermaid\n([\s\S]*?)```/);
if (!match) {
  console.error('NO MERMAID BLOCK FOUND');
  process.exit(1);
}
const diagram = match[1];

const { default: mermaid } = await import('https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs');
mermaid.initialize({ startOnLoad: false });
try {
  const res = await mermaid.parse(diagram);
  console.log('MERMAID PARSE OK, empty?', res);
} catch (e) {
  console.error('MERMAID PARSE ERROR:');
  console.error(e.message || e);
  process.exit(1);
}
