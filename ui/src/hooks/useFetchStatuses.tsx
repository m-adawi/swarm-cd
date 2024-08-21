import { useEffect, useState } from "react"

export interface StackStatus {
  Name: string
  Error: string
  Revision: string
  RepoURL: string
}

const devData: StackStatus[] = [
  {
    Name: "Project Alpha",
    Error: "",
    Revision: "v1.0.1",
    RepoURL: "https://github.com/user/project-alpha"
  },
  {
    Name: "Project Beta",
    Error: "Failed to build",
    Revision: "v2.3.4",
    RepoURL: "https://github.com/user/project-beta"
  },
  {
    Name: "Project Gamma",
    Error: "",
    Revision: "v0.9.8",
    RepoURL: "https://github.com/user/project-gamma"
  }
]

async function fetchData(): Promise<StackStatus[]> {
  const response = await fetch("/stacks")
  if (!response.ok) {
    throw new Error("Network response was not ok")
  }

  return (await response.json()) as StackStatus[]
}

export default function useFetchStatuses(): {
  stacks: StackStatus[]
  error: string | null
} {
  const [statuses, setStatuses] = useState<StackStatus[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (import.meta.env.MODE === "development") {
      setStatuses(devData)
    } else {
      const updateStatuses: () => void = () => {
        fetchData()
          .then(stacks => setStatuses(stacks))
          .catch((err: unknown) => {
            if (err instanceof Error) {
              setError(err.message)
            } else {
              setError("An unknown error occurred")
            }
          })
      }
      updateStatuses() // initial fetch

      const intervalId = setInterval(updateStatuses, 5000)
      return () => clearInterval(intervalId)
    }
  }, [])

  return { stacks: statuses, error }
}
