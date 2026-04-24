import { api } from './client'
import type { MeResponse, User } from '@/types/api'

export function login(email: string, password: string) {
  return api.post<{ user: User }>('/auth/login', { email, password })
}

export function logout() {
  return api.post<void>('/auth/logout')
}

export function fetchMe() {
  return api.get<MeResponse>('/auth/me')
}
