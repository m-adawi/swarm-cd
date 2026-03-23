import { Text } from "@chakra-ui/react"
import React, { useEffect, useState } from "react"
import { StackStatus } from "../hooks/useFetchStatuses"
import StatusCard from "./StatusCard"

function StatusCardList({ statuses, query }: Readonly<{ statuses: StackStatus[]; query: string }>): React.ReactElement {
  const [filteredStatuses, setFilteredStatuses] = useState<StackStatus[]>(statuses)

  useEffect(() => {
    const filtered = statuses.filter(status =>
      Object.values(status).some(value => value != null && value.toString().toLowerCase().includes(query.toLowerCase()))
    )
    setFilteredStatuses(filtered)
  }, [statuses, query])

  return (
    <>
      {filteredStatuses.length === 0 ? (
        <Text fontSize="xl" align="center" mt={4}>
          No items available
        </Text>
      ) : (
        filteredStatuses.map((item, index) => (
          <StatusCard key={index} name={item.name} error={item.error} revision={item.revision} repoURL={item.repo_url} refType={item.ref_type} refValue={item.ref_value} lastChangeAt={item.last_change_at} lastDeployedAt={item.last_deployed_at} />
        ))
      )}
    </>
  )
}

export default StatusCardList
