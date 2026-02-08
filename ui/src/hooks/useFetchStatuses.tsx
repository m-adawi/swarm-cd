import { useCallback, useEffect, useState } from "react"
import devData from "./dummyStackStatuses.json"

export interface StackStatus {
  Name: string
  Error: string
  Revision: string
  RepoURL: string
}

async function fetchFromServer(): Promise<StackStatus[]> {
  const response = await fetch("/stacks")
  if (!response.ok) {
    throw new Error("Network response was not ok")
  }

  return (await response.json()) as StackStatus[]
}

function statusesEqual(a: StackStatus[], b: StackStatus[]): boolean {
  if (a.length !== b.length) return false
  const aJson = JSON.stringify(a.sort((x, y) => x.Name.localeCompare(y.Name)))
  const bJson = JSON.stringify(b.sort((x, y) => x.Name.localeCompare(y.Name)))
  return aJson === bJson
}

export interface UseFetchStatusesResult {
  statuses: StackStatus[]
  error: string | null
  hasUpdate: boolean
  checkForUpdate: () => Promise<void>
  applyUpdate: () => void
  isChecking: boolean
}

export default function useFetchStatuses(): UseFetchStatusesResult {
  const [statuses, setStatuses] = useState<StackStatus[]>([])
  const [pendingStatuses, setPendingStatuses] = useState<StackStatus[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isChecking, setIsChecking] = useState(false)

  const fetchStatuses = useCallback(async (): Promise<StackStatus[]> => {
    const data = import.meta.env.MODE === "development" ? devData : await fetchFromServer()
    return data
  }, [])

  // Initial fetch on mount
  useEffect(() => {
    const initialFetch = async (): Promise<void> => {
      try {
        const data = await fetchStatuses()
        setStatuses(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : "An unknown error occurred")
      }
    }
    void initialFetch()
  }, [fetchStatuses])

  const checkForUpdate = useCallback(async (): Promise<void> => {
    setIsChecking(true)
    try {
      const data = await fetchStatuses()
      if (!statusesEqual(data, statuses)) {
        setPendingStatuses(data)
      } else {
        setPendingStatuses(null)
      }
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : "An unknown error occurred")
    } finally {
      setIsChecking(false)
    }
  }, [fetchStatuses, statuses])

  const applyUpdate = useCallback((): void => {
    if (pendingStatuses !== null) {
      setStatuses(pendingStatuses)
      setPendingStatuses(null)
    }
  }, [pendingStatuses])

  return {
    statuses,
    error,
    hasUpdate: pendingStatuses !== null,
    checkForUpdate,
    applyUpdate,
    isChecking
  }
}
