import { render, screen } from "@testing-library/react"
import StatusCardList from "../../src/components/StatusCardList"
import { StackStatus } from "../../src/hooks/useFetchStatuses"

describe("StatusCardList", () => {
  const statuses: StackStatus[] = [
    { name: "Foobar", error: "", revision: "1.0.0", repo_url: "https://www.url1.com", ref_type: "branch", ref_value: "main", compose_file: "docker-compose.yml", last_change_at: null, last_deployed_at: null },
    { name: "FooFoo", error: "", revision: "2.0.0", repo_url: "https://www.url2.com", ref_type: "tag", ref_value: "v2.0.0", compose_file: "docker-compose.yml", last_change_at: null, last_deployed_at: null },
    { name: "Boobaz", error: "Oh no!!!", revision: "2.0.0", repo_url: "https://www.url3.com", ref_type: "branch", ref_value: "develop", compose_file: "docker-compose.yml", last_change_at: null, last_deployed_at: null }
  ]

  it("should render no statuses if the list of statuses is empty", () => {
    render(<StatusCardList statuses={[]} query="" />)
    expect(screen.getByText(/No items/i)).toBeInTheDocument()
  })

  it("should render a list of statuses", () => {
    render(<StatusCardList statuses={statuses} query="" />)
    for (const status of statuses) {
      expect(screen.getByText(status.name)).toBeInTheDocument()
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
