import { render, screen } from "@testing-library/react"
import HealthFooter from "../../src/components/HealthFooter"
import { HealthData } from "../../src/hooks/useFetchHealth"

describe("HealthFooter", () => {
  const baseHealth: HealthData = {
    status: "healthy",
    version: "1.2.3",
    uptime_seconds: 3661,
    stacks_managed: 5,
    update_interval_seconds: 30,
    mutation_api_enabled: false,
  }

  it("should render version", () => {
    render(<HealthFooter health={baseHealth} />)
    expect(screen.getByText(/SwarmCD 1\.2\.3/)).toBeInTheDocument()
  })

  it("should render update interval", () => {
    render(<HealthFooter health={baseHealth} />)
    expect(screen.getByText("Poll: 30s")).toBeInTheDocument()
  })

  it("should render stack count with plural", () => {
    render(<HealthFooter health={baseHealth} />)
    expect(screen.getByText("5 stacks managed")).toBeInTheDocument()
  })

  it("should render stack count with singular", () => {
    render(<HealthFooter health={{ ...baseHealth, stacks_managed: 1 }} />)
    expect(screen.getByText("1 stack managed")).toBeInTheDocument()
  })

  it("should format uptime as hours and minutes", () => {
    render(<HealthFooter health={{ ...baseHealth, uptime_seconds: 3661 }} />)
    expect(screen.getByText("Uptime: 1h 1m")).toBeInTheDocument()
  })

  it("should format uptime as days and hours", () => {
    render(<HealthFooter health={{ ...baseHealth, uptime_seconds: 90061 }} />)
    expect(screen.getByText("Uptime: 1d 1h")).toBeInTheDocument()
  })

  it("should format uptime as minutes only", () => {
    render(<HealthFooter health={{ ...baseHealth, uptime_seconds: 300 }} />)
    expect(screen.getByText("Uptime: 5m")).toBeInTheDocument()
  })
})
