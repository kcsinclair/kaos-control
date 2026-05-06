import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as schedulerApi from '@/api/scheduler'
import { useUiStore } from '@/stores/ui'
import type { SchedulerJob, SchedulerRun } from '@/types/api'

export const useSchedulerStore = defineStore('scheduler', () => {
  const jobs = ref<SchedulerJob[]>([])
  const selectedJob = ref<SchedulerJob | null>(null)
  const runs = ref<SchedulerRun[]>([])
  const runsTotal = ref(0)
  const loadingJobs = ref(false)
  const loadingRuns = ref(false)
  const logCache = new Map<number, string>()

  async function fetchJobs(project: string): Promise<void> {
    loadingJobs.value = true
    try {
      const data = await schedulerApi.listJobs(project)
      jobs.value = data.jobs ?? []
    } finally {
      loadingJobs.value = false
    }
  }

  async function fetchJob(project: string, name: string): Promise<void> {
    const data = await schedulerApi.getJob(project, name)
    selectedJob.value = data.job
    runs.value = data.runs ?? []
  }

  async function createJob(
    project: string,
    payload: Omit<SchedulerJob, 'created_at' | 'updated_at' | 'next_run_at' | 'last_run_status' | 'last_run_at'>,
  ): Promise<SchedulerJob> {
    const data = await schedulerApi.createJob(project, payload)
    jobs.value.push(data.job)
    return data.job
  }

  async function updateJob(
    project: string,
    name: string,
    payload: Partial<Omit<SchedulerJob, 'name' | 'created_at' | 'updated_at'>>,
  ): Promise<SchedulerJob> {
    const data = await schedulerApi.updateJob(project, name, payload)
    _replaceJob(data.job)
    if (selectedJob.value?.name === name) selectedJob.value = data.job
    return data.job
  }

  async function deleteJob(project: string, name: string): Promise<void> {
    await schedulerApi.deleteJob(project, name)
    jobs.value = jobs.value.filter((j) => j.name !== name)
    if (selectedJob.value?.name === name) selectedJob.value = null
  }

  async function triggerJob(project: string, name: string): Promise<void> {
    const ui = useUiStore()
    await schedulerApi.triggerJob(project, name)
    ui.success(`Job "${name}" triggered`)
  }

  async function pauseJob(project: string, name: string): Promise<void> {
    const data = await schedulerApi.pauseJob(project, name)
    _replaceJob(data.job)
    if (selectedJob.value?.name === name) selectedJob.value = data.job
  }

  async function resumeJob(project: string, name: string): Promise<void> {
    const data = await schedulerApi.resumeJob(project, name)
    _replaceJob(data.job)
    if (selectedJob.value?.name === name) selectedJob.value = data.job
  }

  async function fetchRuns(project: string, jobName: string, page = 1, perPage = 20): Promise<void> {
    loadingRuns.value = true
    try {
      const data = await schedulerApi.listRuns(project, jobName, page, perPage)
      runs.value = data.runs ?? []
      runsTotal.value = data.total ?? 0
    } finally {
      loadingRuns.value = false
    }
  }

  async function fetchRunLog(project: string, jobName: string, runId: number): Promise<string> {
    if (logCache.has(runId)) return logCache.get(runId)!
    const text = await schedulerApi.getRunLog(project, jobName, runId)
    logCache.set(runId, text || '(empty log)')
    return logCache.get(runId)!
  }

  function onWsEvent(type: string, payload: Record<string, unknown>): void {
    const ui = useUiStore()
    const jobName = payload.job as string | undefined
    if (!jobName) return

    if (type === 'scheduler.job.started') {
      const idx = jobs.value.findIndex((j) => j.name === jobName)
      if (idx >= 0) {
        jobs.value[idx] = { ...jobs.value[idx], last_run_status: 'running' }
      }
      if (selectedJob.value?.name === jobName) {
        selectedJob.value = { ...selectedJob.value, last_run_status: 'running' }
      }
    } else if (type === 'scheduler.job.completed') {
      const status = payload.status as string | undefined
      const now = new Date().toISOString()
      const idx = jobs.value.findIndex((j) => j.name === jobName)
      if (idx >= 0) {
        jobs.value[idx] = {
          ...jobs.value[idx],
          last_run_status: (status as SchedulerJob['last_run_status']) ?? undefined,
          last_run_at: now,
        }
      }
      if (selectedJob.value?.name === jobName) {
        selectedJob.value = {
          ...selectedJob.value,
          last_run_status: (status as SchedulerJob['last_run_status']) ?? undefined,
          last_run_at: now,
        }
      }
      if (status === 'failure' || status === 'timeout') {
        ui.error(`Job "${jobName}" ${status === 'timeout' ? 'timed out' : 'failed'}`)
      } else if (status === 'success') {
        ui.success(`Job "${jobName}" completed successfully`)
      }
    }
  }

  function _replaceJob(updated: SchedulerJob): void {
    const idx = jobs.value.findIndex((j) => j.name === updated.name)
    if (idx >= 0) {
      jobs.value[idx] = updated
    }
  }

  return {
    jobs,
    selectedJob,
    runs,
    runsTotal,
    loadingJobs,
    loadingRuns,
    fetchJobs,
    fetchJob,
    createJob,
    updateJob,
    deleteJob,
    triggerJob,
    pauseJob,
    resumeJob,
    fetchRuns,
    fetchRunLog,
    onWsEvent,
  }
})
