#!/usr/bin/env node
import { createHash } from "node:crypto";
import {
  accessSync,
  chmodSync,
  constants,
  copyFileSync,
  createWriteStream,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  rmSync
} from "node:fs";
import https from "node:https";
import os from "node:os";
import path from "node:path";
import { spawn, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const args = process.argv.slice(2);
const repo = process.env.BASELINE_RELEASE_REPO || "apollostreetcompany/baseline";
const version = process.env.BASELINE_VERSION || "latest";
const selfPath = safeRealpath(process.argv[1]);

function safeRealpath(candidate) {
  try {
    return realpathSync(candidate);
  } catch {
    return candidate;
  }
}

function isExecutable(candidate) {
  try {
    accessSync(candidate, constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

function platformAsset() {
  const platform = os.platform();
  const arch = os.arch();
  const osName = platform === "darwin" ? "Darwin" : platform === "linux" ? "Linux" : "";
  const archName = arch === "arm64" ? "arm64" : arch === "x64" ? "x86_64" : "";
  if (!osName || !archName) {
    throw new Error(`unsupported platform for automatic Baseline download: ${platform}/${arch}`);
  }
  return `baseline_${osName}_${archName}.tar.gz`;
}

function releaseBaseURL() {
  if (version === "latest") {
    return `https://github.com/${repo}/releases/latest/download`;
  }
  return `https://github.com/${repo}/releases/download/${version}`;
}

function findBaselineOnPath() {
  const candidates = [];
  if (process.env.BASELINE_BIN) candidates.push(process.env.BASELINE_BIN);
  const localDevBin = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..", "bin", "baseline");
  candidates.push(localDevBin);

  for (const dir of (process.env.PATH || "").split(path.delimiter)) {
    if (!dir) continue;
    candidates.push(path.join(dir, process.platform === "win32" ? "baseline.exe" : "baseline"));
  }

  for (const candidate of candidates) {
    if (!candidate || !existsSync(candidate) || !isExecutable(candidate)) continue;
    if (safeRealpath(candidate) === selfPath) continue;
    return candidate;
  }
  return "";
}

function managedBinaryPath() {
  return path.join(os.homedir(), ".cache", "baseline-ai", "bin", `${os.platform()}-${os.arch()}`, "baseline");
}

function download(url, destination, redirects = 0) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, (response) => {
      if (response.statusCode && response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        if (redirects > 5) {
          reject(new Error(`too many redirects while downloading ${url}`));
          return;
        }
        const next = new URL(response.headers.location, url).toString();
        download(next, destination, redirects + 1).then(resolve, reject);
        return;
      }

      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`download failed for ${url}: HTTP ${response.statusCode}`));
        return;
      }

      const file = createWriteStream(destination);
      response.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    });
    request.on("error", reject);
  });
}

async function installManagedBinary() {
  if (process.env.BASELINE_SKIP_DOWNLOAD === "1") {
    throw new Error("BASELINE_SKIP_DOWNLOAD=1 and no baseline binary was found");
  }

  const asset = platformAsset();
  const baseURL = releaseBaseURL();
  const tmp = mkdtempSync(path.join(os.tmpdir(), "baseline-npm-"));
  const archive = path.join(tmp, asset);
  const checksumsPath = path.join(tmp, "checksums.txt");
  const output = managedBinaryPath();

  try {
    console.error(`Downloading Baseline CLI ${version} for ${os.platform()}/${os.arch()}...`);
    await download(`${baseURL}/${asset}`, archive);
    await download(`${baseURL}/checksums.txt`, checksumsPath);

    const checksums = readFileSync(checksumsPath, "utf8");
    const line = checksums.split(/\r?\n/).find((entry) => entry.trim().endsWith(` ${asset}`));
    if (!line) throw new Error(`checksum entry missing for ${asset}`);
    const expected = line.trim().split(/\s+/)[0];
    const actual = createHash("sha256").update(readFileSync(archive)).digest("hex");
    if (actual !== expected) throw new Error(`checksum mismatch for ${asset}`);

    const tar = spawnSync("tar", ["-xzf", archive, "-C", tmp], { stdio: "inherit" });
    if (tar.status !== 0) throw new Error("tar extraction failed");

    mkdirSync(path.dirname(output), { recursive: true });
    copyFileSync(path.join(tmp, "baseline"), output);
    chmodSync(output, 0o755);
    return output;
  } finally {
    rmSync(tmp, { recursive: true, force: true });
  }
}

function run(binary) {
  const child = spawn(binary, args, { stdio: "inherit" });
  child.on("exit", (code, signal) => {
    if (signal) process.kill(process.pid, signal);
    process.exit(code ?? 1);
  });
  child.on("error", (error) => {
    console.error(error.message);
    process.exit(1);
  });
}

let binary = findBaselineOnPath();
if (!binary) {
  const managed = managedBinaryPath();
  binary = existsSync(managed) && isExecutable(managed) ? managed : "";
}

if (!binary) {
  try {
    binary = await installManagedBinary();
  } catch (error) {
    console.error(`Baseline CLI is not installed: ${error instanceof Error ? error.message : String(error)}`);
    console.error("Try: curl -fsSL https://trackbaseline.com/install.sh | sh");
    process.exit(1);
  }
}

run(binary);
