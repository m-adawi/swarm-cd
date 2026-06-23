import { useEffect, useState } from "react"
import devServices from "./dummyStackServices.json"

export interface TaskStatus {
  ID: string
  Slot: number
  NodeID: string
  State: string
  DesiredState: string
  Error: string
  ContainerID: string
}

export interface ServiceStatus {
  ID: string
  Name: string
  Image: string
  Mode: string
  RunningTasks: number
  DesiredTasks: number
  FailedTasks: number
  Health: string
  Tasks: TaskStatus[]
  Secrets: string[]
  Configs: string[]
}

// Dev-mode fixture keyed by stack name (see dummyStackServices.json). The
// explicit type both documents the contract and fails the build if the JSON
// drifts from the ServiceStatus shape.
const dummyServicesByStack: Record<string, ServiceStatus[]> = devServices

async function fetchFromServer(stackName: string): Promise<ServiceStatus[]> {
  const response = await fetch(`/stacks/${encodeURIComponent(stackName)}/services`)
  if (!response.ok) {
    throw new Error("Network response was not ok")
  }

  return (await response.json()) as ServiceStatus[]
}

export default function useFetchStackServices(
  stackName: string | null,
  intervalMs = 5000
): { services: ServiceStatus[]; error: string | null; loading: boolean } {
  const [services, setServices] = useState<ServiceStatus[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (stackName === null) {
      setServices([])
      setError(null)
      setLoading(false)
      return
    }

    let cancelled = false

    const fetchServices = async (): Promise<void> => {
      console.debug("[useFetchStackServices] fetching", stackName)
      try {
        const data =
          import.meta.env.MODE === "development"
            ? (dummyServicesByStack[stackName] ?? [])
            : await fetchFromServer(stackName)
        if (!cancelled) {
          setServices(data)
          setError(null)
          console.debug("[useFetchStackServices] loaded", stackName, data.length)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "An unknown error occurred")
          console.debug("[useFetchStackServices] error", stackName, err)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    setLoading(true)
    void fetchServices() // initial fetch

    const intervalId = setInterval(() => {
      void fetchServices()
    }, intervalMs)

    return () => {
      cancelled = true
      clearInterval(intervalId)
    }
  }, [stackName, intervalMs])

  return { services, error, loading }
}
