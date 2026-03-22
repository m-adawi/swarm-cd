import { useEffect, useState } from "react"

export interface HealthData {
  status: string
  version: string
  uptime_seconds: number
  stacks_managed: number
  update_interval_seconds: number
  mutation_api_enabled: boolean
  config_warnings?: string[]
}

export default function useFetchHealth(): {
  health: HealthData | null
  error: string | null
} {
  const [health, setHealth] = useState<HealthData | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchHealth = async (): Promise<void> => {
      try {
        const response = await fetch("/health")
        if (!response.ok) {
          throw new Error("Failed to fetch health")
        }
        setHealth((await response.json()) as HealthData)
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to fetch health")
      }
    }

    void fetchHealth()
    const intervalId = setInterval(fetchHealth, 30000)
    return () => clearInterval(intervalId)
  }, [])

  return { health, error }
}
