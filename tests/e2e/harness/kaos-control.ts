import { spawn } from 'node:child_process'
import { mkdtemp, mkdir, writeFile, cp, rm } from 'node:fs/promises'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { existsSync } from 'node:fs'
import { execSync } from 'node:child_process'

export interface KcTestInstance {
  baseURL: string
  kcHomeDir: string
  projectRoot: string
  kill: () => Promise<void>
}

function findRepoRoot(): string {
  let dir = dirname(fileURLToPath(import.meta.url))
  for (let i = 0; i < 10; i++) {
    if (existsSync(join(dir, 'go.mod'))) return dir
    const parent = dirname(dir)
    if (parent === dir) break
    dir = parent
  }
  throw new Error('Could not find repo root (no go.mod found)')
}

async function findFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = createServer()
    server.listen(0, '127.0.0.1', () => {
      const addr = server.address() as { port: number }
      server.close(() => resolve(addr.port))
    })
    server.on('error', reject)
  })
}

async function waitForHealth(baseURL: string, timeoutMs = 10_000): Promise<void> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseURL}/api/health`)
      if (res.ok) return
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 200))
  }
  throw new Error(`kaos-control did not become healthy within ${timeoutMs}ms at ${baseURL}`)
}

export async function spawnKaosControl(): Promise<KcTestInstance> {
  const repoRoot = findRepoRoot()
  const binaryPath = join(repoRoot, 'dist', 'kaos-control')

  // Build if binary is missing
  if (!existsSync(binaryPath)) {
    execSync('make build', { cwd: repoRoot, stdio: 'inherit' })
  }

  // Create temp directories
  const kcHomeDir = await mkdtemp(join(tmpdir(), 'kc-home-'))
  const projectRoot = await mkdtemp(join(tmpdir(), 'kc-proj-'))

  try {
    // Copy fixtures into project root
    const fixturesDir = join(dirname(fileURLToPath(import.meta.url)), '..', 'fixtures', 'lifecycle')
    await cp(fixturesDir, join(projectRoot, 'lifecycle'), { recursive: true })

    // Initialise git repo with initial commit
    const gitEnv = {
      ...process.env,
      GIT_AUTHOR_NAME: 'Test Harness',
      GIT_AUTHOR_EMAIL: 'test@kaos-e2e.local',
      GIT_COMMITTER_NAME: 'Test Harness',
      GIT_COMMITTER_EMAIL: 'test@kaos-e2e.local',
    }
    execSync('git init -b main && git add -A && git commit -m "Initial fixture commit"', {
      cwd: projectRoot,
      env: gitEnv,
      stdio: 'pipe',
    })

    // Find a free port
    const port = await findFreePort()

    // Prepare home dir layout
    const projectsDir = join(kcHomeDir, 'projects')
    const dataDir = join(kcHomeDir, 'data')
    await mkdir(projectsDir, { recursive: true })
    await mkdir(dataDir, { recursive: true })

    // Write app config
    const configPath = join(kcHomeDir, 'config.yaml')
    await writeFile(
      configPath,
      [
        `server:`,
        `  listen: "127.0.0.1:${port}"`,
        `auth:`,
        `  method: local`,
        `  session_ttl: 24h`,
        `projects_dir: ${projectsDir}`,
        `data_dir: ${dataDir}`,
      ].join('\n') + '\n',
    )

    // Register the test project
    await writeFile(
      join(projectsDir, 'testproject.yaml'),
      [
        `name: testproject`,
        `path: ${projectRoot}`,
        `owner: admin@kaos-e2e.local`,
        `description: E2E smoke test project`,
      ].join('\n') + '\n',
    )

    // Spawn the server
    const proc = spawn(binaryPath, ['-config', configPath], {
      env: { ...process.env, LOG_LEVEL: 'warn' },
    })

    const stdoutChunks: Buffer[] = []
    const stderrChunks: Buffer[] = []
    proc.stdout?.on('data', (d: Buffer) => stdoutChunks.push(d))
    proc.stderr?.on('data', (d: Buffer) => stderrChunks.push(d))

    const baseURL = `http://127.0.0.1:${port}`

    try {
      await waitForHealth(baseURL)
    } catch (err) {
      proc.kill('SIGKILL')
      const stdout = Buffer.concat(stdoutChunks).toString()
      const stderr = Buffer.concat(stderrChunks).toString()
      throw new Error(`Server startup failed:\nstdout: ${stdout}\nstderr: ${stderr}\n${err}`)
    }

    const kill = async (): Promise<void> => {
      proc.kill('SIGTERM')
      await new Promise<void>((resolve) => {
        const timer = setTimeout(() => {
          proc.kill('SIGKILL')
          resolve()
        }, 5_000)
        proc.on('exit', () => {
          clearTimeout(timer)
          resolve()
        })
      })
      await rm(kcHomeDir, { recursive: true, force: true })
      await rm(projectRoot, { recursive: true, force: true })
    }

    return { baseURL, kcHomeDir, projectRoot, kill }
  } catch (err) {
    // Cleanup on failure
    await rm(kcHomeDir, { recursive: true, force: true }).catch(() => {})
    await rm(projectRoot, { recursive: true, force: true }).catch(() => {})
    throw err
  }
}
