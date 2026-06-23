import type { IconType } from "react-icons"
import { FaExclamationTriangle, FaHeart, FaHeartBroken, FaQuestionCircle } from "react-icons/fa"

// Centralized ArgoCD-style health → Chakra color + icon mapping, shared by the
// graph nodes and the legend.

export type HealthLevel = "healthy" | "progressing" | "degraded"

export const HEALTH_ORDER: HealthLevel[] = ["healthy", "progressing", "degraded"]

export const HEALTH_LABELS: Record<HealthLevel, string> = {
  healthy: "Healthy",
  progressing: "Progressing",
  degraded: "Degraded"
}

// healthColor returns a Chakra color token for a health string. Unknown values
// fall back to a neutral gray so new backend states still render.
export function healthColor(health: string): string {
  switch (health) {
    case "healthy":
      return "green.400"
    case "progressing":
      return "yellow.400"
    case "degraded":
      return "red.400"
    default:
      return "gray.400"
  }
}

// taskStateColor maps a Swarm task state to a Chakra color: running is green,
// failure states are red, terminal/idle states are gray, and transitional
// states (pending/starting/...) are yellow.
export function taskStateColor(state: string): string {
  switch (state) {
    case "running":
      return "green.400"
    case "failed":
    case "rejected":
    case "orphaned":
      return "red.400"
    case "shutdown":
    case "complete":
      return "gray.400"
    default:
      return "yellow.400"
  }
}

// healthIcon returns the react-icons component for a health string — a heart for
// healthy services (per the ArgoCD-style design), and warning/broken-heart for
// the unhealthy states.
export function healthIcon(health: string): IconType {
  switch (health) {
    case "healthy":
      return FaHeart
    case "progressing":
      return FaExclamationTriangle
    case "degraded":
      return FaHeartBroken
    default:
      return FaQuestionCircle
  }
}
