import { DateTime } from "luxon";

export enum Action {
  Created = "created",
  Updated = "updated",
  Confirmed = "confirmed",
}

type Props = {
  time: string;
  action?: Action;
};

export function DisplayTimeCell({ time, action = Action.Created }: Props) {
  const now = DateTime.now();
  const createdAt = DateTime.fromISO(time);

  if (
    createdAt.toString() ===
    DateTime.fromISO("0001-01-01T00:00:00.000Z").toString()
  ) {
    if (action === Action.Updated) {
      return <div>Not updated yet</div>;
    }
    if (action === Action.Confirmed) {
      return <div>Not confirmed yet</div>;
    }
    return <div>Invalid date</div>;
  }

  const diff = now.diff(createdAt, ["hours", "minutes", "seconds"]);

  if (diff.seconds < 1) {
    return <div>Just now</div>;
  }
  if (diff.minutes < 1) {
    return <div>{diff.seconds.toFixed(0)} seconds ago</div>;
  }
  if (diff.hours < 1) {
    if (diff.minutes === 1) {
      return (
        <div>
          {diff.minutes} minute {diff.seconds.toFixed(0)} seconds ago
        </div>
      );
    }
    return (
      <div>
        {diff.minutes} minutes {diff.seconds.toFixed(0)} seconds ago
      </div>
    );
  }
  if (diff.hours < 24) {
    if (diff.hours === 1) {
      return (
        <div>
          {diff.hours} hour {diff.minutes} minutes ago
        </div>
      );
    }
    return (
      <div>
        {diff.hours} hours {diff.minutes} minutes ago
      </div>
    );
  }
  return (
    <div>{createdAt.toLocaleString(DateTime.DATETIME_FULL_WITH_SECONDS)}</div>
  );
}
