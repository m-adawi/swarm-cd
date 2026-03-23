import { render, screen, fireEvent } from "@testing-library/react"
import WarningBanner from "../../src/components/WarningBanner"

describe("WarningBanner", () => {
  it("should render nothing when warnings array is empty", () => {
    const { container } = render(<WarningBanner warnings={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it("should render a single warning", () => {
    render(<WarningBanner warnings={["Something is wrong"]} />)
    expect(screen.getByText("Something is wrong")).toBeInTheDocument()
  })

  it("should render multiple warnings", () => {
    const warnings = ["First warning", "Second warning", "Third warning"]
    render(<WarningBanner warnings={warnings} />)

    for (const warning of warnings) {
      expect(screen.getByText(warning)).toBeInTheDocument()
    }
  })

  it("should dismiss a warning when close button is clicked", () => {
    render(<WarningBanner warnings={["Warning A", "Warning B"]} />)

    const closeButtons = screen.getAllByRole("button")
    fireEvent.click(closeButtons[0])

    expect(screen.queryByText("Warning A")).not.toBeInTheDocument()
    expect(screen.getByText("Warning B")).toBeInTheDocument()
  })

  it("should render nothing when all warnings are dismissed", () => {
    const { container } = render(<WarningBanner warnings={["Only warning"]} />)

    const closeButton = screen.getByRole("button")
    fireEvent.click(closeButton)

    expect(container.firstChild).toBeNull()
  })

  it("should reappear when warnings change after dismissal", () => {
    const { rerender } = render(<WarningBanner warnings={["Warning A"]} />)

    fireEvent.click(screen.getByRole("button"))
    expect(screen.queryByText("Warning A")).not.toBeInTheDocument()

    rerender(<WarningBanner warnings={["Warning B"]} />)
    expect(screen.getByText("Warning B")).toBeInTheDocument()
  })
})
