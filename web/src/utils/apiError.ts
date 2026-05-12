import { isAxiosError } from 'axios'

/** 后端常见错误响应体：`{ message: string }` */
function messageFromResponseData(data: unknown): string | undefined {
  if (data === null || typeof data !== 'object') {
    return undefined
  }
  const msg = 'message' in data ? (data as { message: unknown }).message : undefined
  if (typeof msg === 'string' && msg.length > 0) {
    return msg
  }
  return undefined
}

export function getApiErrorMessage(err: unknown, fallback: string): string {
  if (isAxiosError(err)) {
    const fromBody = messageFromResponseData(err.response?.data)
    if (fromBody !== undefined) {
      return fromBody
    }
    const m = err.message
    if (typeof m === 'string' && m.length > 0) {
      return m
    }
    return fallback
  }
  if (err instanceof Error && err.message) {
    return err.message
  }
  return fallback
}
