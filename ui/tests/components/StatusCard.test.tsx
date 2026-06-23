import { render, screen } from "@testing-library/react"
import StatusCard from "../../src/components/StatusCard"

describe("StatusCard", () => {
  const status = { name: "Some Name Here", revision: "3.76.1", repoURL: "https://www.github.com/1234" }

  it("should render name, revision, and repoURL properties", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} onSelect={() => {}} />)

    for (const value of Object.values(status)) {
      const valueElement = screen.getByText(new RegExp(value, "i"))
      expect(valueElement).toBeInTheDocument()
    }
  })

  it("should render repoURL as a link", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} onSelect={() => {}} />)

    const repoUrlElement = screen.getByRole("link", { name: status.repoURL })
    expect(repoUrlElement).toBeInTheDocument()
    expect(repoUrlElement).toHaveAttribute("href", status.repoURL)
  })

  it("should not render error if it is empty", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} onSelect={() => {}} />)

    const errorText = screen.queryByText(/error/i)
    expect(errorText).not.toBeInTheDocument()
  })

  it("should render error if it is not empty", () => {
    render(<StatusCard name={status.name} error={"Oh no!"} revision={status.revision} repoURL={status.repoURL} onSelect={() => {}} />)

    const errorText = screen.queryByText(/error/i)
    expect(errorText).toBeInTheDocument()
  })

  it("shows a healthy status icon when there is no error", () => {
    render(<StatusCard name={status.name} error={""} revision={status.revision} repoURL={status.repoURL} onSelect={() => {}} />)
    expect(screen.getByLabelText("healthy")).toBeInTheDocument()
  })

  it("shows a degraded status icon when there is an error", () => {
    render(<StatusCard name={status.name} error={"boom"} revision={status.revision} repoURL={status.repoURL} onSelect={() => {}} />)
    expect(screen.getByLabelText("degraded")).toBeInTheDocument()
  })
})
