"use client";

import ErrorComponent from "@/components/error-component";

export default function ErrorPage({ error }: { error: Error }) {
  return (
    <div className="mt-40">
      <ErrorComponent
        link="/"
        linkText="Go Back to Domain Login"
        error={error}
      />
    </div>
  );
}
