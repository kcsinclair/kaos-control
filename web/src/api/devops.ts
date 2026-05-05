import { api } from '@/api/client'

export interface PipelineStep {
  name: string
  description: string
}

export interface Pipeline {
  slug: string
  name: string
  type: string
  steps: PipelineStep[]
}

export interface PipelinesResponse {
  pipelines: Pipeline[]
}

export interface RunPipelineResponse {
  run_id: string
}

export function listPipelines(project: string): Promise<PipelinesResponse> {
  return api.get<PipelinesResponse>(`/p/${encodeURIComponent(project)}/devops/pipelines`)
}

export function runPipeline(project: string, slug: string): Promise<RunPipelineResponse> {
  return api.post<RunPipelineResponse>(`/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}/run`)
}

export function cancelPipeline(project: string, slug: string): Promise<void> {
  return api.post<void>(`/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}/cancel`)
}

export function getRunLog(project: string, runId: string): Promise<string> {
  return api.get<string>(`/p/${encodeURIComponent(project)}/devops/runs/${encodeURIComponent(runId)}`)
}
