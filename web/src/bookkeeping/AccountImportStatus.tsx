import { useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import { checkedStatus, type StreamStatus } from "./connectionStatus";

// Import completion describes acquisition only, never reconciliation or guaranteed history coverage.
function importLabel(status: StreamStatus): string {
  if (status.state === "disconnected") return "Disconnected";
  if (status.state === "paused") return "Paused";
  if (
    status.last_error ||
    status.decoder_error ||
    status.item_state !== "active"
  )
    return "Needs attention";
  if (status.completed_generation !== status.requested_generation)
    return "Importing";
  return status.readiness === "HISTORICAL_UPDATE_COMPLETE"
    ? "Import complete"
    : "Waiting for bank history";
}

// Read-only refresh stops after five reads or a settled/error result; unmount suppresses in-flight replies.
export function AccountImportStatus({
  api,
  entity,
  book,
  item,
  account,
  locked,
  denied,
}: {
  api: API;
  entity: string;
  book: string;
  item: string;
  account: { stream_id: string; source_account_id: string };
  locked: boolean;
  denied: () => void;
}) {
  const [label, setLabel] = useState("Checking import status…");
  const deniedRef = useRef(denied);
  deniedRef.current = denied;
  useEffect(() => {
    let live = true,
      reads = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;
    setLabel("Checking import status…");
    if (locked) return;
    const refresh = async () => {
      reads++;
      try {
        const value = checkedStatus(
          await api<StreamStatus>(
            `/v1/entities/${entity}/books/${book}/connections/streams/${account.stream_id}/status`,
          ),
          { ...account, sync_item_id: item },
        );
        if (!live) return;
        const next = importLabel(value);
        setLabel(next);
        if (
          reads < 5 &&
          ["Importing", "Waiting for bank history"].includes(next)
        )
          timer = setTimeout(() => void refresh(), 3000);
      } catch (failure) {
        if (live) {
          setLabel("Import status unavailable; check sync status");
          if (
            failure instanceof APIError &&
            [403, 404].includes(failure.status)
          )
            deniedRef.current();
        }
      }
    };
    void refresh();
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [
    api,
    entity,
    book,
    item,
    account.stream_id,
    account.source_account_id,
    locked,
  ]);
  return (
    <span className="registration-secondary" role="status">
      {label}
    </span>
  );
}
