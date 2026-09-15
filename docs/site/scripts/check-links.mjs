import { access, readdir, readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const siteRoot = path.resolve(import.meta.dirname, "..");
const outputRoot = path.join(siteRoot, "dist");
const basePath = "/dovik";
const failures = [];

async function collectHtmlFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await collectHtmlFiles(entryPath)));
    } else if (entry.isFile() && entry.name.endsWith(".html")) {
      files.push(entryPath);
    }
  }

  return files;
}

function targetPath(href) {
  const url = new URL(href, "https://maxie.dev");
  if (url.origin !== "https://maxie.dev" || !url.pathname.startsWith(`${basePath}/`)) {
    return null;
  }

  const relativePath = decodeURIComponent(url.pathname.slice(basePath.length + 1));
  if (relativePath === "") {
    return path.join(outputRoot, "index.html");
  }
  if (path.extname(relativePath)) {
    return path.join(outputRoot, relativePath);
  }
  return path.join(outputRoot, relativePath, "index.html");
}

for (const htmlFile of await collectHtmlFiles(outputRoot)) {
  const html = await readFile(htmlFile, "utf8");
  const hrefs = html.matchAll(/<a\b[^>]*\shref=["']([^"'#?]+)(?:[?#][^"']*)?["'][^>]*>/g);

  for (const [, href] of hrefs) {
    const target = targetPath(href);
    if (!target) {
      continue;
    }

    try {
      await access(target);
    } catch {
      failures.push(`${path.relative(siteRoot, htmlFile)} -> ${href}`);
    }
  }
}

if (failures.length > 0) {
  console.error("Broken internal documentation links:");
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exitCode = 1;
} else {
  console.log("All internal documentation links resolve.");
}
