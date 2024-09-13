"use client";
import { signOut, useSession } from "next-auth/react";
import type React from "react";
import { useCallback, useEffect } from "react";
import { useIdleTimer } from "react-idle-timer";

export default function SessionMaintainer({
  children,
}: {
  children: React.ReactNode;
}) {
  const { data: session, update } = useSession();
  const checkSessionInterval = 60 * 1000; // 60 seconds
  const updateSessionDiffInterval = 120 * 1000; // 120 seconds
  const updateAccessTokenDiffInterval = 120 * 1000; // 120 seconds
  const userActiveTimeout = 60 * 1000; // 60 seconds
  const baseUrl = process.env.NEXT_PUBLIC_BASE_URL;

  const onUserIdle = () => ({});

  const onUserActive = () => ({});

  const { isIdle } = useIdleTimer({
    onIdle: onUserIdle,
    onActive: onUserActive,
    timeout: userActiveTimeout,
    throttle: 500,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: This is a false positive
  const checkUserSession = useCallback(() => {
    const sessionExpiryTimestamp = Math.floor(
      new Date(session?.expires || "").getTime(),
    );
    const accessTokenExpiryTimestamp = Math.floor(
      new Date(session?.accessTokenExpiry || "").getTime(),
    );
    const currentTimestamp = Date.now();
    const sessionRemainingTime = sessionExpiryTimestamp - currentTimestamp;
    const accessTokenRemainingTime =
      accessTokenExpiryTimestamp - currentTimestamp;
    if (
      isIdle() &&
      (sessionRemainingTime < updateSessionDiffInterval ||
        accessTokenRemainingTime < updateAccessTokenDiffInterval)
    ) {
      update();
    } else if (sessionRemainingTime < 0) {
      signOut({ callbackUrl: `${baseUrl}/login?error=SessionExpired` });
    }
  }, [
    session,
    isIdle,
    update,
    updateSessionDiffInterval,
    updateAccessTokenDiffInterval,
    baseUrl,
  ]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: This is a false positive
  useEffect(() => {
    const intervalId = setInterval(checkUserSession, checkSessionInterval);
    return () => clearInterval(intervalId);
  }, [checkUserSession, checkSessionInterval]);

  return <>{children}</>;
}
