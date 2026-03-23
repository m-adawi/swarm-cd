import { render, screen } from "@testing-library/react"
import StatusCard from "../../src/components/StatusCard"
import formatTimestamp from "../../src/formatTimestamp"

describe("StatusCard", () => {
  const status = { name: "Some Name Here", revision: "3.76.1", repoURL: "https://www.github.com/1234", refType: "branch", refValue: "main" }

  it("should render name, revision, and repoURL properties", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={null} lastDeployedAt={null} />)

    expect(screen.getByText(status.name)).toBeInTheDocument()
    expect(screen.getByText(status.revision)).toBeInTheDocument()
    expect(screen.getByText(status.repoURL)).toBeInTheDocument()
  })

  it("should render ref as 'type: value'", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={null} lastDeployedAt={null} />)

    expect(screen.getByText("branch: main")).toBeInTheDocument()
  })

  it("should render repoURL as a link", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={null} lastDeployedAt={null} />)

    const repoUrlElement = screen.getByRole("link", { name: status.repoURL })
    expect(repoUrlElement).toBeInTheDocument()
    expect(repoUrlElement).toHaveAttribute("href", status.repoURL)
  })

  it("should not render error if it is empty", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={null} lastDeployedAt={null} />)

    const errorText = screen.queryByText(/error/i)
    expect(errorText).not.toBeInTheDocument()
  })

  it("should render error if it is not empty", () => {
    render(<StatusCard name={status.name} error={"Oh no!"} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={null} lastDeployedAt={null} />)

    const errorText = screen.queryByText(/error/i)
    expect(errorText).toBeInTheDocument()
  })

  it("should not render timestamps when null", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={null} lastDeployedAt={null} />)

    expect(screen.queryByText("Changed:")).not.toBeInTheDocument()
    expect(screen.queryByText("Deployed:")).not.toBeInTheDocument()
  })

  it("should render timestamps when provided", () => {
    const now = new Date()
    const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString()
    const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000).toISOString()

    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} refType={status.refType} refValue={status.refValue} lastChangeAt={twoHoursAgo} lastDeployedAt={fiveMinutesAgo} />)

    expect(screen.getByText("Changed:")).toBeInTheDocument()
    expect(screen.getByText("2h ago")).toBeInTheDocument()
    expect(screen.getByText("Deployed:")).toBeInTheDocument()
    expect(screen.getByText("5m ago")).toBeInTheDocument()
  })
})

describe("formatTimestamp", () => {
  it("should return 'just now' for recent timestamps", () => {
    const now = new Date().toISOString()
    expect(formatTimestamp(now)).toBe("just now")
  })

  it("should return minutes ago", () => {
    const tenMinutesAgo = new Date(Date.now() - 10 * 60 * 1000).toISOString()
    expect(formatTimestamp(tenMinutesAgo)).toBe("10m ago")
  })

  it("should return hours ago", () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString()
    expect(formatTimestamp(threeHoursAgo)).toBe("3h ago")
  })

  it("should return days ago", () => {
    const twoDaysAgo = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString()
    expect(formatTimestamp(twoDaysAgo)).toBe("2d ago")
  })
})
