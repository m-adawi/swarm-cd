import { render, screen } from "@testing-library/react"
import StatusCardList from "../../src/components/StatusCardList"
import { StackStatus } from "../../src/hooks/useFetchStatuses"

describe("StatusCardList", () => {
  const statuses: StackStatus[] = [
    { Name: "Foobar", Error: "", Revision: "1.0.0", RepoURL: "https://www.url1.com", Ref: "branch:main" },
    { Name: "FooFoo", Error: "", Revision: "2.0.0", RepoURL: "https://www.url2.com", Ref: "tag:v2.0.0" },
    { Name: "Boobaz", Error: "Oh no!!!", Revision: "2.0.0", RepoURL: "https://www.url3.com", Ref: "branch:develop" }
  ]

  it("should render no statuses if the list of statuses is empty", () => {
    render(<StatusCardList statuses={[]} query="" />)
    expect(screen.getByText(/No items/i)).toBeInTheDocument()
  })

  it("should render a list of statuses", () => {
    render(<StatusCardList statuses={statuses} query="" />)
    for (const status of statuses) {
      expect(screen.getByText(status.Name)).toBeInTheDocument()
    }
  })

  it("should filter out the whole list of statuses if query is not found", () => {
    render(<StatusCardList statuses={statuses} query="NOT FOUND!!!" />)
    for(const query of ["Foobar", "FooFoo", "Boobaz"]) {
      expect(screen.queryByText(query)).not.toBeInTheDocument()
    }
    expect(screen.getByText(/No items/i)).toBeInTheDocument()
  })

  it.each([
    { query: "Foo", expectedVisible: ["Foobar", "FooFoo"], expectedHidden: ["Boobaz"] },
    { query: "Foob", expectedVisible: ["Foobar"], expectedHidden: ["FooFoo", "Boobaz"] },
    { query: "oob", expectedVisible: ["Foobar", "Boobaz"], expectedHidden: ["FooFoo"] },
    { query: "2.0.0", expectedVisible: ["FooFoo", "Boobaz"], expectedHidden: ["Foobar"] },
    { query: "https://", expectedVisible: ["Foobar", "FooFoo", "Boobaz"], expectedHidden: [] },
    { query: "Oh no!", expectedVisible: ["Boobaz"], expectedHidden: ["Foobar", "FooFoo"] }
  ])("should filter a list of statuses by query '$query'", ({ query, expectedVisible, expectedHidden }) => {
    render(<StatusCardList statuses={statuses} query={query} />)

    expectedVisible.forEach(name => {
      expect(screen.queryByText(name)).toBeInTheDocument()
    })

    expectedHidden.forEach(name => {
      expect(screen.queryByText(name)).not.toBeInTheDocument()
    })
  })
})
