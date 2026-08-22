// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  getWizard,
  recommend as apiRecommend,
  listStacks as apiListStacks,
  saveWizardState,
  discardWizardState,
  commitWizard,
} from '@/api/architecture'
import { ApiError } from '@/api/client'
import type {
  CatalogItem,
  WizardAnswer,
  WizardBreakingReq,
  WizardCommitResult,
  WizardPriorRun,
  WizardQAPair,
  WizardQuestion,
  WizardRecommendation,
} from '@/types/api'

export type WizardPath = 'browse' | 'guided' | null

const PERSIST_DEBOUNCE_MS = 500

// catalogRelPath strips the "lifecycle/architecture/" prefix a CatalogItem.Path
// carries, since the commit/promote endpoints take paths relative to that
// directory (see internal/http/architecture_wizard.go findCatalogItem).
function catalogRelPath(item: CatalogItem): string {
  return item.path.replace(/^lifecycle\/architecture\//, '')
}

export const useArchitectureWizardStore = defineStore('architectureWizard', () => {
  const path = ref<WizardPath>(null)
  const step = ref('path')
  const questions = ref<WizardQuestion[]>([])
  const defaultArchitecture = ref('')
  const answers = ref<WizardAnswer[]>([])
  const skippedQuestionIds = ref<string[]>([])
  const recommendations = ref<WizardRecommendation[]>([])
  const droppedConstraints = ref<string[]>([])
  const chosenArchitecture = ref<CatalogItem | null>(null)
  const chosenStack = ref<CatalogItem | null>(null)
  const priorRun = ref<WizardPriorRun | null>(null)
  const priorRunResolved = ref(false)
  const commitResult = ref<WizardCommitResult | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  // Set true when ScaffoldStep emits `finish` (Skip / Finish, or post-run
  // Finish) — lets WizardSuccess suppress re-entering the scaffold step
  // (Milestone 4, FR-3).
  const scaffoldSettled = ref(false)

  let persistTimer: ReturnType<typeof setTimeout> | null = null
  let persistProject = ''

  const isPathChosen = computed(() => path.value !== null)
  const isArchitectureChosen = computed(() => chosenArchitecture.value !== null)
  const isStackChosen = computed(() => chosenStack.value !== null)
  const canCommit = computed(() => isArchitectureChosen.value && isStackChosen.value)

  function answerFor(questionId: string): string | undefined {
    return answers.value.find((a) => a.question_id === questionId)?.value
  }

  // The next question neither answered nor skipped this session (FR-7). On
  // resume, only `answers` survives in the persisted state (skipped
  // questions aren't tracked server-side), so a previously-skipped question
  // may be asked again — harmless, since skipping already meant "no strong
  // preference".
  const currentQuestion = computed(
    () =>
      questions.value.find(
        (q) => answerFor(q.id) === undefined && !skippedQuestionIds.value.includes(q.id),
      ) ?? null,
  )
  const answeredQuestionCount = computed(
    () => answers.value.length + skippedQuestionIds.value.length,
  )

  function applyError(e: unknown, fallback: string): void {
    error.value = e instanceof ApiError ? e.message : fallback
  }

  async function start(project: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const res = await getWizard(project)
      questions.value = res.questions
      defaultArchitecture.value = res.default_architecture
      priorRun.value = res.prior_run
      priorRunResolved.value = true
      if (res.resumable_state) {
        const st = res.resumable_state
        path.value = st.path
        answers.value = st.answers
        step.value = st.step
      }
    } catch (e: unknown) {
      applyError(e, 'Failed to load the architecture wizard.')
    } finally {
      loading.value = false
    }
  }

  function setPath(next: WizardPath): void {
    path.value = next
  }

  function setAnswer(questionId: string, value: string): void {
    skippedQuestionIds.value = skippedQuestionIds.value.filter((id) => id !== questionId)
    const existing = answers.value.find((a) => a.question_id === questionId)
    if (existing) {
      existing.value = value
    } else {
      answers.value.push({ question_id: questionId, value })
    }
  }

  function skip(questionId: string): void {
    answers.value = answers.value.filter((a) => a.question_id !== questionId)
    if (!skippedQuestionIds.value.includes(questionId)) {
      skippedQuestionIds.value.push(questionId)
    }
  }

  // Answer/skip the current question and autosave (resume, OQ-3). Returns
  // whether every question has now been answered or skipped.
  function answerCurrentQuestion(project: string, value: string): boolean {
    if (!currentQuestion.value) return true
    setAnswer(currentQuestion.value.id, value)
    persistState(project)
    return currentQuestion.value === null
  }

  function skipCurrentQuestion(project: string): boolean {
    if (!currentQuestion.value) return true
    skip(currentQuestion.value.id)
    persistState(project)
    return currentQuestion.value === null
  }

  async function fetchRecommendations(project: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const res = await apiRecommend(project, answers.value)
      // The API can send these as JSON null (Go nil slices) when empty;
      // default to [] so downstream `.length` access never throws.
      recommendations.value = res.recommendations ?? []
      droppedConstraints.value = res.dropped_constraints ?? []
    } catch (e: unknown) {
      applyError(e, 'Failed to compute recommendations.')
    } finally {
      loading.value = false
    }
  }

  function chooseArchitecture(item: CatalogItem): void {
    chosenArchitecture.value = item
    chosenStack.value = null
  }

  async function fetchStacks(project: string, language?: string): Promise<CatalogItem[]> {
    if (!chosenArchitecture.value) return []
    loading.value = true
    error.value = null
    try {
      return await apiListStacks(project, chosenArchitecture.value.slug, language)
    } catch (e: unknown) {
      applyError(e, 'Failed to load compatible stacks.')
      return []
    } finally {
      loading.value = false
    }
  }

  function chooseStack(item: CatalogItem): void {
    chosenStack.value = item
  }

  function persistState(project: string): void {
    if (!path.value) return
    persistProject = project
    if (persistTimer !== null) clearTimeout(persistTimer)
    persistTimer = setTimeout(() => {
      persistTimer = null
      void saveWizardState(persistProject, {
        path: path.value as 'browse' | 'guided',
        answers: answers.value,
        chosen_architecture: chosenArchitecture.value ? catalogRelPath(chosenArchitecture.value) : undefined,
        chosen_tech_stack: chosenStack.value ? catalogRelPath(chosenStack.value) : undefined,
        step: step.value,
        updated_unix: Math.floor(Date.now() / 1000),
      }).catch((e: unknown) => applyError(e, 'Failed to save wizard progress.'))
    }, PERSIST_DEBOUNCE_MS)
  }

  async function commit(
    project: string,
    opts: { breakingRequirements: WizardBreakingReq[]; qa: WizardQAPair[] },
  ): Promise<WizardCommitResult | null> {
    if (!chosenArchitecture.value || !chosenStack.value) {
      error.value = 'Choose an architecture and a tech stack before committing.'
      return null
    }
    loading.value = true
    error.value = null
    try {
      const result = await commitWizard(project, {
        architecture_path: catalogRelPath(chosenArchitecture.value),
        tech_stack_path: catalogRelPath(chosenStack.value),
        answers: answers.value,
        breaking_requirements: opts.breakingRequirements,
        qa: opts.qa,
      })
      commitResult.value = result
      return result
    } catch (e: unknown) {
      applyError(e, 'Failed to commit the architecture selection.')
      return null
    } finally {
      loading.value = false
    }
  }

  async function discardResumableState(project: string): Promise<void> {
    try {
      await discardWizardState(project)
    } catch (e: unknown) {
      applyError(e, 'Failed to discard saved wizard progress.')
    }
  }

  function reset(): void {
    path.value = null
    step.value = 'path'
    questions.value = []
    defaultArchitecture.value = ''
    answers.value = []
    skippedQuestionIds.value = []
    recommendations.value = []
    droppedConstraints.value = []
    chosenArchitecture.value = null
    chosenStack.value = null
    priorRun.value = null
    priorRunResolved.value = false
    commitResult.value = null
    scaffoldSettled.value = false
    loading.value = false
    error.value = null
    if (persistTimer !== null) {
      clearTimeout(persistTimer)
      persistTimer = null
    }
  }

  return {
    path,
    step,
    questions,
    defaultArchitecture,
    answers,
    skippedQuestionIds,
    recommendations,
    droppedConstraints,
    chosenArchitecture,
    chosenStack,
    priorRun,
    priorRunResolved,
    commitResult,
    scaffoldSettled,
    loading,
    error,
    isPathChosen,
    isArchitectureChosen,
    isStackChosen,
    canCommit,
    currentQuestion,
    answeredQuestionCount,
    answerFor,
    start,
    setPath,
    setAnswer,
    skip,
    answerCurrentQuestion,
    skipCurrentQuestion,
    fetchRecommendations,
    chooseArchitecture,
    fetchStacks,
    chooseStack,
    persistState,
    commit,
    discardResumableState,
    reset,
  }
})
