import { useEffect, useState } from "react"
import devData from "./dummyStackStatuses.json"

export interface StackStatus {
  Name: string
  Error: string
  Revision: string
  RepoURL: string
  Ref: string
}

async function fetchFromServer(): Promise<StackStatus[]> {
  const response = await fetch("/stacks")
  if (!response.ok) {
    throw new Error("Network response was not ok")
  }

  return (await response.json()) as StackStatus[]
}

export default function useFetchStatuses(intervalMs = 5000): {
  statuses: StackStatus[]
  error: string | null
} {
  const [statuses, setStatuses] = useState<StackStatus[]>([])
  const [error, setError] = useState<string | null>(null)

  const fetchStatuses = async (): Promise<void> => {
    try {
      const data = import.meta.env.MODE === "development" ? devData : await fetchFromServer()
      setStatuses(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "An unknown error occurred")
    }
  }

  useEffect(() => {
    void fetchStatuses() // initial fetch

    const intervalId = setInterval(fetchStatuses, intervalMs)
    return () => clearInterval(intervalId)
  }, [intervalMs])

  return { statuses, error }
}
