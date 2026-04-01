import { useRef, useState, type MutableRefObject } from "react";

import { GroupTestersState, TestFlightState } from "../../types";
import { GetTestFlight, GetTestFlightTesters } from "../../../wailsjs/go/main/App";

function emptyTestFlightState(): TestFlightState {
  return { loading: false, groups: [] };
}

export function useTestFlightData(appSelectionRequestRef: MutableRefObject<number>) {
  const [testflightData, setTestflightData] = useState<TestFlightState>(emptyTestFlightState());
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [groupTesters, setGroupTesters] = useState<GroupTestersState>({ loading: false, testers: [] });

  const groupTesterRequestRef = useRef(0);

  function resetGroupSelection() {
    groupTesterRequestRef.current += 1;
    setSelectedGroup(null);
    setGroupTesters({ loading: false, testers: [] });
  }

  function resetSelection() {
    resetGroupSelection();
    setTestflightData(emptyTestFlightState());
  }

  function loadGroupsIfNeeded(appId: string, requestID: number, force = false) {
    if (!force && (testflightData.loading || (testflightData.loadedAppId === appId && !testflightData.error))) {
      return;
    }

    resetGroupSelection();
    setTestflightData({ loading: true, loadedAppId: appId, groups: [] });

    GetTestFlight(appId)
      .then((res) => {
        if (appSelectionRequestRef.current !== requestID) return;
        if (res.error) {
          setTestflightData({ loading: false, loadedAppId: appId, error: res.error, groups: [] });
          return;
        }
        setTestflightData({ loading: false, loadedAppId: appId, groups: res.groups ?? [] });
      })
      .catch((error) => {
        if (appSelectionRequestRef.current !== requestID) return;
        setTestflightData({ loading: false, loadedAppId: appId, error: String(error), groups: [] });
      });
  }

  function handleSelectGroup(groupId: string) {
    const testerRequestID = groupTesterRequestRef.current + 1;
    groupTesterRequestRef.current = testerRequestID;

    setSelectedGroup(groupId);
    setGroupTesters({ loading: true, testers: [] });

    GetTestFlightTesters(groupId)
      .then((res) => {
        if (groupTesterRequestRef.current !== testerRequestID) return;
        if (res.error) {
          setGroupTesters({ loading: false, error: res.error, testers: [] });
          return;
        }
        setGroupTesters({ loading: false, testers: res.testers ?? [] });
      })
      .catch((error) => {
        if (groupTesterRequestRef.current !== testerRequestID) return;
        setGroupTesters({ loading: false, error: String(error), testers: [] });
      });
  }

  function handleBackToGroups() {
    setSelectedGroup(null);
  }

  return {
    testflightData,
    selectedGroup,
    groupTesters,
    resetSelection,
    loadGroupsIfNeeded,
    handleSelectGroup,
    handleBackToGroups,
  };
}
